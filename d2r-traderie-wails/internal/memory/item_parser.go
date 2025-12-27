package memory

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ParsePropertyValue attempts to extract numeric value from property text
func ParsePropertyValue(text string) (int, error) {
	// Find first number in the string
	re := regexp.MustCompile(`[-+]?\d+`)
	match := re.FindString(text)
	if match == "" {
		return 0, fmt.Errorf("no numeric value found in: %s", text)
	}

	value, err := strconv.Atoi(match)
	if err != nil {
		return 0, err
	}

	return value, nil
}

// ParsePropertyRange extracts min-max range from property text
func ParsePropertyRange(text string) (min int, max int, err error) {
	// Look for patterns like "10-20", "10 to 20", "10 - 20"
	re := regexp.MustCompile(`(\d+)\s*(?:-|to)\s*(\d+)`)
	matches := re.FindStringSubmatch(text)
	
	if len(matches) < 3 {
		return 0, 0, fmt.Errorf("no range found in: %s", text)
	}

	min, err = strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, err
	}

	max, err = strconv.Atoi(matches[2])
	if err != nil {
		return 0, 0, err
	}

	return min, max, nil
}

// NormalizePropertyName standardizes property names for mapping
func NormalizePropertyName(name string) string {
	// Remove special characters and extra spaces
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	
	// Remove parentheses and brackets
	name = strings.ReplaceAll(name, "(", "")
	name = strings.ReplaceAll(name, ")", "")
	name = strings.ReplaceAll(name, "[", "")
	name = strings.ReplaceAll(name, "]", "")
	
	// Normalize multiple spaces to single space
	re := regexp.MustCompile(`\s+`)
	name = re.ReplaceAllString(name, " ")
	
	return name
}

// IsSkillProperty checks if a property is a skill bonus
func IsSkillProperty(name string) bool {
	name = strings.ToLower(name)
	return strings.Contains(name, "to ") && 
		(strings.Contains(name, "skill") || 
		 strings.Contains(name, "barbarian") ||
		 strings.Contains(name, "necromancer") ||
		 strings.Contains(name, "paladin") ||
		 strings.Contains(name, "sorceress") ||
		 strings.Contains(name, "amazon") ||
		 strings.Contains(name, "druid") ||
		 strings.Contains(name, "assassin"))
}

// IsResistanceProperty checks if a property is a resistance
func IsResistanceProperty(name string) bool {
	name = strings.ToLower(name)
	return strings.Contains(name, "resist") ||
		strings.Contains(name, "fire") ||
		strings.Contains(name, "cold") ||
		strings.Contains(name, "lightning") ||
		strings.Contains(name, "poison")
}

// GetCharacterClass extracts character class from skill name
func GetCharacterClass(skillText string) string {
	text := strings.ToLower(skillText)
	
	classes := []string{
		"barbarian", "necromancer", "paladin", "sorceress",
		"amazon", "druid", "assassin",
	}
	
	for _, class := range classes {
		if strings.Contains(text, class) {
			return strings.Title(class)
		}
	}
	
	return ""
}

