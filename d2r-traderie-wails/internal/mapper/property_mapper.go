package mapper

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// PropertyMapping represents a learned mapping between D2R and Traderie properties
type PropertyMapping struct {
	D2RProperty      string `json:"d2r_property"`       // Property name from d2go
	TraderieProperty string `json:"traderie_property"`  // Property name in Traderie
	ItemName         string `json:"item_name"`          // Item this mapping applies to (empty = all items)
}

// PropertyMapper handles learning and applying property mappings
type PropertyMapper struct {
	mappings map[string]PropertyMapping // Key: D2RProperty
	mu       sync.RWMutex
	filePath string
}

// NewPropertyMapper creates a new property mapper
func NewPropertyMapper() *PropertyMapper {
	homeDir, _ := os.UserHomeDir()
	mappingsPath := filepath.Join(homeDir, ".d2r-traderie", "property_mappings.json")

	pm := &PropertyMapper{
		mappings: make(map[string]PropertyMapping),
		filePath: mappingsPath,
	}

	// Load existing mappings
	pm.Load()

	return pm
}

// Load loads mappings from disk
func (pm *PropertyMapper) Load() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(pm.filePath)
	os.MkdirAll(dir, 0755)

	// Check if file exists
	if _, err := os.Stat(pm.filePath); os.IsNotExist(err) {
		log.Println("No saved property mappings found, starting fresh")
		return nil
	}

	// Read file
	data, err := os.ReadFile(pm.filePath)
	if err != nil {
		return fmt.Errorf("failed to read mappings: %w", err)
	}

	// Parse JSON
	var mappingsList []PropertyMapping
	if err := json.Unmarshal(data, &mappingsList); err != nil {
		return fmt.Errorf("failed to parse mappings: %w", err)
	}

	// Load into map
	for _, m := range mappingsList {
		key := m.D2RProperty
		if m.ItemName != "" {
			key = m.ItemName + ":" + m.D2RProperty
		}
		pm.mappings[key] = m
	}

	log.Printf("✓ Loaded %d saved property mappings", len(pm.mappings))
	return nil
}

// Save saves mappings to disk
func (pm *PropertyMapper) Save() error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Convert map to slice
	mappingsList := make([]PropertyMapping, 0, len(pm.mappings))
	for _, m := range pm.mappings {
		mappingsList = append(mappingsList, m)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(mappingsList, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal mappings: %w", err)
	}

	// Write to file
	if err := os.WriteFile(pm.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write mappings: %w", err)
	}

	log.Printf("✓ Saved %d property mappings", len(pm.mappings))
	return nil
}

// GetMapping retrieves a learned mapping
func (pm *PropertyMapper) GetMapping(d2rProperty, itemName string) (string, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Try item-specific mapping first
	if itemName != "" {
		key := itemName + ":" + d2rProperty
		if m, ok := pm.mappings[key]; ok {
			return m.TraderieProperty, true
		}
	}

	// Try global mapping
	if m, ok := pm.mappings[d2rProperty]; ok {
		return m.TraderieProperty, true
	}

	return "", false
}

// LearnMapping adds or updates a mapping
func (pm *PropertyMapper) LearnMapping(d2rProperty, traderieProperty, itemName string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	key := d2rProperty
	if itemName != "" {
		key = itemName + ":" + d2rProperty
	}

	pm.mappings[key] = PropertyMapping{
		D2RProperty:      d2rProperty,
		TraderieProperty: traderieProperty,
		ItemName:         itemName,
	}

	log.Printf("✓ Learned mapping: '%s' -> '%s' (item: %s)", d2rProperty, traderieProperty, itemName)
}

// GetAllMappings returns all learned mappings
func (pm *PropertyMapper) GetAllMappings() []PropertyMapping {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	mappings := make([]PropertyMapping, 0, len(pm.mappings))
	for _, m := range pm.mappings {
		mappings = append(mappings, m)
	}
	return mappings
}

// RemoveMapping removes a mapping
func (pm *PropertyMapper) RemoveMapping(d2rProperty, itemName string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	key := d2rProperty
	if itemName != "" {
		key = itemName + ":" + d2rProperty
	}

	delete(pm.mappings, key)
}

