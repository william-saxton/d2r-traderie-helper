package memory

import (
	"fmt"
	"log"
	"strings"
	"syscall"
	"unsafe"

	"github.com/hectorgimenez/d2go/pkg/data"
	"github.com/hectorgimenez/d2go/pkg/data/stat"
	"github.com/hectorgimenez/d2go/pkg/memory"
	"github.com/yourusername/d2r-traderie-wails/pkg/models"
	"golang.org/x/sys/windows"
)

// Reader handles reading item data from D2R game memory
type Reader struct {
	gameReader *memory.GameReader
}

// NewReader creates a new memory reader
func NewReader() (*Reader, error) {
	// Use d2go's built-in process finder - it searches all processes for d2r.exe
	log.Println("Searching for D2R process using d2go...")
	process, err := memory.NewProcess()
	if err != nil {
		return nil, fmt.Errorf("failed to find D2R process: %w", err)
	}

	log.Printf("✅ Successfully attached to D2R (PID: %d)", process.GetPID())
	
	gr := memory.NewGameReader(process)

	log.Println("Memory reader initialized")
	return &Reader{
		gameReader: gr,
	}, nil
}

// findD2RProcess finds the D2R process by enumerating windows
func findD2RProcess() (uint32, error) {
	var foundPID uint32

	// Load user32.dll functions
	user32 := windows.NewLazySystemDLL("user32.dll")
	getWindowTextW := user32.NewProc("GetWindowTextW")

	cb := syscall.NewCallback(func(hwnd windows.HWND, lParam uintptr) uintptr {
		var pid uint32
		windows.GetWindowThreadProcessId(hwnd, &pid)

		// Get window title
		titleBuf := make([]uint16, 256)
		getWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&titleBuf[0])), 256)
		windowTitle := syscall.UTF16ToString(titleBuf)

		// Check if this is D2R window
		if windowTitle == "Diablo II: Resurrected" {
			foundPID = pid
			return 0 // Stop enumeration
		}

		return 1 // Continue enumeration
	})

	windows.EnumWindows(cb, unsafe.Pointer(nil))

	if foundPID == 0 {
		return 0, fmt.Errorf("D2R window not found - make sure the game is running and you're in-game")
	}

	return foundPID, nil
}

// GetHoveredItem reads the currently hovered OR cursor-held item from memory
func (r *Reader) GetHoveredItem() (*models.Item, error) {
	// Get game data
	gameData := r.gameReader.GetData()

	// OPTION 1: Check for item on cursor (picked up) - THIS WORKS!
	// This is a reliable workaround since D2R tracks cursor items perfectly
	for i := range gameData.Inventory.AllItems {
		item := &gameData.Inventory.AllItems[i]
		
		// Check if item is on cursor (LocationCursor)
		if item.Location.LocationType == "cursor" {
			log.Printf("✓ Found item on cursor: %s", item.Name)
			
			// Parse the item from d2go format to our model
			parsedItem := r.parseItem(item)
			return parsedItem, nil
		}
	}

	// OPTION 2: Check for hovered item on ground (this works for ground items only)
	if gameData.HoverData.IsHovered {
		for i := range gameData.Inventory.AllItems {
			item := &gameData.Inventory.AllItems[i]
			
			if item.UnitID == gameData.HoverData.UnitID {
				log.Printf("✓ Found hovered item on ground: %s", item.Name)
				
				// Parse the item from d2go format to our model
				parsedItem := r.parseItem(item)
				return parsedItem, nil
			}
		}
	}

	// No item found
	return nil, fmt.Errorf("no item found - either pick up the item (left-click) and press F9, or hover over ground item and press F9")
}

// parseItem converts d2go item to our internal model
func (r *Reader) parseItem(d2item *data.Item) *models.Item {
	if d2item == nil {
		return nil
	}

	// Determine the item name: use IdentifiedName for unique/set items, otherwise use base name
	itemName := string(d2item.Name)
	quality := d2item.Quality.ToString()
	
	// For Unique, Set, or identified Rare items, use the identified name if available
	if (quality == "Unique" || quality == "Set" || quality == "Rare") && d2item.IdentifiedName != "" {
		itemName = d2item.IdentifiedName
		log.Printf("DEBUG: Using identified name for %s item: %s", quality, itemName)
	} else {
		log.Printf("DEBUG: Using base name: %s (%s)", itemName, quality)
	}

	// Detect crafted item type if applicable
	craftedType := ""
	if quality == "Crafted" {
		craftedType = r.detectCraftedType(d2item)
		if craftedType != "" {
			log.Printf("🔨 Detected crafted type: %s", craftedType)
		} else {
			log.Printf("⚠️ Could not determine crafted type for item")
		}
	}

	// Count sockets
	socketCount := len(d2item.Sockets)
	if socketCount > 0 {
		log.Printf("🔌 Detected %d socket(s) in item", socketCount)
	}

	item := &models.Item{
		Name:        itemName,
		Type:        r.getItemType(d2item),
		Quality:     quality,
		CraftedType: craftedType,
		Properties:  r.parseProperties(d2item),
		Sockets:     socketCount,
		IsEthereal:  d2item.Ethereal,
		IsIdentified: d2item.Identified,
	}

	return item
}

// getItemType determines the item type category
func (r *Reader) getItemType(item *data.Item) string {
	// For Traderie matching, we need the full base item name like "Leather Gloves"
	// The item.Name field contains this but without spaces (e.g., "LeatherGloves")
	baseName := string(item.Name)
	
	// Add spaces before capital letters to match Traderie format
	// "LeatherGloves" -> "Leather Gloves"
	// "ChainMail" -> "Chain Mail"
	spacedName := addSpacesToCamelCase(baseName)
	
	if spacedName != "" {
		return spacedName
	}
	
	// Fallback to itemType if available
	itemType := item.Type()
	if itemType.Name != "" {
		return itemType.Name
	}
	
	if itemType.Code != "" {
		return itemType.Code
	}
	
	// Last resort: return the base name as-is
	return baseName
}

// getGenericItemType returns the generic type (e.g., "Gloves", "Amulet", "Ring")
// This is used for crafted items where Traderie uses generic types like "Caster Gloves"
func (r *Reader) getGenericItemType(item *data.Item) string {
	itemType := item.Type()
	
	// itemType.Name gives us the generic type like "Gloves", "Amulet", "Ring"
	if itemType.Name != "" {
		return itemType.Name
	}
	
	// Fallback: try to extract generic type from base name
	baseName := string(item.Name)
	spacedName := addSpacesToCamelCase(baseName)
	
	// Extract the last word (e.g., "Leather Gloves" -> "Gloves")
	if idx := strings.LastIndex(spacedName, " "); idx != -1 {
		return spacedName[idx+1:]
	}
	
	return spacedName
}

// addSpacesToCamelCase converts "LeatherGloves" to "Leather Gloves"
func addSpacesToCamelCase(s string) string {
	if s == "" {
		return ""
	}
	
	var result []rune
	runes := []rune(s)
	
	for i, r := range runes {
		// Add space before capital letter if:
		// - It's not the first character
		// - Previous character is lowercase
		// - Current character is uppercase
		if i > 0 && r >= 'A' && r <= 'Z' && runes[i-1] >= 'a' && runes[i-1] <= 'z' {
			result = append(result, ' ')
		}
		result = append(result, r)
	}
	
	return string(result)
}

// detectCraftedType identifies the type of crafted item based on its stats
// Returns: "Caster", "Blood", "Hit Power", "Safety", or "" if can't determine
func (r *Reader) detectCraftedType(item *data.Item) string {
	hasManaRegen := false
	hasLifeSteal := false
	hasCrushingBlow := false
	hasKnockback := false
	hasAttackerTakesDamage := false
	hasDamageReduced := false
	hasMagicDamageReduced := false

	// Check item stats for characteristic properties
	log.Printf("🔍 Analyzing %d stats for crafted type detection...", len(item.Stats))
	for _, s := range item.Stats {
		switch s.ID {
		case stat.ID(27): // Regenerate Mana % (the actual stat for crafted Caster items)
			hasManaRegen = true
			log.Printf("  ✓ Found Mana Regeneration %d%% (stat ID: %d) - indicates CASTER", s.Value, s.ID)
		case stat.ManaRecovery: // Also check the other mana recovery stat as backup
			hasManaRegen = true
			log.Printf("  ✓ Found Mana Recovery (stat ID: %d, value: %d) - indicates CASTER", s.ID, s.Value)
		case stat.LifeSteal:
			hasLifeSteal = true
			log.Printf("  ✓ Found Life Steal %d%% (stat ID: %d) - indicates BLOOD", s.Value, s.ID)
		case stat.CrushingBlow:
			hasCrushingBlow = true
			log.Printf("  ✓ Found Crushing Blow %d%% (stat ID: %d) - indicates BLOOD (gloves)", s.Value, s.ID)
		case stat.ID(119): // Knockback
			hasKnockback = true
			log.Printf("  ✓ Found Knockback (stat ID: %d) - indicates HIT POWER (gloves)", s.ID)
		case stat.ID(78): // Attacker Takes Damage
			hasAttackerTakesDamage = true
			log.Printf("  ✓ Found Attacker Takes Damage %d (stat ID: %d) - indicates HIT POWER", s.Value, s.ID)
		case stat.DamageReduced:
			hasDamageReduced = true
			log.Printf("  ✓ Found Damage Reduced %d (stat ID: %d)", s.Value, s.ID)
		case stat.MagicDamageReduction:
			hasMagicDamageReduced = true
			log.Printf("  ✓ Found Magic Damage Reduced %d (stat ID: %d)", s.Value, s.ID)
		}
	}

	// Determine crafted type based on guaranteed stats from D2R crafting recipes
	// Priority order matters since items can have multiple mods
	
	// Caster: ALWAYS has Regenerate Mana 4-10%
	// Check this first because it's the most reliable indicator
	if hasManaRegen {
		return "Caster"
	}
	
	// Safety: ALWAYS has both Damage Reduced AND Magic Damage Reduced
	// Check this before Blood because this combination is unique to Safety
	if hasDamageReduced && hasMagicDamageReduced {
		return "Safety"
	}
	
	// Hit Power: Has Knockback (gloves) OR Attacker Takes Damage (all)
	// Check this before Blood to avoid confusion with Crushing Blow
	if hasKnockback || hasAttackerTakesDamage {
		return "Hit Power"
	}
	
	// Blood: Has Life Stolen per Hit OR Crushing Blow (gloves specific)
	// Check this last since life steal can appear as random mod
	if hasLifeSteal || hasCrushingBlow {
		return "Blood"
	}

	// If we can't determine, return empty string
	log.Printf("⚠️ No guaranteed crafted stats found - this may not be a true crafted item or has unusual mods")
	log.Printf("   Stats: ManaRegen=%v, LifeSteal=%v, CrushingBlow=%v, Knockback=%v, AttackerTakesDmg=%v, DmgReduced=%v, MagicDmgReduced=%v",
		hasManaRegen, hasLifeSteal, hasCrushingBlow, hasKnockback, hasAttackerTakesDamage, hasDamageReduced, hasMagicDamageReduced)
	return ""
}

// parseProperties extracts all properties/stats from the item and formats them for Traderie
func (r *Reader) parseProperties(item *data.Item) []models.Property {
	var properties []models.Property
	
	// Track resistance values for consolidation
	var fireRes, coldRes, lightRes, poisonRes int
	var hasFireRes, hasColdRes, hasLightRes, hasPoisonRes bool
	
	// Track attribute values for consolidation
	var strength, energy, dexterity, vitality int
	var hasStrength, hasEnergy, hasDexterity, hasVitality bool
	
	// Track damage ranges for consolidation
	var minDmg, maxDmg int
	var hasMinDmg, hasMaxDmg bool
	
	// Map to track which stats we've already processed
	processedStats := make(map[string]bool)

	// First pass: collect resistances, attributes, and damage for consolidation
	for _, s := range item.Stats {
		switch s.ID {
		case stat.FireResist:
			fireRes = s.Value
			hasFireRes = true
		case stat.LightningResist:
			lightRes = s.Value
			hasLightRes = true
		case stat.ColdResist:
			coldRes = s.Value
			hasColdRes = true
		case stat.PoisonResist:
			poisonRes = s.Value
			hasPoisonRes = true
		case stat.Strength:
			strength = s.Value
			hasStrength = true
		case stat.Energy:
			energy = s.Value
			hasEnergy = true
		case stat.Dexterity:
			dexterity = s.Value
			hasDexterity = true
		case stat.Vitality:
			vitality = s.Value
			hasVitality = true
		case stat.MinDamage:
			minDmg = s.Value
			hasMinDmg = true
		case stat.MaxDamage:
			maxDmg = s.Value
			hasMaxDmg = true
		}
	}

	// Check if all resistances are equal (All Res charm)
	if hasFireRes && hasColdRes && hasLightRes && hasPoisonRes &&
		fireRes == coldRes && fireRes == lightRes && fireRes == poisonRes {
		// Consolidate to "to All Resistances" (exact Traderie format)
		properties = append(properties, models.Property{
			Name:  "to All Resistances",
			Value: fireRes,
		})
		processedStats["resist_all"] = true
	}

	// Check if all attributes are equal (All Attributes)
	if hasStrength && hasEnergy && hasDexterity && hasVitality &&
		strength == energy && strength == dexterity && strength == vitality {
		// Consolidate to "to All Attributes"
		properties = append(properties, models.Property{
			Name:  "to All Attributes",
			Value: strength,
		})
		processedStats["attr_all"] = true
	}

	// Consolidate damage range
	if hasMinDmg && hasMaxDmg {
		properties = append(properties, models.Property{
			Name:  "Adds Damage",
			Value: fmt.Sprintf("%d-%d", minDmg, maxDmg),
		})
		processedStats["damage_range"] = true
	}

	// Second pass: add remaining stats with proper Traderie naming
	for _, s := range item.Stats {
		// Skip if we already consolidated this stat
		if processedStats["resist_all"] && (s.ID == stat.FireResist || s.ID == stat.LightningResist || s.ID == stat.ColdResist || s.ID == stat.PoisonResist) {
			continue
		}
		if processedStats["attr_all"] && (s.ID == stat.Strength || s.ID == stat.Energy || s.ID == stat.Dexterity || s.ID == stat.Vitality) {
			continue
		}
		if processedStats["damage_range"] && (s.ID == stat.MinDamage || s.ID == stat.MaxDamage) {
			continue
		}
		
		statName := r.mapStatToTraderie(int16(s.ID), s.Value, s.Layer)
		if statName != "" {
			properties = append(properties, models.Property{
				Name:  statName,
				Value: s.Value,
			})
		} else {
			// Log unknown stats for debugging
			log.Printf("DEBUG: Unknown stat ID=%d, Value=%d, Layer=%d", s.ID, s.Value, s.Layer)
		}
	}

	// Add socket information if present
	if len(item.Sockets) > 0 {
		properties = append(properties, models.Property{
			Name:  "Sockets",
			Value: len(item.Sockets),
		})
	}
	
	// Add ethereal if applicable
	if item.Ethereal {
		properties = append(properties, models.Property{
			Name:  "Ethereal",
			Value: true,
		})
	}

	return properties
}

// mapStatToTraderie maps d2go stat IDs to Traderie property names
func (r *Reader) mapStatToTraderie(statID int16, value int, layer int) string {
	// Common stat mappings to Traderie format
	statMap := map[stat.ID]string{
		// Resistances (individual - only used if not all equal)
		stat.FireResist:      "Fire Resistance",
		stat.LightningResist: "Lightning Resistance",
		stat.ColdResist:      "Cold Resistance",
		stat.PoisonResist:    "Poison Resistance",
		stat.ID(110):         "Poison Length Reduced", // Poison Length Reduced by X%
		
		// Life/Mana
		stat.Life:    "Life",
		stat.Mana:    "Mana",
		stat.MaxLife: "Max Life",
		stat.MaxMana: "Max Mana",
		stat.ID(138): "Mana after each Kill", // +X to Mana after each Kill
		
		// Enhanced Defense
		stat.EnhancedDefense: "Enhanced Defense",
		stat.Defense:         "Defense",
		
		// Magic Find
		stat.MagicFind: "Magic Find",
		stat.GoldFind:  "Gold Find",
		
		// Attack Rating
		stat.AttackRating: "Attack Rating",
		
		// Deadly Strike / Critical Strike
		stat.DeadlyStrike: "Deadly Strike",
		
		// Faster Cast/Hit Recovery
		stat.FasterCastRate:     "Faster Cast Rate",
		stat.FasterHitRecovery:  "Faster Hit Recovery",
		stat.FasterRunWalk:      "Faster Run/Walk",
		stat.FasterBlockRate:    "Faster Block Rate",
		
		// Increased Attack Speed
		stat.IncreasedAttackSpeed: "Increased Attack Speed",
		
		// Attributes (individual - only used if not all equal)
		stat.Strength:  "Strength",
		stat.Energy:    "Energy",
		stat.Dexterity: "Dexterity",
		stat.Vitality:  "Vitality",
		
		// Skills
		stat.AllSkills: "All Skills",
		
		// Light Radius
		stat.ID(151): "Light Radius",
		
		// Replenish Life/Mana
		stat.ReplenishLife: "Replenish Life",
		stat.ID(27):        "Regenerate Mana", // Regenerate Mana % (crafted Caster items)
		stat.ManaRecovery:  "Regenerate Mana",
		
		// Damage Reduction
		stat.DamageReduced:        "Damage Reduced",
		stat.MagicDamageReduction: "Magic Damage Reduced",
		
		// Durability
		stat.MaxDurability:        "Max Durability",
		stat.MaxDurabilityPercent: "Enhanced Durability",
		
		// Cannot Be Frozen
		stat.CannotBeFrozen: "Cannot Be Frozen",
		
		// Life/Mana Steal
		stat.LifeSteal: "Life Leech",
		stat.ManaSteal: "Mana Leech",
		
		// Open Wounds, Crushing Blow, Knockback
		stat.OpenWounds:   "Open Wounds",
		stat.CrushingBlow: "Crushing Blow",
		stat.ID(119):      "Knockback",
		
		// Absorb
		stat.AbsorbFire:      "Fire Absorb",
		stat.AbsorbCold:      "Cold Absorb",
		stat.AbsorbLightning: "Lightning Absorb",
		
		// Attacker Takes Damage (Hit Power crafted items)
		stat.ID(78): "Attacker Takes Damage",
		
		// Rainbow Facet Triggers
		stat.ID(197): "On Death",
		stat.ID(199): "On Level-up",

		// Rainbow Facet Elements
		stat.ID(329): "Fire Skill Damage",
		stat.ID(333): "Enemy Fire Resistance",
		stat.ID(330): "Lightning Skill Damage",
		stat.ID(334): "Enemy Lightning Resistance",
		stat.ID(331): "Cold Skill Damage",
		stat.ID(335): "Enemy Cold Resistance",
		stat.ID(332): "Poison Skill Damage",
		stat.ID(336): "Enemy Poison Resistance",

		// Additional Passive Skills (mostly for informational purposes or future use)
		stat.ID(337): "Critical Strike",
		stat.ID(338): "Dodge",
		stat.ID(339): "Avoid",
		stat.ID(340): "Evade",
		
		// Requirements
		stat.Requirements:  "Requirements",
		stat.LevelRequire:  "Required Level",
	}
	
	if name, ok := statMap[stat.ID(statID)]; ok {
		return name
	}
	
	// Handle class-specific skills (+X to Paladin Skills, etc.)
	// layer is used for class-specific bonuses
	if statID == int16(stat.AddClassSkills) {
		classNames := []string{"Amazon", "Sorceress", "Necromancer", "Paladin", "Barbarian", "Druid", "Assassin"}
		if layer >= 0 && layer < len(classNames) {
			return fmt.Sprintf("+%d to %s Skill Levels", value, classNames[layer])
		}
	}
	
	// Handle skill tree bonuses (e.g., +2 to Fire Skills, +2 to Combat Skills)
	// Stat ID 188 is typically used for skill tab bonuses in D2
	// Layer indicates the skill tree (0-2 for each class's three trees)
	if statID == 188 {
		// Skill tree names by class and tree index
		// Each class has 3 skill trees (index 0, 1, 2)
		skillTreeNames := map[int][]string{
			0: {"Bow and Crossbow Skills", "Passive and Magic Skills", "Javelin and Spear Skills"}, // Amazon
			1: {"Fire Skills", "Lightning Skills", "Cold Skills"},                                    // Sorceress
			2: {"Summoning Skills", "Poison and Bone Skills", "Curses"},                             // Necromancer
			3: {"Combat Skills", "Offensive Auras", "Defensive Auras"},                              // Paladin
			4: {"Combat Skills", "Masteries", "Warcries"},                                           // Barbarian
			5: {"Summoning Skills", "Shape Shifting", "Elemental Skills"},                           // Druid
			6: {"Martial Arts", "Shadow Disciplines", "Traps"},                                      // Assassin
		}
		
		// Layer encodes both class and tree: class*3 + tree
		// Extract class index and tree index from layer
		classIndex := layer / 3
		treeIndex := layer % 3
		
		if trees, ok := skillTreeNames[classIndex]; ok && treeIndex < len(trees) {
			className := []string{"Amazon", "Sorceress", "Necromancer", "Paladin", "Barbarian", "Druid", "Assassin"}[classIndex]
			treeName := trees[treeIndex]
			return fmt.Sprintf("to %s (%s Only)", treeName, className)
		}
	}
	
	// For unknown stats, return empty (we'll skip them)
	return ""
}

// Close cleans up the memory reader
func (r *Reader) Close() error {
	// d2go's GameReader doesn't require explicit cleanup currently
	// but we include this for future-proofing
	log.Println("Memory reader closed")
	return nil
}
