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
	"fmt"
	"io"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	"k8s.io/autoscaler/cluster-autoscaler/config"
	"k8s.io/autoscaler/cluster-autoscaler/utils/errors"
	"k8s.io/klog/v2"
	"os"
)

const (
	nodeIDLabel = "recluster.io/id"
)

// reclusterCloudProvider implements CloudProvider interface.
type reclusterCloudProvider struct {
	manager         *Manager
	resourceLimiter *cloudprovider.ResourceLimiter
}

// BuildRecluster builds the reCluster cloud provider.
func BuildRecluster(opts config.AutoscalingOptions, _ cloudprovider.NodeGroupDiscoveryOptions, rl *cloudprovider.ResourceLimiter) cloudprovider.CloudProvider {
	var configFile io.ReadCloser
	if opts.CloudConfig != "" {
		var err error

		configFile, err = os.Open(opts.CloudConfig)
		if err != nil {
			klog.Fatalf("Could not open cloud provider configuration file %s: %w", opts.CloudConfig, err)
		}

		defer configFile.Close()
	}

	manager, err := NewManager(configFile)
	if err != nil {
		klog.Fatalf("Failed to create reCluster manager: %w", err)
	}

	provider := &reclusterCloudProvider{
		manager:         manager,
		resourceLimiter: rl,
	}

	return provider
}

// Name returns name of the cloud provider.
func (r *reclusterCloudProvider) Name() string {
	return cloudprovider.ReClusterProviderName
}

// NodeGroups returns all node groups configured for this cloud provider.
func (r *reclusterCloudProvider) NodeGroups() []cloudprovider.NodeGroup {
	nodeGroups := make([]cloudprovider.NodeGroup, len(r.manager.nodeGroups))
	for i, nodeGroup := range r.manager.nodeGroups {
		nodeGroups[i] = nodeGroup
	}
	return nodeGroups
}

// NodeGroupForNode returns the node group for the given node, nil if the node
// should not be processed by cluster autoscaler, or non-nil error if such occurred.
func (r *reclusterCloudProvider) NodeGroupForNode(node *apiv1.Node) (cloudprovider.NodeGroup, error) {
	// FIXME Should use node.Spec.ProviderID
	nodeID, err := toNodeID(node)
	if err != nil {
		return nil, err
	}

	klog.V(4).Infof("Searching node group for node %s", nodeID)

	for _, nodeGroup := range r.manager.nodeGroups {
		klog.V(4).Infof("Iterating node group %s", nodeGroup.Id())

		nodes, err := nodeGroup.Nodes()
		if err != nil {
			return nil, err
		}

		for _, node := range nodes {
			klog.V(5).Infof("Checking node %s for ID %s", node.Id, nodeID)

			if node.Id == nodeID {
				klog.V(5).Infof("Node %s assigned to node group %s", node.Id, nodeGroup.Id())
				return nodeGroup, nil
			}
		}
	}

	klog.Warningf("Unable to find node group for node %s", nodeID)

	return nil, nil
}

// HasInstance returns whether a given node has a corresponding instance in this cloud provider
func (r *reclusterCloudProvider) HasInstance(node *apiv1.Node) (bool, error) {
	return true, cloudprovider.ErrNotImplemented
}

// Pricing returns pricing model for this cloud provider or error if not available.
func (r *reclusterCloudProvider) Pricing() (cloudprovider.PricingModel, errors.AutoscalerError) {
	return nil, cloudprovider.ErrNotImplemented
}

// GetAvailableMachineTypes get all machine types that can be requested from the cloud provider.
func (r *reclusterCloudProvider) GetAvailableMachineTypes() ([]string, error) {
	return []string{}, cloudprovider.ErrNotImplemented
}

// NewNodeGroup builds a theoretical node group based on the node definition
// provided. The node group is not automatically created on the cloud provider
// side. The node group is not returned by NodeGroups() until it is created.
func (r *reclusterCloudProvider) NewNodeGroup(machineType string, labels map[string]string, systemLabels map[string]string, taints []apiv1.Taint, extraResources map[string]resource.Quantity) (cloudprovider.NodeGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

// GetResourceLimiter returns struct containing limits (max, min) for resources (cores, memory etc.).
func (r *reclusterCloudProvider) GetResourceLimiter() (*cloudprovider.ResourceLimiter, error) {
	return r.resourceLimiter, nil
}

// GPULabel returns the label added to nodes with GPU resource.
func (r *reclusterCloudProvider) GPULabel() string {
	return ""
}

// GetAvailableGPUTypes return all available GPU types cloud provider supports.
func (r *reclusterCloudProvider) GetAvailableGPUTypes() map[string]struct{} {
	return nil
}

// Cleanup cleans up open resources before the cloud provider is destroyed, i.e. go routines etc.
func (r *reclusterCloudProvider) Cleanup() error {
	return nil
}

// Refresh is called before every main loop and can be used to dynamically update cloud provider state.
// In particular the list of node groups returned by NodeGroups can change as a result of CloudProvider.Refresh().
func (r *reclusterCloudProvider) Refresh() error {
	klog.V(4).Info("Refreshing node group cache")
	return r.manager.Refresh()
}

// toNodeID returns the node ID of the given node.
func toNodeID(node *apiv1.Node) (string, error) {
	nodeID, ok := node.Labels[nodeIDLabel]
	if !ok {
		// Check fake node
		_, ok = node.Labels["k8s.io/cluster-autoscaler/fake-node-reason"]
		if ok {
			// Node ID is ProviderID
			return node.Spec.ProviderID, nil
		}

		return "", fmt.Errorf("node ID label %s is missing from node %s with provider ID %s", nodeIDLabel, node.Name, node.Spec.ProviderID)
	}

	return nodeID, nil
}
