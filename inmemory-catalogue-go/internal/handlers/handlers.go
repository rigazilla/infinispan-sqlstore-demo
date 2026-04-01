package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/infinispan/inmemory-catalogue-go/internal/client"
	"github.com/infinispan/inmemory-catalogue-go/internal/config"
)

// Handler manages HTTP handlers for the inmemory catalog API
type Handler struct {
	client *client.InfinispanClient
	config *config.Config
}

// New creates a new Handler
func New(c *client.InfinispanClient, cfg *config.Config) *Handler {
	return &Handler{
		client: c,
		config: cfg,
	}
}

// Health returns the health status with cache sizes
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	catalogSize := 0
	soldProductsSize := 0

	// Get catalogue cache size
	if size, err := h.client.GetCacheSize(h.config.CatalogCacheName); err == nil {
		catalogSize = size
	}

	// Get sold products cache size
	if size, err := h.client.GetCacheSize(h.config.SoldProductsCacheName); err == nil {
		soldProductsSize = size
	}

	response := fmt.Sprintf("Service is up! catalogue[%d] sold_products[%d]", catalogSize, soldProductsSize)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(response))
}

// Reindex triggers reindexing of both caches
func (h *Handler) Reindex(w http.ResponseWriter, r *http.Request) {
	if err := h.client.Reindex(h.config.CatalogCacheName); err != nil {
		http.Error(w, fmt.Sprintf("Failed to reindex catalogue: %v", err), http.StatusInternalServerError)
		return
	}

	if err := h.client.Reindex(h.config.SoldProductsCacheName); err != nil {
		http.Error(w, fmt.Sprintf("Failed to reindex sold products: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Reindex launched"))
}

// GetCatalogue returns products with optional filters
func (h *Handler) GetCatalogue(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	stock := r.URL.Query().Get("stock")
	priceMin := r.URL.Query().Get("price-min")
	priceMax := r.URL.Query().Get("price-max")

	// Build Ickle query
	query := "from retail.RetailProductValue"
	conditions := []string{}

	if name != "" {
		// Full-text search using REST API v3 syntax: name: (+'term')
		conditions = append(conditions, fmt.Sprintf("name: (+'%s')", name))
	}

	if stock != "" {
		conditions = append(conditions, fmt.Sprintf("stock >= %s", stock))
	}

	if priceMin != "" && priceMax != "" {
		conditions = append(conditions, fmt.Sprintf("price : [%s to %s]", priceMin, priceMax))
	}

	if len(conditions) > 0 {
		query += " where " + strings.Join(conditions, " and ")
	}

	// Execute query
	resp, err := h.client.Query(h.config.CatalogCacheName, query, 10000)
	if err != nil {
		http.Error(w, fmt.Sprintf("Query failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Extract hits
	results := make([]map[string]interface{}, 0, len(resp.Hits))
	for _, hit := range resp.Hits {
		results = append(results, hit.Hit)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// GetCatalogueByCode returns a single product by code
func (h *Handler) GetCatalogueByCode(w http.ResponseWriter, r *http.Request) {
	// Extract code from URL path
	path := strings.TrimPrefix(r.URL.Path, "/catalogue/")
	code := strings.TrimSpace(path)

	if code == "" {
		http.Error(w, "Product code is required", http.StatusBadRequest)
		return
	}

	// Build Ickle query
	query := fmt.Sprintf("from retail.RetailProductValue where code = '%s'", code)

	// Execute query
	resp, err := h.client.Query(h.config.CatalogCacheName, query, 1)
	if err != nil {
		http.Error(w, fmt.Sprintf("Query failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if product found
	if len(resp.Hits) == 0 {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp.Hits[0].Hit)
}

// GetSales returns sold products with optional filters
func (h *Handler) GetSales(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	country := r.URL.Query().Get("country")

	// Build Ickle query
	query := "from retail.PurchasedProductValue"
	conditions := []string{}

	if name != "" {
		// Full-text search using REST API v3 syntax
		conditions = append(conditions, fmt.Sprintf("name: (+'%s')", name))
	}

	if country != "" {
		// Full-text search for country
		conditions = append(conditions, fmt.Sprintf("country: (+'%s')", country))
	}

	if len(conditions) > 0 {
		query += " where " + strings.Join(conditions, " and ")
	}

	// Execute query
	resp, err := h.client.Query(h.config.SoldProductsCacheName, query, 10000)
	if err != nil {
		http.Error(w, fmt.Sprintf("Query failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Extract hits
	results := make([]map[string]interface{}, 0, len(resp.Hits))
	for _, hit := range resp.Hits {
		results = append(results, hit.Hit)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
