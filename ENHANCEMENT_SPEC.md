# Enhanced Account Integration - Technical Specification

## Overview
Comprehensive improvements to account management, preset functionality, and security.

## Feature Enhancements

### 1. Account Selection for Preset Launches
**Requirement**: When launching a preset game, show account picker dialog first

**Implementation**:
- Modify preset launch button to call `showAccountSelectionForPreset()`
- Pass game URL to launch after account selection
- Store last-used account per preset (optional enhancement)

**Files to modify**:
- `main.go`: Update preset launch button handler
- Add new dialog function for preset + account selection

### 2. Auto-fetch Game Names and Thumbnails
**Requirement**: Automatically fetch game name and thumbnail when adding preset

**Implementation**:
- Create `roblox_api` package
- Parse Roblox URL to extract Place ID
- Call Roblox API: `https://games.roblox.com/v1/games?universeIds={universeId}`
- Fetch thumbnail: `https://thumbnails.roblox.com/v1/games/icons?universeIds={universeId}&size=512x512&format=Png`
- Update Preset struct to include: `Name`, `Thumbnail` (base64 or URL)
- Display thumbnail in preset list using `canvas.Image`

**New files**:
- `internal/roblox_api/roblox_api.go`

**Files to modify**:
- `internal/preset_manager/preset_manager.go`: Add Name, Thumbnail fields
- `main.go`: Update preset list to show thumbnails

### 3. Track Current Logged-in Account
**Requirement**: Show which account is currently active in Roblox

**Implementation**:
- Detect current account by reading Roblox cookies/local storage
- macOS Roblox stores data in: `~/Library/Application Support/Roblox/`
- Parse LocalStorage or cookies to extract current username
- Create `roblox_session` package
- Display current account in Accounts tab with indicator
- Optional: Add "Current Account" badge/highlight

**New files**:
- `internal/roblox_session/roblox_session.go`

**Files to modify**:
- `main.go`: Add current account indicator in Accounts tab

### 4. Show Account Per Instance
**Requirement**: Display which account each running instance is using

**Implementation**:
- When instance launches with account, store mapping: PID -> Account ID
- Create `instance_account_tracker` to maintain PID->Account mapping
- Merge with existing label_manager or create separate tracker
- Update instance list to show account info alongside label
- Format: `Instance 1 (PID: 1234) - Main Account (@username)`

**New files**:
- `internal/instance_account_tracker/tracker.go`

**Files to modify**:
- `internal/instance_manager/instance_manager.go`: Add AccountID field
- `main.go`: Update instance display to show account

### 5. Enhanced Security
**Requirement**: Improve security for passwords and sensitive data

**Implementation**:

**a) Enhanced Keychain Security**:
- Use Keychain access control lists (ACL)
- Require authentication for password retrieval
- Add `-A` flag to `security add-generic-password` for app-specific access

**b) Memory Protection**:
- Clear password strings from memory after use
- Use `runtime.GC()` after password operations
- Avoid logging passwords in any form

**c) Secure Password Entry**:
- Already using `widget.NewPasswordEntry()` ✓
- Add password strength indicator
- Add show/hide password toggle

**d) Session Token Security**:
- Encrypt session tokens if stored
- Use AES-256 encryption for any local cache
- Implement token refresh mechanism

**e) Code Security**:
- Validate all URL inputs to prevent injection
- Sanitize game IDs before API calls
- Rate limit API requests
- Add timeout to all HTTP requests

**Files to modify**:
- `internal/account_manager/account_manager.go`: Enhanced Keychain flags
- `internal/roblox_api/roblox_api.go`: Secure HTTP client with timeouts
- `main.go`: Add password strength indicator

## Data Structures

### Updated Preset
```go
type Preset struct {
    Name      string `json:"name"`      // Auto-fetched or user-provided
    URL       string `json:"url"`
    PlaceID   string `json:"place_id"`  // Extracted from URL
    Thumbnail string `json:"thumbnail"` // Base64 or URL
    LastAccountUsed string `json:"last_account_used"` // Optional
}
```

### Instance Account Mapping
```go
type InstanceAccountMap struct {
    PID       int    `json:"pid"`
    AccountID string `json:"account_id"`
    LaunchedAt time.Time `json:"launched_at"`
}
```

### Enhanced Account
```go
type Account struct {
    ID       string `json:"id"`
    Username string `json:"username"`
    Label    string `json:"label"`
    IsCurrent bool  `json:"-"` // Runtime only, not persisted
}
```

## API Integrations

### Roblox APIs to Use
1. **Game Info API**: `GET https://games.roblox.com/v1/games?universeIds={id}`
2. **Thumbnail API**: `GET https://thumbnails.roblox.com/v1/games/icons?universeIds={id}&size=512x512`
3. **Place ID from URL**: Parse `roblox.com/games/{placeId}/` or `roblox://placeId={id}`

### URL Parsing
```
Input: https://www.roblox.com/games/1818/Classic-ROBLOX-Crossroads
Extract: placeId = 1818

Input: roblox://placeID=1818
Extract: placeId = 1818
```

## Security Enhancements Detail

### Keychain ACL
```bash
security add-generic-password \
  -s "multi-roblox-manager" \
  -a "{accountID}" \
  -w "{password}" \
  -A \  # App-specific access
  -U    # Update if exists
```

### HTTP Client Security
```go
client := &http.Client{
    Timeout: 10 * time.Second,
    Transport: &http.Transport{
        TLSClientConfig: &tls.Config{
            MinVersion: tls.VersionTLS12,
        },
    },
}
```

## Implementation Order
1. ✅ Create branch
2. Implement `roblox_api` package (game info fetching)
3. Update `preset_manager` with Name/Thumbnail fields
4. Add thumbnail display to preset list UI
5. Implement account selection for preset launches
6. Create `instance_account_tracker` package
7. Update instance display to show accounts
8. Implement `roblox_session` for current account detection
9. Add current account indicator to Accounts tab
10. Enhance security in `account_manager`
11. Add password strength indicator
12. Test all features
13. Commit and merge

## Testing Checklist
- [ ] Preset launches with account selection
- [ ] Game names auto-fetch correctly
- [ ] Thumbnails display in preset list
- [ ] Current account detected and shown
- [ ] Instance shows correct account after launch
- [ ] Passwords secure in Keychain with ACL
- [ ] No password leaks in logs/memory
- [ ] URL validation prevents injection
- [ ] HTTP requests have timeouts
- [ ] Error handling for API failures
