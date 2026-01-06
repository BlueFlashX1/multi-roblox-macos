# Security Architecture - Multi Roblox Manager

## Overview

This document details the security measures implemented to protect user credentials and sensitive data.

## Password Storage

### ❌ NEVER Stored on Disk

**Passwords are NEVER saved to any file, database, or disk storage.**

### ✅ macOS Keychain Only

All passwords are stored exclusively in the **macOS Keychain**, which provides:

- **Hardware encryption** (Secure Enclave on modern Macs)
- **User authentication** required to access
- **Per-app access control**
- **Automatic encryption at rest**
- **System-level security** managed by macOS

### Implementation Details

```bash
# Password is stored with these security flags:
security add-generic-password \
  -s "multi-roblox-manager" \    # Service name
  -a "{account_id}" \             # Account identifier
  -w "{password}" \               # Password (encrypted by Keychain)
  -T "" \                         # Trusted apps: EMPTY = only this app
  -U                              # Update if exists
```

**Key Security Features:**

1. `-T ""` flag = **App-specific access** - Only this app can access the password
2. macOS prompts for user authentication when accessing passwords
3. Passwords encrypted with user's login keychain master key
4. Hardware-backed encryption on T2/M1/M2/M3 Macs

## Data Storage Locations

### Account Metadata (Username, Labels)

**File**: `~/Library/Application Support/multi_roblox_macos/accounts.json`
**Contains**: Usernames, labels, account IDs
**Does NOT contain**: Passwords, cookies, tokens

**Permissions**: `0600` (owner read/write only)

```json
{
  "id": "account_1",
  "username": "example_user",
  "label": "Main Account"
}
```

### Instance Tracking

**File**: `~/Library/Application Support/multi_roblox_macos/instance_accounts.json`
**Contains**: PID → Account ID mappings
**Permissions**: `0600`

### Presets

**File**: `~/Library/Application Support/multi_roblox_macos/presets.json`
**Contains**: Game names, URLs, thumbnails, last used account ID, private server links
**Permissions**: `0600` (owner read/write only)

## Security Guarantees

### ✅ What IS Secure

1. **Passwords**: macOS Keychain with hardware encryption
2. **File permissions**: User-only access (`0600`)
3. **No plaintext secrets**: Passwords never in memory longer than needed
4. **Memory clearing**: Passwords cleared after use
5. **HTTPS only**: All API calls use TLS 1.2+
6. **URL validation**: Injection prevention
7. **Timeout protection**: 10s limit on all HTTP requests

### ⚠️ User Responsibility

1. **macOS login password** protects Keychain
2. **System security** - Keep macOS updated
3. **FileVault** recommended for full disk encryption
4. **Don't share Mac user account** - Each macOS user has separate Keychain

## Multi-User Safety

### ✅ Automatic Isolation

**Each macOS user account has:**

- Separate Keychain (cannot access other users' passwords)
- Separate `~/Library` directory
- Separate app data storage

**If someone else downloads the app on the SAME Mac:**

- ✅ **Different macOS user** = Complete isolation, cannot access your data
- ⚠️ **Same macOS user** = Can access data (as intended for that user)

### Protection Against Shared Downloads

The app is **safe to share** because:

1. Each macOS user has their own Keychain
2. File paths use `~` (home directory) which is user-specific
3. No global/system-wide storage
4. Keychain requires macOS user authentication

**Example:**

- User A downloads app → Stores passwords in User A's Keychain
- User B downloads same app → Stores passwords in User B's Keychain
- **User A cannot access User B's passwords** (and vice versa)

## Network Security

### HTTPS Client Configuration

```go
client := &http.Client{
    Timeout: 10 * time.Second,
    Transport: &http.Transport{
        TLSClientConfig: &tls.Config{
            MinVersion: tls.VersionTLS12,  // TLS 1.2 minimum
        },
    },
}
```

### API Endpoints (Read-Only)

- `games.roblox.com` - Game information
- `thumbnails.roblox.com` - Game icons
- `apis.roblox.com` - Universe/Place ID lookup

**No authentication tokens sent to Roblox APIs** - All calls are public/read-only

## Threat Model

### ✅ Protected Against

1. **Malware reading password files** - No password files exist
2. **Other apps accessing passwords** - Keychain app-specific access
3. **Network eavesdropping** - TLS 1.2+ encryption
4. **Shared computer users** - macOS user isolation
5. **Disk theft** - FileVault + Keychain encryption
6. **Memory dumps** - Passwords cleared after use

### ⚠️ Cannot Protect Against

1. **Compromised macOS user account** - If attacker has your Mac login, they have Keychain access
2. **Keyloggers** - If malware logs keyboard, it can capture passwords when entered
3. **Screen recording malware** - Can see passwords if shown on screen
4. **Physical access + unlocked Mac** - Keychain accessible when Mac is unlocked

## Best Practices for Users

### Recommended

1. ✅ Use strong macOS login password
2. ✅ Enable FileVault (full disk encryption)
3. ✅ Lock Mac when away (⌘L)
4. ✅ Keep macOS updated
5. ✅ Use different password for each Roblox account

### Optional

1. Use Touch ID/Face ID for Keychain access
2. Enable firmware password (prevents boot from external drive)
3. Regular security audits: `security dump-keychain`

## Compliance Notes

### Data Retention

- **Passwords**: Deleted when account is removed from app
- **Metadata**: Deleted when account is removed
- **Instance tracking**: Auto-cleaned when instances close
- **Presets**: User-managed, persists until deleted

### Privacy

- **No telemetry** - App does not send usage data anywhere
- **No analytics** - No tracking of any kind
- **Local-only** - All data stays on your Mac
- **No cloud sync** - Accounts are device-specific

## Security Audit Checklist

- [x] Passwords in Keychain only (never on disk)
- [x] File permissions set to user-only (`0600`)
- [x] Memory cleared after password use
- [x] HTTPS with TLS 1.2+ minimum
- [x] URL validation and sanitization
- [x] Request timeouts (10s)
- [x] App-specific Keychain access (`-T ""`)
- [x] No hardcoded secrets
- [x] No logging of sensitive data
- [x] Multi-user isolation via macOS user accounts

## Reporting Security Issues

If you discover a security vulnerability, please:

1. **Do NOT** open a public GitHub issue
2. Contact the maintainer privately
3. Provide details of the vulnerability
4. Allow time for a fix before public disclosure

---

**Last Updated**: 2026-01-05
**Version**: 3.1.0
