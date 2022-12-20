/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package recluster

import (
	"context"
	"fmt"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/recluster/gorecluster"
	"k8s.io/autoscaler/cluster-autoscaler/config"
	"k8s.io/klog/v2"
	schedulerframework "k8s.io/kubernetes/pkg/scheduler/framework"
)

// NodeGroup implements cloudprovider.NodeGroup interface. NodeGroup contains
// configuration info and functions to control a set of nodes that have the
// same capacity and set of labels.
type NodeGroup struct {
	client   Client
	nodePool *gorecluster.NodePool
}

// MaxSize returns maximum size of the node group.
func (ng *NodeGroup) MaxSize() int {
	return ng.nodePool.MaxNodes
}

// MinSize returns minimum size of the node group.
func (ng *NodeGroup) MinSize() int {
	return ng.nodePool.MinNodes
}

// TargetSize returns the current target size of the node group. It is possible
// that the number of nodes in Kubernetes is different at the moment but should
// be equal to Size() once everything stabilizes (new nodes finish startup and
// registration or removed nodes are deleted completely).
func (ng *NodeGroup) TargetSize() (int, error) {
	return ng.nodePool.Count, nil
}

// IncreaseSize increases the size of the node group. To delete a node you need
// to explicitly name it and use DeleteNode. This function should wait until
// node group size is updated.
func (ng *NodeGroup) IncreaseSize(delta int) error {
	if delta <= 0 {
		return fmt.Errorf("delta must be positive, have: %d", delta)
	}

	currentSize, err := ng.TargetSize()
	if err != nil {
		return fmt.Errorf("failed to get node group target size")
	}

	targetSize := currentSize + delta
	if targetSize > ng.MaxSize() {
		return fmt.Errorf("size increase is too large, current: %d, desired: %d, max: %d",
			currentSize, targetSize, ng.MaxSize())
	}

	ctx := context.Background()
	opts := &gorecluster.UpdateNodePoolOpts{Count: uint(targetSize)}
	nodePool, err := ng.client.UpdateNodePool(ctx, ng.nodePool.ID, opts)
	if err != nil {
		return err
	}

	if nodePool.Count != targetSize {
		return fmt.Errorf("couldn't increase size to %d (delta: %d). Current size is: %d",
			targetSize, delta, nodePool.Count)
	}

	ng.nodePool.Count = targetSize
	return nil
}

// DeleteNodes deletes nodes from this node group (and also increasing the size
// of the node group with that). Error is returned either on failure or if the
// given node doesn't belong to this node group. This function should wait
// until node group size is updated.
func (ng *NodeGroup) DeleteNodes(nodes []*apiv1.Node) error {
	currentSize, err := ng.TargetSize()
	if err != nil {
		return fmt.Errorf("failed to get node group target size")
	}

	targetSize := currentSize - len(nodes)
	if targetSize < ng.MinSize() {
		return fmt.Errorf("node group size is below minimum, desired: %d, min: %d", targetSize, ng.MinSize())
	}

	klog.V(4).Infof("Deleting %d nodes", len(nodes))

	ctx := context.Background()
	for _, node := range nodes {
		nodeID, err := toNodeID(node)
		if err != nil {
			// CA creates fake node objects to represent upcoming Nodes that haven't registered as nodes yet.
			// We cannot delete the node at this point.
			return fmt.Errorf("cannot delete node %s with provider ID %s on NodeGroup %s: %w", node.Name, node.Spec.ProviderID, ng.nodePool.ID, err)
		}

		klog.V(4).Infof("Deleting node %s", nodeID)

		opts := &gorecluster.DeleteNodePoolNodeOpts{}
		_, err = ng.client.DeleteNodePoolNode(ctx, ng.nodePool.ID, nodeID, opts)
		if err != nil {
			return fmt.Errorf("failed to delete node %s from node pool %s: %w",
				nodeID, ng.nodePool.ID, err)
		}

		ng.nodePool.Count--
	}

	return nil
}

// DecreaseTargetSize decreases the target size of the node group. This function
// doesn't permit to delete any existing node and can be used only to reduce the
// request for new nodes that have not been yet fulfilled. Delta should be negative.
// It is assumed that cloud provider will not delete the existing nodes when there
// is an option to just decrease the target.
func (ng *NodeGroup) DecreaseTargetSize(delta int) error {
	if delta >= 0 {
		return fmt.Errorf("delta must be negative, have: %d", delta)
	}

	currentSize, err := ng.TargetSize()
	if err != nil {
		return fmt.Errorf("failed to get node group target size")
	}

	targetSize := currentSize + delta
	if targetSize < ng.MinSize() {
		return fmt.Errorf("size decrease is too small, current: %d, desired: %d, min: %d",
			currentSize, targetSize, ng.MinSize())
	}

	ctx := context.Background()
	opts := &gorecluster.UpdateNodePoolOpts{Count: uint(targetSize)}
	nodePool, err := ng.client.UpdateNodePool(ctx, ng.nodePool.ID, opts)
	if err != nil {
		return err
	}

	if nodePool.Count != targetSize {
		return fmt.Errorf("couldn't decrease size to %d (delta: %d). Current size is: %d",
			targetSize, delta, nodePool.Count)
	}

	ng.nodePool.Count = targetSize
	return nil
}

// Id returns an unique identifier of the node group.
func (ng *NodeGroup) Id() string {
	return ng.nodePool.ID
}

// Debug returns a string containing all information regarding this node group.
func (ng *NodeGroup) Debug() string {
	return fmt.Sprintf("node group %s (min: %d, max: %d, current: %d)", ng.Id(), ng.MinSize(), ng.MaxSize(), len(ng.nodePool.Nodes))
}

// Nodes returns a list of all nodes that belong to this node group.  It is
// required that Instance objects returned by this method have ID field set.
// Other fields are optional.
func (ng *NodeGroup) Nodes() ([]cloudprovider.Instance, error) {
	if ng.nodePool == nil {
		return nil, fmt.Errorf("node pool instance is not created")
	}

	instances := make([]cloudprovider.Instance, 0, len(ng.nodePool.Nodes))
	for _, node := range ng.nodePool.Nodes {
		instance := cloudprovider.Instance{
			Id:     node.ID,
			Status: toInstanceStatus(&node.Status),
		}

		instances = append(instances, instance)
	}

	return instances, nil
}

// TemplateNodeInfo returns a schedulerframework.NodeInfo structure of an empty
// (as if just started) node. This will be used in scale-up simulations to
// predict what would a new node look like if a node group was expanded. The
// returned NodeInfo is expected to have a fully populated Node object, with
// all the labels, capacity and allocatable information as well as all pods
// that are started on the node by default, using manifest (most likely only
// kube-proxy). Implementation optional.
func (ng *NodeGroup) TemplateNodeInfo() (*schedulerframework.NodeInfo, error) {
	return nil, cloudprovider.ErrNotImplemented
}

// Exist checks if the node group really exists on the cloud provider side.
// Allows to tell the theoretical node group from the real one.
func (ng *NodeGroup) Exist() bool {
	return ng.nodePool != nil
}

// Create creates the node group on the cloud provider side.
func (ng *NodeGroup) Create() (cloudprovider.NodeGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

// Delete deletes the node group on the cloud provider side.  This will be
// executed only for autoprovisioned node groups, once their size drops to 0.
func (ng *NodeGroup) Delete() error {
	return cloudprovider.ErrNotImplemented
}

// Autoprovisioned returns true if the node group is autoprovisioned. An
// autoprovisioned group was created by CA and can be deleted when scaled to 0.
func (ng *NodeGroup) Autoprovisioned() bool {
	return false
}

// GetOptions returns NodeGroupAutoscalingOptions that should be used for this particular
// NodeGroup. Returning a nil will result in using default options.
func (ng *NodeGroup) GetOptions(_ config.NodeGroupAutoscalingOptions) (*config.NodeGroupAutoscalingOptions, error) {
	return nil, cloudprovider.ErrNotImplemented
}

// toInstanceStatus converts the given *gorecluster.Status to a cloudprovider.InstanceStatus
func toInstanceStatus(status *gorecluster.Status) *cloudprovider.InstanceStatus {
	if status == nil {
		return nil
	}

	instanceStatus := &cloudprovider.InstanceStatus{}
	switch status.Status {
	case "BOOTING", "ACTIVE":
		instanceStatus.State = cloudprovider.InstanceCreating
	case "ACTIVE_READY", "ACTIVE_NOT_READY":
		instanceStatus.State = cloudprovider.InstanceRunning
	case "ACTIVE_DELETING":
		instanceStatus.State = cloudprovider.InstanceDeleting
	default:
		instanceStatus.ErrorInfo = &cloudprovider.InstanceErrorInfo{
			ErrorClass:   cloudprovider.OtherErrorClass,
			ErrorCode:    "NoCodeReCluster",
			ErrorMessage: status.Message,
		}
	}

	return instanceStatus
}
