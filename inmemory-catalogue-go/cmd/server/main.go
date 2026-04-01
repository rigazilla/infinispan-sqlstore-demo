package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/infinispan/inmemory-catalogue-go/internal/client"
	"github.com/infinispan/inmemory-catalogue-go/internal/config"
	"github.com/infinispan/inmemory-catalogue-go/internal/handlers"
)

func main() {
	fmt.Println("Infinispan SQL Store - Go Implementation")
	fmt.Println("  _   _   _   _   _   _   _   _")
	fmt.Println(" / \\ / \\ / \\ / \\ / \\ / \\ / \\ / \\")
	fmt.Println("( S | q | l | S | t | o | r | e )")
	fmt.Println(" \\_/ \\_/ \\_/ \\_/ \\_/ \\_/ \\_/ \\_/")
	fmt.Println()

	// Load configuration
	cfg := config.Load()
	fmt.Printf("Connecting to Infinispan at %s\n", cfg.InfinispanURL)

	// Create Infinispan client
	infinispanClient, err := client.NewInfinispanClient(cfg.InfinispanURL, cfg.Username, cfg.Password)
	if err != nil {
		log.Fatalf("Failed to create Infinispan client: %v", err)
	}

	// Check Infinispan health
	health, err := infinispanClient.CheckHealth()
	if err != nil {
		log.Fatalf("Infinispan health check failed: %v", err)
	}
	fmt.Printf("Infinispan health: %s\n", health)

	// Initialize caches
	if err := initializeCaches(infinispanClient, cfg); err != nil {
		log.Fatalf("Failed to initialize caches: %v", err)
	}

	// Create HTTP handlers
	h := handlers.New(infinispanClient, cfg)

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/reindex", h.Reindex)
	mux.HandleFunc("/catalogue/", h.GetCatalogueByCode) // Must come before /catalogue
	mux.HandleFunc("/catalogue", h.GetCatalogue)
	mux.HandleFunc("/sales", h.GetSales)

	// Start server
	addr := ":" + cfg.ServerPort
	fmt.Printf("\n🚀 Server starting on http://localhost%s\n", addr)
	fmt.Println("\nAvailable endpoints:")
	fmt.Println("  GET  /health              - Health check with cache sizes")
	fmt.Println("  GET  /reindex             - Trigger reindexing")
	fmt.Println("  GET  /catalogue           - List all products")
	fmt.Println("  GET  /catalogue?name=X    - Search products by name")
	fmt.Println("  GET  /catalogue/:code     - Get product by code")
	fmt.Println("  GET  /sales               - List all sales")
	fmt.Println("  GET  /sales?name=X        - Search sales by product name")
	fmt.Println("  GET  /sales?country=X     - Search sales by country")
	fmt.Println()

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// initializeCaches creates and configures the SQL Store caches
func initializeCaches(c *client.InfinispanClient, cfg *config.Config) error {
	fmt.Println("\n📦 Initializing caches...")

	// Read and prepare table store configuration
	tableStoreXML, err := os.ReadFile("tableStore.xml")
	if err != nil {
		return fmt.Errorf("failed to read tableStore.xml: %w", err)
	}

	tableStoreConfig := replacePlaceholders(string(tableStoreXML), cfg)

	// Read and prepare query store configuration
	queryStoreXML, err := os.ReadFile("queryStore.xml")
	if err != nil {
		return fmt.Errorf("failed to read queryStore.xml: %w", err)
	}

	queryStoreConfig := replacePlaceholders(string(queryStoreXML), cfg)

	// Create catalogue cache
	fmt.Printf("Creating cache: %s\n", cfg.CatalogCacheName)
	if err := c.CreateCache(cfg.CatalogCacheName, tableStoreConfig); err != nil {
		return fmt.Errorf("failed to create catalogue cache: %w", err)
	}

	// Create sold products cache
	fmt.Printf("Creating cache: %s\n", cfg.SoldProductsCacheName)
	if err := c.CreateCache(cfg.SoldProductsCacheName, queryStoreConfig); err != nil {
		return fmt.Errorf("failed to create sold products cache: %w", err)
	}

	// CRITICAL: Trigger reindexing for SQL Store caches
	fmt.Println("\n🔄 Triggering reindexing (required for SQL Store caches)...")
	if err := c.Reindex(cfg.CatalogCacheName); err != nil {
		return fmt.Errorf("failed to reindex catalogue cache: %w", err)
	}

	if err := c.Reindex(cfg.SoldProductsCacheName); err != nil {
		return fmt.Errorf("failed to reindex sold products cache: %w", err)
	}

	// Verify cache sizes
	fmt.Println("\n✅ Caches initialized successfully!")
	catalogSize, _ := c.GetCacheSize(cfg.CatalogCacheName)
	soldProductsSize, _ := c.GetCacheSize(cfg.SoldProductsCacheName)
	fmt.Printf("  - %s: %d entries\n", cfg.CatalogCacheName, catalogSize)
	fmt.Printf("  - %s: %d entries\n", cfg.SoldProductsCacheName, soldProductsSize)

	return nil
}

// replacePlaceholders replaces configuration placeholders in XML
func replacePlaceholders(xmlConfig string, cfg *config.Config) string {
	replacer := strings.NewReplacer(
		"CONNECTION_URL", cfg.DBConnectionURL,
		"USER_NAME", cfg.DBUsername,
		"PASSWORD", cfg.DBPassword,
		"DIALECT", cfg.DBDialect,
		"DRIVER", cfg.DBDriver,
	)
	return replacer.Replace(xmlConfig)
}
