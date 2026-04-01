package config

import "os"

// Config holds application configuration
type Config struct {
	// Infinispan connection
	InfinispanURL  string
	Username       string
	Password       string

	// Cache names
	CatalogCacheName      string
	SoldProductsCacheName string

	// Database connection for SQL Store
	DBConnectionURL string
	DBUsername      string
	DBPassword      string
	DBDialect       string
	DBDriver        string

	// Server config
	ServerPort string
}

// Load returns configuration from environment variables with defaults
func Load() *Config {
	return &Config{
		InfinispanURL:         getEnv("INFINISPAN_URL", "http://localhost:11222"),
		Username:              getEnv("INFINISPAN_USERNAME", "admin"),
		Password:              getEnv("INFINISPAN_PASSWORD", "secret"),
		CatalogCacheName:      getEnv("CATALOG_CACHE_NAME", "catalogue-table-store"),
		SoldProductsCacheName: getEnv("SOLD_PRODUCTS_CACHE_NAME", "sold-products-query-store"),
		DBConnectionURL:       getEnv("DB_CONNECTION_URL", "jdbc:postgresql://database:5432/retailstore"),
		DBUsername:            getEnv("DB_USERNAME", "infinispan"),
		DBPassword:            getEnv("DB_PASSWORD", "secret"),
		DBDialect:             getEnv("DB_DIALECT", "POSTGRES"),
		DBDriver:              getEnv("DB_DRIVER", "org.postgresql.Driver"),
		ServerPort:            getEnv("SERVER_PORT", "8280"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
