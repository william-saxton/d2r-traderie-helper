package memory

import (
	"fmt"
	"log"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/hectorgimenez/d2go/pkg/memory"
	"golang.org/x/sys/windows"
)

// ProcessInfo contains information about a D2R process
type ProcessInfo struct {
	PID        uint32
	WindowName string
	LastSeen   time.Time
}

// ProcessManager manages D2R process detection and switching
// NOTE: Due to d2go limitations, we can only have ONE active reader at a time
// The reader will dynamically switch to whichever D2R window is active
type ProcessManager struct {
	processes     map[uint32]*ProcessInfo // All detected D2R processes
	currentReader *Reader                 // Single active reader
	currentPID    uint32                  // PID of current reader
	mutex         sync.RWMutex
	stopChan      chan struct{}
	onUpdate      func([]ProcessInfo)
	refreshTicker *time.Ticker
}

// NewProcessManager creates and initializes a new process manager
func NewProcessManager(onUpdate func([]ProcessInfo)) (*ProcessManager, error) {
	pm := &ProcessManager{
		processes: make(map[uint32]*ProcessInfo),
		stopChan:  make(chan struct{}),
		onUpdate:  onUpdate,
	}

	// Initial scan for D2R processes
	if err := pm.RefreshProcesses(); err != nil {
		log.Printf("⚠️ Initial process scan found no D2R instances: %v", err)
		log.Println("💡 Process manager will continue monitoring for D2R launches...")
	} else {
		// Try to create initial reader for first process found
		pm.mutex.Lock()
		for pid := range pm.processes {
			if err := pm.switchToProcessLocked(pid); err != nil {
				log.Printf("⚠️ Failed to create initial reader for PID %d: %v", pid, err)
			} else {
				break // Successfully created reader for first process
			}
		}
		pm.mutex.Unlock()
	}

	// Start background monitoring
	pm.startMonitoring()

	return pm, nil
}

// startMonitoring begins the background process monitoring goroutine
func (pm *ProcessManager) startMonitoring() {
	pm.refreshTicker = time.NewTicker(10 * time.Second)
	
	go func() {
		for {
			select {
			case <-pm.stopChan:
				pm.refreshTicker.Stop()
				return
			case <-pm.refreshTicker.C:
				if err := pm.RefreshProcesses(); err != nil {
					log.Printf("Process refresh error: %v", err)
				}
			}
		}
	}()
	
	log.Println("✅ Process manager background monitoring started")
}

// RefreshProcesses scans for D2R processes and updates the process list
func (pm *ProcessManager) RefreshProcesses() error {
	pids, err := findAllD2RProcesses()
	if err != nil {
		return err
	}

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Track which PIDs are still alive
	activePIDs := make(map[uint32]bool)
	for _, pid := range pids {
		activePIDs[pid] = true
	}

	// Remove dead processes
	for pid := range pm.processes {
		if !activePIDs[pid] {
			log.Printf("🔴 D2R process %d has terminated", pid)
			delete(pm.processes, pid)
			
			// If this was our current reader's process, close it
			if pm.currentPID == pid && pm.currentReader != nil {
				log.Printf("Current reader was attached to dead process %d, closing...", pid)
				pm.currentReader.Close()
				pm.currentReader = nil
				pm.currentPID = 0
			}
		} else {
			// Update last seen time
			pm.processes[pid].LastSeen = time.Now()
		}
	}

	// Add new processes
	newProcessCount := 0
	for _, pid := range pids {
		if _, exists := pm.processes[pid]; !exists {
			pm.processes[pid] = &ProcessInfo{
				PID:        pid,
				WindowName: "Diablo II: Resurrected",
				LastSeen:   time.Now(),
			}
			log.Printf("🟢 New D2R process detected: PID %d", pid)
			newProcessCount++
		}
	}

	// Notify callback if processes changed
	if newProcessCount > 0 || len(pm.processes) != len(pids) {
		pm.notifyUpdate()
	}

	if len(pm.processes) == 0 {
		return fmt.Errorf("no D2R processes found")
	}

	return nil
}

// GetActiveProcessReader returns the reader for the currently active D2R window
// Automatically switches to the active process if it's different from current
func (pm *ProcessManager) GetActiveProcessReader() (*Reader, uint32, error) {
	activePID, err := getActiveD2RPID()
	if err != nil {
		return nil, 0, fmt.Errorf("no active D2R window: %w", err)
	}

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Check if this process is in our tracked list
	if _, exists := pm.processes[activePID]; !exists {
		return nil, 0, fmt.Errorf("active window is D2R (PID %d) but not in tracked processes - try refreshing", activePID)
	}

	// If we already have a reader for this PID, return it
	if pm.currentPID == activePID && pm.currentReader != nil {
		return pm.currentReader, activePID, nil
	}

	// Need to switch to this process
	log.Printf("🔄 Switching to D2R process PID %d (was %d)", activePID, pm.currentPID)
	if err := pm.switchToProcessLocked(activePID); err != nil {
		return nil, 0, fmt.Errorf("failed to switch to process %d: %w", activePID, err)
	}

	return pm.currentReader, pm.currentPID, nil
}

// switchToProcessLocked switches the current reader to a different process
// Must be called with mutex locked
func (pm *ProcessManager) switchToProcessLocked(pid uint32) error {
	// Close existing reader if any
	if pm.currentReader != nil {
		log.Printf("Closing reader for PID %d", pm.currentPID)
		pm.currentReader.Close()
		pm.currentReader = nil
		pm.currentPID = 0
	}

	// Create new reader
	reader, err := newReaderForPID(pid)
	if err != nil {
		return fmt.Errorf("failed to create reader for PID %d: %w", pid, err)
	}

	pm.currentReader = reader
	pm.currentPID = pid
	log.Printf("✅ Now attached to D2R process PID %d", pid)
	
	return nil
}

// GetProcessCount returns the number of detected D2R processes
func (pm *ProcessManager) GetProcessCount() int {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	return len(pm.processes)
}

// GetProcessInfoList returns a copy of all process info
func (pm *ProcessManager) GetProcessInfoList() []ProcessInfo {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	infos := make([]ProcessInfo, 0, len(pm.processes))
	for _, info := range pm.processes {
		infos = append(infos, *info)
	}
	return infos
}

// notifyUpdate calls the update callback if set (must hold lock)
func (pm *ProcessManager) notifyUpdate() {
	if pm.onUpdate != nil {
		infos := make([]ProcessInfo, 0, len(pm.processes))
		for _, info := range pm.processes {
			infos = append(infos, *info)
		}
		// Call in goroutine to avoid blocking
		go pm.onUpdate(infos)
	}
}

// Close shuts down the process manager and cleans up the reader
func (pm *ProcessManager) Close() error {
	log.Println("Shutting down process manager...")
	
	close(pm.stopChan)

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.currentReader != nil {
		pm.currentReader.Close()
		log.Printf("Closed reader for PID %d", pm.currentPID)
		pm.currentReader = nil
		pm.currentPID = 0
	}
	
	pm.processes = make(map[uint32]*ProcessInfo)

	log.Println("Process manager shut down complete")
	return nil
}

// findAllD2RProcesses enumerates all windows and finds all D2R process PIDs
func findAllD2RProcesses() ([]uint32, error) {
	var foundPIDs []uint32
	pidMap := make(map[uint32]bool) // Prevent duplicates

	// Load user32.dll functions
	user32 := windows.NewLazySystemDLL("user32.dll")
	getWindowTextW := user32.NewProc("GetWindowTextW")

	cb := syscall.NewCallback(func(hwnd windows.HWND, lParam uintptr) uintptr {
		var pid uint32
		windows.GetWindowThreadProcessId(hwnd, &pid)

		// Get window title
		titleBuf := make([]uint16, 256)
		getWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&titleBuf[0])), 256)
		windowTitle := syscall.UTF16ToString(titleBuf)

		// Check if this is D2R window
		if windowTitle == "Diablo II: Resurrected" {
			if !pidMap[pid] {
				foundPIDs = append(foundPIDs, pid)
				pidMap[pid] = true
			}
		}

		return 1 // Continue enumeration
	})

	windows.EnumWindows(cb, unsafe.Pointer(nil))

	if len(foundPIDs) == 0 {
		return nil, fmt.Errorf("no D2R windows found - make sure the game is running and you're in-game")
	}

	return foundPIDs, nil
}

// getActiveD2RPID gets the PID of the currently active foreground D2R window
func getActiveD2RPID() (uint32, error) {
	// Get foreground window
	hwnd := windows.GetForegroundWindow()
	if hwnd == 0 {
		return 0, fmt.Errorf("no foreground window")
	}

	// Get PID of foreground window
	var pid uint32
	windows.GetWindowThreadProcessId(hwnd, &pid)

	// Verify it's a D2R window by checking title
	user32 := windows.NewLazySystemDLL("user32.dll")
	getWindowTextW := user32.NewProc("GetWindowTextW")
	
	titleBuf := make([]uint16, 256)
	getWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&titleBuf[0])), 256)
	windowTitle := syscall.UTF16ToString(titleBuf)

	if windowTitle != "Diablo II: Resurrected" {
		return 0, fmt.Errorf("active window is not D2R (title: %s)", windowTitle)
	}

	return pid, nil
}

// newReaderForPID creates a Reader for a specific PID
func newReaderForPID(pid uint32) (*Reader, error) {
	// NOTE: d2go library's memory.NewProcess() finds the first D2R process
	// For true multi-process support, we'd need to:
	// 1. Fork d2go and add NewProcessForPID()
	// 2. Or use custom Windows API memory reading
	
	// For now, we'll use the standard method with verification
	process, err := memory.NewProcess()
	if err != nil {
		return nil, fmt.Errorf("failed to attach to D2R process: %w", err)
	}
	
	// Check if we got the right process
	attachedPID := uint32(process.GetPID())
	if attachedPID != pid {
		log.Printf("⚠️ Requested PID %d but d2go attached to PID %d (d2go limitation)", pid, attachedPID)
		// This is expected with current d2go - it always attaches to first process found
		// We'll still create the reader, but log the discrepancy
	}

	gr := memory.NewGameReader(process)
	
	reader := &Reader{
		gameReader: gr,
	}

	log.Printf("✅ Created reader for D2R process (requested PID: %d, actual: %d)", pid, attachedPID)
	return reader, nil
}

