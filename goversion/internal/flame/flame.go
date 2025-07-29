// Package flame provides flame score calculation functionality
package flame

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// StatType represents the main character stats
type StatType string

const (
	STR StatType = "STR"
	DEX StatType = "DEX"
	INT StatType = "INT"
	LUK StatType = "LUK"
)

// FlameStats represents the extracted flame statistics
type FlameStats struct {
	MainStat        int
	SecondaryStat   int
	WeaponAttack    int
	MagicAttack     int
	AllStatPercent  int
	CPIncrease      int  // CP increase value
	HasCPIncrease   bool // Whether CP increase was detected
}

// FlameConfig holds the configuration for flame scoring
type FlameConfig struct {
	MainStat      StatType
	SecondaryStat StatType
}

// ExtractFlameStats extracts flame-related stats from OCR text
func ExtractFlameStats(text string, config *FlameConfig) (*FlameStats, error) {
	stats := &FlameStats{}
	
	lines := strings.Split(strings.ToLower(text), "\n")
	
	for _, line := range lines {
		// Remove spaces around + signs
		line = strings.ReplaceAll(strings.ReplaceAll(line, " +", "+"), "+ ", "+")
		
		// Extract main stat
		if strings.Contains(line, strings.ToLower(string(config.MainStat))) {
			if value := extractNumberAfterPlus(line); value != -1 {
				stats.MainStat = value
			}
		}
		
		// Extract secondary stat  
		if strings.Contains(line, strings.ToLower(string(config.SecondaryStat))) {
			if value := extractNumberAfterPlus(line); value != -1 {
				stats.SecondaryStat = value
			}
		}
		
		// Extract weapon attack
		if strings.Contains(line, "weapon attack") || strings.Contains(line, "weapon att") {
			if value := extractNumberAfterPlus(line); value != -1 {
				stats.WeaponAttack = value
			}
		}
		
		// Extract magic attack
		if strings.Contains(line, "magic attack") || strings.Contains(line, "magic att") {
			if value := extractNumberAfterPlus(line); value != -1 {
				stats.MagicAttack = value
			}
		}
		
		// Extract all stats percentage
		if strings.Contains(line, "all stats") {
			if value := extractPercentageAfterPlus(line); value != -1 {
				stats.AllStatPercent = value
			}
		}
		
		// Extract CP increase (can be positive or negative)
		if strings.Contains(line, "cp increase") {
			value := extractNumberAfterPlusOrMinus(line)
			stats.CPIncrease = value
			stats.HasCPIncrease = true
		}
	}
	
	return stats, nil
}

// extractNumberAfterPlus extracts a number after a + sign from a line
func extractNumberAfterPlus(line string) int {
	re := regexp.MustCompile(`\+(\d+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		if value, err := strconv.Atoi(matches[1]); err == nil {
			return value
		}
	}
	return -1
}

// extractNumberAfterPlusOrMinus extracts a number after a + or - sign from a line
func extractNumberAfterPlusOrMinus(line string) int {
	// Try positive first
	re := regexp.MustCompile(`\+(\d+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		if value, err := strconv.Atoi(matches[1]); err == nil {
			return value
		}
	}
	
	// Try negative
	re = regexp.MustCompile(`-(\d+)`)
	matches = re.FindStringSubmatch(line)
	if len(matches) > 1 {
		if value, err := strconv.Atoi(matches[1]); err == nil {
			return -value // Return negative value
		}
	}
	
	return 0 // Return 0 if no match (different from -1 for other functions)
}

// extractPercentageAfterPlus extracts a percentage number after a + sign from a line
func extractPercentageAfterPlus(line string) int {
	re := regexp.MustCompile(`\+(\d+)%`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		if value, err := strconv.Atoi(matches[1]); err == nil {
			return value
		}
	}
	return -1
}

// CalculateFlameScore calculates the flame score using the formula:
// Main Stat + (Attack × 4) + (All Stat % × 10) + (Secondary Stat ÷ 8)
func CalculateFlameScore(stats *FlameStats, config *FlameConfig) float64 {
	mainStatValue := float64(stats.MainStat)
	
	// Use magic attack for INT classes, weapon attack for others
	var attackValue float64
	if config.MainStat == INT {
		attackValue = float64(stats.MagicAttack) * 4
	} else {
		attackValue = float64(stats.WeaponAttack) * 4
	}
	
	allStatValue := float64(stats.AllStatPercent) * 10
	secondaryStatValue := float64(stats.SecondaryStat) / 8
	
	return mainStatValue + attackValue + allStatValue + secondaryStatValue
}

// FormatFlameScoreBreakdown returns a formatted breakdown of the flame score calculation
func FormatFlameScoreBreakdown(stats *FlameStats, config *FlameConfig, score float64) string {
	var breakdown strings.Builder
	
	breakdown.WriteString("Flame Score Breakdown:\n")
	breakdown.WriteString(fmt.Sprintf("Main Stat (%s): %d\n", config.MainStat, stats.MainStat))
	
	if config.MainStat == INT {
		breakdown.WriteString(fmt.Sprintf("Magic Attack: %d → %.0f\n", stats.MagicAttack, float64(stats.MagicAttack)*4))
	} else {
		breakdown.WriteString(fmt.Sprintf("Weapon Attack: %d → %.0f\n", stats.WeaponAttack, float64(stats.WeaponAttack)*4))
	}
	
	breakdown.WriteString(fmt.Sprintf("All Stat %%: %d%% → %.0f\n", stats.AllStatPercent, float64(stats.AllStatPercent)*10))
	breakdown.WriteString(fmt.Sprintf("Secondary Stat (%s): %d → %.3f\n", config.SecondaryStat, stats.SecondaryStat, float64(stats.SecondaryStat)/8))
	breakdown.WriteString(fmt.Sprintf("Total Flame Score: %.3f", score))
	
	return breakdown.String()
}