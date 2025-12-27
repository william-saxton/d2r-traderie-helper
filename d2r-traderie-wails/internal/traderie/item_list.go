package traderie

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed traderie-item-list.json
var embeddedJSON []byte

// TraderieItemList represents the full traderie item list structure
type TraderieItemList struct {
	Items []TraderieItem `json:"items"`
}

// TraderieItem represents a single item in the traderie list
type TraderieItem struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Img         string             `json:"img"`
	Type        string             `json:"type"`
	Description string             `json:"description"`
	Properties  []TraderieProperty `json:"properties"`
	Tags        []TraderieTag      `json:"tags"`
}

// TraderieProperty represents a property that can be set on an item
type TraderieProperty struct {
	PropertyID int      `json:"property_id"`
	Property   string   `json:"property"`
	Options    []string `json:"options"` // For dropdown properties
	Type       string   `json:"type"`    // "string", "number", "array"
	Required   bool     `json:"required"`
	Min        *int     `json:"min"`
	Max        *int     `json:"max"`
}

// TraderieTag represents a tag for categorizing items
type TraderieTag struct {
	TagID    int    `json:"tag_id"`
	Tag      string `json:"tag"`
	Category string `json:"category"`
}

// LoadItemListFromEmbedded loads the traderie item list from embedded data
func LoadItemListFromEmbedded() (*TraderieItemList, error) {
	if len(embeddedJSON) == 0 {
		return nil, fmt.Errorf("embedded item list is empty")
	}
	return ParseItemListJSON(embeddedJSON)
}

// ParseItemListJSON parses the traderie item list from JSON bytes
func ParseItemListJSON(data []byte) (*TraderieItemList, error) {
	var itemList TraderieItemList
	err := json.Unmarshal(data, &itemList)
	if err != nil {
		return nil, fmt.Errorf("failed to parse item list: %w", err)
	}

	return &itemList, nil
}

// FindItemByName searches for an item by name in the traderie list (case-insensitive)
func (til *TraderieItemList) FindItemByName(name string) (*TraderieItem, bool) {
	for i := range til.Items {
		if strings.EqualFold(til.Items[i].Name, name) {
			return &til.Items[i], true
		}
	}
	return nil, false
}

// GetPropertyOptions returns the valid options for a property
func (ti *TraderieItem) GetPropertyOptions(propertyName string) []string {
	for _, prop := range ti.Properties {
		if prop.Property == propertyName {
			return prop.Options
		}
	}
	return nil
}

// GetPropertyType returns the type of a property (string, number, array)
func (ti *TraderieItem) GetPropertyType(propertyName string) string {
	for _, prop := range ti.Properties {
		if prop.Property == propertyName {
			return prop.Type
		}
	}
	return ""
}
