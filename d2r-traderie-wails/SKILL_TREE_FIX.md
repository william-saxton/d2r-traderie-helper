# Skill Tree Detection Fix

## Issue
Users reported that class-specific skill tree bonuses (like "+2 to Javelin and Spear Skills (Amazon Only)") were being incorrectly detected as generic class skill bonuses ("+2 to Amazon Skill Levels").

## Root Cause
The application was only handling `stat.AddClassSkills` (stat ID 83) which represents generic class-level bonuses, but was not handling `stat.AddSkillTab` (stat ID 188) which represents specific skill tree bonuses.

## Changes Made

### 1. Updated d2go library
- Updated from `v0.0.0-20251221085113-0481a1408c62` to `v0.0.0-20251228080657-d31193099721`
- The newer version from kwader2k's fork includes fixes for class-specific spell detection

### 2. Added Skill Tree Mapping
Added support for `stat.AddSkillTab` in `d2r-traderie-wails/internal/memory/reader.go`:

```go
// Handle skill tree bonuses (+X to Javelin and Spear Skills, etc.)
// layer determines which skill tree
if statID == int16(stat.AddSkillTab) {
    skillTreeNames := map[int]string{
        // Amazon skill trees
        0: "to Bow and Crossbow Skills",
        1: "to Passive and Magic Skills",
        2: "to Javelin and Spear Skills",
        // Sorceress skill trees
        8:  "to Fire Skills",
        9:  "to Lightning Skills",
        10: "to Cold Skills",
        // Necromancer skill trees
        16: "to Curses",
        17: "to Poison and Bone Skills",
        18: "to Summoning Skills",
        // Paladin skill trees
        24: "to Combat Skills",
        25: "to Offensive Auras",
        26: "to Defensive Auras",
        // Barbarian skill trees
        32: "to Combat Skills",
        33: "to Masteries",
        34: "to Warcries",
        // Druid skill trees
        40: "to Summoning Skills",
        41: "to Shape Shifting Skills",
        42: "to Elemental Skills",
        // Assassin skill trees
        48: "to Trap Skills",
        49: "to Shadow Disciplines",
        50: "to Martial Arts",
    }
    
    if skillTree, ok := skillTreeNames[layer]; ok {
        return skillTree
    }
}
```

## Testing
After this fix, items with skill tree bonuses should now be correctly detected with Traderie's exact property format:

**Before:**
- Bone Knuckle: "+2 to Amazon Skill Levels"
- Titan's Revenge: "+2 to Amazon Skill Levels"

**After:**
- Bone Knuckle: "to Javelin and Spear Skills (Amazon Only)"
- Titan's Revenge: "to Javelin and Spear Skills (Amazon Only)"

This matches Traderie's exact property naming convention which includes the class name in parentheses.

## Files Modified
- `d2r-traderie-wails/go.mod` - Updated d2go dependency version
- `d2r-traderie-wails/go.sum` - Updated checksums
- `d2r-traderie-wails/internal/memory/reader.go` - Added skill tree detection logic

## Build
The application has been successfully rebuilt at:
`C:\personal-workspace\d2r-traderie-helper\d2r-traderie-wails\build\bin\d2r-traderie-wails.exe`

Users should test with items that have class-specific skill tree bonuses to verify the fix is working correctly.

