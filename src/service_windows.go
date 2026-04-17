//go:build windows

package main

import (
	"log"
	"os"
	"path/filepath"
	"veda-anchor-engine/src/api"
	"veda-anchor-engine/src/internal/agent"
	"veda-anchor-engine/src/internal/config"
	"veda-anchor-engine/src/internal/data"
	"veda-anchor-engine/src/internal/data/logger"
	"veda-anchor-engine/src/internal/ipc"
	"veda-anchor-engine/src/internal/monitoring"

	"golang.org/x/sys/windows/svc"
)

// vedaAnchorService implements svc.Handler
type vedaAnchorService struct{}

func (s *vedaAnchorService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown

	changes <- svc.Status{State: svc.StartPending}

	// Setup logging
	logPath, err := config.GetLogPath()
	if err != nil {
		return true, 1
	}
	_ = os.MkdirAll(filepath.Dir(logPath), 0755)

	logFile, _ := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if logFile != nil {
		defer func() { _ = logFile.Close() }()
		log.SetOutput(logFile)
	}

	log.Printf("=== %s ENGINE SERVICE STARTING ===", config.AppName)

	// Initialize core
	db, err := data.InitDB()
	if err != nil {
		log.Printf("Failed to initialize database: %v", err)
		changes <- svc.Status{State: svc.StopPending}
		return true, 1
	}

	logger.NewLogger(db)
	l := logger.GetLogger()
	server := api.NewServer(db)

	// Start monitoring (screentime now handled by Agent)
	monitoring.StartDefault(l, server.Apps, nil)

	// Start IPC Server in background
	ipcServer := ipc.NewServer(server)
	go func() {
		log.Printf("%s Engine is starting IPC Server...", config.AppName)
		if err := ipcServer.Start(); err != nil {
			log.Printf("IPC server error: %v", err)
		}
	}()

	// Launch Agent in user session
	agentStopCh := make(chan struct{})
	go agent.SuperviseAgent(agentStopCh)

	// Service is now running
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
	log.Printf("=== %s ENGINE SERVICE RUNNING ===", config.AppName)

	// Wait for stop/shutdown signal
	for {
		c := <-r
		switch c.Cmd {
		case svc.Stop, svc.Shutdown:
			log.Printf("=== %s ENGINE SERVICE STOPPING ===", config.AppName)
			changes <- svc.Status{State: svc.StopPending}
			// Cleanup
			close(agentStopCh)
			l.Close()
			_ = db.Close()
			return false, 0
		case svc.Interrogate:
			changes <- c.CurrentStatus
		default:
			log.Printf("Unexpected service control request: %d", c.Cmd)
		}
	}
}
