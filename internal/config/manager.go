package config

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/look/kiddo/internal/logger"
)

var log = logger.Get()

// Manager handles GitHub polling and rule synchronization
type Manager struct {
	config *AppConfig
	client *http.Client
	rules  []Rule
}

// NewManager creates a new config manager
func NewManager(cfg *AppConfig) *Manager {
	return &Manager{
		config: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		rules: []Rule{},
	}
}

// Fetch attempts to download the rules from GitHub
// Returns true if rules were updated, false otherwise
func (m *Manager) Fetch() (bool, error) {
	// Construct raw GitHub URL
	url := fmt.Sprintf(
		"https://raw.githubusercontent.com/%s/%s/%s",
		m.config.GitHubRepo,
		m.config.GitHubBranch,
		m.config.ConfigFile,
	)

	log.Debugf("Fetching configuration from: %s", url)

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication if token provided
	if m.config.GitHubToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", m.config.GitHubToken))
	}

	// Add If-None-Match for ETag caching
	cachedETag := LoadETAG(m.config.ETAGFile)
	if cachedETag != "" {
		req.Header.Set("If-None-Match", cachedETag)
		log.Debugf("Using cached ETag: %s", cachedETag)
	}

	// Execute request
	resp, err := m.client.Do(req)
	if err != nil {
		log.Warnf("Failed to fetch configuration: %w", err)
		// Try to load from cache on error
		return m.loadFromCache()
	}
	defer resp.Body.Close()

	// Handle 304 Not Modified
	if resp.StatusCode == http.StatusNotModified {
		log.Debugf("Configuration unchanged (304 Not Modified)")
		return m.loadFromCache()
	}

	// Handle errors
	if resp.StatusCode != http.StatusOK {
		log.Warnf("Failed to fetch configuration: HTTP %d", resp.StatusCode)
		return m.loadFromCache()
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Warnf("Failed to read response body: %v", err)
		return m.loadFromCache()
	}

	content := string(body)

	// Parse rules
	newRules, err := ParseRules(content)
	if err != nil {
		log.Errorf("Failed to parse rules: %v", err)
		return m.loadFromCache()
	}

	// Save ETag for future requests
	if etag := resp.Header.Get("ETag"); etag != "" {
		if err := SaveETAG(m.config.ETAGFile, etag); err != nil {
			log.Warnf("Failed to save ETag: %v", err)
		} else {
			log.Debugf("Saved ETag: %s", etag)
		}
	}

	// Save rules to cache
	if err := SaveRulesCache(m.config.CacheFile, content); err != nil {
		log.Warnf("Failed to cache rules: %v", err)
		// Still continue with the new rules
	} else {
		log.Infof("Cached %d rules locally", len(newRules))
	}

	// Check if rules actually changed
	if m.rulesEqual(m.rules, newRules) {
		log.Debugf("Rules unchanged from last fetch")
		return false, nil
	}

	// Update rules
	m.rules = newRules
	log.Infof("Successfully fetched and updated %d rules from GitHub", len(m.rules))

	return true, nil
}

// LoadFromCache attempts to load rules from local cache
func (m *Manager) loadFromCache() (bool, error) {
	cachedRules, err := LoadRulesFromCache(m.config.CacheFile)
	if err != nil {
		return false, fmt.Errorf("failed to load cached rules: %w", err)
	}

	if len(cachedRules) == 0 {
		return false, fmt.Errorf("no cached rules available")
	}

	// Check if rules changed
	if m.rulesEqual(m.rules, cachedRules) {
		log.Debugf("Cached rules unchanged from last load")
		return false, nil
	}

	m.rules = cachedRules
	log.Infof("Loaded %d rules from local cache", len(m.rules))
	return true, nil
}

// GetRules returns the current set of rules
func (m *Manager) GetRules() []Rule {
	return m.rules
}

// rulesEqual compares two rule sets for equality
func (m *Manager) rulesEqual(a, b []Rule) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i].Username != b[i].Username ||
			a[i].DayOfWeek != b[i].DayOfWeek ||
			a[i].StartHour != b[i].StartHour ||
			a[i].EndHour != b[i].EndHour {
			return false
		}
	}

	return true
}
