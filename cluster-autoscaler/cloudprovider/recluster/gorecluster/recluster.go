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
	"github.com/Khan/genqlient/graphql"
	"net/http"
	"net/url"
	"time"
)

const (
	// GraphQLClientTimeout represents GraphQL client timeout in seconds.
	GraphQLClientTimeout = 60 * time.Second
	// GraphQLClientUserAgent represents GraphQL client user agent.
	GraphQLClientUserAgent = "kubernetes/cluster-autoscaler"
)

// Client represents a client to call the reCluster API.
type Client struct {
	// graphQLClient is the underlying GraphQL client used to run the requests.
	graphQLClient graphql.Client
}

// ReclusterTransport transport.
type reclusterTransport struct {
	// rt is the HTTP round tripper.
	rt http.RoundTripper
	// token is the auth token.
	token string
}

// NewClient represents a new client to call the API.
func NewClient(URL string, token string) (*Client, error) {
	baseURL, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}

	client := Client{
		graphQLClient: graphql.NewClient(baseURL.String(), &http.Client{
			Timeout:   GraphQLClientTimeout,
			Transport: &reclusterTransport{rt: http.DefaultTransport, token: token},
		}),
	}

	return &client, nil
}

// RoundTrip executes a single HTTP transaction, returning
// a Response for the provided Request.
func (t *reclusterTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", GraphQLClientUserAgent)
	req.Header.Set("Authorization", "Bearer "+t.token)
	return t.RoundTrip(req)
}
