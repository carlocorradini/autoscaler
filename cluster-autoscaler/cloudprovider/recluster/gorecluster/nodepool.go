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

package gorecluster

import (
	"context"
)

// ListNodePoolOpts defines fields to list node pools.
type ListNodePoolOpts struct{}

// UpdateNodePoolOpts defines fields to update a node pool.
type UpdateNodePoolOpts = UpdateNodePoolInput

// DeleteNodePoolNodeOpts defines fields to delete Node from NodePool
type DeleteNodePoolNodeOpts struct{}

// ListNodePools lists all the node pools.
func (c *Client) ListNodePools(ctx context.Context, _ *ListNodePoolOpts) ([]NodePool, error) {
	resp, err := NodePools(ctx, c.graphQLClient)
	if err != nil {
		return nil, err
	}

	return resp.NodePools, nil
}

// UpdateNodePool updates a specific node pool.
func (c *Client) UpdateNodePool(ctx context.Context, ID string, opts *UpdateNodePoolOpts) (*NodePool, error) {
	resp, err := UpdateNodePool(ctx, c.graphQLClient, ID, *opts)
	if err != nil {
		return nil, err
	}

	return &resp.UpdateNodePool, nil
}

// DeleteNodePoolNode deletes a specific node in a node pool.
func (c *Client) DeleteNodePoolNode(ctx context.Context, ID string, nodeID string, _ *DeleteNodePoolNodeOpts) (*Node, error) {
	resp, err := DeleteNodePoolNode(ctx, c.graphQLClient, ID, nodeID)
	if err != nil {
		return nil, err
	}

	return &resp.DeleteNodePoolNode, nil
}
