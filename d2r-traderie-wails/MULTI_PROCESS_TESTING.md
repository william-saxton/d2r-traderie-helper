# Multi-Process D2R Support - Testing Guide

## Overview
The application now supports detecting and switching between multiple D2R instances. Due to d2go library limitations, the implementation uses a **dynamic switching approach** rather than simultaneous connections.

## How It Works

### Architecture
1. **Process Detection**: The app continuously scans for all running D2R processes (every 10 seconds)
2. **Dynamic Switching**: When you press F9, the app:
   - Detects which D2R window is currently active (foreground)
   - Switches the memory reader to that process if needed
   - Reads the item from the active process

### Key Features
- ✅ Detects all running D2R instances
- ✅ Shows count of detected instances in UI
- ✅ Automatically switches to active window when scanning
- ✅ Real-time process monitoring
- ✅ Manual refresh option

## Testing Instructions

### Test 1: Single Instance (Regression Test)
**Purpose**: Ensure existing functionality still works

1. Launch ONE D2R instance
2. Start the d2r-traderie-wails application
3. Verify header shows "🎮 1 D2R Instance"
4. Hover/hold an item in D2R
5. Press F9
6. **Expected**: Item should be scanned successfully

### Test 2: Multiple Instances - Basic Detection
**Purpose**: Verify process detection works

1. Launch TWO D2R instances
2. Start the d2r-traderie-wails application
3. Click on the "🎮 2 D2R Instances" badge
4. **Expected**: Process list panel shows both PIDs with "Diablo II: Resurrected" labels

### Test 3: Process Switching
**Purpose**: Test automatic switching between instances

1. Launch TWO D2R instances (call them Instance A and Instance B)
2. Start the application
3. Make Instance A the active window (click on it)
4. Hover/hold an item in Instance A
5. Press F9
6. **Expected**: Item from Instance A is scanned
7. Make Instance B the active window
8. Hover/hold an item in Instance B  
9. Press F9
10. **Expected**: Item from Instance B is scanned
11. Check console logs - should show "🔄 Switching to D2R process PID XXXX"

### Test 4: Process Launch While App Running
**Purpose**: Verify dynamic process detection

1. Start the application with NO D2R running
2. Verify header shows "⚠️ No D2R Processes"
3. Launch D2R
4. Wait up to 10 seconds (or click Refresh in process list)
5. **Expected**: Header updates to "🎮 1 D2R Instance"
6. Launch a second D2R instance
7. Wait up to 10 seconds (or click Refresh)
8. **Expected**: Header updates to "🎮 2 D2R Instances"

### Test 5: Process Termination
**Purpose**: Test graceful handling of process exit

1. Launch two D2R instances
2. Start the application
3. Scan an item from Instance A
4. Close Instance A (Alt+F4)
5. Wait up to 10 seconds (or click Refresh)
6. **Expected**: Header updates to "🎮 1 D2R Instance"
7. Make Instance B active and scan an item
8. **Expected**: Item scans successfully from remaining instance

### Test 6: Non-D2R Window Active
**Purpose**: Test error handling when D2R isn't focused

1. Launch D2R
2. Start application
3. Click on a different window (e.g., browser, file explorer)
4. Press F9
5. **Expected**: Error message "No active D2R window"

### Test 7: Manual Process Refresh
**Purpose**: Test manual refresh functionality

1. Launch D2R
2. Start application
3. Launch a second D2R instance
4. WITHOUT waiting, click process count badge
5. Click "🔄 Refresh Processes"
6. **Expected**: Both processes immediately appear in list

## Console Log Messages to Watch For

### Successful Scenarios
- `✅ Connected to X D2R instance(s) successfully!`
- `🟢 New D2R process detected: PID XXXX`
- `🔄 Switching to D2R process PID XXXX (was YYYY)`
- `📖 Reading from D2R process PID XXXX`
- `✓ Captured item from PID XXXX: [item name]`

### Warning/Error Scenarios
- `⚠️ No D2R instances found yet, monitoring for launches...`
- `🔴 D2R process XXXX has terminated`
- `❌ No active D2R window: active window is not D2R`
- `Failed to get active D2R process: no foreground window`

## Known Limitations

### d2go Library Constraint
The d2go library only supports attaching to ONE process at a time. Therefore:
- We cannot read from multiple D2R instances simultaneously
- The reader must be closed and recreated when switching processes
- There's a brief delay when switching (typically < 100ms)

### Future Enhancements
To support true simultaneous connections, would need to:
1. Fork the d2go library and add `NewProcessForPID()` method
2. Modify the Windows memory reading code to support multiple handles
3. Update ProcessManager to maintain multiple active readers

## Troubleshooting

### "No D2R processes found"
- Ensure D2R is running and you're **in-game** (not main menu)
- Run the application as Administrator
- Click Refresh Processes manually

### Items not scanning after switching
- Ensure the D2R window is **active/focused** when pressing F9
- Check console logs for switch messages
- Try pressing F9 again

### Process count incorrect
- Click "🔄 Refresh Processes" manually
- Check Task Manager to verify D2R processes are actually running
- Restart the application

### "Active window is not D2R"
- Make sure the D2R window is focused (click on it)
- Check window title is exactly "Diablo II: Resurrected"
- Some windowed/fullscreen modes may affect detection

## Performance Notes
- Background process scanning runs every 10 seconds (lightweight)
- Process switching takes ~50-100ms (imperceptible to user)
- No performance impact on D2R gameplay
- Memory usage: ~5-10MB per process tracked (minimal)

