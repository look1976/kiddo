package cleanup

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/look/kiddo/internal/config"
	"github.com/look/kiddo/internal/logger"
)

var log = logger.Get()

// Cleaner handles cleanup of unauthorized users
type Cleaner struct {
	rules []config.Rule
}

// NewCleaner creates a new cleaner instance
func NewCleaner() *Cleaner {
	return &Cleaner{
		rules: []config.Rule{},
	}
}

// UpdateRules updates the cleaner with new rules
func (c *Cleaner) UpdateRules(rules []config.Rule) {
	c.rules = rules
	log.Debugf("Cleaner updated with %d rules", len(rules))
}

// Clean removes unauthorized user accounts
func (c *Cleaner) Clean() error {
	log.Debugf("Starting cleanup of unauthorized users")

	// Build list of authorized users
	authorizedUsers := make(map[string]bool)
	for _, rule := range c.rules {
		authorizedUsers[strings.ToLower(rule.Username)] = true
	}

	log.Debugf("Authorized users: %v", c.getRuleUsernames())

	// System/protected accounts that should never be deleted
	protectedAccounts := map[string]bool{
		"administrator": true,
		"guest": true,
		"system": true,
		"local service": true,
		"network service": true,
		"defaultaccount": true,
		"wdagutilityaccount": true,
	}

	if !isWindows() {
		log.Debugf("[DRY RUN] Would clean unauthorized users from list: %v", c.getRuleUsernames())
		return nil
	}

	// Get list of all local users
	users, err := c.getLocalUsers()
	if err != nil {
		log.Errorf("Failed to get local users: %v", err)
		return err
	}

	log.Debugf("Found %d local users", len(users))

	// Check each user
	for _, user := range users {
		userLower := strings.ToLower(user)

		// Skip protected accounts
		if protectedAccounts[userLower] {
			log.Debugf("Skipping protected account: %s", user)
			continue
		}

		// Check if user is authorized
		if !authorizedUsers[userLower] {
			log.Warnf("Found unauthorized user: %s", user)

			// Logoff the user first
			if err := c.logoffUser(user); err != nil {
				log.Errorf("Failed to logoff user %s: %v", user, err)
			}

			// Delete the user account
			if err := c.deleteUser(user); err != nil {
				log.Errorf("Failed to delete user %s: %v", user, err)
			}
		}
	}

	log.Infof("Cleanup cycle complete")
	return nil
}

// getLocalUsers returns a list of local user accounts
func (c *Cleaner) getLocalUsers() ([]string, error) {
	cmd := exec.Command("net", "user")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute net user: %w", err)
	}

	var users []string
	lines := strings.Split(string(output), "\n")

	// Parse net user output
	// Format is typically:
	// \\COMPUTERNAME
	// User accounts for \\COMPUTERNAME
	// -----------------------------------------------
	// Administrator    [otherseparator]
	// Guest
	// User1            [otherseparator]
	// ...
	// -----------------------------------------------
	// The command completed successfully.

	inUserList := false
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and headers
		if line == "" || strings.HasPrefix(line, "\\\\") || strings.HasPrefix(line, "User accounts") ||
			strings.HasPrefix(line, "---") || strings.HasPrefix(line, "The command") {
			continue
		}

		// Extract username (first word on the line)
		parts := strings.Fields(line)
		if len(parts) > 0 {
			username := parts[0]
			// Skip if it looks like a separator or header
			if !strings.Contains(username, "-") {
				users = append(users, username)
				inUserList = true
			}
		}
	}

	return users, nil
}

// logoffUser logs off a user
func (c *Cleaner) logoffUser(username string) error {
	log.Warnf("Logging off user: %s", username)

	// Use logoff command to forcefully log off the user
	// Note: This might not work if user is not logged in, so we ignore errors
	cmd := exec.Command("logoff", username, "/server:localhost")
	cmd.CombinedOutput() // Ignore error

	return nil
}

// deleteUser deletes a user account
func (c *Cleaner) deleteUser(username string) error {
	log.Warnf("Deleting user account: %s", username)

	cmd := exec.Command("net", "user", username, "/delete")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("failed to delete user %s: %v - %s", username, err, string(output))
	}

	log.Infof("Successfully deleted user: %s", username)
	return nil
}

// getRuleUsernames returns a list of usernames from current rules
func (c *Cleaner) getRuleUsernames() []string {
	var usernames []string
	seen := make(map[string]bool)

	for _, rule := range c.rules {
		username := strings.ToLower(rule.Username)
		if !seen[username] {
			seen[username] = true
			usernames = append(usernames, rule.Username)
		}
	}

	return usernames
}

// isWindows checks if we're running on Windows
func isWindows() bool {
	return os.Getenv("OS") == "Windows_NT"
}
