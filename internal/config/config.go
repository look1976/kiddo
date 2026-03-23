package config

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Rule represents a single rule entry from the configuration file
// Format: Username, Day-of-week, Start-hour, End-hour
// Example: Janek, Fri, 17-21
type Rule struct {
	Username  string    // Username allowed to log in
	DayOfWeek string    // Day of week (Mon-Sun or abbreviation)
	StartHour int       // Start hour (0-23)
	EndHour   int       // End hour (0-23)
	ParsedAt  time.Time // When this rule was parsed
}

// AppConfig represents the application configuration
type AppConfig struct {
	// GitHub configuration
	GitHubToken string `json:"github_token"` // Personal Access Token (from env or registry)
	GitHubRepo  string `json:"github_repo"`  // Repository in format "owner/repo"
	GitHubBranch string `json:"github_branch"` // Branch name (default: main)
	ConfigFile  string `json:"github_config_file"` // File path in repo (default: schedules.csv)

	// Check settings
	CheckIntervalSeconds int `json:"check_interval_seconds"` // How often to check GitHub (default: 60)
	CleanupIntervalSeconds int `json:"cleanup_interval_seconds"` // How often to clean unauthorized users (default: 300)

	// Logging
	LogLevel string `json:"log_level"` // debug, info, warn, error (default: info)
	LogDir   string `json:"log_dir"`   // Directory for logs

	// Local paths
	ConfigDir  string // Local config directory (C:\ProgramData\Kiddo\)
	CacheFile  string // Cached rules file
	ETAGFile   string // Cached ETAG for change detection
}

// DefaultConfig returns a new config with sensible defaults
func DefaultConfig() *AppConfig {
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = "C:\\ProgramData"
	}
	configDir := filepath.Join(programData, "Kiddo")

	return &AppConfig{
		GitHubBranch: "main",
		ConfigFile: "schedules.csv",
		CheckIntervalSeconds: 60,
		CleanupIntervalSeconds: 300,
		LogLevel: "info",
		LogDir: filepath.Join(configDir, "logs"),
		ConfigDir: configDir,
		CacheFile: filepath.Join(configDir, "rules_cache.csv"),
		ETAGFile: filepath.Join(configDir, ".etag"),
	}
}

// LoadConfig loads configuration from JSON file
func LoadConfig(filePath string) (*AppConfig, error) {
	config := DefaultConfig()

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file doesn't exist, use defaults
			return config, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Get GitHub token from environment variable if not in config
	if config.GitHubToken == "" {
		config.GitHubToken = os.Getenv("KIDDO_GITHUB_TOKEN")
	}

	return config, nil
}

// SaveConfig saves configuration to JSON file
func SaveConfig(filePath string, config *AppConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Set restrictive permissions (admin only)
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ParseRules parses a CSV content into Rule objects
func ParseRules(content string) ([]Rule, error) {
	reader := csv.NewReader(strings.NewReader(content))
	reader.FieldsPerRecord = -1 // Allow variable number of fields
	reader.TrimLeadingSpace = true

	var rules []Rule

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse CSV: %w", err)
		}

		// Skip empty lines and comments
		if len(record) == 0 || (len(record) > 0 && record[0] == "") {
			continue
		}
		if len(record) > 0 && strings.HasPrefix(record[0], "#") {
			continue
		}

		if len(record) < 4 {
			return nil, fmt.Errorf("invalid rule format (need 4 fields): %v", record)
		}

		username := strings.TrimSpace(record[0])
		dayOfWeek := strings.TrimSpace(record[1])
		timeRange := strings.TrimSpace(record[2])
		
		// Parse time range (e.g., "17-21")
		timeParts := strings.Split(timeRange, "-")
		if len(timeParts) != 2 {
			return nil, fmt.Errorf("invalid time range format %q (expected HH-HH): %s", timeRange, username)
		}

		startHour, err := strconv.Atoi(strings.TrimSpace(timeParts[0]))
		if err != nil {
			return nil, fmt.Errorf("invalid start hour %q for user %s: %w", timeParts[0], username, err)
		}

		endHour, err := strconv.Atoi(strings.TrimSpace(timeParts[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid end hour %q for user %s: %w", timeParts[1], username, err)
		}

		if startHour < 0 || startHour > 23 || endHour < 0 || endHour > 23 {
			return nil, fmt.Errorf("hour out of range (0-23) for user %s", username)
		}

		rule := Rule{
			Username:  username,
			DayOfWeek: normalizeDayOfWeek(dayOfWeek),
			StartHour: startHour,
			EndHour:   endHour,
			ParsedAt:  time.Now(),
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

// normalizeDayOfWeek converts day names to full English names
func normalizeDayOfWeek(day string) string {
	dayLower := strings.ToLower(day)
	dayMap := map[string]string{
		"mon": "Monday",
		"monday": "Monday",
		"tue": "Tuesday",
		"tuesday": "Tuesday",
		"wed": "Wednesday",
		"wednesday": "Wednesday",
		"thu": "Thursday",
		"thursday": "Thursday",
		"fri": "Friday",
		"friday": "Friday",
		"sat": "Saturday",
		"saturday": "Saturday",
		"sun": "Sunday",
		"sunday": "Sunday",
	}

	if full, ok := dayMap[dayLower]; ok {
		return full
	}

	return dayLower // Return as-is if not recognized
}

// LoadRulesFromCache loads cached rules from local file
func LoadRulesFromCache(filePath string) ([]Rule, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Rule{}, nil
		}
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	return ParseRules(string(data))
}

// SaveRulesCache saves rules to local cache file
func SaveRulesCache(filePath string, content string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// SaveETAG saves the ETAG value to cache file
func SaveETAG(filePath string, etag string) error {
	if err := os.WriteFile(filePath, []byte(etag), 0600); err != nil {
		return fmt.Errorf("failed to write ETAG file: %w", err)
	}
	return nil
}

// LoadETAG loads the cached ETAG value
func LoadETAG(filePath string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	return string(data)
}
