package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/yourusername/d2r-traderie-wails/internal/traderie"
	"github.com/yourusername/d2r-traderie-wails/pkg/models"
)

// CloudflareClient handles communication with Traderie API using a browser extension bridge
// to bypass Cloudflare protection
type CloudflareClient struct {
	baseURL string
	mapper  *PropertyMapper
	cookies []*http.Cookie
	apiKey  string // Added to store the Bearer token
	bridge  *ExtensionBridge
}

// NewCloudflareClient creates a new Cloudflare-bypassing Traderie API client using the extension bridge
func NewCloudflareClient(cookies []*http.Cookie, apiKey string, bridge *ExtensionBridge) *CloudflareClient {
	// Ensure Bearer is prepended if missing
	auth := apiKey
	if auth != "" && !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		auth = "Bearer " + auth
	}

	return &CloudflareClient{
		baseURL: "https://traderie.com",
		mapper:  NewPropertyMapper(),
		cookies: cookies,
		apiKey:  auth,
		bridge:  bridge,
	}
}

// PostItem posts an item to Traderie using the browser extension bridge
func (c *CloudflareClient) PostItem(item *models.Item, tItem *traderie.TraderieItem, platform, mode string, ladder bool, region string, prices []models.CurrencyGroupPrice, manualMappings []map[string]interface{}, makeOffer bool, itemList *traderie.TraderieItemList) error {
	log.Printf("Posting item to Traderie via extension: %s (ID: %s)", item.Name, tItem.ID)

	// Convert to Traderie format
	traderieItem := c.mapper.MapItemToTraderie(item, tItem, platform, mode, ladder, region, manualMappings, makeOffer, prices, itemList)

	if c.bridge == nil {
		return fmt.Errorf("extension bridge not initialized")
	}

	// Queue the command for the extension
	payload := map[string]interface{}{
		"item":    traderieItem,
		"auth":    c.apiKey,
		"baseURL": c.baseURL,
	}
	cmdID := c.bridge.AddCommand("post_listing", payload)

	// Wait for the result
	result, err := c.bridge.WaitForResult(cmdID, 1*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to get result from extension: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("extension failed to post item: %s", result.Error)
	}

	log.Println("✅ Item posted successfully via extension")
	return nil
}

// LearnMapping adds a mapping to the internal mapper
func (c *CloudflareClient) LearnMapping(d2rProp, traderieProp string) {
	c.mapper.LearnMapping(d2rProp, traderieProp)
}

// MapPropertyName maps in-game property name to Traderie format
func (c *CloudflareClient) MapPropertyName(inGameText, itemClass string) string {
	return c.mapper.MapPropertyName(inGameText, itemClass)
}

// TestConnection tests the connection via the extension
func (c *CloudflareClient) TestConnection() error {
	log.Println("Testing Traderie connection via extension...")

	if c.bridge == nil {
		return fmt.Errorf("extension bridge not initialized")
	}

	cmdID := c.bridge.AddCommand("test_connection", map[string]interface{}{
		"baseURL": c.baseURL,
	})

	result, err := c.bridge.WaitForResult(cmdID, 30*time.Second)
	if err != nil {
		return fmt.Errorf("extension bridge timeout: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("connection test failed: %s", result.Error)
	}

	log.Println("✅ Successfully connected to Traderie via extension")
	return nil
}

// GetUserListings retrieves user's current listings via extension
func (c *CloudflareClient) GetUserListings() ([]models.TraderieItem, error) {
	log.Println("Fetching user listings via extension...")

	if c.bridge == nil {
		return nil, fmt.Errorf("extension bridge not initialized")
	}

	cmdID := c.bridge.AddCommand("get_listings", map[string]interface{}{
		"baseURL": c.baseURL,
	})

	result, err := c.bridge.WaitForResult(cmdID, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("extension bridge timeout: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("failed to fetch listings: %s", result.Error)
	}

	var listings []models.TraderieItem
	if err := json.Unmarshal(result.Data, &listings); err != nil {
		return nil, fmt.Errorf("failed to parse listings: %w", err)
	}

	log.Printf("✅ Retrieved %d listings via extension", len(listings))
	return listings, nil
}


