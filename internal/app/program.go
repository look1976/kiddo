package app

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/look/kiddo/internal/cleanup"
	"github.com/look/kiddo/internal/config"
	"github.com/look/kiddo/internal/enforcer"
	"github.com/look/kiddo/internal/logger"
	"github.com/look/kiddo/internal/scheduler"
)

var log = logger.Get()

// Program represents the main Windows service program
type Program struct {
	cfg       *config.AppConfig
	cfgMgr    *config.Manager
	enf       *enforcer.Enforcer
	sched     *scheduler.Scheduler
	clean     *cleanup.Cleaner
	stopChan  chan struct{}
	ticker    *time.Ticker
}

// Start implements service.Service interface - called when service starts
func (p *Program) Start(svc interface{}) error {
	log.Infof("=== Kiddo Service Starting ===")

	// Create config directory
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = "C:\\ProgramData"
	}
	configDir := filepath.Join(programData, "Kiddo")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Errorf("Failed to create config directory: %v", err)
		return err
	}

	// Load configuration
	configPath := filepath.Join(configDir, "config.json")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Errorf("Failed to load configuration: %v", err)
		return err
	}

	// Validate required configuration
	if cfg.GitHubRepo == "" {
		log.Errorf("GitHub repository not configured")
		return fmt.Errorf("github_repo must be set in config.json")
	}

	if cfg.GitHubToken == "" {
		log.Warnf("GitHub token not configured - public repos only")
	}

	log.Infof("Configuration loaded from: %s", configPath)
	log.Infof("GitHub Repo: %s/%s", cfg.GitHubRepo, cfg.GitHubBranch)

	// Initialize components
	p.cfg = cfg
	p.cfgMgr = config.NewManager(cfg)
	p.enf = enforcer.NewEnforcer()
	p.sched = scheduler.NewScheduler()
	p.clean = cleanup.NewCleaner()
	p.stopChan = make(chan struct{})
	p.ticker = time.NewTicker(time.Duration(cfg.CheckIntervalSeconds) * time.Second)

	// Initial config fetch
	log.Infof("Performing initial configuration fetch...")
	if updated, err := p.cfgMgr.Fetch(); err != nil {
		log.Warnf("Initial fetch failed: %v", err)
	} else if updated {
		p.updateComponentRules()
	}

	// Start main service loop in a goroutine
	go p.run()

	log.Infof("Service started successfully")
	return nil
}

// Stop implements service.Service interface - called when service stops
func (p *Program) Stop(svc interface{}) error {
	log.Infof("=== Kiddo Service Stopping ===")

	// Signal the run loop to stop
	if p.stopChan != nil {
		close(p.stopChan)
	}

	if p.ticker != nil {
		p.ticker.Stop()
	}

	log.Infof("Service stopped")
	return nil
}

// run is the main service loop
func (p *Program) run() {
	log.Infof("Service loop started")

	// Track cleanup timing separately
	lastCleanup := time.Now()
	cleanupInterval := time.Duration(p.cfg.CleanupIntervalSeconds) * time.Second

	// Handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-p.stopChan:
			log.Infof("Service loop stopped")
			return

		case <-sigChan:
			log.Infof("Received signal, stopping service loop")
			return

		case <-p.ticker.C:
			// Time to check for config updates
			p.checkAndFetch()

			// Check if it's time for cleanup
			if time.Since(lastCleanup) >= cleanupInterval {
				p.performCleanup()
				lastCleanup = time.Now()
			}

			// Perform enforcement checks
			p.performEnforcement()
		}
	}
}

// checkAndFetch fetches configuration from GitHub
func (p *Program) checkAndFetch() {
	updated, err := p.cfgMgr.Fetch()

	if err != nil {
		log.Warnf("Configuration fetch error: %v", err)
		return
	}

	if updated {
		log.Infof("Configuration updated, applying new rules")
		p.updateComponentRules()
	}
}

// updateComponentRules updates all components with new rules
func (p *Program) updateComponentRules() {
	rules := p.cfgMgr.GetRules()

	p.enf.UpdateRules(rules)
	p.sched.UpdateRules(rules)
	p.clean.UpdateRules(rules)

	log.Infof("Updated %d rules across all components", len(rules))
}

// performEnforcement runs policy enforcement checks
func (p *Program) performEnforcement() {
	if err := p.enf.Enforce(); err != nil {
		log.Errorf("Enforcement error: %v", err)
	}

	if err := p.sched.CheckAndExecuteShutdowns(); err != nil {
		log.Errorf("Shutdown check error: %v", err)
	}
}

// performCleanup runs user cleanup
func (p *Program) performCleanup() {
	log.Debugf("Running cleanup cycle...")

	if err := p.clean.Clean(); err != nil {
		log.Errorf("Cleanup error: %v", err)
	}
}
