package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/yourusername/d2r-traderie-wails/internal/traderie"
	"github.com/yourusername/d2r-traderie-wails/pkg/models"
)

// Client handles communication with Traderie API
type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	mapper     *PropertyMapper
}

// NewClient creates a new Traderie API client
func NewClient(apiKey string) *Client {
	return &Client{
		baseURL: "https://traderie.com/api",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey: apiKey,
		mapper: NewPropertyMapper(),
	}
}

// PostItem posts an item to Traderie
// NOTE: This method will likely be blocked by Cloudflare
// Use CloudflareClient instead for reliable posting
func (c *Client) PostItem(item *models.Item, tItem *traderie.TraderieItem, platform, mode string, ladder bool, region string, prices []models.CurrencyGroupPrice, manualMappings []map[string]interface{}, makeOffer bool, itemList *traderie.TraderieItemList) error {
	log.Printf("Posting item to Traderie (Standard API): %s", item.Name)

	traderieItem := c.mapper.MapItemToTraderie(item, tItem, platform, mode, ladder, region, manualMappings, makeOffer, prices, itemList)

	// Create multipart body
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	
	// Marshal to JSON
	payload, err := json.Marshal(traderieItem)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}
	
	fw, err := w.CreateFormField("body")
	if err != nil {
		return fmt.Errorf("failed to create form field: %w", err)
	}
	if _, err := fw.Write(payload); err != nil {
		return fmt.Errorf("failed to write payload: %w", err)
	}
	w.Close()

	log.Printf("Payload: %s", string(payload))

	// Create request
	req, err := http.NewRequest("POST", "https://traderie.com/api/diablo2resurrected/listings/create", &b)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	if c.apiKey != "" {
		// Ensure we don't double prepend Bearer
		auth := c.apiKey
		if !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			auth = "Bearer " + auth
		}
		req.Header.Set("Authorization", auth)
	}

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	log.Println("Item posted successfully")
	return nil
}

// LearnMapping adds a mapping to the internal mapper
func (c *Client) LearnMapping(d2rProp, traderieProp string) {
	c.mapper.LearnMapping(d2rProp, traderieProp)
}

// MapPropertyName maps in-game property name to Traderie format
func (c *Client) MapPropertyName(inGameText, itemClass string) string {
	return c.mapper.MapPropertyName(inGameText, itemClass)
}

// TestConnection tests the connection to Traderie API
func (c *Client) TestConnection() error {
	req, err := http.NewRequest("GET", c.baseURL+"/status", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	log.Println("Successfully connected to Traderie API")
	return nil
}

// GetUserListings retrieves user's current listings
func (c *Client) GetUserListings() ([]models.TraderieItem, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/listings/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	var listings []models.TraderieItem
	if err := json.NewDecoder(resp.Body).Decode(&listings); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return listings, nil
}

