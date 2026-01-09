# Multi Roblox Manager for macOS

[![Go](https://img.shields.io/badge/Go-00ADD8?style=flat-square&logo=go&logoColor=white)](https://golang.org/)
[![macOS](https://img.shields.io/badge/macOS-000000?style=flat-square&logo=apple&logoColor=white)](https://www.apple.com/macos/)

> Enhanced fork of [Insadem/multi-roblox-macos](https://github.com/Insadem/multi-roblox-macos)

Run multiple Roblox instances with full account management.

<p align="center">
  <a href="https://github.com/BlueFlashX1/multi-roblox-macos/releases/latest">
    <img src="https://img.shields.io/badge/Download-Latest%20Release-brightgreen?style=for-the-badge&logo=apple" alt="Download" />
  </a>
</p>

---

## Features Added

| Feature              | Description                                     |
| -------------------- | ----------------------------------------------- |
| **Account Switcher** | Switch accounts with macOS Keychain integration |
| **Friends Manager**  | View and manage friends across accounts         |
| **Instance Tracker** | See which account runs in each instance         |
| **Resource Monitor** | CPU/memory usage per instance                   |
| **Preset Manager**   | Save and load launch configurations             |
| **Label Manager**    | Color-coded instance organization               |

---

## Installation

1. Download from [Releases](https://github.com/BlueFlashX1/multi-roblox-macos/releases/latest)
2. Run in Terminal: `xattr -c /path/to/Multi\ Roblox\ Manager.app`
3. Launch the app

---

## Build from Source

```bash
go install fyne.io/fyne/v2/cmd/fyne@latest
git clone https://github.com/BlueFlashX1/multi-roblox-macos.git
cd multi-roblox-macos
fyne package -os darwin -icon ./resources/app_icon.png
```

---

## Hardware

| Platform                 | Status         |
| ------------------------ | -------------- |
| Apple Silicon (M1/M2/M3) | ✅ Works       |
| Intel Mac                | ✅ Should work |

---

## Credits

Original by [Insadem](https://github.com/Insadem) • Enhanced by [BlueFlashX1](https://github.com/BlueFlashX1)
