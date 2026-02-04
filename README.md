# Multi Roblox Manager for macOS

[![Go](https://img.shields.io/badge/Go-00ADD8?style=flat-square&logo=go&logoColor=white)](https://golang.org/)
[![macOS](https://img.shields.io/badge/macOS-000000?style=flat-square&logo=apple&logoColor=white)](https://www.apple.com/macos/)

> Enhanced fork of [Insadem/multi-roblox-macos](https://github.com/Insadem/multi-roblox-macos)

Run multiple Roblox instances with full account management.

## ⚠️ Vivaldi Browser Required

**Cookie capture currently only works with [Vivaldi browser](https://vivaldi.com/download/).**

I forked this project because the original wasn't working reliably, and enhanced it to work with Vivaldi (my default browser). If this repo gains wider attention, I may expand support to other browsers (Chrome, Firefox, Safari, Arc) for cookie capture—enabling quick account switching without needing to re-login.

<p align="center">
  <a href="https://github.com/BlueFlashX1/multi-roblox-macos/releases/latest">
    <img src="https://img.shields.io/badge/Download-Latest%20Release-brightgreen?style=for-the-badge&logo=apple" alt="Download" />
  </a>
</p>

---

## Features Added

| Feature              | Description                                     |
| -------------------- | ----------------------------------------------- |
| **Account Switcher** | Switch accounts via Vivaldi cookie capture + macOS Keychain |
| **Friends Manager**  | View and manage friends across accounts         |
| **Instance Tracker** | See which account runs in each instance         |
| **Resource Monitor** | CPU/memory usage per instance                   |
| **Preset Manager**   | Save and load launch configurations             |
| **Label Manager**    | Color-coded instance organization               |

---

## Installation

### Step 1: Install Vivaldi Browser (Required)

Cookie capture only works with Vivaldi. Other browsers are not currently supported.

[Download Vivaldi](https://vivaldi.com/download/)

### Step 2: Install Python (Required for Cookie Capture)

The app uses Python to read Vivaldi's cookies. If you don't have Python installed:

**Option A: Homebrew (Recommended)**

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
brew install python
```

**Option B: Download from [python.org/downloads](https://www.python.org/downloads/)**

Verify with: `python3 --version`

### Step 3: Install Required Python Modules

```bash
# Ensure pip is available (safe to run even if already installed)
python3 -m ensurepip --upgrade

# Upgrade pip and install required modules
python3 -m pip install --upgrade pip
python3 -m pip install browser-cookie3 lz4 pycryptodomex
```

<details>
<summary><strong>Troubleshooting</strong></summary>

**Permission denied errors:**

```bash
python3 -m pip install --user browser-cookie3 lz4 pycryptodomex
```

**Cookie errors with Chrome/Vivaldi 130+:**

```bash
pip install --force-reinstall git+https://github.com/borisbabic/browser_cookie3.git@refs/pull/215/head
```

</details>

### Step 4: Download & Install

1. Download from [Releases](https://github.com/BlueFlashX1/multi-roblox-macos/releases/latest)
2. Drag to `/Applications`
3. Run in Terminal: `xattr -c '/Applications/Multi Roblox Manager.app'`
4. Launch the app

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
| Apple Silicon (M1/M2/M3/M4) | ✅ Works       |
| Intel Mac                | ⚠️ Untested    |

---

## Credits

Original by [Insadem](https://github.com/Insadem) • Enhanced by [BlueFlashX1](https://github.com/BlueFlashX1)
