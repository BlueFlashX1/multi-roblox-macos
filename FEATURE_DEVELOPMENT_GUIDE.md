# Feature Development Guide

## Setup Required

### Install Go (Required for Development)

```bash
# Install Go using Homebrew
brew install go

# Verify installation
go version
```

### Install Fyne CLI

```bash
go install fyne.io/fyne/v2/cmd/fyne@latest
```

## Feature Branches Created

We've created 5 feature branches for the top improvements:

1. **feature/instance-counter-manager** - Display and manage running instances
2. **feature/quick-launch-presets** - Save favorite games and quick launch
3. **feature/resource-monitor** - Monitor CPU/Memory per instance
4. **feature/instance-labeling** - Label instances for easy identification
5. **feature/account-switcher** - Store and switch between account profiles

## Build and Test

```bash
cd ~/Desktop/multi-roblox-macos

# Build the app
go build -o multi_roblox_macos .

# Run directly
./multi_roblox_macos

# Or package as macOS app
fyne package -os darwin -icon ./resources/app_icon.png
```

## Development Workflow

For each feature:

1. Checkout the feature branch
2. Implement the feature
3. Test thoroughly
4. Commit with descriptive messages
5. Optionally create PR to main branch

Example:
```bash
# Work on instance counter
git checkout feature/instance-counter-manager

# Make changes...
# Test...

git add .
git commit -m "Add instance counter with live refresh"

# Push to your fork
git push myfork feature/instance-counter-manager
```

## Current Repository State

```
fix-macos-damaged-app-error (pushed to PR #13)
├── feature/instance-counter-manager
├── feature/quick-launch-presets
├── feature/resource-monitor
├── feature/instance-labeling
└── feature/account-switcher
```

## Next Steps

1. Install Go and Fyne
2. Choose a feature branch to work on
3. Implement the feature
4. Test and iterate

Would you like me to start implementing any of these features?
