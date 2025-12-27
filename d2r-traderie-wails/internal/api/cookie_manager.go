package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// CookieStore represents stored cookies
type CookieStore struct {
	Cookies []StoredCookie `json:"cookies"`
	Updated time.Time      `json:"updated"`
}

// StoredCookie represents a cookie that can be serialized
type StoredCookie struct {
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Domain   string    `json:"domain"`
	Path     string    `json:"path"`
	Expires  time.Time `json:"expires,omitempty"`
	Secure   bool      `json:"secure"`
	HttpOnly bool      `json:"http_only"`
}

// CookieManager handles saving and loading cookies
type CookieManager struct {
	cookiePath string
}

// NewCookieManager creates a new cookie manager
func NewCookieManager() *CookieManager {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	
	cookiePath := filepath.Join(homeDir, ".d2r-traderie", "cookies.json")
	return &CookieManager{
		cookiePath: cookiePath,
	}
}

// SaveCookies saves cookies to disk
func (cm *CookieManager) SaveCookies(cookies []*http.Cookie) error {
	// Convert to storable format
	var stored []StoredCookie
	for _, c := range cookies {
		stored = append(stored, StoredCookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  c.Expires,
			Secure:   c.Secure,
			HttpOnly: c.HttpOnly,
		})
	}

	store := CookieStore{
		Cookies: stored,
		Updated: time.Now(),
	}

	// Ensure directory exists
	dir := filepath.Dir(cm.cookiePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cookies: %w", err)
	}

	// Write to file
	if err := os.WriteFile(cm.cookiePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write cookies: %w", err)
	}

	return nil
}

// LoadCookies loads cookies from disk
func (cm *CookieManager) LoadCookies() ([]*http.Cookie, error) {
	// Check if file exists
	if _, err := os.Stat(cm.cookiePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no saved cookies found - please set up authentication first")
	}

	// Read file
	data, err := os.ReadFile(cm.cookiePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cookies: %w", err)
	}

	// Unmarshal
	var store CookieStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("failed to parse cookies: %w", err)
	}

	// Convert back to http.Cookie
	var cookies []*http.Cookie
	for _, sc := range store.Cookies {
		cookies = append(cookies, &http.Cookie{
			Name:     sc.Name,
			Value:    sc.Value,
			Domain:   sc.Domain,
			Path:     sc.Path,
			Expires:  sc.Expires,
			Secure:   sc.Secure,
			HttpOnly: sc.HttpOnly,
		})
	}

	return cookies, nil
}

// ParseCookieString parses a cookie string from browser devtools
// Format: "name1=value1; name2=value2; ..."
func (cm *CookieManager) ParseCookieString(cookieStr string, domain string) ([]*http.Cookie, error) {
	if cookieStr == "" {
		return nil, fmt.Errorf("empty cookie string")
	}

	// Parse the cookie string
	header := http.Header{}
	header.Add("Cookie", cookieStr)
	req := &http.Request{Header: header}

	cookies := req.Cookies()
	if len(cookies) == 0 {
		return nil, fmt.Errorf("no cookies found in string")
	}

	// Set domain for all cookies
	for _, c := range cookies {
		if c.Domain == "" {
			c.Domain = domain
		}
		if c.Path == "" {
			c.Path = "/"
		}
	}

	return cookies, nil
}

// ParseNetscapeCookies parses cookies from Netscape cookie file format
// (compatible with browser cookie export extensions)
func (cm *CookieManager) ParseNetscapeCookies(cookieFile string) ([]*http.Cookie, error) {
	data, err := os.ReadFile(cookieFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read cookie file: %w", err)
	}

	var cookies []*http.Cookie
	lines := splitLines(string(data))
	
	for _, line := range lines {
		// Skip comments and empty lines
		if line == "" || line[0] == '#' {
			continue
		}

		// Parse Netscape cookie format
		// domain\tflag\tpath\tsecure\texpiration\tname\tvalue
		fields := splitTabs(line)
		if len(fields) != 7 {
			continue
		}

		expires := time.Unix(parseInt64(fields[4]), 0)
		
		cookies = append(cookies, &http.Cookie{
			Name:     fields[5],
			Value:    fields[6],
			Domain:   fields[0],
			Path:     fields[2],
			Expires:  expires,
			Secure:   fields[3] == "TRUE",
			HttpOnly: false,
		})
	}

	if len(cookies) == 0 {
		return nil, fmt.Errorf("no cookies found in file")
	}

	return cookies, nil
}

// Helper functions
func splitLines(s string) []string {
	// Simple split on newlines
	result := []string{}
	current := ""
	for _, c := range s {
		if c == '\n' || c == '\r' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func splitTabs(s string) []string {
	result := []string{}
	current := ""
	for _, c := range s {
		if c == '\t' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func parseInt64(s string) int64 {
	var result int64
	fmt.Sscanf(s, "%d", &result)
	return result
}

// HasSavedCookies checks if cookies are saved
func (cm *CookieManager) HasSavedCookies() bool {
	_, err := os.Stat(cm.cookiePath)
	return err == nil
}


