package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/kardianos/service"
	"github.com/look/kiddo/internal/app"
	"github.com/look/kiddo/internal/logger"
)

var log = logger.Get()

func main() {
	// Initialize logger
	if err := logger.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// Create service config
	svcConfig := &service.Config{
		Name:        "Kiddo",
		DisplayName: "System Maintenance Service",
		Description: "Performs routine system maintenance tasks",
		Option: service.KeyValue{
			"Restart": "on-failure",
		},
	}

	// Create application instance
	prg := &app.Program{}

	// Create service instance
	svc, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// Handle command-line arguments
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "install":
			err = svc.Install()
			if err != nil {
				log.Fatalf("Failed to install service: %v", err)
			}
			fmt.Println("Service installed successfully")
			return
		case "uninstall":
			// Clean up scheduled tasks created by the service
			log.Info("Cleaning up scheduled tasks...")
			cmd := exec.Command("powershell", "-Command", "Get-ScheduledTask | Where-Object {$_.TaskName -like 'Kiddo_*'} | Unregister-ScheduledTask -Confirm:$false")
			if output, err := cmd.CombinedOutput(); err != nil {
				log.Warnf("Failed to clean up scheduled tasks: %v - %s", err, string(output))
			} else {
				log.Info("Scheduled tasks cleaned up successfully")
			}

			err = svc.Uninstall()
			if err != nil {
				log.Fatalf("Failed to uninstall service: %v", err)
			}
			fmt.Println("Service uninstalled successfully")
			return
		case "start":
			err = svc.Start()
			if err != nil {
				log.Fatalf("Failed to start service: %v", err)
			}
			fmt.Println("Service started successfully")
			return
		case "stop":
			err = svc.Stop()
			if err != nil {
				log.Fatalf("Failed to stop service: %v", err)
			}
			fmt.Println("Service stopped successfully")
			return
		case "restart":
			err = svc.Restart()
			if err != nil {
				log.Fatalf("Failed to restart service: %v", err)
			}
			fmt.Println("Service restarted successfully")
			return
		}
	}

	// Run service
	err = svc.Run()
	if err != nil {
		log.Errorf("Service run failed: %v", err)
	}
}
