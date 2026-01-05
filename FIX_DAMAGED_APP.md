# Fix for "Application is Damaged" Error on macOS

## Problem
When downloading the pre-built `multi_roblox_arm.app` from releases, macOS may show:
- "The application is damaged and can't be opened"
- "Operation not supported" error
- The app works from command line but fails when clicked

## Root Cause
The issue occurs because:
1. The app needs additional Info.plist entries for proper macOS compatibility
2. Code signature becomes invalid after modifying Info.plist
3. Gatekeeper blocks apps with invalid or missing signatures

## Solution

### Automated Fix Script

Save this script as `fix_multi_roblox.sh` and run it:

```bash
#!/bin/bash
set -euo pipefail

# Script to fix multi_roblox_arm.app with proper Info.plist modifications

# Default source path - users should update this
APP_SOURCE="${1:-/Volumes/multi_roblox_mac/multi_roblox_arm.app}"
APP_DEST="$HOME/Desktop/multi_roblox_arm_fixed.app"

# Validate source exists
if [ ! -d "$APP_SOURCE" ]; then
    echo "Error: Source app not found at: $APP_SOURCE"
    echo "Usage: $0 /path/to/multi_roblox_arm.app"
    exit 1
fi

# Safely remove destination if it exists
if [ -d "$APP_DEST" ]; then
    echo "Removing existing fixed app..."
    rm -rf "$APP_DEST"
fi

echo "Copying app to Desktop..."
ditto "$APP_SOURCE" "$APP_DEST"

echo "Making app writable..."
chmod -R u+w "$APP_DEST"

echo "Creating temporary Info.plist with modifications..."
TEMP_PLIST=$(mktemp "${TMPDIR:-/tmp}/info_plist.XXXXXX")
trap 'rm -f "$TEMP_PLIST"' EXIT

cat > "$TEMP_PLIST" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple Computer//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key>
	<string>multi_roblox_macos</string>
	<key>CFBundleExecutable</key>
	<string>multi_roblox_macos</string>
	<key>CFBundleIdentifier</key>
	<string>com.example.multi_roblox_macos</string>
	<key>CFBundleIconFile</key>
	<string>icon.icns</string>
	<key>CFBundleShortVersionString</key>
	<string>0.0.1</string>
	<key>CFBundleSupportedPlatforms</key>
	<array>
		<string>MacOSX</string>
	</array>
	<key>CFBundleVersion</key>
	<string>1</string>
	<key>NSHighResolutionCapable</key>
	<true/>
	<key>NSSupportsAutomaticGraphicsSwitching</key>
	<true/>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>LSApplicationCategoryType</key>
	<string>public.app-category.</string>
	<key>LSMinimumSystemVersion</key>
	<string>10.11</string>
	<key>LSRequiresNativeExecution</key>
	<true/>
	<key>LSArchitecturePriority</key>
	<array>
		<string>arm64</string>
		<string>x86_64</string>
	</array>
	<key>NSCameraUsageDescription</key>
	<string>Requires Camera for 13+ Authentication check</string>
	<key>NSMicrophoneUsageDescription</key>
	<string>Requires Microphone for talking</string>
</dict>
</plist>
EOF

echo "Copying new Info.plist..."
cp "$TEMP_PLIST" "$APP_DEST/Contents/Info.plist"

echo "Removing extended attributes..."
xattr -cr "$APP_DEST"

echo "Ensuring executable permissions..."
chmod +x "$APP_DEST/Contents/MacOS/multi_roblox_macos"

echo "Removing old signature..."
codesign --remove-signature "$APP_DEST" 2>/dev/null || true

echo "Signing with ad-hoc signature..."
codesign --force --deep --sign - "$APP_DEST"

echo "Registering with Launch Services..."
/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister -f "$APP_DEST"

echo ""
echo "✅ Done! The fixed app is at: $APP_DEST"
echo ""
echo "To open the app:"
echo "1. Right-click the app and select 'Open'"
echo "2. Click 'Open' in the security dialog"
echo ""
echo "Or copy to Applications:"
echo "cp -R \"$APP_DEST\" /Applications/"
```

### Usage

1. Download the script or copy the content above
2. Make it executable:
   ```bash
   chmod +x fix_multi_roblox.sh
   ```
3. Run the script with the path to your app:
   ```bash
   ./fix_multi_roblox.sh /path/to/multi_roblox_arm.app
   ```
   Or use the default path (if mounted at `/Volumes/multi_roblox_mac/`):
   ```bash
   ./fix_multi_roblox.sh
   ```
4. The fixed app will be created at `~/Desktop/multi_roblox_arm_fixed.app`
5. Copy to Applications (optional):
   ```bash
   cp -R ~/Desktop/multi_roblox_arm_fixed.app /Applications/
   ```

### Manual Fix (Alternative)

If you prefer to fix it manually:

1. **Copy the app to a writable location:**
   ```bash
   cp -R /path/to/multi_roblox_arm.app ~/Desktop/
   ```

2. **Edit Info.plist:**
   - Right-click the app → Show Package Contents
   - Navigate to `Contents/Info.plist`
   - Add these keys before `</dict>`:
   ```xml
   <key>LSRequiresNativeExecution</key>
   <true/>
   <key>LSArchitecturePriority</key>
   <array>
       <string>arm64</string>
       <string>x86_64</string>
   </array>
   <key>NSCameraUsageDescription</key>
   <string>Requires Camera for 13+ Authentication check</string>
   <key>NSMicrophoneUsageDescription</key>
   <string>Requires Microphone for talking</string>
   ```

3. **Remove quarantine and re-sign:**
   ```bash
   xattr -cr ~/Desktop/multi_roblox_arm.app
   codesign --force --deep --sign - ~/Desktop/multi_roblox_arm.app
   ```

4. **Open the app:**
   - Right-click → Open (holding Option/Alt key)
   - Or go to System Settings → Privacy & Security and click "Open Anyway"

## What Each Step Does

1. **Info.plist modifications:**
   - `LSRequiresNativeExecution` - Forces native execution (prevents Rosetta issues)
   - `LSArchitecturePriority` - Prioritizes arm64 architecture
   - `NSCameraUsageDescription` - Required for camera permissions
   - `NSMicrophoneUsageDescription` - Required for microphone permissions

2. **xattr -cr** - Removes extended attributes including quarantine flags

3. **codesign --force --deep --sign -** - Creates a new ad-hoc signature

4. **lsregister** - Registers the app with macOS Launch Services

## Security Considerations

The script includes the following security features:

1. **Strict error handling** (`set -euo pipefail`) - Script fails fast on errors
2. **Input validation** - Checks if source app exists before proceeding
3. **Safe temporary files** - Uses `mktemp` with random names instead of predictable paths
4. **Cleanup on exit** - `trap` ensures temporary files are removed even if script fails
5. **Proper quoting** - All variables are quoted to prevent injection
6. **Command line argument support** - Allows passing app path as argument
7. **No privileged operations** - Script runs entirely with user permissions

**Important:** Always verify the source of the app before running this script. Only use it on apps downloaded from trusted sources.

## For Developers

To prevent this issue in future releases, add these keys to the Info.plist during the build process using Fyne's metadata configuration or by modifying the Info.plist template before packaging.

## Tested On
- macOS Sequoia (15.x)
- Apple Silicon (M1/M2/M3)
- App works both from Finder and command line after fix

## Credits
- **Original Developer**: [Insadem](https://github.com/Insadem) - Creator of [multi-roblox-macos](https://github.com/Insadem/multi-roblox-macos)
- **Fix Solution**: Developed through troubleshooting macOS Gatekeeper and code signing requirements for the stable branch release
