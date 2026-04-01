package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/icholy/digest"
)

// InfinispanClient provides methods to interact with Infinispan REST API
type InfinispanClient struct {
	baseURL    string
	httpClient *http.Client
	endpoints  *Endpoints
}

// Endpoints holds discovered API endpoints from OpenAPI schema
type Endpoints struct {
	PostCache             string
	CacheExists           string
	GetCacheSize          string
	Reindex               string
	PostQueryCache        string
	ContainerHealthStatus string
}

// QueryRequest represents the JSON body for query POST requests
type QueryRequest struct {
	Query            string `json:"query"`
	MaxResults       int    `json:"max_results,omitempty"`
	StartOffset      int    `json:"start_offset,omitempty"`
	HitCountAccuracy int    `json:"hit_count_accuracy,omitempty"`
}

// QueryResponse represents the response from a query
type QueryResponse struct {
	HitCount      int           `json:"hit_count"`
	HitCountExact bool          `json:"hit_count_exact"`
	Hits          []QueryHit    `json:"hits"`
}

// QueryHit represents a single hit in the query response
type QueryHit struct {
	Hit map[string]interface{} `json:"hit"`
}

// NewInfinispanClient creates a new Infinispan client with digest authentication
func NewInfinispanClient(baseURL, username, password string) (*InfinispanClient, error) {
	// Create HTTP client with digest transport
	client := &http.Client{
		Transport: &digest.Transport{
			Username: username,
			Password: password,
		},
	}

	ic := &InfinispanClient{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		httpClient: client,
	}

	// Discover endpoints from OpenAPI schema
	if err := ic.discoverEndpoints(); err != nil {
		return nil, fmt.Errorf("failed to discover endpoints: %w", err)
	}

	return ic, nil
}

// discoverEndpoints fetches the OpenAPI schema and extracts endpoint patterns
func (c *InfinispanClient) discoverEndpoints() error {
	url := c.baseURL + "/rest/v3/openapi"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch OpenAPI schema: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OpenAPI schema returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Parse OpenAPI schema
	var schema map[string]interface{}
	if err := json.Unmarshal(body, &schema); err != nil {
		return fmt.Errorf("failed to parse OpenAPI schema: %w", err)
	}

	// Extract endpoints based on operationId
	c.endpoints = &Endpoints{
		PostCache:             "/rest/v3/caches/{cacheName}",
		CacheExists:           "/rest/v3/caches/{cacheName}",
		GetCacheSize:          "/rest/v3/caches/{cacheName}?action=size",
		Reindex:               "/rest/v3/caches/{cacheName}/_reindex",
		PostQueryCache:        "/rest/v3/caches/{cacheName}/_search",
		ContainerHealthStatus: "/rest/v3/container/health/status",
	}

	return nil
}

// CheckHealth checks if Infinispan is healthy
func (c *InfinispanClient) CheckHealth() (string, error) {
	url := c.baseURL + c.endpoints.ContainerHealthStatus

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// CacheExists checks if a cache exists
func (c *InfinispanClient) CacheExists(cacheName string) (bool, error) {
	url := c.baseURL + strings.Replace(c.endpoints.CacheExists, "{cacheName}", cacheName, 1)

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent, nil
}

// CreateCache creates a cache with the given XML configuration
func (c *InfinispanClient) CreateCache(cacheName, xmlConfig string) error {
	exists, err := c.CacheExists(cacheName)
	if err != nil {
		return fmt.Errorf("failed to check cache existence: %w", err)
	}

	if exists {
		fmt.Printf("Cache %s already exists, skipping creation\n", cacheName)
		return nil
	}

	url := c.baseURL + strings.Replace(c.endpoints.PostCache, "{cacheName}", cacheName, 1)

	req, err := http.NewRequest("POST", url, bytes.NewBufferString(xmlConfig))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/xml")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		// Check if it's a race condition (cache already exists)
		if strings.Contains(string(body), "already exists") {
			fmt.Printf("Cache %s was created by another process, continuing\n", cacheName)
			return nil
		}
		return fmt.Errorf("failed to create cache, status: %d, body: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Cache %s created successfully\n", cacheName)
	return nil
}

// GetCacheSize returns the number of entries in a cache
func (c *InfinispanClient) GetCacheSize(cacheName string) (int, error) {
	url := c.baseURL + strings.Replace(c.endpoints.GetCacheSize, "{cacheName}", cacheName, 1)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to get cache size, status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var size int
	if err := json.Unmarshal(body, &size); err != nil {
		return 0, fmt.Errorf("failed to parse cache size: %w", err)
	}

	return size, nil
}

// Reindex triggers reindexing of a cache
func (c *InfinispanClient) Reindex(cacheName string) error {
	url := c.baseURL + strings.Replace(c.endpoints.Reindex, "{cacheName}", cacheName, 1)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to reindex cache, status: %d, body: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Cache %s reindexed successfully\n", cacheName)
	return nil
}

// Query executes an Ickle query on a cache using POST method
func (c *InfinispanClient) Query(cacheName, ickleQuery string, maxResults int) (*QueryResponse, error) {
	url := c.baseURL + strings.Replace(c.endpoints.PostQueryCache, "{cacheName}", cacheName, 1)

	if maxResults == 0 {
		maxResults = 10000 // Default safe limit
	}

	queryReq := QueryRequest{
		Query:      ickleQuery,
		MaxResults: maxResults,
	}

	jsonBody, err := json.Marshal(queryReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("query failed, status: %d, body: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var queryResp QueryResponse
	if err := json.Unmarshal(body, &queryResp); err != nil {
		return nil, fmt.Errorf("failed to parse query response: %w", err)
	}

	return &queryResp, nil
}
