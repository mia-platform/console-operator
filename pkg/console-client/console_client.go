// Copyright 2016-2025 Mia-Platform
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package consoleclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	corev1alpha1 "github.com/mia-platform/console-operator/api/core/v1alpha1"
)

func NewClient(config ClientConfig, ctx context.Context) (*Client, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	if config.ClientID == "" {
		return nil, fmt.Errorf("client ID is required")
	}
	if config.ClientSecret == "" {
		return nil, fmt.Errorf("client secret is required")
	}

	if _, err := url.Parse(config.BaseURL); err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		baseURL:      config.BaseURL,
		clientID:     config.ClientID,
		clientSecret: config.ClientSecret,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		ctx: ctx,
	}, nil
}

// GetToken obtains a bearer token using client credentials flow
func (c *Client) GetToken(ctx context.Context) error {
	// Check if token is still valid
	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		return nil
	}

	// Prepare form data for client credentials grant
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	// Create token request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/m2m/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}

	// Set basic auth for client credentials
	req.SetBasicAuth(c.clientID, c.clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// Execute token request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	// Parse token response
	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	// Store token and calculate expiry
	c.token = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return nil
}

func (c *Client) Get(ctx context.Context, endpoint string) (*http.Response, error) {
	return c.doRequest(ctx, http.MethodGet, endpoint, nil)
}

func (c *Client) Post(ctx context.Context, endpoint string, body interface{}) (*http.Response, error) {
	return c.doRequest(ctx, http.MethodPost, endpoint, body)
}

func (c *Client) Put(ctx context.Context, endpoint string, body interface{}) (*http.Response, error) {
	return c.doRequest(ctx, http.MethodPut, endpoint, body)
}

func (c *Client) Delete(ctx context.Context, endpoint string) (*http.Response, error) {
	return c.doRequest(ctx, http.MethodDelete, endpoint, nil)
}

func (c *Client) doRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	// Ensure we have a valid token before making the request
	if err := c.GetToken(ctx); err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	fullURL := c.baseURL + endpoint

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add Bearer token authentication
	req.Header.Set("Authorization", "Bearer "+c.token)

	// Set content type for requests with body
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Set accept header
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

// GetJSON is a convenience method that performs GET and unmarshals JSON response
func (c *Client) GetJSON(ctx context.Context, endpoint string, result interface{}) error {
	resp, err := c.Get(ctx, endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func (c *Client) CompanyExists(ctx context.Context, companyName string) (bool, string, error) {
	var companies []ConsoleCompany

	err := c.GetJSON(ctx, "/api/backend/tenants/", &companies)
	if err != nil {
		return false, "", fmt.Errorf("failed to get companies: %w", err)
	}

	for _, company := range companies {
		if company.Name == companyName {
			return true, company.CompanyId, nil
		}
	}

	return false, "", nil
}

func (c *Client) CreateCompany(ctx context.Context, company ConsoleCompany) (string, error) {
	var result struct {
		ID        string `json:"_id"`
		CompanyId string `json:"tenantId"`
	}
	c.PostJSON(ctx, "/api/backend/tenants", company, &result)
	if result.CompanyId != "" {
		return result.CompanyId, nil
	}
	return "", fmt.Errorf("failed to create company")
}

func (c *Client) AddCompanyOwners(ctx context.Context, company corev1alpha1.Company, companyId string) []error {
	var companyOwners = company.Spec.CompanyOwners
	var log = log.Default()
	var errors []error
	for _, ownerEmail := range companyOwners {
		addUserURL := fmt.Sprintf("/api/companies/%s/users", companyId)
		log.Printf("Adding user %s to company %s. Calling URL %s", ownerEmail, companyId, addUserURL)
		if err := c.PostJSON(ctx, addUserURL, map[string]string{"email": ownerEmail, "role": "company-owner"}, nil); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (c *Client) AddCompanyCluster(ctx context.Context, cluster corev1alpha1.Cluster, companyId string, serviceAccountToken string) (string, error) {

	var log = log.Default()
	addClusterURL := fmt.Sprintf("/api/tenants/%s/clusters/", companyId)
	var clusterPayload struct {
		ClusterId  string `json:"clusterId"`
		Connection struct {
			Url                 string `json:"url"`
			Base64CA            string `json:"base64CA"`
			ServiceAccountToken string `json:"serviceAccountToken"`
		} `json:"connection"`
		Description string `json:"description,omitempty"`
	}
	clusterPayload.ClusterId = cluster.ClusterId
	clusterPayload.Description = cluster.Description
	clusterPayload.Connection.Base64CA = cluster.Connection.Base64CA
	clusterPayload.Connection.Url = cluster.Connection.Url
	clusterPayload.Connection.ServiceAccountToken = serviceAccountToken

	log.Printf("Adding cluster %s to company %s. Calling URL %s", cluster.ClusterId, companyId, addClusterURL)
	log.Printf("Payload %s", clusterPayload)

	var result struct {
		ClusterId string `json:"_id"`
	}
	if err := c.PostJSON(ctx, addClusterURL, clusterPayload, &result); err != nil {
		log.Printf("Error %s", err)
		return "", err
	}

	return result.ClusterId, nil
}

func (c *Client) AddCompanyEnvironments(ctx context.Context, environments []corev1alpha1.Environment, companyId string, clusters map[string]string) error {
	var log = log.Default()
	var addEnvPath = fmt.Sprintf("/tenants/%s/project-blueprint/environments", companyId)
	var environmentsPayload []corev1alpha1.Environment = []corev1alpha1.Environment{}
	for _, env := range environments {
		clusterId, exists := clusters[env.Cluster.ClusterId]
		if exists && clusterId != "" {
			env.Cluster.ClusterId = clusterId
			environmentsPayload = append(environmentsPayload, env)
		} else {
			log.Fatal("ClusterId not found for environment ", env.EnvId, " with cluster ", env.Cluster.ClusterId)
		}
	}
	log.Printf("Adding environments to company %s. Calling URL %s", companyId, addEnvPath)
	log.Printf("Payload %+v", environmentsPayload)
	return c.PostJSON(ctx, addEnvPath, environmentsPayload, nil)
}

// PostJSON is a convenience method that performs POST and unmarshals JSON response
func (c *Client) PostJSON(ctx context.Context, endpoint string, body interface{}, result interface{}) error {
	resp, err := c.Post(ctx, endpoint, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}

	return nil
}
