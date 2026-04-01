#!/bin/bash

# Color codes
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

BASE_URL="http://localhost:8280"

echo -e "${BLUE}Testing Infinispan Inmemory Catalogue - Go Implementation${NC}\n"

# Test health endpoint
echo -e "${GREEN}1. Testing /health${NC}"
curl -s "${BASE_URL}/health"
echo -e "\n"

# Test catalogue - all products
echo -e "${GREEN}2. Testing /catalogue (all products)${NC}"
curl -s "${BASE_URL}/catalogue" | jq -c '.[] | {code, name, price, stock}' | head -5
echo -e "\n"

# Test catalogue - search by name
echo -e "${GREEN}3. Testing /catalogue?name=Party${NC}"
curl -s "${BASE_URL}/catalogue?name=Party" | jq -c '.[] | {code, name, price, stock}'
echo -e "\n"

# Test catalogue - filter by stock
echo -e "${GREEN}4. Testing /catalogue?stock=100${NC}"
curl -s "${BASE_URL}/catalogue?stock=100" | jq -c '.[] | {code, name, stock}' | head -3
echo -e "\n"

# Test catalogue - price range
echo -e "${GREEN}5. Testing /catalogue?price-min=20&price-max=100${NC}"
curl -s "${BASE_URL}/catalogue?price-min=20&price-max=100" | jq -c '.[] | {code, name, price}' | head -3
echo -e "\n"

# Test catalogue - get by code
echo -e "${GREEN}6. Testing /catalogue/c123${NC}"
curl -s "${BASE_URL}/catalogue/c123" | jq '.'
echo -e "\n"

# Test sales - all
echo -e "${GREEN}7. Testing /sales (all sales)${NC}"
curl -s "${BASE_URL}/sales" | jq -c '.[] | {name, country}' | head -5
echo -e "\n"

# Test sales - search by name
echo -e "${GREEN}8. Testing /sales?name=Skirt${NC}"
curl -s "${BASE_URL}/sales?name=Skirt" | jq -c '.[] | {name, country}' | head -3
echo -e "\n"

# Test sales - filter by country
echo -e "${GREEN}9. Testing /sales?country=Spain${NC}"
curl -s "${BASE_URL}/sales?country=Spain" | jq -c '.[] | {name, country}' | head -3
echo -e "\n"

# Test reindex
echo -e "${GREEN}10. Testing /reindex${NC}"
curl -s "${BASE_URL}/reindex"
echo -e "\n\n"

echo -e "${BLUE}All tests completed!${NC}"
