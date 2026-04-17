package api

import (
	"fmt"
	"os"
	"strings"
	"time"
	"veda-anchor-engine/src/internal/agent"
	"veda-anchor-engine/src/internal/auth"
	app_blocklist "veda-anchor-engine/src/internal/blocklist/app"
	"veda-anchor-engine/src/internal/config"
	"veda-anchor-engine/src/internal/data/history"
	"veda-anchor-engine/src/internal/monitoring"
	"veda-anchor-engine/src/internal/platform/autostart"
	"veda-anchor-engine/src/internal/platform/proc_sensing"
	"veda-anchor-engine/src/internal/platform/uninstall"
)

const appName = "Veda"

// --- Lifecycle ---

func (s *Server) Shutdown() {
	s.Logger.Println("Received stop request. Shutting down...")

	go func() {
		// Kill agent first
		agent.KillAgent()

		time.Sleep(1 * time.Second)
		s.Logger.Close()
		if err := s.db.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to close database: %v\n", err)
		}
		os.Exit(0)
	}()
}

func (s *Server) Uninstall(password string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	if !auth.CheckPasswordHash(password, cfg.PasswordHash) {
		return fmt.Errorf("invalid password")
	}

	go func() {
		// Kill agent first
		agent.KillAgent()

		s.Logger.Close()
		if err := s.db.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to close database: %v\n", err)
		}

		s.killOtherVedaProcesses()

		if err := s.unblockAll(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to unblock all files: %v\n", err)
		}

		// Stop and delete the Windows Service
		_ = autostart.StopAndDeleteService()

		if err := uninstall.SelfDestruct(appName); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initiate self-deletion: %v\n", err)
		}
		os.Exit(0)
	}()
	return nil
}

// --- Settings & History ---

func (s *Server) GetAutostartStatus() (bool, error) {
	return autostart.GetServiceStartType()
}

func (s *Server) EnableAutostart() error {
	_, err := autostart.EnsureAutostart()
	return err
}

func (s *Server) DisableAutostart() error {
	return autostart.RemoveAutostart()
}

func (s *Server) ClearAppHistory(password string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	if !auth.CheckPasswordHash(password, cfg.PasswordHash) {
		return fmt.Errorf("invalid password")
	}

	history.ClearAppHistory()
	monitoring.ResetGlobalManager()
	return nil
}

func (s *Server) ClearWebHistory(password string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	if !auth.CheckPasswordHash(password, cfg.PasswordHash) {
		return fmt.Errorf("invalid password")
	}

	history.ClearWebHistory()
	return nil
}

// --- Internal Helpers ---

func (s *Server) killOtherVedaProcesses() {
	currentPid := os.Getpid()
	procs, err := proc_sensing.GetAllProcesses()
	if err != nil {
		return
	}

	for _, p := range procs {
		if int(p.PID) == currentPid {
			continue
		}
		if strings.HasPrefix(strings.ToLower(p.Name), "veda") {
			if osProc, err := os.FindProcess(int(p.PID)); err == nil {
				_ = osProc.Kill()
			}
		}
	}
}

func (s *Server) unblockAll() error {
	list, err := app_blocklist.LoadAppBlocklist()
	if err != nil {
		return err
	}

	for _, name := range list {
		if before, ok := strings.CutSuffix(name, ".blocked"); ok {
			newName := before
			_ = os.Rename(name, newName)
		}
	}
	return nil
}
