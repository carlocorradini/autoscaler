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
	"encoding/json"
	"fmt"
	"io"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/recluster/gorecluster"
	"k8s.io/klog/v2"
)

// Client defines all mandatory methods to be exposed as a client (mock or API).
type Client interface {
	// ListNodePools lists all the node pools.
	ListNodePools(ctx context.Context, opts *gorecluster.ListNodePoolOpts) ([]gorecluster.NodePool, error)

	// UpdateNodePool updates a specific node pool.
	UpdateNodePool(ctx context.Context, ID string, opts *gorecluster.UpdateNodePoolOpts) (*gorecluster.NodePool, error)

	// DeleteNodePoolNode deletes a specific node from a node pool.
	DeleteNodePoolNode(ctx context.Context, ID string, nodeID string, opts *gorecluster.DeleteNodePoolNodeOpts) (*gorecluster.Node, error)
}

// Manager handles reCluster communication and holds information about the node groups.
type Manager struct {
	client     Client
	nodeGroups []*NodeGroup
}

// Config is the configuration reCluster cloud provider.
type Config struct {
	// Token is the access Token associated with the cluster.
	Token string `json:"token"`

	// URL points to reCluster API.
	URL string `json:"url"`
}

// NewManager initializes an API client given a cloud provider configuration file.
func NewManager(configReader io.Reader) (*Manager, error) {
	config := &Config{}

	// Read configuration
	if configReader != nil {
		body, err := io.ReadAll(configReader)
		if err != nil {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}

		err = json.Unmarshal(body, config)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	// Validate configuration
	if config.Token == "" {
		return nil, fmt.Errorf("configuration 'token' is not provided")
	}
	if config.URL == "" {
		return nil, fmt.Errorf("configuration 'URL' is not provided")
	}

	// Create API client
	client, err := gorecluster.NewClient(config.URL, config.Token)

	if err != nil {
		return nil, fmt.Errorf("failed to initialize API client: %w", err)
	}

	manager := &Manager{
		client:     client,
		nodeGroups: make([]*NodeGroup, 0),
	}

	return manager, nil
}

// Refresh refreshes the cache holding the node groups.
func (m *Manager) Refresh() error {
	ctx := context.Background()

	nodePools, err := m.client.ListNodePools(ctx, nil)
	if err != nil {
		return err
	}

	var nodeGroups []*NodeGroup
	for _, nodePool := range nodePools {
		if !nodePool.AutoScale {
			continue
		}

		klog.V(4).Infof("Adding node pool %q, name: %s, min: %d, max: %d",
			nodePool.ID, nodePool.Name, nodePool.MinNodes, nodePool.MaxNodes)

		nodeGroups = append(nodeGroups, &NodeGroup{
			id:       nodePool.ID,
			client:   m.client,
			nodePool: &nodePool,
			minSize:  nodePool.MinNodes,
			maxSize:  nodePool.MaxNodes,
		})
	}

	if len(nodeGroups) == 0 {
		klog.V(4).Info("Cluster-Autoscaler is disabled, no node pools are configured")
	}

	m.nodeGroups = nodeGroups
	return nil
}
