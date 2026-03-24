package enforcer

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/look/kiddo/internal/config"
	"github.com/look/kiddo/internal/logger"
)

var log = logger.Get()

// LoggedInUser represents a logged in user session
type LoggedInUser struct {
	Username    string
	SessionID   int
	SessionName string
	State       string
}

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

	// Log off users who are not allowed to be logged in right now
	if err := e.logOffUnauthorizedUsers(allowedToLoginNow); err != nil {
		log.Errorf("Error logging off unauthorized users: %v", err)
	}

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

// logOffUnauthorizedUsers logs off users who are not allowed to be logged in
func (e *Enforcer) logOffUnauthorizedUsers(allowedUsers map[string]bool) error {
	if !isWindows() {
		log.Debugf("[DRY RUN] Would log off unauthorized users")
		return nil
	}

	// Get list of logged in users
	loggedInUsers, err := e.getLoggedInUsers()
	if err != nil {
		return fmt.Errorf("failed to get logged in users: %w", err)
	}

	for _, user := range loggedInUsers {
		usernameLower := strings.ToLower(user.Username)
		if !allowedUsers[usernameLower] {
			log.Warnf("User %s is logged in but not allowed, logging off session %d", user.Username, user.SessionID)
			if err := e.logOffUser(user.SessionID); err != nil {
				log.Errorf("Failed to log off user %s: %v", user.Username, err)
			}
		}
	}

	return nil
}

// getLoggedInUsers returns a list of currently logged in users
func (e *Enforcer) getLoggedInUsers() ([]LoggedInUser, error) {
	cmd := exec.Command("quser")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute quser: %w", err)
	}

	var users []LoggedInUser
	lines := strings.Split(string(output), "\n")

	// Skip the header line
	if len(lines) > 0 {
		lines = lines[1:]
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse line: USERNAME SESSIONNAME ID STATE IDLE_TIME LOGON_TIME
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		// First part is username, might have > prefix
		username := strings.TrimPrefix(parts[0], ">")

		// Find the ID - it's usually the 3rd or 4th field
		var sessionID int
		var sessionName string
		var state string

		// quser output varies, but typically: USERNAME SESSIONNAME ID STATE ...
		if len(parts) >= 4 {
			sessionName = parts[1]
			if id, err := strconv.Atoi(parts[2]); err == nil {
				sessionID = id
				state = parts[3]
			} else if id, err := strconv.Atoi(parts[1]); err == nil {
				// Sometimes SESSIONNAME is missing
				sessionID = id
				state = parts[2]
			}
		}

		if sessionID > 0 {
			users = append(users, LoggedInUser{
				Username:    username,
				SessionID:   sessionID,
				SessionName: sessionName,
				State:       state,
			})
		}
	}

	return users, nil
}

// logOffUser logs off a user session by ID
func (e *Enforcer) logOffUser(sessionID int) error {
	cmd := exec.Command("logoff", strconv.Itoa(sessionID))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to logoff session %d: %v - %s", sessionID, err, string(output))
	}
	log.Infof("Successfully logged off session %d", sessionID)
	return nil
}

// isWindows checks if we're running on Windows
func isWindows() bool {
	return os.Getenv("OS") == "Windows_NT"
}
