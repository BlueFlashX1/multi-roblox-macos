# Multi Roblox Manager for macOS

[![Go](https://img.shields.io/badge/Go-00ADD8?style=flat-square&logo=go&logoColor=white)](https://golang.org/)
[![macOS](https://img.shields.io/badge/macOS-000000?style=flat-square&logo=apple&logoColor=white)](https://www.apple.com/macos/)

> Enhanced fork of [Insadem/multi-roblox-macos](https://github.com/Insadem/multi-roblox-macos)

Run multiple Roblox instances simultaneously with full account management.

<p align="center">
  <a href="https://github.com/BlueFlashX1/multi-roblox-macos/releases/latest">
    <img src="https://img.shields.io/badge/Download-Latest%20Release-brightgreen?style=for-the-badge&logo=apple" alt="Download" />
  </a>
</p>

---

## Installation

### Step 1: Install Python (Required for Cookie Capture)

The app uses Python to read your browser cookies. If you've never installed Python, follow these steps:

**Option A: Install via Homebrew (Recommended)**

```bash
# Install Homebrew if you don't have it
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install Python
brew install python
```

**Option B: Download from Python.org**

1. Go to [python.org/downloads](https://www.python.org/downloads/)
2. Download the latest Python 3.x for macOS
3. Run the installer and follow the prompts

**Verify Python is installed:**

```bash
python3 --version
```

You should see something like `Python 3.12.x`

---

### Step 2: Install Required Python Modules

Open Terminal and run these commands:

```bash
# Upgrade pip first
python3 -m pip install --upgrade pip

# Install all required modules
python3 -m pip install browser-cookie3 lz4 pycryptodomex
```

<details>
<summary><strong>Troubleshooting Python/pip issues</strong></summary>

**If you get "pip not found" error:**

```bash
python3 -m ensurepip --upgrade
python3 -m pip install --upgrade pip
python3 -m pip install browser-cookie3 lz4 pycryptodomex
```

**If you get "Permission denied" errors:**

```bash
python3 -m pip install --user browser-cookie3 lz4 pycryptodomex
```

**If you get cookie decryption errors with Chrome/Vivaldi 130+:**

```bash
pip install --force-reinstall git+https://github.com/borisbabic/browser_cookie3.git@refs/pull/215/head
```

</details>

---

### Step 3: Download the App

Download the latest `.dmg` file from [Releases](https://github.com/BlueFlashX1/multi-roblox-macos/releases/latest).

---

### Step 4: Install the App

1. **Double-click** the downloaded `.dmg` file to mount it
2. **Drag** `Multi Roblox Manager.app` into your `/Applications` folder
3. **Eject** the disk image (right-click → Eject, or drag to Trash)

> **Important:** You must copy the app to Applications first. Running it directly from the DMG will not work.

---

### Step 5: Remove Quarantine (Required)

macOS blocks apps from unidentified developers. To fix this:

1. Open **Terminal** (search "Terminal" in Spotlight, or find it in Applications → Utilities)
2. Copy and paste this command, then press Enter:

```bash
xattr -c '/Applications/Multi Roblox Manager.app'
```

1. Enter your password if prompted (you won't see characters as you type - this is normal)

<details>
<summary><strong>Troubleshooting: "No such file or directory"</strong></summary>

This error means the app isn't where the command expects. Try these alternatives:

**If the app is still in Downloads:**

```bash
xattr -c ~/Downloads/Multi\ Roblox\ Manager.app
```

**If you're unsure where the app is, run this to find it:**

```bash
find /Applications ~/Downloads -name "Multi Roblox Manager.app" 2>/dev/null
```

Then use the path it finds in the `xattr -c` command.

</details>

---

### Step 6: Launch and Use

1. **Make sure you're logged into Roblox** in your browser (Vivaldi/Chrome/Safari)
2. Open `Multi Roblox Manager` from your Applications folder
3. Go to [roblox.com](https://www.roblox.com) in your browser
4. Click **Play** on any game - the app will handle launching a new instance
5. Repeat for each account you want to run simultaneously

---

## Quick Setup (Copy & Paste)

For those who want all commands in one place:

```bash
# 1. Install Homebrew (skip if you already have it)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# 2. Install Python and required modules
brew install python
python3 -m pip install --upgrade pip
python3 -m pip install browser-cookie3 lz4 pycryptodomex

# 3. Remove quarantine from the app (after downloading and installing)
xattr -c '/Applications/Multi Roblox Manager.app'

echo "Setup complete! Make sure you're logged into Roblox in your browser, then launch the app."
```

---

## Features

| Feature              | Description                                     |
| -------------------- | ----------------------------------------------- |
| **Multi-Instance**   | Run multiple Roblox games at the same time      |
| **Account Switcher** | Switch accounts with macOS Keychain integration |
| **Friends Manager**  | View and manage friends across accounts         |
| **Instance Tracker** | See which account runs in each instance         |
| **Resource Monitor** | CPU/memory usage per instance                   |
| **Preset Manager**   | Save and load launch configurations             |
| **Label Manager**    | Color-coded instance organization               |
| **Teleport Support** | Works with in-game teleports                    |

---

## Hardware Compatibility

| Platform                    | Status                |
| --------------------------- | --------------------- |
| Apple Silicon (M1/M2/M3/M4) | ✅ Tested and working |
| Intel Mac                   | ⚠️ Untested           |

---

## Common Issues

### "No module named 'browser_cookie3'" or "Failed to capture cookie"

Python modules aren't installed. Run:

```bash
python3 -m pip install browser-cookie3 lz4 pycryptodomex
```

If that doesn't work, see Step 2 above for troubleshooting.

### Cookie decryption errors (Chrome/Vivaldi 130+)

Newer browser versions need a patched version:

```bash
pip install --force-reinstall git+https://github.com/borisbabic/browser_cookie3.git@refs/pull/215/head
```

Also make sure your browser is **fully closed** before running the app.

### "python3: command not found"

Python isn't installed. Follow Step 1 above to install it.

### "App is damaged and can't be opened"

Run the `xattr -c` command from Step 5 above.

### "Eject the disk image" when opening

You're running the app from the DMG. Copy it to `/Applications` first (Step 4).

### "No such file or directory" in Terminal

The app path is wrong. See the troubleshooting dropdown in Step 5.

### App won't open after xattr command

Try right-clicking the app → Open → Open (this bypasses Gatekeeper once).

### "Make sure you're logged into Roblox in Vivaldi"

1. Open your browser and go to [roblox.com](https://www.roblox.com)
2. Log in to your Roblox account
3. **Close the browser completely** (Cmd+Q, not just closing the window)
4. Try running the app again

---

## Build from Source (Optional)

```bash
go install fyne.io/fyne/v2/cmd/fyne@latest
git clone https://github.com/BlueFlashX1/multi-roblox-macos.git
cd multi-roblox-macos
fyne package -os darwin -icon ./resources/app_icon.png
```

---

## Credits

Original by [Insadem](https://github.com/Insadem) • Enhanced by [BlueFlashX1](https://github.com/BlueFlashX1)
