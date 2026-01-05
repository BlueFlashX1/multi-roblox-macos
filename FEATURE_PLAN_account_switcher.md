# Feature: Account Switcher

## Overview
Store and manage multiple Roblox account profiles for quick switching.

## UI Design
```
┌─────────────────────────────────────┐
│  Multi Roblox MacOS - Accounts      │
├─────────────────────────────────────┤
│  Saved Profiles:                    │
│                                     │
│  ┌─────────────────────────────┐   │
│  │ 📝 Main Account          [Launch]│
│  │ 📝 Trading Alt           [Launch]│
│  │ 📝 AFK Account           [Launch]│
│  └─────────────────────────────┘   │
│                                     │
│  [+ Add Profile] [⚙️ Manage]        │
└─────────────────────────────────────┘
```

## Implementation Steps

### 1. Profile Storage Module
File: `internal/account_profiles/profiles.go`

```go
package account_profiles

type Profile struct {
    Name        string
    CookieData  string // Encrypted
    Notes       string
    LastUsed    time.Time
    Color       string // For UI
}

func LoadProfiles() ([]Profile, error)
func SaveProfile(p Profile) error
func DeleteProfile(name string) error
func LaunchWithProfile(profile Profile) error
```

### 2. Cookie Management
**IMPORTANT SECURITY CONSIDERATIONS:**
- Store cookies encrypted (AES-256)
- Use macOS Keychain for encryption key
- Never log cookie data
- Clear from memory after use

File: `internal/cookie_manager/cookie.go`

```go
package cookie_manager

func EncryptCookie(data string) (string, error)
func DecryptCookie(encrypted string) (string, error)
func InjectCookieToRoblox(pid int, cookie string) error
```

### 3. UI Components
- Profile list with colored tags
- Add/Edit profile dialog
- Launch button per profile
- Settings to manage profiles

## Technical Details

### Security Architecture
1. **Encryption**: AES-256-GCM for cookie storage
2. **Key Storage**: macOS Keychain API
3. **File Permissions**: 0600 (user read/write only)
4. **Memory**: Clear sensitive data immediately after use

### Storage Location
```
~/Library/Application Support/multi_roblox_macos/
├── profiles.json (encrypted)
└── config.json
```

### Cookie Injection Method
1. Read Roblox cookie file location
2. Backup existing cookie
3. Write new cookie
4. Launch Roblox instance
5. Optionally restore original cookie

**Roblox Cookie Locations:**
- macOS: `~/Library/Application Support/Roblox/LocalStorage/`
- Chrome: `~/Library/Application Support/Google/Chrome/Default/Cookies`

## Implementation Phases

### Phase 1: Basic Storage
- Create profile structure
- Implement file I/O
- Add basic encryption

### Phase 2: Cookie Management
- Locate Roblox cookie files
- Implement safe cookie injection
- Add backup/restore mechanism

### Phase 3: UI Integration
- Profile list widget
- Add/Edit dialogs
- Launch integration

### Phase 4: Security Hardening
- Implement Keychain integration
- Add cookie validation
- Secure memory handling

## Security Warnings

⚠️ **IMPORTANT**: This feature handles sensitive user data. Must include:

1. Clear privacy disclosure
2. Local-only storage (never cloud)
3. Secure deletion option
4. User consent dialog

## Testing

### Security Testing
- Verify encryption works
- Check file permissions
- Test Keychain integration
- Validate cookie injection

### Functional Testing
- Add/Edit/Delete profiles
- Launch with different profiles
- Handle concurrent instances
- Test error cases

## Ethical Considerations

This feature must:
1. Comply with Roblox Terms of Service
2. Only work with legitimate accounts
3. Not enable automation/botting
4. Respect user privacy

## Future Enhancements
- Profile export/import (encrypted)
- Auto-switch based on time
- Profile groups/categories
- Two-factor auth integration
