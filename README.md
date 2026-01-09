# Multi Roblox Manager for macOS

[![Go](https://img.shields.io/badge/Go-00ADD8?style=flat-square&logo=go&logoColor=white)](https://golang.org/)
[![macOS](https://img.shields.io/badge/macOS-000000?style=flat-square&logo=apple&logoColor=white)](https://www.apple.com/macos/)
[![Fyne](https://img.shields.io/badge/Fyne-GUI-blue?style=flat-square)](https://fyne.io/)

> **Enhanced fork** of [Insadem/multi-roblox-macos](https://github.com/Insadem/multi-roblox-macos) with major feature additions.

Run multiple Roblox instances simultaneously on macOS with full account management.

---

## My Contributions

This fork extends the original multi-instance launcher with a complete account management system:

| Feature                 | Description                                                    |
| ----------------------- | -------------------------------------------------------------- |
| **Account Switcher**    | Switch between Roblox accounts with macOS Keychain integration |
| **Friends Manager**     | View and manage friends across accounts                        |
| **Instance Tracker**    | Track which account is running in each instance                |
| **Cookie Manager**      | Secure credential handling                                     |
| **Resource Monitor**    | Monitor CPU/memory usage per instance                          |
| **Preset Manager**      | Save and load launch configurations                            |
| **Label Manager**       | Color-coded instance labels for organization                   |
| **Discord Integration** | Parse Discord invite links for quick game joins                |

### Technical Highlights

- **+7,400 lines of Go code** added to original project
- **macOS Keychain** integration for secure credential storage
- **Fyne GUI framework** for native macOS interface
- Comprehensive error handling and session conflict fixes

---

## Installation

### Download (Recommended)

1. Download latest release from [Releases](https://github.com/BlueFlashX1/multi-roblox-macos/releases)
2. Open Terminal and run: `xattr -c /path/to/Multi\ Roblox\ Manager.app`
3. Launch the app

### Build from Source

```bash
# Install Fyne first
go install fyne.io/fyne/v2/cmd/fyne@latest

# Clone and build
git clone https://github.com/BlueFlashX1/multi-roblox-macos.git
cd multi-roblox-macos
fyne package -os darwin -icon ./resources/app_icon.png
```

---

## Usage

1. Launch Multi Roblox Manager
2. Add your Roblox accounts (stored securely in macOS Keychain)
3. Click play on Roblox website for each account
4. Use the manager to switch, label, and organize instances

---

## Hardware Support

| Platform                 | Status                      |
| ------------------------ | --------------------------- |
| Apple Silicon (M1/M2/M3) | ✅ Full compatibility       |
| Intel Mac                | ✅ Should work (not tested) |

---

## Project Structure

```
internal/
├── account_manager/      # Account CRUD operations
├── cookie_manager/       # Secure cookie handling
├── friends_manager/      # Friends list integration
├── instance_manager/     # Multi-instance orchestration
├── label_manager/        # Instance labeling
├── preset_manager/       # Launch presets
├── resource_monitor/     # CPU/memory monitoring
├── roblox_api/          # Roblox API integration
├── roblox_login/        # Authentication
├── roblox_session/      # Session management
└── thumbnail_cache/     # Avatar caching
```

---

## Credits

- Original project by [Insadem](https://github.com/Insadem/multi-roblox-macos)
- Enhanced by [BlueFlashX1](https://github.com/BlueFlashX1)
