package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Command represents a task for the browser extension
type Command struct {
	ID        string      `json:"id"`
	Action    string      `json:"action"` // "post_listing", "refresh_listings", "search"
	Payload   interface{} `json:"payload"`
	CreatedAt time.Time   `json:"created_at"`
}

// CommandResult represents the outcome of an extension command
type CommandResult struct {
	ID      string          `json:"id"`
	Success bool            `json:"success"`
	Error   string          `json:"error,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// ExtensionBridge handles communication between Wails and the Browser Extension
type ExtensionBridge struct {
	commands    []*Command
	results     map[string]*CommandResult
	mu          sync.RWMutex
	server      *http.Server
	port        int
	commandChan chan *CommandResult
}

// NewExtensionBridge creates a new bridge on a specific port
func NewExtensionBridge(port int) *ExtensionBridge {
	return &ExtensionBridge{
		commands:    []*Command{},
		results:     make(map[string]*CommandResult),
		port:        port,
		commandChan: make(chan *CommandResult, 10),
	}
}

// Start starts the local HTTP server
func (b *ExtensionBridge) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/commands", b.handleCommands)
	mux.HandleFunc("/results", b.handleResults)

	b.server = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", b.port),
		Handler: mux,
	}

	log.Printf("Extension bridge server starting on http://127.0.0.1:%d", b.port)
	go func() {
		if err := b.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Extension bridge server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the server
func (b *ExtensionBridge) Stop() error {
	if b.server != nil {
		return b.server.Close()
	}
	return nil
}

// AddCommand queues a command for the extension
func (b *ExtensionBridge) AddCommand(action string, payload interface{}) string {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := fmt.Sprintf("%d", time.Now().UnixNano())
	cmd := &Command{
		ID:        id,
		Action:    action,
		Payload:   payload,
		CreatedAt: time.Now(),
	}
	b.commands = append(b.commands, cmd)
	return id
}

// WaitForResults waits for a specific command result with timeout
func (b *ExtensionBridge) WaitForResult(id string, timeout time.Duration) (*CommandResult, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		b.mu.RLock()
		result, ok := b.results[id]
		b.mu.RUnlock()

		if ok {
			return result, nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil, fmt.Errorf("timeout waiting for result %s", id)
}

func (b *ExtensionBridge) handleCommands(w http.ResponseWriter, r *http.Request) {
	// Enable CORS for the extension
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Extension polls for commands
	if len(b.commands) > 0 {
		log.Printf("Extension bridge: Sending %d commands to extension: %s (ID: %s)", len(b.commands), b.commands[0].Action, b.commands[0].ID)
	} else {
		// Log very occasionally to show life without flooding
		if time.Now().Unix()%30 == 0 {
			log.Println("Extension bridge: Received heartbeat poll from extension")
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(b.commands)

	// Clear commands after polling
	b.commands = []*Command{}
}

func (b *ExtensionBridge) handleResults(w http.ResponseWriter, r *http.Request) {
	// Enable CORS for the extension
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var result CommandResult
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	b.mu.Lock()
	b.results[result.ID] = &result
	b.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

