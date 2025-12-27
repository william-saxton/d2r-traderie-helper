package hotkey

import (
	"fmt"
	"log"

	"github.com/moutend/go-hook/pkg/keyboard"
	"github.com/moutend/go-hook/pkg/types"
)

// Listener handles global hotkey detection
type Listener struct {
	hotkey   string
	callback func()
	stopChan chan struct{}
	keyCode  types.VKCode
}

// NewListener creates a new hotkey listener
func NewListener(hotkey string, callback func()) *Listener {
	return &Listener{
		hotkey:   hotkey,
		callback: callback,
		stopChan: make(chan struct{}),
		keyCode:  parseHotkey(hotkey),
	}
}

// Start begins listening for hotkey presses
func (l *Listener) Start() error {
	log.Printf("Starting hotkey listener for: %s", l.hotkey)

	// Install keyboard hook
	keyboardChan := make(chan types.KeyboardEvent, 100)
	
	if err := keyboard.Install(nil, keyboardChan); err != nil {
		return fmt.Errorf("failed to install keyboard hook: %w", err)
	}

	// Start listening in a goroutine
	go l.listen(keyboardChan)

	log.Println("Hotkey listener started successfully")
	return nil
}

// listen processes keyboard events
func (l *Listener) listen(keyboardChan chan types.KeyboardEvent) {
	for {
		select {
		case <-l.stopChan:
			return
		case event := <-keyboardChan:
			// Only trigger on key down, not key up
			if event.Message == types.WM_KEYDOWN || event.Message == types.WM_SYSKEYDOWN {
				if event.VKCode == l.keyCode {
					log.Println("Hotkey detected!")
					l.callback()
				}
			}
		}
	}
}

// Stop stops the hotkey listener
func (l *Listener) Stop() {
	log.Println("Stopping hotkey listener...")
	close(l.stopChan)
	keyboard.Uninstall()
	log.Println("Hotkey listener stopped")
}

// parseHotkey converts hotkey string to VK code
func parseHotkey(hotkey string) types.VKCode {
	// Map common hotkey names to VK codes
	hotkeyMap := map[string]types.VKCode{
		"F1":  types.VK_F1,
		"F2":  types.VK_F2,
		"F3":  types.VK_F3,
		"F4":  types.VK_F4,
		"F5":  types.VK_F5,
		"F6":  types.VK_F6,
		"F7":  types.VK_F7,
		"F8":  types.VK_F8,
		"F9":  types.VK_F9,
		"F10": types.VK_F10,
		"F11": types.VK_F11,
		"F12": types.VK_F12,
	}

	if vkCode, ok := hotkeyMap[hotkey]; ok {
		return vkCode
	}

	// Default to F9
	return types.VK_F9
}

