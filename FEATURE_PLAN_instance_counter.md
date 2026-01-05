# Feature: Instance Counter & Manager

## Overview
Display running Roblox instances with ability to manage them individually.

## UI Design
```
┌─────────────────────────────────────┐
│  Multi Roblox MacOS                 │
├─────────────────────────────────────┤
│  Running Instances: 3               │
│                                     │
│  ┌─────────────────────────────┐   │
│  │ Instance 1 (PID: 12345)     │ ❌ │
│  │ Instance 2 (PID: 12346)     │ ❌ │
│  │ Instance 3 (PID: 12347)     │ ❌ │
│  └─────────────────────────────┘   │
│                                     │
│  [New Instance] [Close All]         │
│  [Discord Server]                   │
└─────────────────────────────────────┘
```

## Implementation Steps

### 1. Create Instance Manager Module
File: `internal/instance_manager/instance_manager.go`

```go
package instance_manager

type Instance struct {
    PID         int
    StartTime   time.Time
    WindowTitle string
}

func GetRunningInstances() ([]Instance, error)
func CloseInstance(pid int) error
func GetInstanceCount() int
```

### 2. Update Main UI
- Add scrollable list of instances
- Add refresh timer (every 2 seconds)
- Add individual close buttons
- Add instance counter badge

### 3. Integration
- Use existing `ps_darwin` package
- Filter for "RobloxPlayer" processes
- Display with icons and close buttons

## Technical Details

### Dependencies
- Existing `ps_darwin` package for process listing
- Fyne `widget.List` for scrollable instance list
- `time.Ticker` for auto-refresh

### Data Flow
1. Timer triggers every 2s
2. Call `GetRunningInstances()`
3. Update UI list
4. User clicks close → `CloseInstance(pid)`
5. Refresh list

## Testing
- Launch 3+ Roblox instances
- Verify count updates
- Test individual close
- Test "Close All"
- Verify UI refreshes automatically

## Future Enhancements
- Show window titles
- Show memory usage per instance
- Sort by PID/start time
