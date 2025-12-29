package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/yourusername/d2r-traderie-wails/internal/api"
	"github.com/yourusername/d2r-traderie-wails/internal/config"
	"github.com/yourusername/d2r-traderie-wails/internal/hotkey"
	"github.com/yourusername/d2r-traderie-wails/internal/mapper"
	"github.com/yourusername/d2r-traderie-wails/internal/memory"
	"github.com/yourusername/d2r-traderie-wails/internal/traderie"
	"github.com/yourusername/d2r-traderie-wails/pkg/models"
)

// App struct
type App struct {
	ctx            context.Context
	memReader      *memory.Reader
	traderieClient interface {
		PostItem(item *models.Item, tItem *traderie.TraderieItem, platform, mode string, ladder bool, region string, prices []models.CurrencyGroupPrice, manualMappings []map[string]interface{}, makeOffer bool, itemList *traderie.TraderieItemList) error
		TestConnection() error
		LearnMapping(d2rProp, traderieProp string)
		MapPropertyName(inGameText, itemClass string) string
	}
	hotkeyListener *hotkey.Listener
	itemList       *traderie.TraderieItemList
	propertyMapper *mapper.PropertyMapper
	config         *config.Config
	cookieManager  *api.CookieManager
	bridge         *api.ExtensionBridge
	refreshTicker  *time.Ticker
	refreshStop    chan struct{}
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.Println("Starting D2R Traderie Wails Application...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Failed to load config, using defaults: %v", err)
		cfg = config.Default()
	}
	a.config = cfg

	// Initialize memory reader
	log.Println("Looking for Diablo 2 Resurrected process...")
	memReader, err := memory.NewReader()
	if err != nil {
		errMsg := fmt.Sprintf("❌ Could not connect to D2R: %v\n\nPlease run as Administrator!", err)
		runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "D2R Connection Error",
			Message: errMsg,
		})
		log.Println(errMsg)
		return
	}
	a.memReader = memReader
	log.Println("✅ Connected to D2R successfully!")

	// Initialize cookie manager
	a.cookieManager = api.NewCookieManager()

	// Initialize extension bridge
	a.bridge = api.NewExtensionBridge(8081)
	if err := a.bridge.Start(); err != nil {
		log.Printf("❌ Failed to start extension bridge: %v", err)
	} else {
		log.Println("✅ Extension bridge started on port 8081")
	}

	// Initialize property mapper
	a.propertyMapper = mapper.NewPropertyMapper()

	// Initialize Traderie API client with Cloudflare bypass
	if a.cookieManager.HasSavedCookies() {
		cookies, err := a.cookieManager.LoadCookies()
		if err != nil {
			log.Printf("⚠️ Failed to load cookies: %v", err)
			log.Println("⚠️ Falling back to standard HTTP client (may be blocked by Cloudflare)")
			a.traderieClient = api.NewClient(cfg.Traderie.APIKey)
		} else {
			a.traderieClient = api.NewCloudflareClient(cookies, cfg.Traderie.APIKey, a.bridge)
			log.Println("✅ Using Cloudflare bypass with saved cookies")
		}
	} else {
		log.Println("⚠️ No saved cookies found")
		log.Println("⚠️ Standard HTTP client may be blocked by Cloudflare")
		log.Println("💡 Use 'Setup Cookies' button in UI to configure Cloudflare bypass")
		a.traderieClient = api.NewClient(cfg.Traderie.APIKey)
	}

	// Apply saved mappings to the traderie client
	if a.traderieClient != nil {
		mappings := a.propertyMapper.GetAllMappings()
		for _, m := range mappings {
			a.traderieClient.LearnMapping(m.D2RProperty, m.TraderieProperty)
		}
		log.Printf("✓ Applied %d saved property mappings to Traderie client", len(mappings))
	}

	// Load traderie item list from embedded data
	log.Println("Loading traderie item list from embedded data...")
	itemList, err := traderie.LoadItemListFromEmbedded()
	if err != nil {
		log.Printf("❌ Failed to load embedded item list: %v", err)
	} else {
		a.itemList = itemList
		log.Printf("✓ Loaded %d items from embedded Traderie list", len(itemList.Items))
	}

	// Initialize hotkey listener
	a.hotkeyListener = hotkey.NewListener(cfg.Hotkey, func() {
		log.Println("Hotkey pressed! Scanning item...")
		a.handleHotkey()
	})

	if err := a.hotkeyListener.Start(); err != nil {
		log.Printf("Failed to start hotkey listener: %v", err)
	} else {
		log.Printf("✅ Hotkey listener started. Press %s to capture item.", cfg.Hotkey)
	}

	// Start auto-refresh if enabled
	a.StartAutoRefresh()

	log.Println("===========================================")
	log.Println("Application ready!")
	log.Println("Press F9 while hovering/holding an item to scan")
	log.Println("===========================================")

	// Emit ready event to frontend
	runtime.EventsEmit(ctx, "backend-ready", map[string]string{
		"version": "1.4.0-PRICING-FIX",
	})
}

// shutdown is called when the app is shutting down
func (a *App) shutdown(ctx context.Context) {
	if a.hotkeyListener != nil {
		a.hotkeyListener.Stop()
	}
	if a.memReader != nil {
		a.memReader.Close()
	}
	log.Println("Application shut down")
}

// getUniqueItemNameMapping returns the correct Traderie name for items where d2go name differs
func getUniqueItemNameMapping(d2goName string) string {
	// Map of d2go unique item names to Traderie names (case-insensitive)
	mappings := map[string]string{
		"death's fathom":  "Death's Fathom",
		"deaths fathom":   "Death's Fathom",
		"death fathom":    "Death's Fathom",
		// Add more mappings here as issues are discovered
	}
	
	normalized := strings.ToLower(strings.TrimSpace(d2goName))
	if traderieName, ok := mappings[normalized]; ok {
		return traderieName
	}
	return ""
}

// FindTraderieItem finds a matching Traderie item using name, type, and fuzzy matching
func (a *App) FindTraderieItem(item *models.Item) (*traderie.TraderieItem, bool) {
	if a.itemList == nil || len(a.itemList.Items) == 0 {
		log.Println("❌ ERROR: Traderie item list is empty or nil!")
		return nil, false
	}

	itemName := item.Name
	itemType := item.Type
	
	// Check for known unique item name mappings
	if item.Quality == "Unique" || item.Quality == "Set" {
		if mappedName := getUniqueItemNameMapping(itemName); mappedName != "" {
			log.Printf("🔄 Applying name mapping: '%s' → '%s'", itemName, mappedName)
			itemName = mappedName
		}
	}

	// Special case: Rainbow Facet
	if strings.Contains(strings.ToLower(itemName), "rainbow facet") {
		element := ""
		trigger := ""

		for _, prop := range item.Properties {
			pName := strings.ToLower(prop.Name)
			if strings.Contains(pName, "fire skill damage") || strings.Contains(pName, "enemy fire resistance") {
				element = "Fire"
			} else if strings.Contains(pName, "lightning skill damage") || strings.Contains(pName, "enemy lightning resistance") {
				element = "Lightning"
			} else if strings.Contains(pName, "cold skill damage") || strings.Contains(pName, "enemy cold resistance") {
				element = "Cold"
			} else if strings.Contains(pName, "poison skill damage") || strings.Contains(pName, "enemy poison resistance") {
				element = "Poison"
			}

			if strings.Contains(pName, "on level-up") {
				trigger = "Level-up"
			} else if strings.Contains(pName, "on death") {
				trigger = "Death"
			}
		}

		if element != "" && trigger != "" {
			facetName := fmt.Sprintf("Rainbow Facet: %s %s", element, trigger)
			log.Printf("💎 Identified Rainbow Facet: %s", facetName)
			tItem, found := a.itemList.FindItemByName(facetName)
			if found {
				return tItem, true
			}
		}
	}

	// For Rare, Magic, or Crafted items, skip name matching and use base type directly
	// This is because these items have generated names (e.g., "Havoc Jack") but should
	// search Traderie by their base type (e.g., "Plate Mail")
	if item.Quality == "Rare" || item.Quality == "Magic" || item.Quality == "Crafted" {
		searchName := itemType
		
		// For crafted items with a detected type, construct the full name
		// Traderie uses GENERIC types: "Caster Gloves", not "Caster Leather Gloves"
		// So we need to extract the generic type from the base type
		if item.Quality == "Crafted" && item.CraftedType != "" {
			// Extract generic type: "Leather Gloves" -> "Gloves", "Gold Amulet" -> "Amulet"
			genericType := itemType
			if idx := strings.LastIndex(itemType, " "); idx != -1 {
				genericType = itemType[idx+1:]
			}
			searchName = fmt.Sprintf("%s %s", item.CraftedType, genericType)
			log.Printf("🔨 Crafted item detected, searching for: %s (base: %s)", searchName, itemType)
		} else {
			log.Printf("🔍 Rare/Magic item detected (%s), searching by base type: %s", item.Quality, itemType)
		}
		
		if searchName != "" {
			tItem, found := a.itemList.FindItemByName(searchName)
			if found {
				log.Printf("✅ Found matching Traderie item: %s (ID: %s)", tItem.Name, tItem.ID)
				return tItem, true
			}
		}
		
		// If crafted type search failed, try just the base type as fallback
		if item.Quality == "Crafted" && item.CraftedType != "" && itemType != "" {
			log.Printf("⚠ Crafted item '%s' not found, trying base type: %s", searchName, itemType)
			tItem, found := a.itemList.FindItemByName(itemType)
			if found {
				log.Printf("✅ Found matching Traderie item by base type: %s (ID: %s)", tItem.Name, tItem.ID)
				return tItem, true
			}
		}
		
		// For rare/magic/crafted items, if type match fails, fall through to fuzzy matching
		log.Printf("⚠ Type '%s' not found, trying fuzzy match", searchName)
	} else {
		// For Unique, Set, Runeword, Normal items - try exact name match first
		tItem, found := a.itemList.FindItemByName(itemName)
		if found {
			log.Printf("✅ Found matching Traderie item by name: %s (ID: %s)", tItem.Name, tItem.ID)
			return tItem, true
		}

		// Try exact type match as fallback
		if itemType != "" {
			log.Printf("⚠ Item '%s' not found, trying fallback to type '%s'", itemName, itemType)
			tItem, found = a.itemList.FindItemByName(itemType)
			if found {
				log.Printf("✅ Found matching Traderie item by type: %s (ID: %s)", tItem.Name, tItem.ID)
				return tItem, true
			}
		}
	}

	// 3. Try fuzzy/partial matching
	log.Printf("⚠ Fallback failed, trying fuzzy match for '%s' or '%s'", itemName, itemType)
	iName := strings.ToLower(itemName)
	iType := strings.ToLower(itemType)

	// List of crafted item prefixes to exclude from fuzzy matching for non-crafted items
	craftedPrefixes := []string{"blood ", "caster ", "hit power ", "safety "}

	for i := range a.itemList.Items {
		tName := strings.ToLower(a.itemList.Items[i].Name)

		// Skip crafted items in fuzzy matching if:
		// 1. Source is rare/magic (to prevent "Gloves" from matching "Blood Gloves")
		// 2. Source is crafted but type couldn't be determined (empty CraftedType)
		//    This prevents unidentified crafted items from fuzzy matching to wrong crafted types
		shouldSkipCrafted := (item.Quality == "Rare" || item.Quality == "Magic") ||
			(item.Quality == "Crafted" && item.CraftedType == "")

		if shouldSkipCrafted {
			isCraftedItem := false
			for _, prefix := range craftedPrefixes {
				if strings.HasPrefix(tName, prefix) {
					isCraftedItem = true
					break
				}
			}
			if isCraftedItem {
				log.Printf("⚠️ Skipping crafted item '%s' in fuzzy match (source: %s, craftedType: '%s')", 
					a.itemList.Items[i].Name, item.Quality, item.CraftedType)
				continue // Skip this crafted item
			}
		}

		// Check if Traderie item name is contained in the item name or vice versa
		// Skip very short names to avoid noise (e.g. "Amu" matching "Amulet")
		if (len(tName) > 3 && (strings.Contains(iName, tName) || strings.Contains(iType, tName))) ||
			(len(iType) > 2 && strings.Contains(tName, iType)) {
			log.Printf("✅ Found partial match: %s (ID: %s)", a.itemList.Items[i].Name, a.itemList.Items[i].ID)
			return &a.itemList.Items[i], true
		}
	}

	return nil, false
}

// handleHotkey is called when the hotkey is pressed
func (a *App) handleHotkey() {
	// Read item from memory
	item, err := a.memReader.GetHoveredItem()
	if err != nil {
		log.Printf("Failed to read item: %v", err)
		runtime.EventsEmit(a.ctx, "item-scan-error", err.Error())
		return
	}

	if item == nil {
		log.Println("No item currently hovered or on cursor")
		runtime.EventsEmit(a.ctx, "item-scan-error", "No item hovered")
		return
	}

	log.Printf("✓ Captured item: %s (%s) Type: %s", item.Name, item.Quality, item.Type)

	// Find matching traderie item
	traderieItem, found := a.FindTraderieItem(item)
	if !found {
		log.Printf("❌ Could not find matching Traderie item for %s / %s", item.Name, item.Type)
	}

	// Get Traderie properties for this item
	traderieProperties := []string{}
	commonProperties := map[string]bool{
		"Platform": true,
		"Mode":     true,
		"Ladder":   true,
		"Region":   true,
	}

	// Prepare initial mappings based on saved preferences
	itemMappings := []map[string]string{}

	if traderieItem != nil {
		for _, prop := range traderieItem.Properties {
			if !commonProperties[prop.Property] {
				traderieProperties = append(traderieProperties, prop.Property)
			}
		}
		log.Printf("✓ Loaded %d Traderie properties for item", len(traderieProperties))

		// If it's a rare/magic/crafted item and we found a generic Traderie item,
		// try to pre-map the Rarity property
		tNameLower := strings.ToLower(traderieItem.Name)
		iTypeLower := strings.ToLower(item.Type)
		isGeneric := strings.EqualFold(traderieItem.Name, item.Type) || 
					 (len(iTypeLower) > 2 && strings.Contains(tNameLower, iTypeLower)) ||
					 (len(tNameLower) > 2 && strings.Contains(iTypeLower, tNameLower))
		
		if isGeneric || item.Quality == "Magic" || item.Quality == "Rare" || item.Quality == "Crafted" {
			for _, tProp := range traderieItem.Properties {
				if tProp.Property == "Rarity" {
					// Map D2R Quality to Traderie Rarity options
					traderieRarity := strings.ToLower(item.Quality)
					// "Crafted" usually maps to "rare" or has its own logic in some categories, 
					// but for Amulets/Rings it's often its own option if available.
					// If not available, "rare" is a safe fallback for Crafted items in terms of stats.
					
					// Let's check if the current traderieItem specifically has this option
					hasOption := false
					for _, opt := range tProp.Options {
						if strings.EqualFold(opt, traderieRarity) {
							traderieRarity = opt
							hasOption = true
							break
						}
					}
					
					if !hasOption && traderieRarity == "crafted" {
						traderieRarity = "rare" // Fallback for crafted if not explicitly listed
					}

					itemMappings = append(itemMappings, map[string]string{
						"d2rProp":      fmt.Sprintf("Quality: %s", item.Quality),
						"traderieProp": "Rarity",
					})
					
					// We should also store the actual mapped value somewhere so the frontend can pre-select it.
					// The property mappings currently only store the property NAME mapping.
					// The mapper.MapItemToTraderie handles the actual value mapping.
					break
				}
			}
		}
	}

	for _, prop := range item.Properties {
		d2rPropStr := fmt.Sprintf("%s: %v", prop.Name, prop.Value)
		
		// 1. Try persistent learned mappings first
		mapping, found := a.propertyMapper.GetMapping(prop.Name, "")
		if found {
			itemMappings = append(itemMappings, map[string]string{
				"d2rProp":      d2rPropStr,
				"traderieProp": mapping,
			})
		} else {
			// 2. Fallback to automatic matching logic
			autoMapping := a.traderieClient.MapPropertyName(prop.Name, "")
			if autoMapping != "" {
				itemMappings = append(itemMappings, map[string]string{
					"d2rProp":      d2rPropStr,
					"traderieProp": autoMapping,
				})
			} else {
				// 3. Just add the D2R property with empty Traderie mapping so user can select
				itemMappings = append(itemMappings, map[string]string{
					"d2rProp":      d2rPropStr,
					"traderieProp": "",
				})
			}
		}
	}

	// Send item to frontend
	runtime.EventsEmit(a.ctx, "item-scanned", map[string]interface{}{
		"item":               item,
		"traderieProperties": traderieProperties,
		"mappings":           itemMappings,
	})
}

// GetAllItems returns all items from the Traderie list for autocomplete
func (a *App) GetAllItems() []string {
	if a.itemList == nil {
		log.Println("⚠️ GetAllItems called but itemList is nil!")
		return []string{}
	}

	items := make([]string, 0, len(a.itemList.Items))
	for _, item := range a.itemList.Items {
		items = append(items, item.Name)
	}
	log.Printf("✓ GetAllItems returning %d items", len(items))
	return items
}

// GetTradingOptions returns the saved trading options
func (a *App) GetTradingOptions() map[string]interface{} {
	return map[string]interface{}{
		"platform":            a.config.Traderie.Platform,
		"mode":                a.config.Traderie.Mode,
		"ladder":              a.config.Traderie.Ladder,
		"region":              a.config.Traderie.Region,
		"autoRefreshEnabled":  a.config.Traderie.AutoRefreshEnabled,
		"autoRefreshInterval": a.config.Traderie.AutoRefreshInterval,
		"searchRange":         a.config.Traderie.SearchRange,
	}
}

// PostItem posts an item to Traderie
func (a *App) PostItem(item *models.Item, platform string, tradingOpts map[string]interface{}, pricingOpts map[string]interface{}) error {
	log.Println("Posting item to Traderie...")
	log.Printf("Item: %s", item.Name)

	// Learn any mappings provided from the UI
	if mappings, ok := tradingOpts["mappings"].([]interface{}); ok {
		for _, m := range mappings {
			if mapping, ok := m.(map[string]interface{}); ok {
				d2rProp, _ := mapping["d2rProp"].(string)
				traderieProp, _ := mapping["traderieProp"].(string)
				
				if d2rProp != "" && traderieProp != "" {
					// Clean up the name (e.g. "to Life: 50" -> "to Life")
					cleanD2R := d2rProp
					if idx := strings.Index(d2rProp, ":"); idx != -1 {
						cleanD2R = strings.TrimSpace(d2rProp[:idx])
					}
					
					// Learn in the active client
					a.traderieClient.LearnMapping(cleanD2R, traderieProp)
					// Also save to the persistent mapper
					a.propertyMapper.LearnMapping(cleanD2R, traderieProp, "")
				}
			}
		}
		// Save mappings to disk
		_ = a.propertyMapper.Save()
	}

	// Find the Traderie Item ID
	tItem, found := a.FindTraderieItem(item)
	if !found {
		return fmt.Errorf("could not find item '%s' (or type '%s') in Traderie database. Please ensure the name matches Traderie's exactly.", item.Name, item.Type)
	}

	// Extract offer items from pricing options
	currencyGroupPrices := []models.CurrencyGroupPrice{}
	if offers, ok := pricingOpts["offers"].([]interface{}); ok {
		for _, o := range offers {
			if offer, ok := o.(map[string]interface{}); ok {
				group := models.CurrencyGroupPrice{
					Items: []models.PriceItem{},
				}

				if items, ok := offer["items"].([]interface{}); ok {
					for _, it := range items {
						if itemObj, ok := it.(map[string]interface{}); ok {
							itemName, _ := itemObj["itemName"].(string)
							quantityVal, _ := itemObj["quantity"].(float64)
							quantity := int(quantityVal)
							if quantity <= 0 {
								quantity = 1
							}

							if itemName != "" {
								// Find the offer item in our database to get its ID and Type
								if otItem, found := a.itemList.FindItemByName(itemName); found {
									priceItem := models.PriceItem{
										Quantity: quantity,
										Item:     otItem.ID,
										ItemType: otItem.Type,
									}
									group.Items = append(group.Items, priceItem)
								} else {
									// If not found in our database, we shouldn't send it as it will cause "Invalid pricing"
									log.Printf("⚠️ Warning: Price item '%s' not found in database, skipping to avoid API error", itemName)
								}
							}
						}
					}
				}

				if len(group.Items) > 0 {
					currencyGroupPrices = append(currencyGroupPrices, group)
				}
			}
		}
	}

	// Prepare manual mappings for the client
	var manualMappings []map[string]interface{}
	if mappings, ok := tradingOpts["mappings"].([]interface{}); ok {
		for _, m := range mappings {
			if mapping, ok := m.(map[string]interface{}); ok {
				manualMappings = append(manualMappings, mapping)
			}
		}
	}

	// Use configured values if not provided in opts
	p := platform
	if p == "" {
		p = a.config.Traderie.Platform
	}
	m := "softcore"
	if val, ok := tradingOpts["mode"].(string); ok {
		m = val
	}
	l := true
	if val, ok := tradingOpts["ladder"].(bool); ok {
		l = val
	}
	r := "Americas"
	if val, ok := tradingOpts["region"].(string); ok {
		r = val
	}

	// Extract pricing options
	makeOffer := true
	if val, ok := pricingOpts["askForOffers"].(bool); ok {
		makeOffer = val
	}

	// Safety check: If no specific prices are provided, we MUST set makeOffer to true
	// otherwise the Traderie API will return "Invalid pricing"
	hasValidPrices := false
	for _, group := range currencyGroupPrices {
		if len(group.Items) > 0 {
			hasValidPrices = true
			break
		}
	}

	if !hasValidPrices {
		makeOffer = true
		log.Println("ℹ️ No valid specific prices provided, forcing makeOffer=true to avoid API error")
	}

	err := a.traderieClient.PostItem(
		item,
		tItem,
		p,
		m,
		l,
		r,
		currencyGroupPrices,
		manualMappings,
		makeOffer,
		a.itemList,
	)
	if err != nil {
		log.Printf("❌ Failed to post item: %v", err)
		return err
	}

	log.Println("✅ Item posted successfully!")
	return nil
}

// GetPropertyMapping returns a saved mapping for a D2R property
func (a *App) GetPropertyMapping(d2rProp string) string {
	// Clean up the name (e.g. "Max Life: 50" -> "Max Life")
	cleanD2R := d2rProp
	if idx := strings.Index(d2rProp, ":"); idx != -1 {
		cleanD2R = strings.TrimSpace(d2rProp[:idx])
	}

	mapping, _ := a.propertyMapper.GetMapping(cleanD2R, "")
	return mapping
}

// SavePropertyMappings saves the current manual mappings to the persistent store
func (a *App) SavePropertyMappings(mappings []map[string]interface{}) error {
	log.Println("Saving manual property mappings...")
	for _, mapping := range mappings {
		d2rProp, _ := mapping["d2rProp"].(string)
		traderieProp, _ := mapping["traderieProp"].(string)
		if d2rProp != "" && traderieProp != "" {
			// Clean up the name (e.g. "to Life: 50" -> "to Life")
			cleanD2R := d2rProp
			if idx := strings.Index(d2rProp, ":"); idx != -1 {
				cleanD2R = strings.TrimSpace(d2rProp[:idx])
			}
			
			// Learn in the active client
			a.traderieClient.LearnMapping(cleanD2R, traderieProp)
			// Also save to the persistent mapper
			a.propertyMapper.LearnMapping(cleanD2R, traderieProp, "")
		}
	}
	return a.propertyMapper.Save()
}

// SetAuthToken updates the Traderie auth token/API key
func (a *App) SetAuthToken(token string) error {
	log.Println("Updating Traderie auth token...")
	a.config.Traderie.APIKey = token
	
	// Only fall back to standard client if we don't have cookies saved
	// If we HAVE cookies, we should keep using the CloudflareClient
	if !a.cookieManager.HasSavedCookies() {
		a.traderieClient = api.NewClient(token)
		log.Println("Using standard HTTP client with new token")
	} else {
		log.Println("Token saved, but keeping existing Cloudflare bypass active")
	}
	
	return a.config.Save()
}

// GetAuthToken returns the current auth token
func (a *App) GetAuthToken() string {
	return a.config.Traderie.APIKey
}

// SetupCookies handles cookie setup from frontend
func (a *App) SetupCookies(cookieString string) error {
	log.Println("Setting up Traderie cookies...")
	
	if cookieString == "" {
		return fmt.Errorf("cookie string is empty")
	}

	// Parse cookies
	cookies, err := a.cookieManager.ParseCookieString(cookieString, "traderie.com")
	if err != nil {
		log.Printf("❌ Failed to parse cookies: %v", err)
		return fmt.Errorf("failed to parse cookies: %w", err)
	}

	log.Printf("✅ Parsed %d cookies", len(cookies))

	// Save cookies
	if err := a.cookieManager.SaveCookies(cookies); err != nil {
		log.Printf("❌ Failed to save cookies: %v", err)
		return fmt.Errorf("failed to save cookies: %w", err)
	}

	log.Println("✅ Cookies saved successfully!")

	// Test connection
	client := api.NewCloudflareClient(cookies, a.config.Traderie.APIKey, a.bridge)
	if err := client.TestConnection(); err != nil {
		log.Printf("⚠️ Connection test failed: %v", err)
		return fmt.Errorf("connection test failed: %w", err)
	}

	log.Println("✅ Connection test successful!")
	
	// Update the app's client to use Cloudflare bypass
	a.traderieClient = client
	
	return nil
}

// TestConnection tests the current Traderie connection
func (a *App) TestConnection() error {
	log.Println("Testing Traderie connection...")
	
	if a.traderieClient == nil {
		return fmt.Errorf("no Traderie client initialized")
	}

	err := a.traderieClient.TestConnection()
	if err != nil {
		log.Printf("❌ Connection test failed: %v", err)
		return err
	}

	log.Println("✅ Connection test successful!")
	return nil
}

// HasSavedCookies checks if cookies are already saved
func (a *App) HasSavedCookies() bool {
	return a.cookieManager.HasSavedCookies()
}

// GetCookieSetupInstructions returns instructions for setting up cookies
func (a *App) GetCookieSetupInstructions() string {
	return `To setup Cloudflare bypass:

1. Open https://traderie.com/d2r in your browser
2. Make sure you're logged in
3. Press F12 to open DevTools
4. Go to the Console tab
5. Type: copy(document.cookie)
6. Press Enter (cookies are now in clipboard)
7. Paste the cookies into the input field below
8. Click "Save Cookies"

Your cookies will be securely stored and the bot will use them to bypass Cloudflare protection.`
}

// GenerateSearchURL creates a Traderie search URL for an item with fuzzy property ranges
func (a *App) GenerateSearchURL(item *models.Item, searchRange int, propertyMappings []map[string]string, excludedProps []string, opts map[string]interface{}) (string, error) {
	log.Printf("Generating search URL for %s with %d%% range", item.Name, searchRange)

	tItem, found := a.FindTraderieItem(item)
	if !found {
		return "", fmt.Errorf("item '%s' (or type '%s') not found in Traderie database", item.Name, item.Type)
	}

	// Extract options or use defaults
	platform := a.config.Traderie.Platform
	if v, ok := opts["platform"].(string); ok && v != "" {
		platform = v
	}
	// Traderie expects "PC" not "pc"
	if strings.EqualFold(platform, "pc") {
		platform = "PC"
	}

	mode := a.config.Traderie.Mode
	if v, ok := opts["mode"].(string); ok && v != "" {
		mode = v
	}

	ladder := "true"
	if v, ok := opts["ladder"].(bool); ok {
		if !v {
			ladder = "false"
		}
	} else {
		// Use config if not in opts
		if !a.config.Traderie.Ladder {
			ladder = "false"
		}
	}

	// Region. Default to all if not specific
	region := ""
	if v, ok := opts["region"].(string); ok && v != "" && v != "Any" && !strings.Contains(v, ",") {
		region = v
	}

	baseURL := fmt.Sprintf("https://traderie.com/diablo2resurrected/product/%s?", tItem.ID)
	params := []string{}
	
	if platform != "" && !strings.EqualFold(platform, "any") {
		params = append(params, fmt.Sprintf("prop_Platform=%s", platform))
	}
	if mode != "" && !strings.EqualFold(mode, "any") {
		params = append(params, fmt.Sprintf("prop_Mode=%s", mode))
	}
	params = append(params, fmt.Sprintf("prop_Ladder=%s", ladder))
	
	if region != "" {
		params = append(params, fmt.Sprintf("prop_Region=%s", region))
	}

	if item.IsEthereal {
		params = append(params, "prop_Ethereal=true")
	}

	// Helper to check if property is excluded
	isExcluded := func(propName string) bool {
		for _, p := range excludedProps {
			if p == propName {
				return true
			}
		}
		return false
	}

	// Iterate over the mappings provided from the UI instead of raw item properties
	for _, mapping := range propertyMappings {
		d2rPropStr := mapping["d2rProp"]
		traderiePropName := mapping["traderieProp"]

		if traderiePropName == "" || traderiePropName == "Rarity" {
			continue
		}

		// Extract the actual property name from d2rProp (e.g. "to All Resistances: 15" -> "to All Resistances")
		cleanD2R := d2rPropStr
		if idx := strings.Index(d2rPropStr, ":"); idx != -1 {
			cleanD2R = strings.TrimSpace(d2rPropStr[:idx])
		}

		if isExcluded(cleanD2R) {
			continue
		}

		// Find the property ID in tItem
		var tProp *traderie.TraderieProperty
		for i := range tItem.Properties {
			if tItem.Properties[i].Property == traderiePropName {
				tProp = &tItem.Properties[i]
				break
			}
		}

		if tProp == nil {
			continue
		}

		// Add property to URL
		if tProp.Type == "number" {
			// Extract value from d2rPropStr
			var val float64
			valInterface := a.extractNumericValue(d2rPropStr)
			if v, ok := valInterface.(int); ok {
				val = float64(v)
			} else if v, ok := valInterface.(float64); ok {
				val = v
			} else {
				continue
			}

			minVal := int(val * (1.0 - float64(searchRange)/100.0))
			maxVal := int(val * (1.0 + float64(searchRange)/100.0))

			// If it's something like sockets or skills, we might want exact min/max or just min
			params = append(params, fmt.Sprintf("prop_%dMin=%d", tProp.PropertyID, minVal))
			params = append(params, fmt.Sprintf("prop_%dMax=%d", tProp.PropertyID, maxVal))
		} else if tProp.Type == "string" {
			// For string properties (like sockets), we probably want exact match
			valInterface := a.extractNumericValue(d2rPropStr)
			if valInterface != nil {
				params = append(params, fmt.Sprintf("prop_%d=%v", tProp.PropertyID, valInterface))
			}
		}
	}

	return baseURL + strings.Join(params, "&"), nil
}

// extractNumericValue helper to get a number from a property string
func (a *App) extractNumericValue(s string) interface{} {
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
		if (s[i] >= '0' && s[i] <= '9') {
			var intVal int
			fmt.Sscanf(s[i:], "%d", &intVal)
			return intVal
		}
	}
	return nil
}

// OpenURLInExtension tells the browser extension to open a specific URL in a new tab
func (a *App) OpenURLInExtension(url string) error {
	log.Printf("Requesting extension to open URL: %s", url)
	if a.bridge == nil {
		return fmt.Errorf("extension bridge not initialized")
	}

	cmdID := a.bridge.AddCommand("open_tab", map[string]interface{}{
		"url": url,
	})

	// We don't necessarily need to wait for a result here as it's a fire-and-forget UI action,
	// but we'll do a quick check to ensure the command was at least queued.
	result, err := a.bridge.WaitForResult(cmdID, 30*time.Second)
	if err != nil {
		log.Printf("⚠️ Extension timed out for open_tab, but URL was queued: %v", err)
		// Fallback: If extension is not responding, we could try opening it from Go,
		// but that might open in the default browser which might not have the extension/cookies.
		return fmt.Errorf("extension is not responding (timeout). Please make sure the Traderie extension is installed and active in your browser.")
	}

	if !result.Success {
		return fmt.Errorf("extension failed to open URL: %s", result.Error)
	}

	return nil
}

// StartAutoRefresh starts or restarts the auto-refresh timer
func (a *App) StartAutoRefresh() {
	a.StopAutoRefresh()

	if !a.config.Traderie.AutoRefreshEnabled {
		return
	}

	interval := time.Duration(a.config.Traderie.AutoRefreshInterval) * time.Minute
	// Enforce 1h min, 24h max as per user request
	if interval < 1*time.Hour {
		interval = 1 * time.Hour
	}
	if interval > 24*time.Hour {
		interval = 24 * time.Hour
	}

	log.Printf("Starting auto-refresh timer: every %v", interval)
	a.refreshTicker = time.NewTicker(interval)
	a.refreshStop = make(chan struct{})

	go func() {
		for {
			select {
			case <-a.refreshTicker.C:
				log.Println("Auto-refresh triggered...")
				a.RefreshListings()
			case <-a.refreshStop:
				return
			}
		}
	}()
}

// StopAutoRefresh stops the auto-refresh timer
func (a *App) StopAutoRefresh() {
	if a.refreshTicker != nil {
		a.refreshTicker.Stop()
		close(a.refreshStop)
		a.refreshTicker = nil
		a.refreshStop = nil
		log.Println("Auto-refresh timer stopped")
	}
}

// RefreshListings triggers a listings refresh via the extension
func (a *App) RefreshListings() error {
	log.Println("Refreshing all listings...")
	if a.bridge == nil {
		return fmt.Errorf("extension bridge not initialized")
	}

	cmdID := a.bridge.AddCommand("refresh_listings", map[string]interface{}{
		"baseURL": "https://traderie.com",
	})

	result, err := a.bridge.WaitForResult(cmdID, 1*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to get refresh result from extension: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("extension failed to refresh listings: %s", result.Error)
	}

	log.Println("✅ Listings refreshed successfully!")
	runtime.EventsEmit(a.ctx, "listings-refreshed", true)
	return nil
}

// SaveTradingOptions saves the trading options for future use
func (a *App) SaveTradingOptions(opts map[string]interface{}) error {
	// Update config
	if platform, ok := opts["platform"].(string); ok {
		a.config.Traderie.Platform = platform
	}
	if mode, ok := opts["mode"].(string); ok {
		a.config.Traderie.Mode = mode
	}
	if ladder, ok := opts["ladder"].(bool); ok {
		a.config.Traderie.Ladder = ladder
	}
	if region, ok := opts["region"].(string); ok {
		a.config.Traderie.Region = region
	}
	if enabled, ok := opts["autoRefreshEnabled"].(bool); ok {
		a.config.Traderie.AutoRefreshEnabled = enabled
	}
	if interval, ok := opts["autoRefreshInterval"].(float64); ok {
		a.config.Traderie.AutoRefreshInterval = int(interval)
	}
	if rangeVal, ok := opts["searchRange"].(float64); ok {
		a.config.Traderie.SearchRange = int(rangeVal)
	}

	// Update the auto-refresh timer if settings changed
	a.StartAutoRefresh()

	// Save config
	return a.config.Save()
}
