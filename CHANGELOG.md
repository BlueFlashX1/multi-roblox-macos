# Changelog

All notable changes to this project are documented here.

The format is loosely based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [3.2.0] — 2026-05-16

Security + correctness release. ~18 commits across 3 TIER-prioritized batches with 2 independent audit rounds (Go-reviewer + security-reviewer + general code-reviewer agents).

### Security

- **Cookie/secret command injection eliminated** (`e001576`) — all 4 secret-bearing Python invocations in `cookie_manager` now pass values via `cmd.Env` (`MRM_ROBLOX_COOKIE`, `MRM_CRYPTO_KEY`, `MRM_ENC_B64`) with `const` script strings. A `.ROBLOSECURITY` cookie containing a single quote, newline, or backtick can no longer escape its string literal to execute arbitrary Python.
- **Auth ticket leakage closed** (`daa7e80`, `3a48b4d`) — `protocolString` was being logged in raw form (~59 chars of the live ticket landed in `preset_launch.log`). All log sites now emit `len + sha256[:4]` only.
- **Subprocess output leakage closed** (`e481668`) — every `fmt.Errorf("output: %s", string(output))` in `cookie_manager.go` redacted to length+hash; raw cookies no longer surface to `dialog.ShowError`.
- **Python supply-chain hardened** (`045215e`, `216dfa6`) — `browser_cookie3==0.19.1`, `pycryptodomex==3.20.0`, `lz4==4.3.3` pinned. System-wide install fallback removed. Unmapped packages now error instead of silently installing unpinned. Consent surfaced to log.
- **`/tmp` symlink-attack staging moved** (`f1d8bdc`, `f831918`) — multi-instance Roblox.app copies now stage at `~/Library/Caches/multi_roblox_macos/instances/` (mode 0700, current-user only). Pre-existing dirs `Chmod`'d to 0700 for self-healing.
- **HTTPStorages writes hardened** (`a725254`, `1b7ee4a`) — Go-side `lstat` + Python-side `os.open(O_NOFOLLOW)` close the symlink-replacement TOCTOU on Roblox cookie writes.
- **Vivaldi DB locking** (`6145817`, `f831918`) — `sqlite3` shell-out replaced with `database/sql`; `?_locking_mode=exclusive` applied to both `SetRobloxCookie` and `ClearVivaldiRobloxCookies` to fail-closed when Vivaldi is running.
- **`golang.org/x/net` v0.26.0 → v0.54.0** (`56287e2`) — CVE-2025-22870 (HTTP proxy bypass via IPv6 zone IDs), CVE-2025-22872 (XSS).
- **`LogSecret` helper + convention** (`c690c65`) — `logger.LogSecret(name, value)` emits `name(len=N, sha8=xxxxxxxx)` for safe secret-adjacent logging.

### Fixed

- **Mutex deadlock in `instance_account_tracker`** (`313d2ee`) — `LoadMappings` and `SaveMappings` both grabbed the same non-reentrant mutex; `TrackInstance` called both → deadlocked every multi-instance launch attempt. Refactored to single critical section per public method.
- **Fyne UI data races** (`63a3d09`, `d1f6ab8`, `fffd90e`) — `instanceList`, `counterLabel`, `systemStatsLabel`, and `selectWidget.Options` all migrated to goroutine-safe patterns (`sync.RWMutex`-guarded `currentInstances` + `widget.NewLabelWithData` + `SetOptions()`). Both background goroutines now `select` on a `context.Context` tied to `window.SetOnClosed`; `ticker.Stop()` runs via `defer`.
- **`os.RemoveAll` error swallowed in slot-full cleanup** (`f831918`) — `preset_manager.go:276` now logs via `logger.LogError`.
- **`/tmp` cleanup hardcoded to slots 2-10** (`f1d8bdc`) — replaced with `filepath.Glob("Roblox*.app")` over the new staging dir; instance >=11 no longer leaks.
- **Cookie validation freezing UI on "New Instance"** (`86f63e3`) — synchronous loop over accounts moved to a goroutine; dialog shows `[checking...]` placeholders that update reactively via `binding.String`.
- **Silenced errors surfaced** (`0f4f756`) — `rand.Read`, `os.UserHomeDir`, `LoadPresets`/`LoadAccounts` all return errors now. Fresh-install "file not found" treated as empty-state success rather than dialog spam (`5aed66a`).

### Added

- **"Close All" confirmation + zero-instance disable** (`44ea8c3`, `5aed66a`) — confirmation dialog with running-instance count; button disabled when count is 0.
- **Vivaldi-not-installed friendly error** (`44ea8c3`) — replaces raw file-path error with: *"Vivaldi browser is required for cookie capture. Download it at vivaldi.com, log into Roblox there, then try again."*
- **Color-only label save** (`44ea8c3`) — previously silently dropped if text was empty; now `SetLabel` fires whenever text OR color is set.
- **Typed Roblox API response structs** (`c690c65`) — high-frequency endpoints decode into concrete structs instead of `map[string]interface{}`. Remaining sites use comma-ok assertions.

### Changed

- **Build artifacts no longer committed** (`05ad125` + manual `git rm --cached`) — `multi_roblox_manager`, `multi_roblox_macos`, `*.dmg`, `Multi Roblox Manager.app/` removed from git tracking (kept on disk locally). `.gitignore` covers anchored binary paths plus `*.tar.gz`, `*.zip`, `*.exe`, `/build/`, `/dist/`.
- **Account-add without password no longer stores `"cookie_auth"` placeholder in Keychain** (`44ea8c3`) — skips Keychain entirely; cookie-only auth path is clean.

### Removed

- **~120 LOC of dead code** (`d4c443c`) — `copyFile`, `decryptChromeValue`, `decryptV10`, `decryptWithPython` (unused; the Python decrypt path is the actively-maintained one).
- **Unused exported globals** (`d4c443c`) — `AccountSwitchNeeded`, `PendingUsername` from `internal/roblox_login`.

### Audit trail

3 independent reviewer agents (`go-reviewer`, `security-reviewer`, `hydra-analyst`) across 2 rounds. Initial audit caught the TIER 1 set; re-audit caught residual gaps in 3 of those 5 fixes (sibling URL log path, off-thread label `SetText` calls, unpinned-package silent fallback). All gaps closed before the version bump.

---

## [3.1.0] — 2026-01-09

- Account switching, Friends, Private Servers (major feature drop)
- Multi-instance session conflict fix (Error 267)
- Cryptographically unique account IDs (timestamp + random bytes)
- Duplicate-username check on account add
- Python dep auto-install on first run
- Cookie capture restricted to Vivaldi only

## Earlier versions

See `git log --before=2026-01-09 --oneline` for the upstream `Insadem/multi-roblox-macos` history that this fork started from.
