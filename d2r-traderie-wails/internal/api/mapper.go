package api

import (
	"fmt"
	"strings"

	"github.com/yourusername/d2r-traderie-wails/internal/traderie"
	"github.com/yourusername/d2r-traderie-wails/pkg/models"
)

// PropertyMapper maps d2go item properties to Traderie format
type PropertyMapper struct {
	propertyMappings map[string]interface{}
}

// NewPropertyMapper creates a new property mapper
func NewPropertyMapper() *PropertyMapper {
	return &PropertyMapper{
		propertyMappings: getPropertyMappings(),
	}
}

// MapItemToTraderie converts an item to Traderie API format (listings/create)
func (pm *PropertyMapper) MapItemToTraderie(
	item *models.Item,
	tItem *traderie.TraderieItem,
	platform, mode string,
	ladder bool,
	region string,
	manualMappings []map[string]interface{},
	makeOffer bool,
	prices []models.CurrencyGroupPrice,
	itemList *traderie.TraderieItemList,
) *models.TraderieItem {
	traderieListing := &models.TraderieItem{
		AcceptListingPrice:  false,
		Captcha:             "",
		CaptchaManaged:      false,
		EndTime:             "",
		Free:                false,
		Item:                tItem.ID,
		ItemMode:            nil,
		ItemType:            tItem.Type,
		MakeOffer:           makeOffer,
		NeedMaterials:       false,
		OfferBells:          false,
		OfferNmt:            false,
		OfferWishlist:       false,
		OfferWishlistId:     "",
		Selling:             true,
		StandingListing:     false,
		StockListing:        false,
		TouchTrading:        false,
		Wishlist:            "",
		Amount:              "1",
		Properties:          []models.TraderieListingProp{},
		CurrencyGroupPrices: []models.CurrencyGroupPrice{},
		Items:               []models.TraderieListingItem{}, // Start empty for D2R
	}

	// Helper to find property info from the Traderie item list
	findPropInfo := func(name string) *traderie.TraderieProperty {
		for i := range tItem.Properties {
			if strings.EqualFold(tItem.Properties[i].Property, name) {
				return &tItem.Properties[i]
			}
		}
		return nil
	}

	// Keep track of property IDs already added to avoid duplicates
	addedPropIDs := make(map[int]bool)

	// 1. Add common properties (Platform, Mode, Ladder, Region) - Preferred: true
	commonProps := []struct {
		name  string
		value interface{}
	}{
		{"Platform", platform},
		{"Mode", mode},
		{"Ladder", ladder},
		{"Region", region},
	}

	for _, cp := range commonProps {
		if info := findPropInfo(cp.name); info != nil {
			if !addedPropIDs[info.PropertyID] {
				traderieListing.Properties = append(traderieListing.Properties, models.TraderieListingProp{
					ID:        info.PropertyID,
					Property:  info.Property,
					Option:    cp.value,
					Type:      info.Type,
					Preferred: true,
				})
				addedPropIDs[info.PropertyID] = true
			}
		}
	}

	// 2. Add manual mappings from the UI (High Priority) - Preferred: true
	for _, mapping := range manualMappings {
		d2rProp, _ := mapping["d2rProp"].(string)
		traderieProp, _ := mapping["traderieProp"].(string)

		if traderieProp != "" {
			if info := findPropInfo(traderieProp); info != nil {
				if !addedPropIDs[info.PropertyID] {
					var val interface{}
					
					// Special handling for Rarity/Quality
					if info.Property == "Rarity" {
						if strings.HasPrefix(d2rProp, "Quality: ") {
							val = strings.ToLower(strings.TrimPrefix(d2rProp, "Quality: "))
							// Ensure the value exists in Traderie options
							foundOption := false
							for _, opt := range info.Options {
								if strings.EqualFold(opt, val.(string)) {
									val = opt
									foundOption = true
									break
								}
							}
							if !foundOption {
								if val == "crafted" {
									val = "rare" // Fallback
								} else {
									val = info.Options[0] // Default to first option
								}
							}
						}
					}

					if val == nil {
						val = pm.extractNumericValue(d2rProp)
					}

					if val != nil {
						if info.Type == "bool" {
							val = true
						}
						traderieListing.Properties = append(traderieListing.Properties, models.TraderieListingProp{
							ID:        info.PropertyID,
							Property:  info.Property,
							Option:    val,
							Type:      info.Type,
							Preferred: true,
						})
						addedPropIDs[info.PropertyID] = true
					}
				}
			}
		}
	}

	// REMOVED: 3. Add automatic mappings for any remaining properties
	// This was causing too many unwanted properties to be sent.

	// 4. Ethereal/Sockets - Preferred: true
	if item.IsEthereal {
		if info := findPropInfo("Ethereal"); info != nil && !addedPropIDs[info.PropertyID] {
			traderieListing.Properties = append(traderieListing.Properties, models.TraderieListingProp{
				ID:        info.PropertyID,
				Property:  info.Property,
				Option:    true,
				Type:      info.Type,
				Preferred: true,
			})
			addedPropIDs[info.PropertyID] = true
		}
	}

	if item.Sockets > 0 {
		if info := findPropInfo("Sockets"); info != nil && !addedPropIDs[info.PropertyID] {
			traderieListing.Properties = append(traderieListing.Properties, models.TraderieListingProp{
				ID:        info.PropertyID,
				Property:  info.Property,
				Option:    item.Sockets,
				Type:      info.Type,
				Preferred: true,
			})
			addedPropIDs[info.PropertyID] = true
		}
	}

	// 5. Pricing (match curl example)
	// For D2R, the 'items' array should contain the items you WANT (your price)
	if !makeOffer && len(prices) > 0 {
		firstGroup := prices[0]
		for _, pItem := range firstGroup.Items {
			// Find metadata for this price item from our global list
			var otMetadata *traderie.TraderieItem
			if itemList != nil {
				for i := range itemList.Items {
					if itemList.Items[i].ID == pItem.Item {
						otMetadata = &itemList.Items[i]
						break
					}
				}
			}

			if otMetadata == nil {
				// Fallback: Use the item ID as label if not found in list
				traderieListing.Items = append(traderieListing.Items, models.TraderieListingItem{
					Quantity:   pItem.Quantity,
					Value:      pItem.Item,
					Label:      pItem.Item,
					Properties: []interface{}{},
				})
				continue
			}

			// Convert metadata properties to full objects for validation
			var props []interface{}
			for _, p := range otMetadata.Properties {
				props = append(props, p)
			}

			// Add requested PRICE item to the items list
			traderieListing.Items = append(traderieListing.Items, models.TraderieListingItem{
				Quantity:   pItem.Quantity,
				Diy:        false,
				CanCatalog: false,
				Properties: props,
				Value:      otMetadata.ID, // Correctly set to price item's ID
				Label:      otMetadata.Name,
				ImgURL:     otMetadata.Img,
				Index:      0,
				Group:      0,
				OfferProps: []string{},
			})
		}

		// Ensure top-level items array is populated and currencyGroupPrices is empty
		traderieListing.CurrencyGroupPrices = []models.CurrencyGroupPrice{}
	} else {
		// Just use CurrencyGroupPrices if makeOffer=true or no prices
		traderieListing.CurrencyGroupPrices = prices
	}

	return traderieListing
}

// extractNumericValue helper to get a number from a property string
func (pm *PropertyMapper) extractNumericValue(s string) interface{} {
	// Look for ranges first (e.g. 3-32)
	if strings.Contains(s, "-") {
		parts := strings.Split(s, "-")
		if len(parts) >= 2 {
			var maxVal int
			fmt.Sscanf(strings.TrimSpace(parts[len(parts)-1]), "%d", &maxVal)
			if maxVal > 0 {
				return maxVal
			}
		}
	}

	// Scan through string to find first number
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			var intVal int
			fmt.Sscanf(s[i:], "%d", &intVal)
			return intVal
		}
	}
	return nil
}

// MapPropertyName maps in-game property name to Traderie format
func (pm *PropertyMapper) MapPropertyName(inGameText, itemClass string) string {
	// Clean up the in-game text
	cleanInGame := inGameText
	if idx := strings.Index(inGameText, ":"); idx != -1 {
		cleanInGame = strings.TrimSpace(inGameText[:idx])
	}

	normalized := normalizeText(cleanInGame)

	// Try exact match first
	for key, value := range pm.propertyMappings {
		if normalizeText(key) == normalized {
			return pm.resolveMapping(value, itemClass)
		}
	}

	// Try partial match
	for key, value := range pm.propertyMappings {
		normalizedKey := normalizeText(key)
		if strings.Contains(normalized, normalizedKey) || strings.Contains(normalizedKey, normalized) {
			return pm.resolveMapping(value, itemClass)
		}
	}

	return ""
}

// LearnMapping adds a new mapping to the mapper
func (pm *PropertyMapper) LearnMapping(d2rProp, traderieProp string) {
	cleanD2R := d2rProp
	if idx := strings.Index(d2rProp, ":"); idx != -1 {
		cleanD2R = strings.TrimSpace(d2rProp[:idx])
	}
	
	pm.propertyMappings[cleanD2R] = traderieProp
}

// resolveMapping handles class-specific skill mappings
func (pm *PropertyMapper) resolveMapping(value interface{}, itemClass string) string {
	switch v := value.(type) {
	case string:
		return v
	case map[string]string:
		if itemClass != "" && v[itemClass] != "" {
			return v[itemClass]
		}
		if v["any"] != "" {
			return v["any"]
		}
	}
	return ""
}

// mapCategory converts item type to Traderie category
func (pm *PropertyMapper) mapCategory(itemType string) string {
	categoryMap := map[string]string{
		"Ring":    "jewelry",
		"Amulet":  "jewelry",
		"Charm":   "charms",
		"Jewel":   "jewels",
		"Rune":    "runes",
		"Weapon":  "weapons",
		"Armor":   "armor",
		"Helm":    "armor",
		"Shield":  "armor",
		"Gloves":  "armor",
		"Belt":    "armor",
		"Boots":   "armor",
	}

	if category, ok := categoryMap[itemType]; ok {
		return category
	}
	return "other"
}

// mapQuality converts item quality to Traderie format
func (pm *PropertyMapper) mapQuality(quality string) string {
	qualityMap := map[string]string{
		"Unique":  "unique",
		"Set":     "set",
		"Rare":    "rare",
		"Magic":   "magic",
		"Crafted": "crafted",
		"Normal":  "normal",
	}

	if q, ok := qualityMap[quality]; ok {
		return q
	}
	return "magic"
}

// normalizeText standardizes text for comparison
func normalizeText(text string) string {
	text = strings.ToLower(text)
	text = strings.TrimSpace(text)
	
	// Remove special characters except spaces
	var result strings.Builder
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			result.WriteRune(r)
		} else {
			result.WriteRune(' ')
		}
	}
	
	// Normalize multiple spaces
	normalized := result.String()
	normalized = strings.Join(strings.Fields(normalized), " ")
	
	return normalized
}

// getPropertyMappings returns the property mapping table
func getPropertyMappings() map[string]interface{} {
	return map[string]interface{}{
		// Skills - Barbarian
		"Battle Orders": map[string]string{
			"barbarian": "to Battle Orders (Barbarian Only)",
			"any":       "to Battle Orders (Any Class)",
		},
		"Battle Command": map[string]string{
			"barbarian": "to Battle Command (Barbarian Only)",
			"any":       "to Battle Command (Any Class)",
		},
		"Whirlwind": map[string]string{
			"barbarian": "to Whirlwind (Barbarian Only)",
			"any":       "to Whirlwind (Any Class)",
		},
		"Berserk": map[string]string{
			"barbarian": "to Berserk (Barbarian Only)",
			"any":       "to Berserk (Any Class)",
		},
		
		// Skills - Sorceress
		"Fireball": map[string]string{
			"sorceress": "to Fireball (Sorceress Only)",
			"any":       "to Fireball (Any Class)",
		},
		"Fire Ball": map[string]string{
			"sorceress": "to Fireball (Sorceress Only)",
			"any":       "to Fireball (Any Class)",
		},
		"Frozen Orb": map[string]string{
			"sorceress": "to Frozen Orb (Sorceress Only)",
			"any":       "to Frozen Orb (Any Class)",
		},
		"Blizzard": map[string]string{
			"sorceress": "to Blizzard (Sorceress Only)",
			"any":       "to Blizzard (Any Class)",
		},
		"Chain Lightning": map[string]string{
			"sorceress": "to Chain Lightning (Sorceress Only)",
			"any":       "to Chain Lightning (Any Class)",
		},
		"Lightning": map[string]string{
			"sorceress": "to Lightning (Sorceress Only)",
			"any":       "to Lightning (Any Class)",
		},
		"Teleport": map[string]string{
			"sorceress": "to Teleport (Sorceress Only)",
			"any":       "to Teleport (Any Class)",
		},
		
		// Skills - Paladin
		"Holy Shield": map[string]string{
			"paladin": "to Holy Shield (Paladin Only)",
			"any":     "to Holy Shield (Any Class)",
		},
		"Blessed Hammer": map[string]string{
			"paladin": "to Blessed Hammer (Paladin Only)",
			"any":     "to Blessed Hammer (Any Class)",
		},
		"Concentration": map[string]string{
			"paladin": "to Concentration (Paladin Only)",
			"any":     "to Concentration (Any Class)",
		},
		"Fanaticism": map[string]string{
			"paladin": "to Fanaticism (Paladin Only)",
			"any":     "to Fanaticism (Any Class)",
		},
		"Conviction": map[string]string{
			"paladin": "to Conviction (Paladin Only)",
			"any":     "to Conviction (Any Class)",
		},
		
		// General skill modifiers
		"All Skills":                "to All Skills",
		"All Skill Levels":          "to All Skills",
		"Barbarian Skill Levels":    "to Barbarian Skill Levels",
		"Sorceress Skill Levels":    "to Sorceress Skill",
		"Paladin Skill Levels":      "to Paladin Skill",
		"Necromancer Skill Levels":  "to Necromancer Skill",
		"Amazon Skill Levels":       "to Amazon Skill",
		"Druid Skill Levels":        "to Druid Skill",
		"Assassin Skill Levels":     "to Assassin Skill",
		"Lightning Damage":          "Lightning Damage",
		"Adds Lightning Damage":     "Lightning Damage",
		
		// Stats
		"Faster Cast Rate":            "% Faster Cast Rate",
		"Faster Hit Recovery":         "% Faster Hit Recovery",
		"Increased Attack Speed":      "% Increased Attack Speed",
		"Faster Run/Walk":             "% Faster Run/Walk",
		"Faster Block Rate":           "% Faster Block Rate",
		"Enhanced Damage":             "% Enhanced Damage",
		"Enhanced Defense":            "% Enhanced Defense",
		"Increased Maximum Durability": "% Increased Maximum Durability",
		
		// Life and Mana
		"Life":         "to Life",
		"Maximum Life": "to Life",
		"Mana":         "to Mana",
		"Maximum Mana": "to Mana",
		
		// Attributes
		"Strength":  "to Strength",
		"Dexterity": "to Dexterity",
		"Vitality":  "to Vitality",
		"Energy":    "to Energy",
		
		// Stealing
		"Life Stolen Per Hit": "% Life Stolen Per Hit",
		"Mana Stolen Per Hit": "% Mana Stolen Per Hit",
		
		// Resistances
		"All Resistances":   "to All Resistances",
		"Fire Resist":       "% Fire Resist",
		"Cold Resist":       "% Cold Resist",
		"Lightning Resist":  "% Lightning Resist",
		"Poison Resist":     "% Poison Resist",
		
		// Magic Find
		"Magic Find":                           "% Better Chance of Getting Magic Items",
		"Better Chance of Getting Magic Items": "% Better Chance of Getting Magic Items",
		
		// Combat modifiers
		"Deadly Strike":       "% Deadly Strike",
		"Crushing Blow":       "% Crushing Blow",
		"Open Wounds":         "% Open Wounds",
		"Chance of Open Wounds": "% Open Wounds",
		
		// Other
		"Defense":               "Defense",
		"Damage":                "Damage",
		"Attack Rating":         "to Attack Rating",
		"Bonus to Attack Rating": "to Attack Rating",
		"Sockets":               "Sockets",
		"Socket":                "Sockets",
		"Replenish Life":        "Replenish Life",
		"Regenerate Mana":       "Regenerate Mana",
		"Damage Reduced":        "Damage Reduced",
		"Magic Damage Reduced":  "Magic Damage Reduced",
		"Cannot Be Frozen":      "Cannot Be Frozen",
		"Ignore Target Defense": "Ignore Target's Defense",
		"Prevent Monster Heal":  "Prevent Monster Heal",
		"Half Freeze Duration":  "Half Freeze Duration",
		"Light Radius":          "to Light Radius",
	}
}
