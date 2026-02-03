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

### Step 1: Download

Download the latest `.dmg` file from [Releases](https://github.com/BlueFlashX1/multi-roblox-macos/releases/latest).

### Step 2: Install the App

1. **Double-click** the downloaded `.dmg` file to mount it
2. **Drag** `Multi Roblox Manager.app` into your `/Applications` folder
3. **Eject** the disk image (right-click → Eject, or drag to Trash)

> **Important:** You must copy the app to Applications first. Running it directly from the DMG will not work.

### Step 3: Remove Quarantine (Required)

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

### Step 4: Launch and Use

1. Open `Multi Roblox Manager` from your Applications folder
2. Go to [roblox.com](https://www.roblox.com) in your browser
3. Click **Play** on any game - the app will handle launching a new instance
4. Repeat for each account you want to run simultaneously

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

### "App is damaged and can't be opened"

Run the `xattr -c` command from Step 3 above.

### "Eject the disk image" when opening

You're running the app from the DMG. Copy it to `/Applications` first (Step 2).

### "No such file or directory" in Terminal

The app path is wrong. See the troubleshooting dropdown in Step 3.

### App won't open after xattr command

Try right-clicking the app → Open → Open (this bypasses Gatekeeper once).

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
