//go:build windows

package agent

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"veda-anchor-engine/src/internal/platform/proc_sensing"

	"golang.org/x/sys/windows"
)

const (
	INVALID_SESSION_ID = ^uint32(0)
)

var (
	agentPID   uint32
	agentPIDMu sync.Mutex
)

func GetAgentPID() uint32 {
	agentPIDMu.Lock()
	defer agentPIDMu.Unlock()
	return agentPID
}

func setAgentPID(pid uint32) {
	agentPIDMu.Lock()
	agentPID = pid
	agentPIDMu.Unlock()
}

func KillAgent() {
	pid := GetAgentPID()
	if pid == 0 {
		return
	}
	log.Printf("[AgentLauncher] Killing agent (PID: %d)", pid)
	if proc, err := os.FindProcess(int(pid)); err == nil {
		_ = proc.Kill()
	}
	setAgentPID(0)
}

func LaunchAgent() (uint32, error) {
	agentPath := filepath.Join(os.Getenv("ProgramFiles"), "VedaAnchor", "veda-anchor-agent.exe")

	if _, err := os.Stat(agentPath); err != nil {
		log.Printf("[AgentLauncher] Agent binary not found: %s", agentPath)
		return 0, err
	}

	sessionID, err := getActiveConsoleSession()
	if err != nil {
		log.Printf("[AgentLauncher] Failed to get active session: %v", err)
		return 0, err
	}

	if sessionID == INVALID_SESSION_ID {
		log.Printf("[AgentLauncher] No active user session")
		return 0, nil
	}

	log.Printf("[AgentLauncher] Found active session: %d", sessionID)

	token, err := getUserToken(sessionID)
	if err != nil {
		log.Printf("[AgentLauncher] Failed to get user token: %v", err)
		return 0, err
	}
	defer token.Close()

	log.Printf("[AgentLauncher] Launching Agent in user session...")
	pid, err := createProcessAsUser(token, agentPath)
	if err != nil {
		log.Printf("[AgentLauncher] Failed to launch Agent: %v", err)
		return 0, err
	}

	setAgentPID(pid)
	log.Printf("[AgentLauncher] Agent launched successfully (PID: %d)", pid)
	return pid, nil
}

func getActiveConsoleSession() (uint32, error) {
	sessionID := windows.WTSGetActiveConsoleSessionId()
	if sessionID == INVALID_SESSION_ID {
		return INVALID_SESSION_ID, nil
	}
	return sessionID, nil
}

func getUserToken(sessionID uint32) (windows.Token, error) {
	var token windows.Token
	err := windows.WTSQueryUserToken(sessionID, &token)
	if err != nil {
		return 0, err
	}

	var tokenDup windows.Token
	err = windows.DuplicateTokenEx(
		token,
		windows.MAXIMUM_ALLOWED,
		nil,
		windows.SecurityIdentification,
		windows.TokenPrimary,
		&tokenDup,
	)
	token.Close()
	if err != nil {
		return 0, err
	}

	return tokenDup, nil
}

func createProcessAsUser(token windows.Token, exePath string) (uint32, error) {
	appName, err := windows.UTF16PtrFromString(exePath)
	if err != nil {
		return 0, err
	}

	// Build user environment block
	envBlock, err := createEnvironmentBlock(token)
	if err != nil {
		log.Printf("[AgentLauncher] Warning: CreateEnvironmentBlock failed: %v, using inherited env", err)
		// Fall through with nil envBlock
	}
	if envBlock != nil {
		defer destroyEnvironmentBlock(envBlock)
	}

	si := windows.StartupInfo{
		Desktop:    windows.StringToUTF16Ptr("Winsta0\\default"),
		Flags:      windows.STARTF_USESHOWWINDOW,
		ShowWindow: windows.SW_HIDE,
	}
	var pi windows.ProcessInformation

	creationFlags := uint32(windows.CREATE_UNICODE_ENVIRONMENT)

	err = windows.CreateProcessAsUser(
		token,
		appName,
		nil,
		nil,
		nil,
		false,
		creationFlags,
		envBlock,
		nil,
		&si,
		&pi,
	)
	if err != nil {
		return 0, err
	}

	pid := pi.ProcessId
	windows.CloseHandle(pi.Process)
	windows.CloseHandle(pi.Thread)
	return pid, nil
}

var (
	modUserenv                  = syscall.NewLazyDLL("userenv.dll")
	procCreateEnvironmentBlock  = modUserenv.NewProc("CreateEnvironmentBlock")
	procDestroyEnvironmentBlock = modUserenv.NewProc("DestroyEnvironmentBlock")
)

func createEnvironmentBlock(token windows.Token) (*uint16, error) {
	var envBlock *uint16
	ret, _, err := procCreateEnvironmentBlock.Call(
		uintptr(unsafe.Pointer(&envBlock)),
		uintptr(token),
		0, // don't inherit parent env
	)
	if ret == 0 {
		return nil, err
	}
	return envBlock, nil
}

func destroyEnvironmentBlock(envBlock *uint16) {
	procDestroyEnvironmentBlock.Call(uintptr(unsafe.Pointer(envBlock)))
}

func SuperviseAgent(stopCh <-chan struct{}) {
	// Wait for IPC pipe to be ready before first launch
	waitForIPCReady()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Initial launch
	if !isAgentRunning() {
		LaunchAgent()
	}

	for {
		select {
		case <-stopCh:
			log.Printf("[AgentSupervisor] Stopping")
			return
		case <-ticker.C:
			if !isAgentRunning() {
				log.Printf("[AgentSupervisor] Agent not running, restarting...")
				if pid, err := LaunchAgent(); err == nil {
					setAgentPID(pid)
				} else {
					log.Printf("[AgentSupervisor] Failed to restart agent: %v", err)
				}
			}
		}
	}
}

func isAgentRunning() bool {
	procs, err := proc_sensing.GetAllProcesses()
	if err != nil {
		return false
	}
	for _, p := range procs {
		if strings.EqualFold(p.Name, "veda-anchor-agent.exe") {
			return true
		}
	}
	return false
}

func waitForIPCReady() {
	pipePath := `\\.\pipe\veda-anchor`
	for range 30 { // Wait up to 30 seconds
		_, err := os.Stat(pipePath)
		if err == nil {
			log.Printf("[AgentSupervisor] IPC pipe ready")
			return
		}
		time.Sleep(1 * time.Second)
	}
	log.Printf("[AgentSupervisor] IPC pipe not ready after 30s, proceeding anyway")
}
