package enforcer

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/look/kiddo/internal/config"
	"github.com/look/kiddo/internal/logger"
)

var log = logger.Get()

// Enforcer handles policy enforcement (user login times)
type Enforcer struct {
	rules []config.Rule
}

// NewEnforcer creates a new policy enforcer
func NewEnforcer() *Enforcer {
	return &Enforcer{
		rules: []config.Rule{},
	}
}

// UpdateRules updates the enforcer with new rules
func (e *Enforcer) UpdateRules(rules []config.Rule) {
	e.rules = rules
	log.Debugf("Enforcer updated with %d rules", len(e.rules))
}

// Enforce checks all users and enforces current policies
func (e *Enforcer) Enforce() error {
	now := time.Now()
	currentDay := now.Weekday().String()
	currentHour := now.Hour()

	log.Debugf("Enforcing policies - Current time: %s %02d:00", currentDay, currentHour)

	// Get list of all allowed users from rules
	allowedUsers := make(map[string]bool)
	for _, rule := range e.rules {
		allowedUsers[strings.ToLower(rule.Username)] = true
	}

	// Check each rule to determine if user should be allowed right now
	allowedToLoginNow := make(map[string]bool)
	for _, rule := range e.rules {
		username := strings.ToLower(rule.Username)

		// Check if rule applies to current day
		if !e.dayMatches(rule.DayOfWeek, currentDay) {
			continue
		}

		// Check if current hour is within allowed time range
		if currentHour >= rule.StartHour && currentHour < rule.EndHour {
			allowedToLoginNow[username] = true
			log.Debugf("User %s is currently allowed (within %d-%d)", rule.Username, rule.StartHour, rule.EndHour)
		} else {
			log.Debugf("User %s is NOT allowed right now (outside %d-%d)", rule.Username, rule.StartHour, rule.EndHour)
		}
	}

	// For now, just log what we would enforce
	// Full implementation would actually modify user accounts
	log.Debugf("Total allowed users in rules: %d", len(allowedUsers))
	log.Debugf("Users allowed to login now: %d", len(allowedToLoginNow))

	// If no users are allowed to login and there are rules configured, shutdown the PC
	if len(allowedToLoginNow) == 0 && len(allowedUsers) > 0 {
		log.Warnf("No users allowed to login at current time, initiating PC shutdown")
		cmd := exec.Command("shutdown", "/s", "/t", "60") // 60 second warning
		if err := cmd.Run(); err != nil {
			log.Errorf("Failed to initiate shutdown: %v", err)
		}
	}

	return nil
}

// dayMatches checks if a rule day matches the current day
func (e *Enforcer) dayMatches(ruleDay, currentDay string) bool {
	// Normalize rule day
	ruleDay = normalizeDay(ruleDay)
	currentDay = normalizeDay(currentDay)
	return ruleDay == currentDay
}

// normalizeDay normalizes day names
func normalizeDay(day string) string {
	dayLower := strings.ToLower(day)
	dayNames := map[string]string{
		"mon": "Monday", "monday": "Monday",
		"tue": "Tuesday", "tuesday": "Tuesday",
		"wed": "Wednesday", "wednesday": "Wednesday",
		"thu": "Thursday", "thursday": "Thursday",
		"fri": "Friday", "friday": "Friday",
		"sat": "Saturday", "saturday": "Saturday",
		"sun": "Sunday", "sunday": "Sunday",
	}

	if normalized, ok := dayNames[dayLower]; ok {
		return normalized
	}
	return dayLower
}

// SetUserLoginHours sets Windows logon hours for a user
// This would normally be done via Group Policy, but for now we log the intent
func (e *Enforcer) SetUserLoginHours(username string, startHour, endHour int) error {
	log.Infof("Setting login hours for user %s: %02d:00-%02d:00", username, startHour, endHour)

	// Note: Actual implementation would require:
	// - Group Policy Object (GPO) manipulation
	// - Or periodic user logoff at end time
	// - For now, this is a placeholder

	return nil
}

// DisableUser disables a user account
func (e *Enforcer) DisableUser(username string) error {
	log.Warnf("Disabling user account: %s", username)

	// Check if running on Windows
	if !isWindows() {
		log.Debugf("[DRY RUN] Would disable user: %s", username)
		return nil
	}

	// Execute: net user <username> /active:no
	cmd := exec.Command("net", "user", username, "/active:no")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Errorf("Failed to disable user %s: %v - %s", username, err, string(output))
		return err
	}

	log.Infof("Successfully disabled user: %s", username)
	return nil
}

// EnableUser enables a user account
func (e *Enforcer) EnableUser(username string) error {
	log.Infof("Enabling user account: %s", username)

	if !isWindows() {
		log.Debugf("[DRY RUN] Would enable user: %s", username)
		return nil
	}

	cmd := exec.Command("net", "user", username, "/active:yes")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Errorf("Failed to enable user %s: %v - %s", username, err, string(output))
		return err
	}

	log.Infof("Successfully enabled user: %s", username)
	return nil
}

// GetLocalUsers returns a list of all local user accounts (except system accounts)
func (e *Enforcer) GetLocalUsers() ([]string, error) {
	if !isWindows() {
		log.Debugf("[DRY RUN] Would list local users")
		return []string{}, nil
	}

	cmd := exec.Command("net", "user")
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Errorf("Failed to list users: %v", err)
		return nil, err
	}

	// Parse output - simple approach
	// Output format shows usernames, but parsing is complex
	// For now, return empty list - implementation would parse the output
	lines := strings.Split(string(output), "\n")
	log.Debugf("Got %d lines from net user output", len(lines))

	return []string{}, nil
}

// isWindows checks if we're running on Windows
func isWindows() bool {
	return os.Getenv("OS") == "Windows_NT"
}
