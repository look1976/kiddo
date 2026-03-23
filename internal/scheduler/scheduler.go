package scheduler

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

// Scheduler handles PC shutdown scheduling
type Scheduler struct {
	rules []config.Rule
}

// NewScheduler creates a new shutdown scheduler
func NewScheduler() *Scheduler {
	return &Scheduler{
		rules: []config.Rule{},
	}
}

// UpdateRules updates the scheduler with new rules
func (s *Scheduler) UpdateRules(rules []config.Rule) {
	s.rules = rules
	log.Debugf("Scheduler updated with %d rules", len(rules))
}

// ScheduleShutdowns creates Windows scheduled tasks for user shutdowns
func (s *Scheduler) ScheduleShutdowns() error {
	if !isWindows() {
		log.Debugf("[DRY RUN] Would schedule shutdowns")
		return nil
	}

	log.Debugf("Scheduling shutdown tasks for %d rules", len(s.rules))

	// Track unique users and their earliest shutdown times
	shutdownTimes := make(map[string]int) // username -> end hour

	for _, rule := range s.rules {
		username := strings.ToLower(rule.Username)

		// For each user, find their earliest end-of-session time
		if existingEndHour, exists := shutdownTimes[username]; !exists {
			shutdownTimes[username] = rule.EndHour
		} else if rule.EndHour < existingEndHour {
			shutdownTimes[username] = rule.EndHour
		}
	}

	// Schedule a shutdown task for each user
	for username, endHour := range shutdownTimes {
		if err := s.scheduleUserShutdown(username, endHour); err != nil {
			log.Errorf("Failed to schedule shutdown for %s: %v", username, err)
		}
	}

	return nil
}

// scheduleUserShutdown creates a scheduled task for a user shutdown
func (s *Scheduler) scheduleUserShutdown(username string, endHour int) error {
	taskName := fmt.Sprintf("Kiddo_Shutdown_%s", username)

	// Format end time as HH:MM for schtasks
	shutdownTime := fmt.Sprintf("%02d:00", endHour)

	log.Infof("Scheduling shutdown task: %s at %s", taskName, shutdownTime)

	// First, try to delete existing task
	deleteCmd := exec.Command("schtasks", "/delete", "/tn", taskName, "/f")
	deleteCmd.CombinedOutput() // Ignore error if task doesn't exist

	// Create new scheduled task
	// schtasks /create /tn "TaskName" /tr "shutdown /s /t 300" /sc daily /st 21:00
	createCmd := exec.Command(
		"schtasks",
		"/create",
		"/tn", taskName,
		"/tr", "shutdown /s /t 300", // 5 minute warning before shutdown
		"/sc", "daily",
		"/st", shutdownTime,
		"/f", // Force creation/overwrite
	)

	output, err := createCmd.CombinedOutput()
	if err != nil {
		log.Errorf("Failed to create scheduled task %s: %v - %s", taskName, err, string(output))
		return err
	}

	log.Infof("Successfully scheduled shutdown task: %s", taskName)
	return nil
}

// CheckAndExecuteShutdowns checks if any PC should be shut down right now
func (s *Scheduler) CheckAndExecuteShutdowns() error {
	now := time.Now()
	currentHour := now.Hour()
	currentMinute := now.Minute()

	log.Debugf("Checking for shutdown conditions - Current time: %02d:%02d", currentHour, currentMinute)

	// Track which users should be logged off
	usersToShutdown := make(map[string]bool)

	for _, rule := range s.rules {
		// Check if we're within shutdown minute window (e.g., 21:00 - 21:01)
		if currentHour == rule.EndHour && currentMinute < 2 {
			usersToShutdown[strings.ToLower(rule.Username)] = true
		}
	}

	if len(usersToShutdown) > 0 {
		log.Warnf("Need to shutdown %d users", len(usersToShutdown))

		// For now, we let Windows Task Scheduler handle actual shutdowns
		// This function primarily logs and verifies conditions
	}

	return nil
}

// isWindows checks if we're running on Windows
func isWindows() bool {
	return os.Getenv("OS") == "Windows_NT"
}
