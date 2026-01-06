package main

import (
	"fmt"
	"image/color"
	"insadem/multi_roblox_macos/internal/account_manager"
	"insadem/multi_roblox_macos/internal/close_all_app_instances"
	"insadem/multi_roblox_macos/internal/cookie_manager"
	"insadem/multi_roblox_macos/internal/discord_link_parser"
	"insadem/multi_roblox_macos/internal/discord_redirect"
	"insadem/multi_roblox_macos/internal/friends_manager"
	"insadem/multi_roblox_macos/internal/instance_account_tracker"
	"insadem/multi_roblox_macos/internal/instance_manager"
	"insadem/multi_roblox_macos/internal/label_manager"
	"insadem/multi_roblox_macos/internal/logger"
	"insadem/multi_roblox_macos/internal/preset_manager"
	"insadem/multi_roblox_macos/internal/resource_monitor"
	"insadem/multi_roblox_macos/internal/roblox_api"
	"insadem/multi_roblox_macos/internal/roblox_login"
	"insadem/multi_roblox_macos/internal/roblox_session"
	"insadem/multi_roblox_macos/internal/thumbnail_cache"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

//go:generate fyne bundle -o bundled.go ./resources/discord.png
//go:generate fyne bundle -o bundled.go -append ./resources/more.png
//go:generate fyne bundle -o bundled.go -append ./resources/mop.png

func main() {
	// Initialize logger
	if err := logger.InitLogger(); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
	}
	defer logger.Close()

	logger.LogInfo("Multi Roblox Manager started")

	// Cleanup old /tmp Roblox copies on startup
	cookie_manager.CleanupTempRobloxCopies()

	// Auto-refresh expired cookies on startup and periodically
	go func() {
		refreshCookies := func() {
			accounts, err := account_manager.LoadAccounts()
			if err == nil && len(accounts) > 0 {
				var accountIDs []string
				accountUsernames := make(map[string]string)
				for _, acc := range accounts {
					accountIDs = append(accountIDs, acc.ID)
					accountUsernames[acc.ID] = acc.Username
				}

				results := cookie_manager.AutoRefreshExpiredCookies(accountIDs, accountUsernames)
				for _, r := range results {
					if r.Refreshed {
						logger.LogInfo("Auto-refreshed cookie for %s", r.Username)
					} else if r.WasExpired && r.Error != "" {
						logger.LogDebug("Could not auto-refresh %s: %s", r.Username, r.Error)
					}
				}
			}
		}

		// Initial refresh on startup
		refreshCookies()

		// Periodic refresh every 30 minutes
		ticker := time.NewTicker(30 * time.Minute)
		for range ticker.C {
			logger.LogInfo("Periodic cookie refresh check...")
			refreshCookies()
		}
	}()

	mainApp := app.New()
	window := mainApp.NewWindow("Multi Roblox Manager")
	window.Resize(fyne.NewSize(500, 600))

	// Create tabs
	tabs := container.NewAppTabs(
		container.NewTabItem("Instances", createInstancesTab(window)),
		container.NewTabItem("Presets", createPresetsTab(window)),
		container.NewTabItem("Accounts", createAccountsTab(window)),
		container.NewTabItem("Friends", createFriendsTab(window)),
		container.NewTabItem("About", createAboutTab(window)),
	)

	window.SetContent(tabs)

	// Cleanup on app close
	window.SetOnClosed(func() {
		logger.LogInfo("App closing, cleaning up temporary files...")
		cookie_manager.CleanupTempRobloxCopies()
		logger.LogInfo("Cleanup complete, goodbye!")
	})

	window.ShowAndRun()
}

func createInstancesTab(window fyne.Window) fyne.CanvasObject {
	// Instance counter and system stats labels
	counterLabel := widget.NewLabel("Running Instances: 0")
	counterLabel.TextStyle = fyne.TextStyle{Bold: true}

	systemStatsLabel := widget.NewLabel("System: CPU 0% | Memory 0 MB / 0 MB")

	// Instance list with resource stats and labels
	instanceList := widget.NewList(
		func() int { return 0 },
		func() fyne.CanvasObject {
			colorIndicator := canvas.NewRectangle(color.Transparent)
			colorIndicator.SetMinSize(fyne.NewSize(4, 40))

			return container.NewBorder(
				nil, nil,
				colorIndicator,
				container.NewVBox(
					widget.NewButton("Label", nil),
					widget.NewButton("Close", nil),
				),
				container.NewVBox(
					widget.NewLabel("Instance"),
					widget.NewLabel("Resources"),
				),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {},
	)

	var currentInstances []instance_manager.Instance
	var updateInstances func()

	// Update instance list function
	updateInstances = func() {
		instances, err := instance_manager.GetRunningInstances()
		if err != nil {
			return
		}

		currentInstances = instances
		counterLabel.SetText(fmt.Sprintf("Running Instances: %d", len(instances)))

		// Update system stats
		cpuPercent, memUsed, memTotal, err := resource_monitor.GetSystemStats()
		if err == nil {
			systemStatsLabel.SetText(fmt.Sprintf("System: CPU %.1f%% | Memory %s / %s",
				cpuPercent,
				resource_monitor.FormatMemory(memUsed),
				resource_monitor.FormatMemory(memTotal)))
		}

		instanceList.Length = func() int {
			return len(currentInstances)
		}

		instanceList.UpdateItem = func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(currentInstances) {
				return
			}

			instance := currentInstances[id]
			border := obj.(*fyne.Container)

			// BorderLayout NewBorder(top, bottom, left, right, center) stores objects as:
			// Objects[0] = center, Objects[1] = left, Objects[2] = right
			// (only non-nil objects are stored)
			labelBox := border.Objects[0].(*fyne.Container)         // Center
			colorIndicator := border.Objects[1].(*canvas.Rectangle) // Left
			buttonBox := border.Objects[2].(*fyne.Container)        // Right

			instanceLabel := labelBox.Objects[0].(*widget.Label)
			resourceLabel := labelBox.Objects[1].(*widget.Label)

			labelButton := buttonBox.Objects[0].(*widget.Button)
			closeButton := buttonBox.Objects[1].(*widget.Button)

			// Set color indicator
			if instance.Color != "" {
				if c, err := parseHexColor(instance.Color); err == nil {
					colorIndicator.FillColor = c
				} else {
					colorIndicator.FillColor = color.Transparent
				}
			} else {
				colorIndicator.FillColor = color.Transparent
			}
			colorIndicator.Refresh()

			// Get resource stats for this instance
			stats, err := resource_monitor.GetProcessStats(instance.PID)
			resourceInfo := ""
			if err == nil {
				resourceInfo = fmt.Sprintf("CPU: %.1f%% | Memory: %s",
					stats.CPUPercent,
					resource_monitor.FormatMemory(stats.MemoryMB))
			} else {
				resourceInfo = "Stats unavailable"
			}

			// Set instance label text with account info
			labelText := ""
			if instance.Label != "" {
				labelText = fmt.Sprintf("%s (PID: %d)", instance.Label, instance.PID)
			} else {
				labelText = fmt.Sprintf("Instance %d (PID: %d)", id+1, instance.PID)
			}

			// Add account info if available
			if accountID, found := instance_account_tracker.GetAccountForInstance(instance.PID); found {
				if account, err := account_manager.GetAccount(accountID); err == nil {
					accountLabel := account.Username
					if account.Label != "" {
						accountLabel = account.Label
					}
					labelText += fmt.Sprintf(" - 👤 %s", accountLabel)
				}
			} else if instance.Label == "" {
				// Untracked instance - prompt user to label it
				labelText += " - ❓ Unknown account"
			}

			instanceLabel.SetText(labelText)
			resourceLabel.SetText(resourceInfo)

			// Label button
			labelButton.OnTapped = func() {
				showLabelDialog(window, instance.PID, updateInstances)
			}

			// Close button
			closeButton.OnTapped = func() {
				instance_manager.CloseInstance(instance.PID)
				updateInstances()
			}
		}

		instanceList.Refresh()
	}

	// Auto-refresh every 2 seconds
	go func() {
		for {
			time.Sleep(2 * time.Second)
			updateInstances()
		}
	}()

	// Initial update
	updateInstances()

	// Buttons
	newInstanceButton := widget.NewButtonWithIcon("New Instance", resourceMorePng, func() {
		showAccountSelectionDialog(window, func() {
			time.Sleep(500 * time.Millisecond)
			updateInstances()
		})
	})

	closeAllButton := widget.NewButtonWithIcon("Close All", resourceMopPng, func() {
		close_all_app_instances.Close("RobloxPlayer")
		updateInstances()
	})

	// Layout
	return container.NewBorder(
		container.NewVBox(
			counterLabel,
			systemStatsLabel,
			widget.NewSeparator(),
		),
		container.NewVBox(
			widget.NewSeparator(),
			newInstanceButton,
			closeAllButton,
		),
		nil,
		nil,
		instanceList,
	)
}

func createPresetsTab(window fyne.Window) fyne.CanvasObject {
	// Load presets
	presets, _ := preset_manager.LoadPresets()
	var presetList *widget.List

	// Preset list with card layout
	presetList = widget.NewList(
		func() int { return len(presets) },
		func() fyne.CanvasObject {
			// Create card-style layout with thumbnail using HBox for predictable structure
			thumbnail := canvas.NewImageFromFile("")
			thumbnail.FillMode = canvas.ImageFillContain
			thumbnail.SetMinSize(fyne.NewSize(80, 80))

			nameLabel := widget.NewLabel("Game Name")
			nameLabel.TextStyle = fyne.TextStyle{Bold: true}

			urlLabel := widget.NewLabel("URL")
			urlLabel.TextStyle = fyne.TextStyle{Italic: true}

			serverLabel := widget.NewLabel("") // Shows private server status

			buttonBox := container.NewHBox(
				widget.NewButton("Launch", nil),
				widget.NewButton("Settings", nil),
				widget.NewButton("Delete Preset", nil),
			)

			infoBox := container.NewVBox(
				nameLabel,
				urlLabel,
				serverLabel,
				buttonBox,
			)

			return container.NewHBox(
				thumbnail,
				infoBox,
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(presets) {
				return
			}

			preset := presets[id]
			hbox := obj.(*fyne.Container)

			thumbnail := hbox.Objects[0].(*canvas.Image)
			infoBox := hbox.Objects[1].(*fyne.Container)

			nameLabel := infoBox.Objects[0].(*widget.Label)
			urlLabel := infoBox.Objects[1].(*widget.Label)
			serverLabel := infoBox.Objects[2].(*widget.Label)
			buttonBox := infoBox.Objects[3].(*fyne.Container)

			launchBtn := buttonBox.Objects[0].(*widget.Button)
			settingsBtn := buttonBox.Objects[1].(*widget.Button)
			deleteBtn := buttonBox.Objects[2].(*widget.Button)

			// Set game name
			nameLabel.SetText(preset.Name)

			// Set URL (truncated)
			urlText := preset.URL
			if len(urlText) > 40 {
				urlText = urlText[:37] + "..."
			}
			urlLabel.SetText(urlText)

			// Show private server status
			if preset.PrivateServerLinkCode != "" {
				serverLabel.SetText("🔒 Private Server configured")
			} else {
				serverLabel.SetText("")
			}

			// Load and display thumbnail
			if preset.ThumbnailURL != "" {
				// Try to get cached thumbnail first
				if cachedPath, found := thumbnail_cache.GetCachedThumbnail(preset.ThumbnailURL); found {
					thumbnail.File = cachedPath
					thumbnail.Refresh()
				} else {
					// Download and cache in background
					go func() {
						if localPath, err := thumbnail_cache.DownloadAndCacheThumbnail(preset.ThumbnailURL); err == nil {
							thumbnail.File = localPath
							thumbnail.Refresh()
						}
					}()
				}
			} else {
				// No thumbnail - show placeholder
				thumbnail.Resource = nil
				thumbnail.File = ""
			}

			launchBtn.OnTapped = func() {
				showAccountSelectionForPreset(window, preset, id, func() {
					time.Sleep(500 * time.Millisecond)
				})
			}

			settingsBtn.OnTapped = func() {
				showPresetSettingsDialog(window, preset, id, func() {
					presets, _ = preset_manager.LoadPresets()
					presetList.Refresh()
				})
			}

			deleteBtn.OnTapped = func() {
				dialog.ShowConfirm("Delete Preset",
					fmt.Sprintf("Delete preset '%s'?", preset.Name),
					func(yes bool) {
						if yes {
							preset_manager.DeletePreset(id)
							presets, _ = preset_manager.LoadPresets()
							presetList.Refresh()
						}
					}, window)
			}
		},
	)

	// Add preset button
	addButton := widget.NewButton("Add Preset", func() {
		nameEntry := widget.NewEntry()
		nameEntry.SetPlaceHolder("Name (optional - auto-fetched from URL)")

		urlEntry := widget.NewEntry()
		urlEntry.SetPlaceHolder("Roblox URL (e.g., roblox://placeId=123456)")

		formItems := []*widget.FormItem{
			widget.NewFormItem("Name", nameEntry),
			widget.NewFormItem("URL", urlEntry),
		}

		infoLabel := widget.NewLabel("Leave name blank to auto-fetch game name and thumbnail")
		infoLabel.Wrapping = fyne.TextWrapWord

		formContainer := container.NewVBox()
		for _, item := range formItems {
			formContainer.Add(widget.NewLabel(item.Text))
			formContainer.Add(item.Widget)
		}

		content := container.NewVBox(
			infoLabel,
			widget.NewSeparator(),
			formContainer,
		)

		customDialog := dialog.NewCustomConfirm("Add Preset", "Add", "Cancel", content, func(ok bool) {
			if ok && urlEntry.Text != "" {
				// Show loading message
				progress := dialog.NewProgressInfinite("Fetching Game Info", "Please wait...", window)
				progress.Show()

				// Add preset (will auto-fetch if name is empty)
				go func() {
					preset_manager.AddPreset(nameEntry.Text, urlEntry.Text)
					presets, _ = preset_manager.LoadPresets()
					progress.Hide()
					presetList.Refresh()
				}()
			}
		}, window)

		customDialog.Show()
	})

	// Layout
	return container.NewBorder(
		nil,
		container.NewVBox(
			widget.NewSeparator(),
			addButton,
			widget.NewLabel("Tip: Find game URLs on roblox.com, they look like:\nroblox://placeId=123456 or https://www.roblox.com/games/123456/"),
		),
		nil,
		nil,
		presetList,
	)
}

// Global friends state for periodic refresh
var (
	friendsListWidget    *widget.List
	friendsData          []friends_manager.Friend
	friendsStatusCache   map[int64]roblox_api.UserPresence
	friendsStatusLock    sync.RWMutex
	friendsRefreshTicker *time.Ticker
)

func createFriendsTab(window fyne.Window) fyne.CanvasObject {
	logger.LogInfo("Creating Friends tab")

	friendsStatusCache = make(map[int64]roblox_api.UserPresence)

	// Header with count
	headerLabel := widget.NewLabel("👥 Friends List")
	headerLabel.TextStyle = fyne.TextStyle{Bold: true}

	countLabel := widget.NewLabel("Loading...")

	// Load friends
	var err error
	friendsData, err = friends_manager.LoadFriends()
	if err != nil {
		logger.LogError("Failed to load friends: %v", err)
		friendsData = []friends_manager.Friend{}
	}
	countLabel.SetText(fmt.Sprintf("%d friends saved", len(friendsData)))

	// Create friends list
	friendsListWidget = widget.NewList(
		func() int { return len(friendsData) },
		func() fyne.CanvasObject {
			// Template for each row
			nameLabel := widget.NewLabel("Username (Display Name)")
			nameLabel.TextStyle = fyne.TextStyle{Bold: true}
			statusLabel := widget.NewLabel("⚫ Offline")
			gameLabel := widget.NewLabel("")
			joinBtn := widget.NewButton("Join", nil)
			joinBtn.Importance = widget.HighImportance
			deleteBtn := widget.NewButton("Remove", nil)

			leftBox := container.NewVBox(nameLabel, statusLabel, gameLabel)
			rightBox := container.NewHBox(joinBtn, deleteBtn)

			return container.NewBorder(nil, nil, nil, rightBox, leftBox)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(friendsData) {
				return
			}
			friend := friendsData[id]

			borderContainer := obj.(*fyne.Container)
			leftBox := borderContainer.Objects[0].(*fyne.Container)
			rightBox := borderContainer.Objects[1].(*fyne.Container)

			nameLabel := leftBox.Objects[0].(*widget.Label)
			statusLabel := leftBox.Objects[1].(*widget.Label)
			gameLabel := leftBox.Objects[2].(*widget.Label)
			joinBtn := rightBox.Objects[0].(*widget.Button)
			deleteBtn := rightBox.Objects[1].(*widget.Button)

			// Display name
			displayText := friend.Username
			if friend.DisplayName != "" && friend.DisplayName != friend.Username {
				displayText = fmt.Sprintf("%s (@%s)", friend.DisplayName, friend.Username)
			}
			nameLabel.SetText(displayText)

			// Check status from cache
			friendsStatusLock.RLock()
			presence, hasStatus := friendsStatusCache[friend.UserID]
			friendsStatusLock.RUnlock()

			if hasStatus {
				switch presence.UserPresenceType {
				case 0:
					statusLabel.SetText("⚫ Offline")
					gameLabel.SetText("")
					joinBtn.Disable()
				case 1:
					statusLabel.SetText("🟢 Online (Website)")
					gameLabel.SetText("")
					joinBtn.Disable()
				case 2:
					statusLabel.SetText("🎮 In Game")
					if presence.LastLocation != "" {
						gameLabel.SetText(presence.LastLocation)
					}
					joinBtn.Enable()
				case 3:
					statusLabel.SetText("🔧 In Studio")
					gameLabel.SetText("")
					joinBtn.Disable()
				}
			} else {
				statusLabel.SetText("⏳ Checking...")
				gameLabel.SetText("")
				joinBtn.Disable()
			}

			// Join button - launches game to join this friend
			joinBtn.OnTapped = func() {
				showJoinFriendDialog(window, friend, presence)
			}

			// Delete button
			deleteBtn.OnTapped = func() {
				dialog.ShowConfirm("Remove Friend",
					fmt.Sprintf("Remove %s from your friends list?", friend.Username),
					func(ok bool) {
						if ok {
							if err := friends_manager.RemoveFriend(friend.UserID); err != nil {
								dialog.ShowError(err, window)
							} else {
								refreshFriendsList(countLabel)
							}
						}
					}, window)
			}
		},
	)

	// Add friend button
	addFriendBtn := widget.NewButton("➕ Add Friend", func() {
		showAddFriendDialog(window, countLabel)
	})
	addFriendBtn.Importance = widget.HighImportance

	// Refresh button
	refreshBtn := widget.NewButton("🔄 Refresh Status", func() {
		go refreshFriendsStatus()
	})

	buttonBox := container.NewHBox(addFriendBtn, refreshBtn)

	// Start periodic status refresh
	go startFriendsStatusRefresh()

	// Initial status check
	go refreshFriendsStatus()

	return container.NewBorder(
		container.NewVBox(
			headerLabel,
			countLabel,
			buttonBox,
			widget.NewSeparator(),
		),
		nil, nil, nil,
		friendsListWidget,
	)
}

func refreshFriendsList(countLabel *widget.Label) {
	var err error
	friendsData, err = friends_manager.LoadFriends()
	if err != nil {
		logger.LogError("Failed to reload friends: %v", err)
		friendsData = []friends_manager.Friend{}
	}
	countLabel.SetText(fmt.Sprintf("%d friends saved", len(friendsData)))
	if friendsListWidget != nil {
		friendsListWidget.Refresh()
	}
	go refreshFriendsStatus()
}

func startFriendsStatusRefresh() {
	if friendsRefreshTicker != nil {
		friendsRefreshTicker.Stop()
	}
	friendsRefreshTicker = time.NewTicker(30 * time.Second) // Refresh every 30 seconds

	for range friendsRefreshTicker.C {
		refreshFriendsStatus()
	}
}

func refreshFriendsStatus() {
	if len(friendsData) == 0 {
		return
	}

	logger.LogDebug("Refreshing friend statuses...")

	// Get user IDs
	var userIDs []int64
	for _, f := range friendsData {
		userIDs = append(userIDs, f.UserID)
	}

	// Get a cookie for the presence API (uses any saved cookie)
	accounts, _ := account_manager.LoadAccounts()
	var cookie string
	for _, acc := range accounts {
		if cookieData, err := cookie_manager.GetCookieForAccount(acc.ID); err == nil {
			cookie = cookieData.Value
			break
		}
	}

	// Fetch presence
	presences, err := roblox_api.GetUserPresence(userIDs, cookie)
	if err != nil {
		logger.LogError("Failed to get friend presence: %v", err)
		return
	}

	// Update cache
	friendsStatusLock.Lock()
	for _, p := range presences {
		friendsStatusCache[p.UserID] = p
	}
	friendsStatusLock.Unlock()

	// Refresh UI
	if friendsListWidget != nil {
		friendsListWidget.Refresh()
	}

	logger.LogDebug("Refreshed %d friend statuses", len(presences))
}

func showAddFriendDialog(window fyne.Window, countLabel *widget.Label) {
	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("Enter username or user ID...")

	form := dialog.NewForm("Add Friend", "Add", "Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Username or ID", usernameEntry),
		},
		func(ok bool) {
			if !ok {
				return
			}

			input := strings.TrimSpace(usernameEntry.Text)
			if input == "" {
				return
			}

			// Try to parse as user ID first
			var userInfo *roblox_api.UserInfo
			var err error

			if userID, parseErr := strconv.ParseInt(input, 10, 64); parseErr == nil {
				// It's a numeric ID
				userInfo, err = roblox_api.LookupUserByID(userID)
			} else {
				// It's a username
				userInfo, err = roblox_api.LookupUserByUsername(input)
			}

			if err != nil {
				dialog.ShowError(fmt.Errorf("Failed to find user: %v", err), window)
				return
			}

			// Confirm adding the friend
			confirmMsg := fmt.Sprintf("Add %s (@%s) to your friends list?",
				userInfo.DisplayName, userInfo.Username)

			dialog.ShowConfirm("Confirm Add Friend", confirmMsg, func(ok bool) {
				if ok {
					if err := friends_manager.AddFriend(userInfo.UserID, userInfo.Username, userInfo.DisplayName); err != nil {
						if strings.Contains(err.Error(), "already exists") {
							dialog.ShowInformation("Already Added",
								fmt.Sprintf("%s is already in your friends list.", userInfo.Username),
								window)
						} else {
							dialog.ShowError(err, window)
						}
					} else {
						refreshFriendsList(countLabel)
						dialog.ShowInformation("Friend Added",
							fmt.Sprintf("Added %s to your friends list!", userInfo.Username),
							window)
					}
				}
			}, window)
		}, window)

	form.Resize(fyne.NewSize(350, 150))
	form.Show()
}

func showJoinFriendDialog(window fyne.Window, friend friends_manager.Friend, presence roblox_api.UserPresence) {
	if presence.PlaceID == 0 {
		dialog.ShowInformation("Cannot Join",
			fmt.Sprintf("%s is not in a joinable game.", friend.Username),
			window)
		return
	}

	// Show account selection for joining via browser
	accounts, err := account_manager.LoadAccounts()
	if err != nil || len(accounts) == 0 {
		// Launch without account selection
		launchJoinFriendViaBrowser(presence.PlaceID, friend.UserID, window)
		dialog.ShowInformation("Join Friend",
			fmt.Sprintf("Opening game page to join %s!\n\nClick Play on the game page.", friend.Username),
			window)
		return
	}

	// Create account selection with cookie status
	var options []string
	for _, acc := range accounts {
		displayText := acc.Username
		if acc.Label != "" {
			displayText = fmt.Sprintf("%s (%s)", acc.Label, acc.Username)
		}
		options = append(options, displayText)
	}

	selectWidget := widget.NewSelect(options, nil)
	if len(options) > 0 {
		selectWidget.SetSelected(options[0])
	}

	infoLabel := widget.NewLabel(fmt.Sprintf("Join %s in:\n%s\n\nOpens game page in browser - click Play to join.", friend.Username, presence.LastLocation))
	infoLabel.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(
		infoLabel,
		widget.NewSeparator(),
		widget.NewLabel("Account to use:"),
		selectWidget,
	)

	dialog.ShowCustomConfirm("Join Friend", "Open Game Page", "Cancel", content,
		func(ok bool) {
			if !ok {
				return
			}

			selectedIndex := 0
			for i, opt := range options {
				if opt == selectWidget.Selected {
					selectedIndex = i
					break
				}
			}

			account := accounts[selectedIndex]

			// Check browser account mismatch
			browserUsername, _ := cookie_manager.GetCurrentBrowserCookieUsername()

			if browserUsername != "" && !strings.EqualFold(browserUsername, account.Username) {
				dialog.ShowConfirm("Account Mismatch",
					fmt.Sprintf("Browser logged in as: %s\nYou selected: %s\n\nClear browser session to log in as %s?",
						browserUsername, account.Username, account.Username),
					func(clearSession bool) {
						if clearSession {
							// Save current browser cookie before clearing
							saveBrowserCookieBeforeClear()
							cookie_manager.ClearVivaldiRobloxCookies()
							logger.LogInfo("Cleared Vivaldi cookies for friend join")
						}
						launchJoinFriendViaBrowser(presence.PlaceID, friend.UserID, window)
						if clearSession {
							dialog.ShowInformation("Join Friend",
								fmt.Sprintf("Browser cleared! Log in as %s, then click Play.", account.Username),
								window)
						}
					}, window)
				return
			}

			launchJoinFriendViaBrowser(presence.PlaceID, friend.UserID, window)
			if browserUsername == "" {
				dialog.ShowInformation("Join Friend",
					fmt.Sprintf("Opening game page! Log in as %s, then click Play.", account.Username),
					window)
			}
		}, window)
}

func launchJoinFriendViaBrowser(placeID, userID int64, window fyne.Window) {
	// Open the game page directly - this will show the Play button and auto-detect friend
	// Using the experiences URL which has better join functionality
	gameURL := fmt.Sprintf("https://www.roblox.com/games/%d", placeID)
	logger.LogInfo("Opening game page in browser: %s (friend userID: %d)", gameURL, userID)
	exec.Command("open", gameURL).Start()
}

// saveBrowserCookieBeforeClear saves the current browser cookie to the matching account before clearing
func saveBrowserCookieBeforeClear() {
	accounts, err := account_manager.LoadAccounts()
	if err != nil {
		logger.LogError("Failed to load accounts for cookie save: %v", err)
		return
	}

	// Convert to the format expected by SaveCurrentBrowserCookieToAccount
	var accountList []struct{ ID, Username string }
	for _, acc := range accounts {
		accountList = append(accountList, struct{ ID, Username string }{acc.ID, acc.Username})
	}

	if err := cookie_manager.SaveCurrentBrowserCookieToAccount(accountList); err != nil {
		logger.LogError("Failed to save browser cookie before clear: %v", err)
	}
}

func launchJoinFriend(placeID, followUserID int64, authTicket string, window fyne.Window) {
	logger.LogInfo("Launching to join friend (placeID: %d, followUserID: %d)", placeID, followUserID)

	launchTime := fmt.Sprintf("%d", time.Now().UnixNano()/int64(time.Millisecond))
	browserTrackerId := fmt.Sprintf("%d", time.Now().UnixNano()%1000000000)

	var protocolString string
	if authTicket != "" {
		// With auth ticket - use RequestGame with the place ID (more reliable than RequestFollowUser)
		// The friend join happens automatically when joining the same server
		protocolString = fmt.Sprintf("roblox-player:1+launchmode:play+gameinfo:%s+launchtime:%s+placelauncherurl:https://assetgame.roblox.com/game/PlaceLauncher.ashx?request=RequestGame&browserTrackerId=%s&placeId=%d&isPlayTogetherGame=false+browsertrackerid:%s+robloxLocale:en_us+gameLocale:en_us+channel:",
			authTicket, launchTime, browserTrackerId, placeID, browserTrackerId)
	} else {
		// Without auth ticket - use roblox:// protocol
		protocolString = fmt.Sprintf("roblox://placeId=%d", placeID)
	}

	// Check if Roblox is running for multi-instance
	robloxApp := "/Applications/Roblox.app/Contents/MacOS/RobloxPlayer"

	if authTicket != "" {
		cmd := exec.Command(robloxApp, "-protocolString", protocolString)
		if err := cmd.Start(); err != nil {
			logger.LogError("Failed to launch join friend: %v", err)
			dialog.ShowError(fmt.Errorf("Failed to join friend: %v", err), window)
			return
		}
		logger.LogInfo("Launched join friend, PID: %d", cmd.Process.Pid)
	} else {
		cmd := exec.Command("open", protocolString)
		if err := cmd.Run(); err != nil {
			logger.LogError("Failed to launch join friend: %v", err)
			dialog.ShowError(fmt.Errorf("Failed to join friend: %v", err), window)
			return
		}
	}

	dialog.ShowInformation("Joining Friend",
		fmt.Sprintf("Launching Roblox to join your friend!"),
		window)
}

func createAboutTab(window fyne.Window) fyne.CanvasObject {
	title := widget.NewLabel("Multi Roblox Manager")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	version := widget.NewLabel("Version 3.1.0")
	version.Alignment = fyne.TextAlignCenter

	description := widget.NewLabel(`Run multiple Roblox accounts simultaneously on macOS!

Features:
• 🎮 Multi-Instance: Run multiple Roblox windows at once
• 🍪 Cookie Auth: Instant account switching via Vivaldi cookies
• ✅ Cookie Validation: See which accounts are ready to launch
• 📋 Presets: Quick-launch saved games with specific accounts
• 🔐 Keychain Security: Cookies stored securely in macOS Keychain
• 📊 Resource Monitor: Track CPU/memory per instance

How it works:
1. Add accounts in Accounts tab
2. Log into each account in Vivaldi browser
3. Click "Capture" to save the session cookie
4. Use Presets or New Instance to launch with any account!`)
	description.Wrapping = fyne.TextWrapWord

	discordButton := widget.NewButtonWithIcon("Discord Server", resourceDiscordPng, func() {
		discord_redirect.RedirectToServer(discord_link_parser.DiscordLink())
	})

	viewLogButton := widget.NewButton("View Debug Log", func() {
		logPath := logger.GetLogPath()
		logger.LogInfo("User requested to view log file")

		// Create options dialog
		content := widget.NewLabel(fmt.Sprintf("Log file:\n%s", logPath))
		content.Wrapping = fyne.TextWrapWord

		openInConsoleBtn := widget.NewButton("Open in Console.app", func() {
			exec.Command("open", "-a", "Console", logPath).Start()
		})

		openInFinderBtn := widget.NewButton("Reveal in Finder", func() {
			exec.Command("open", "-R", logPath).Start()
		})

		tailLogBtn := widget.NewButton("View Last 50 Lines", func() {
			output, err := exec.Command("tail", "-50", logPath).Output()
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			// Show log content in a scrollable dialog
			logContent := widget.NewLabel(string(output))
			logContent.Wrapping = fyne.TextWrapWord
			scroll := container.NewVScroll(logContent)
			scroll.SetMinSize(fyne.NewSize(450, 400))
			dialog.ShowCustom("Debug Log (Last 50 Lines)", "Close", scroll, window)
		})

		dialogContent := container.NewVBox(
			content,
			widget.NewSeparator(),
			openInConsoleBtn,
			openInFinderBtn,
			tailLogBtn,
		)

		dialog.ShowCustom("Debug Log Options", "Close", dialogContent, window)
	})

	return container.NewVBox(
		widget.NewSeparator(),
		title,
		version,
		widget.NewSeparator(),
		description,
		widget.NewSeparator(),
		discordButton,
		viewLogButton,
	)
}

func createAccountsTab(window fyne.Window) fyne.CanvasObject {
	// Load accounts
	accounts, _ := account_manager.LoadAccounts()
	var accountList *widget.List

	// Detect current logged-in account from Vivaldi cookie
	currentAccountLabel := widget.NewLabel("🔍 Detecting current account...")
	currentAccountLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Async detect current account from browser cookie
	go func() {
		browserUsername, err := cookie_manager.GetCurrentBrowserCookieUsername()
		if err == nil && browserUsername != "" {
			currentAccountLabel.SetText(fmt.Sprintf("🌐 Browser Session: %s", browserUsername))
		} else {
			// Fallback to session detection
			sessionUsername, _ := roblox_session.GetCurrentUsername()
			if sessionUsername != "" {
				currentAccountLabel.SetText(fmt.Sprintf("🎮 Active Session: %s", sessionUsername))
			} else {
				currentAccountLabel.SetText("⚠️ No active session detected")
			}
		}
	}()

	// Cookie status label showing valid cookies
	cookieStatusLabel := widget.NewLabel("🍪 Cookie-based account switching available")
	cookieStatusLabel.TextStyle = fyne.TextStyle{Italic: true}

	// Count valid cookies
	go func() {
		validCount := 0
		for _, acc := range accounts {
			if cookie_manager.HasSavedCookie(acc.ID) {
				// Verify cookie is still valid
				cookie, err := cookie_manager.GetCookieForAccount(acc.ID)
				if err == nil {
					if _, verifyErr := cookie_manager.VerifyCookieUsername(cookie.Value); verifyErr == nil {
						validCount++
					}
				}
			}
		}
		if validCount > 0 {
			cookieStatusLabel.SetText(fmt.Sprintf("🍪 %d account(s) ready for instant switching", validCount))
		} else {
			cookieStatusLabel.SetText("⚠️ No valid cookies - capture accounts to enable switching")
		}
	}()

	// Cache for cookie validation results (to avoid repeated API calls)
	cookieStatusCache := make(map[string]cookie_manager.CookieValidationResult)
	var cacheMutex sync.Mutex

	// Background validation of all cookies
	go func() {
		for _, acc := range accounts {
			result := cookie_manager.ValidateCookieForAccount(acc.ID)
			cacheMutex.Lock()
			cookieStatusCache[acc.ID] = result
			cacheMutex.Unlock()
		}
		// Refresh list after validation completes
		accountList.Refresh()
	}()

	// Account list with status indicators
	accountList = widget.NewList(
		func() int { return len(accounts) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Account Name Here"),
				widget.NewLabel("⚪"), // Cookie status indicator
				widget.NewButton("Capture", nil),
				widget.NewButton("Edit", nil),
				widget.NewButton("Delete", nil),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(accounts) {
				return
			}

			account := accounts[id]
			box := obj.(*fyne.Container)
			label := box.Objects[0].(*widget.Label)
			statusLabel := box.Objects[1].(*widget.Label)
			captureBtn := box.Objects[2].(*widget.Button)
			editBtn := box.Objects[3].(*widget.Button)
			deleteBtn := box.Objects[4].(*widget.Button)

			displayText := account.Username
			if account.Label != "" {
				displayText = fmt.Sprintf("%s (%s)", account.Label, account.Username)
			}
			label.SetText(displayText)

			// Check cached validation result
			cacheMutex.Lock()
			result, hasResult := cookieStatusCache[account.ID]
			cacheMutex.Unlock()

			if hasResult {
				switch result.Status {
				case cookie_manager.CookieStatusValid:
					if result.ExpiresWarning {
						// Cookie expiring within 7 days
						statusLabel.SetText(fmt.Sprintf("⚠️ %dd", result.DaysUntilExpiry))
						captureBtn.SetText("🔄 Refresh")
					} else {
						statusLabel.SetText("✅")
						captureBtn.SetText("Recapture")
					}
				case cookie_manager.CookieStatusExpired:
					statusLabel.SetText("❌")
					captureBtn.SetText("⚠️ Recapture")
				case cookie_manager.CookieStatusNone:
					statusLabel.SetText("⚪")
					captureBtn.SetText("Capture")
				default:
					statusLabel.SetText("⚠️")
					captureBtn.SetText("Capture")
				}
			} else {
				// Still loading
				if cookie_manager.HasSavedCookie(account.ID) {
					statusLabel.SetText("🔄")
					captureBtn.SetText("Recapture")
				} else {
					statusLabel.SetText("⚪")
					captureBtn.SetText("Capture")
				}
			}

			// Capture cookie button
			captureBtn.OnTapped = func() {
				showCaptureDialog(window, account, func() {
					// Re-validate after capture
					go func() {
						newResult := cookie_manager.ValidateCookieForAccount(account.ID)
						cacheMutex.Lock()
						cookieStatusCache[account.ID] = newResult
						cacheMutex.Unlock()
						accountList.Refresh()
					}()
				})
			}

			editBtn.OnTapped = func() {
				showEditAccountDialog(window, account.ID, func() {
					accounts, _ = account_manager.LoadAccounts()
					accountList.Refresh()
				})
			}

			deleteBtn.OnTapped = func() {
				dialog.ShowConfirm("Delete Account",
					fmt.Sprintf("Delete account '%s'?\n\nPassword and saved cookie will be removed.", account.Username),
					func(yes bool) {
						if yes {
							account_manager.DeleteAccount(account.ID)
							cookie_manager.ClearSavedCookie(account.ID)
							cacheMutex.Lock()
							delete(cookieStatusCache, account.ID)
							cacheMutex.Unlock()
							accounts, _ = account_manager.LoadAccounts()
							accountList.Refresh()
						}
					}, window)
			}
		},
	)

	// Add account button
	addButton := widget.NewButton("Add Account", func() {
		usernameEntry := widget.NewEntry()
		usernameEntry.SetPlaceHolder("Roblox Username")

		passwordEntry := widget.NewPasswordEntry()
		passwordEntry.SetPlaceHolder("Roblox Password (optional for cookie method)")

		labelEntry := widget.NewEntry()
		labelEntry.SetPlaceHolder("Label (e.g., Main Account, Alt 1)")

		formItems := []*widget.FormItem{
			widget.NewFormItem("Username", usernameEntry),
			widget.NewFormItem("Password", passwordEntry),
			widget.NewFormItem("Label", labelEntry),
		}

		dialog.ShowForm("Add Account", "Add", "Cancel", formItems, func(ok bool) {
			if ok && usernameEntry.Text != "" {
				// Password is now optional - cookie method doesn't need it
				password := passwordEntry.Text
				if password == "" {
					password = "cookie_auth" // Placeholder
				}
				if err := account_manager.AddAccount(usernameEntry.Text, password, labelEntry.Text); err != nil {
					dialog.ShowError(fmt.Errorf("Failed to add account: %w", err), window)
				} else {
					accounts, _ = account_manager.LoadAccounts()
					accountList.Refresh()
				}
			}
		}, window)
	})

	infoLabel := widget.NewLabel("🍪 Cookie Method: Log into Roblox in Vivaldi, then click 'Capture' to save the session.\n\n✅ = Valid cookie (ready to switch)  ❌ = Expired (recapture needed)  ⚪ = No cookie")
	infoLabel.Wrapping = fyne.TextWrapWord

	// Layout
	return container.NewBorder(
		container.NewVBox(
			currentAccountLabel,
			cookieStatusLabel,
			widget.NewSeparator(),
		),
		container.NewVBox(
			widget.NewSeparator(),
			infoLabel,
			widget.NewSeparator(),
			addButton,
		),
		nil,
		nil,
		accountList,
	)
}

// showCaptureDialog shows a dialog to capture the current Roblox cookie for an account
func showCaptureDialog(window fyne.Window, account account_manager.Account, refreshCallback func()) {
	displayName := account.Username
	if account.Label != "" {
		displayName = fmt.Sprintf("%s (%s)", account.Label, account.Username)
	}

	instructionsText := fmt.Sprintf(`🍪 Capture Cookie for: %s

Steps:
1. Make sure Vivaldi is open
2. Log into Roblox as %s
3. Click 'Capture Now' below

The cookie will be saved securely in your Keychain.
You can then switch to this account instantly when launching games!`, displayName, account.Username)

	instructionsLabel := widget.NewLabel(instructionsText)
	instructionsLabel.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(instructionsLabel)

	customDialog := dialog.NewCustomWithoutButtons("Capture Cookie", content, window)

	// Helper function to save cookie and show notification (defined before use)
	var saveCookieAndNotify func(cookie *cookie_manager.RobloxCookie, acc account_manager.Account, dispName string, dlg dialog.Dialog, win fyne.Window, callback func())
	saveCookieAndNotify = func(cookie *cookie_manager.RobloxCookie, acc account_manager.Account, dispName string, dlg dialog.Dialog, win fyne.Window, callback func()) {
		cookie.AccountID = acc.ID
		if err := cookie_manager.SaveCookieForAccount(acc.ID, cookie); err != nil {
			logger.LogError("Failed to save cookie: %v", err)
			dialog.ShowError(fmt.Errorf("Failed to save cookie: %v", err), win)
			return
		}

		logger.LogInfo("Successfully captured cookie for account: %s", acc.Username)
		dialog.ShowInformation("Success",
			fmt.Sprintf("✅ Cookie captured for %s!\n\nYou can now instantly switch to this account when launching games.", dispName),
			win)

		dlg.Hide()
		callback()
	}

	captureBtn := widget.NewButton("Capture Now", func() {
		logger.LogInfo("Attempting to capture cookie for account: %s", account.Username)

		// Read current cookie from Vivaldi
		cookie, err := cookie_manager.GetCurrentRobloxCookie()
		if err != nil {
			logger.LogError("Failed to capture cookie: %v", err)
			dialog.ShowError(fmt.Errorf("Failed to capture cookie:\n%v\n\nMake sure you're logged into Roblox in Vivaldi.", err), window)
			return
		}

		// Verify the cookie belongs to the expected account
		verifiedUsername, verifyErr := cookie_manager.VerifyCookieUsername(cookie.Value)
		if verifyErr != nil {
			logger.LogError("Failed to verify cookie: %v", verifyErr)
			dialog.ShowError(fmt.Errorf("Failed to verify cookie:\n%v", verifyErr), window)
			return
		}

		logger.LogInfo("Cookie verified - belongs to: %s", verifiedUsername)

		// Check if username matches (case-insensitive)
		if !strings.EqualFold(verifiedUsername, account.Username) {
			dialog.ShowConfirm("Username Mismatch",
				fmt.Sprintf("⚠️ Cookie belongs to: %s\nExpected: %s\n\nSave anyway?", verifiedUsername, account.Username),
				func(saveAnyway bool) {
					if saveAnyway {
						saveCookieAndNotify(cookie, account, displayName, customDialog, window, refreshCallback)
					}
				}, window)
			return
		}

		saveCookieAndNotify(cookie, account, displayName, customDialog, window, refreshCallback)
	})

	cancelBtn := widget.NewButton("Cancel", func() {
		customDialog.Hide()
	})

	openVivaldiBtn := widget.NewButton("Open Roblox Login", func() {
		exec.Command("open", "-a", "Vivaldi", "https://www.roblox.com/login").Start()
	})

	buttonBox := container.NewHBox(
		cancelBtn,
		openVivaldiBtn,
		captureBtn,
	)

	fullContent := container.NewVBox(
		content,
		widget.NewSeparator(),
		buttonBox,
	)

	customDialog = dialog.NewCustomWithoutButtons("Capture Cookie", fullContent, window)
	customDialog.Show()
}

// showLabelDialog shows a dialog to label an instance
func showLabelDialog(window fyne.Window, pid int, refreshCallback func()) {
	labelEntry := widget.NewEntry()
	labelEntry.SetPlaceHolder("Enter label (e.g., Main Account, Alt 1)")

	// Get current label if exists
	if existingLabel, found := label_manager.GetLabel(pid); found {
		labelEntry.SetText(existingLabel.Label)
	}

	// Color selection
	colorSelect := widget.NewSelect([]string{
		"Red", "Cyan", "Blue", "Orange", "Mint", "Yellow", "Purple", "Light Blue", "None",
	}, nil)
	colorSelect.SetSelected("Red")

	formItems := []*widget.FormItem{
		widget.NewFormItem("Label", labelEntry),
		widget.NewFormItem("Color", colorSelect),
	}

	dialog.ShowForm("Label Instance", "Save", "Cancel", formItems, func(ok bool) {
		if ok {
			colorValue := ""
			colors := label_manager.DefaultColors()
			colorMap := map[string]string{
				"Red":        colors[0],
				"Cyan":       colors[1],
				"Blue":       colors[2],
				"Orange":     colors[3],
				"Mint":       colors[4],
				"Yellow":     colors[5],
				"Purple":     colors[6],
				"Light Blue": colors[7],
				"None":       "",
			}

			if selected := colorSelect.Selected; selected != "" {
				colorValue = colorMap[selected]
			}

			if labelEntry.Text != "" {
				label_manager.SetLabel(pid, labelEntry.Text, colorValue)
			} else if labelEntry.Text == "" && colorValue == "" {
				label_manager.DeleteLabel(pid)
			}

			refreshCallback()
		}
	}, window)
}

// parseHexColor converts a hex color string to color.Color
func parseHexColor(s string) (color.Color, error) {
	c := color.NRGBA{R: 0, G: 0, B: 0, A: 255}

	if len(s) != 7 || s[0] != '#' {
		return c, fmt.Errorf("invalid color format")
	}

	r, err := strconv.ParseUint(s[1:3], 16, 8)
	if err != nil {
		return c, err
	}
	g, err := strconv.ParseUint(s[3:5], 16, 8)
	if err != nil {
		return c, err
	}
	b, err := strconv.ParseUint(s[5:7], 16, 8)
	if err != nil {
		return c, err
	}

	c.R = uint8(r)
	c.G = uint8(g)
	c.B = uint8(b)

	return c, nil
}

// showEditAccountDialog shows a dialog to edit account label
func showEditAccountDialog(window fyne.Window, accountID string, refreshCallback func()) {
	account, err := account_manager.GetAccount(accountID)
	if err != nil {
		dialog.ShowError(err, window)
		return
	}

	labelEntry := widget.NewEntry()
	labelEntry.SetText(account.Label)
	labelEntry.SetPlaceHolder("Label (e.g., Main Account, Alt 1)")

	formItems := []*widget.FormItem{
		widget.NewFormItem("Username", widget.NewLabel(account.Username)),
		widget.NewFormItem("Label", labelEntry),
	}

	dialog.ShowForm("Edit Account", "Save", "Cancel", formItems, func(ok bool) {
		if ok {
			account_manager.UpdateAccountLabel(accountID, labelEntry.Text)
			refreshCallback()
		}
	}, window)
}

// showAccountSelectionDialog shows account selection when launching new instance
func showAccountSelectionDialog(window fyne.Window, launchCallback func()) {
	logger.LogInfo("showAccountSelectionDialog called - New Instance launch")

	accounts, err := account_manager.LoadAccounts()
	if err != nil || len(accounts) == 0 {
		logger.LogInfo("No accounts configured, launching without account selection")
		roblox_login.LaunchWithoutAccount()
		launchCallback()
		return
	}

	logger.LogDebug("Found %d accounts for selection", len(accounts))

	// Create account selection options with cookie status
	var options []string
	options = append(options, "Launch without account (opens Roblox home)")
	for _, acc := range accounts {
		displayText := acc.Username
		if acc.Label != "" {
			displayText = fmt.Sprintf("%s (%s)", acc.Label, acc.Username)
		}
		// Add status indicator
		result := cookie_manager.ValidateCookieForAccount(acc.ID)
		switch result.Status {
		case cookie_manager.CookieStatusValid:
			displayText = "✅ " + displayText
		case cookie_manager.CookieStatusExpired:
			displayText = "❌ " + displayText + " (expired)"
		case cookie_manager.CookieStatusNone:
			displayText = "⚪ " + displayText + " (no cookie)"
		}
		options = append(options, displayText)
	}

	selectWidget := widget.NewSelect(options, nil)
	selectWidget.SetSelected(options[0])

	infoLabel := widget.NewLabel("Select an account to launch a new Roblox instance.\n\n✅ = Cookie ready (instant auth)\n❌ = Cookie expired (go to Accounts tab to recapture)\n⚪ = No cookie saved")
	infoLabel.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(
		infoLabel,
		widget.NewSeparator(),
		widget.NewLabel("Account:"),
		selectWidget,
	)

	customDialog := dialog.NewCustomWithoutButtons("Launch New Instance", content, window)

	// Launch button
	launchBtn := widget.NewButton("🚀 Launch Instance", func() {
		selectedIndex := -1
		for i, opt := range options {
			if opt == selectWidget.Selected {
				selectedIndex = i
				break
			}
		}

		logger.LogDebug("Selected index: %d, option: %s", selectedIndex, selectWidget.Selected)

		if selectedIndex == 0 {
			logger.LogInfo("Launching new instance without account")
			roblox_login.LaunchWithoutAccount()
			customDialog.Hide()
			launchCallback()
		} else if selectedIndex > 0 {
			account := accounts[selectedIndex-1]
			logger.LogInfo("Launching new instance with account: %s (ID: %s)", account.Username, account.ID)

			// Check cookie status
			result := cookie_manager.ValidateCookieForAccount(account.ID)
			if result.Status == cookie_manager.CookieStatusValid {
				// Get the cookie for this account
				cookie, err := cookie_manager.GetCookieForAccount(account.ID)
				if err != nil {
					dialog.ShowError(fmt.Errorf("Failed to get cookie: %v", err), window)
					return
				}

				// Write cookie to Roblox's storage before launching
				if err := cookie_manager.SetRobloxAppCookie(cookie.Value); err != nil {
					logger.LogError("Failed to set Roblox app cookie: %v", err)
					// Continue anyway - might still work
				}

				// Launch Roblox home with multi-instance support
				pid, err := preset_manager.LaunchRobloxHomeWithAccount(cookie.Value)
				if err != nil {
					dialog.ShowError(fmt.Errorf("Failed to launch: %v", err), window)
					return
				}

				// Track which account this instance belongs to
				if pid > 0 {
					instance_account_tracker.TrackInstance(pid, account.ID)
					logger.LogInfo("Tracked instance PID %d with account %s", pid, account.Username)
				}

				customDialog.Hide()
				launchCallback()

				dialog.ShowInformation("Instance Launched",
					fmt.Sprintf("Launched Roblox home as %s!\n\nThe app will open to the home screen.", account.Username),
					window)
			} else if result.Status == cookie_manager.CookieStatusExpired {
				dialog.ShowError(fmt.Errorf("Cookie for %s has expired!\n\nGo to Accounts tab and click 'Recapture' to refresh it.", account.Username), window)
			} else {
				dialog.ShowError(fmt.Errorf("No cookie for %s!\n\nGo to Accounts tab, log into Roblox as this user in Vivaldi, then click 'Capture'.", account.Username), window)
			}
		}
	})

	cancelBtn := widget.NewButton("Cancel", func() {
		logger.LogInfo("User cancelled instance launch")
		customDialog.Hide()
	})

	buttonBox := container.NewHBox(
		cancelBtn,
		launchBtn,
	)

	fullContent := container.NewVBox(
		content,
		widget.NewSeparator(),
		buttonBox,
	)

	customDialog = dialog.NewCustomWithoutButtons("Launch New Instance", fullContent, window)
	customDialog.Show()
}

// showPresetSettingsDialog shows settings dialog for a preset
func showPresetSettingsDialog(window fyne.Window, preset preset_manager.Preset, presetIndex int, refreshCallback func()) {
	logger.LogInfo("Opening settings for preset: %s", preset.Name)

	// Private server link entry
	privateServerEntry := widget.NewEntry()
	privateServerEntry.SetPlaceHolder("Paste private server link here...")
	if preset.PrivateServerLinkCode != "" {
		privateServerEntry.SetText("https://www.roblox.com/games/" + fmt.Sprintf("%d", preset.PlaceID) + "?privateServerLinkCode=" + preset.PrivateServerLinkCode)
	}

	privateServerInfo := widget.NewLabel("Paste a full private server link to enable direct joining to your private server.")
	privateServerInfo.Wrapping = fyne.TextWrapWord

	currentStatus := widget.NewLabel("")
	if preset.PrivateServerLinkCode != "" {
		currentStatus.SetText("🔒 Private server link: " + preset.PrivateServerLinkCode[:min(20, len(preset.PrivateServerLinkCode))] + "...")
	} else {
		currentStatus.SetText("No private server configured")
	}

	content := container.NewVBox(
		widget.NewLabel("Private Server"),
		widget.NewSeparator(),
		privateServerInfo,
		privateServerEntry,
		currentStatus,
		widget.NewSeparator(),
	)

	dialog.ShowCustomConfirm("Preset Settings: "+preset.Name, "Save", "Cancel", content,
		func(ok bool) {
			if !ok {
				return
			}

			// Extract link code from the entered URL
			linkCode := preset_manager.ExtractPrivateServerLinkCode(privateServerEntry.Text)

			// If nothing entered but there was a code, they might want to clear it
			if privateServerEntry.Text == "" {
				linkCode = ""
			}

			if err := preset_manager.UpdatePresetPrivateServer(presetIndex, linkCode); err != nil {
				dialog.ShowError(err, window)
			} else {
				if linkCode != "" {
					dialog.ShowInformation("Saved",
						"Private server link saved! When you launch this preset with an account, it will join the private server.",
						window)
				} else {
					dialog.ShowInformation("Cleared",
						"Private server link cleared. Launches will go to the public server.",
						window)
				}
				refreshCallback()
			}
		}, window)
}

// showAccountSelectionForPreset shows account selection for preset launch with cookie switching
func showAccountSelectionForPreset(window fyne.Window, preset preset_manager.Preset, presetIndex int, launchCallback func()) {
	logger.LogInfo("showAccountSelectionForPreset called for preset: %s (index: %d)", preset.Name, presetIndex)

	accounts, err := account_manager.LoadAccounts()
	if err != nil || len(accounts) == 0 {
		logger.LogInfo("No accounts found, launching without account selection")
		preset_manager.LaunchPreset(preset)
		launchCallback()
		return
	}

	logger.LogDebug("Found %d accounts", len(accounts))

	// Detect current logged-in account from browser cookie (more reliable)
	var currentUsername string
	browserUsername, err := cookie_manager.GetCurrentBrowserCookieUsername()
	if err == nil && browserUsername != "" {
		currentUsername = browserUsername
	} else {
		// Fallback to session detection
		currentUsername, _ = roblox_session.GetCurrentUsername()
	}
	logger.LogDebug("Current detected username: %s", currentUsername)

	// Create account selection options with cookie status indicators
	var options []string
	options = append(options, "Launch without account")
	defaultSelection := 0

	for i, acc := range accounts {
		displayText := acc.Username
		if acc.Label != "" {
			displayText = fmt.Sprintf("%s (%s)", acc.Label, acc.Username)
		}
		// Add cookie status indicator with expiry warning
		result := cookie_manager.ValidateCookieForAccount(acc.ID)
		switch result.Status {
		case cookie_manager.CookieStatusValid:
			if result.ExpiresWarning {
				displayText = fmt.Sprintf("⚠️ %s (%dd left)", displayText, result.DaysUntilExpiry)
			} else {
				displayText = "✅ " + displayText
			}
		case cookie_manager.CookieStatusExpired:
			displayText = "❌ " + displayText + " (expired)"
		case cookie_manager.CookieStatusNone:
			displayText = "⚪ " + displayText
		}

		// Pre-select last used account for this preset
		if preset.LastAccountUsed == acc.ID {
			defaultSelection = i + 1
		}

		options = append(options, displayText)
	}

	selectWidget := widget.NewSelect(options, nil)
	selectWidget.SetSelected(options[defaultSelection])

	// Build info text
	infoText := fmt.Sprintf("Select account to launch '%s'", preset.Name)
	if currentUsername != "" && currentUsername != "[Browser Session Active]" && currentUsername != "[Active Session]" {
		infoText += fmt.Sprintf("\n\n🌐 Browser logged in as: %s", currentUsername)
	}
	infoText += "\n\n✅ = Ready (instant switch)  ❌ = Expired (recapture needed)  ⚪ = No cookie"

	infoLabel := widget.NewLabel(infoText)
	infoLabel.Wrapping = fyne.TextWrapWord

	// Server type selection (only show if preset has private server configured)
	var serverTypeSelect *widget.Select
	var serverTypeContainer fyne.CanvasObject

	if preset.PrivateServerLinkCode != "" {
		serverTypeSelect = widget.NewSelect([]string{"Private Server", "Public Server"}, nil)
		serverTypeSelect.SetSelected("Private Server")
		serverTypeContainer = container.NewVBox(
			widget.NewLabel("Server Type:"),
			serverTypeSelect,
			widget.NewSeparator(),
		)
	} else {
		serverTypeContainer = container.NewVBox() // Empty container
	}

	content := container.NewVBox(
		infoLabel,
		widget.NewSeparator(),
		serverTypeContainer,
		widget.NewLabel("Account:"),
		selectWidget,
	)

	customDialog := dialog.NewCustomWithoutButtons("Launch Preset", content, window)

	// Switch Account button - uses cookie switching
	switchAccountBtn := widget.NewButton("⚡ Switch & Launch", func() {
		selectedIndex := -1
		for i, opt := range options {
			if opt == selectWidget.Selected {
				selectedIndex = i
				break
			}
		}

		if selectedIndex > 0 {
			account := accounts[selectedIndex-1]
			logger.LogInfo("Switching to account: %s using cookie", account.Username)

			// Pre-launch cookie check with auto-refresh
			cookieValue, err := cookie_manager.PreLaunchCookieCheck(account.ID, account.Username)
			if err != nil {
				logger.LogError("Pre-launch cookie check failed: %v", err)
				dialog.ShowError(fmt.Errorf("Cookie issue for %s:\n%v\n\nGo to Accounts tab to recapture.", account.Username, err), window)
				return
			}

			// Create a temporary cookie struct for compatibility
			cookie := &cookie_manager.RobloxCookie{Value: cookieValue}

			// Check if Vivaldi is running
			if isVivaldiRunning() {
				dialog.ShowConfirm("Close Vivaldi",
					"Vivaldi must be closed to switch accounts.\n\nClose Vivaldi and switch to "+account.Username+"?",
					func(yes bool) {
						if yes {
							// Close Vivaldi
							exec.Command("pkill", "-x", "Vivaldi").Run()
							time.Sleep(500 * time.Millisecond)

							// Get auth ticket for the selected account
							// Note: We don't clear Roblox cache anymore - auth ticket is independent
							// This allows multi-instance to work properly without closing other windows
							logger.LogInfo("Getting auth ticket for: %s", account.Username)

							// Get auth ticket directly from cookie (no cache clearing needed)
							authTicket, err := cookie_manager.GetAuthTicket(cookie.Value)
							if err != nil {
								logger.LogError("Failed to get auth ticket: %v", err)
								dialog.ShowError(fmt.Errorf("Failed to get auth ticket: %v\n\nThe cookie may have expired. Try recapturing it.", err), window)
								return
							}

							logger.LogInfo("Got auth ticket, launching as: %s", account.Username)

							// Step 4: Check if private server was selected
							usePrivateServer := serverTypeSelect != nil && serverTypeSelect.Selected == "Private Server"

							if usePrivateServer && preset.PrivateServerLinkCode != "" {
								// Check if browser is logged in as the correct account
								browserUsername, _ := cookie_manager.GetCurrentBrowserCookieUsername()

								if browserUsername != "" && !strings.EqualFold(browserUsername, account.Username) {
									// Browser is logged in as different account - offer to clear
									dialog.ShowConfirm("Account Mismatch",
										fmt.Sprintf("Browser is logged in as: %s\nYou selected: %s\n\nClear browser session to log in as %s?",
											browserUsername, account.Username, account.Username),
										func(clearSession bool) {
											if clearSession {
												// Save current browser cookie before clearing
												saveBrowserCookieBeforeClear()
												// Clear Vivaldi cookies for Roblox
												cookie_manager.ClearVivaldiRobloxCookies()
												logger.LogInfo("Cleared Vivaldi Roblox cookies for account switch")
											}

											// Open private server via browser
											shareURL := fmt.Sprintf("https://www.roblox.com/share?code=%s&type=Server", preset.PrivateServerLinkCode)
											logger.LogInfo("Opening private server via browser: %s", shareURL)
											exec.Command("open", shareURL).Start()

											preset_manager.UpdatePresetLastAccount(presetIndex, account.ID)
											customDialog.Hide()
											launchCallback()

											if clearSession {
												dialog.ShowInformation("Private Server",
													fmt.Sprintf("Browser session cleared!\n\nPlease log in as %s when the page loads.", account.Username),
													window)
											}
										}, window)
									return
								}

								// Browser matches or no session - just open
								shareURL := fmt.Sprintf("https://www.roblox.com/share?code=%s&type=Server", preset.PrivateServerLinkCode)
								logger.LogInfo("Opening private server via browser: %s", shareURL)
								exec.Command("open", shareURL).Start()

								preset_manager.UpdatePresetLastAccount(presetIndex, account.ID)
								customDialog.Hide()
								launchCallback()

								if browserUsername == "" {
									dialog.ShowInformation("Private Server",
										fmt.Sprintf("Opening private server!\n\nPlease log in as %s when the page loads.", account.Username),
										window)
								} else {
									dialog.ShowInformation("Private Server",
										fmt.Sprintf("Opening private server as %s!", account.Username),
										window)
								}
							} else {
								// Launch public server directly with auth ticket
								// Clear private server link temporarily for public launch
								tempPreset := preset
								tempPreset.PrivateServerLinkCode = "" // Force public server

								pid, err := preset_manager.LaunchPresetWithTicket(tempPreset, authTicket)
								if err != nil {
									logger.LogError("Failed to launch preset: %v", err)
									dialog.ShowError(fmt.Errorf("Failed to launch: %v", err), window)
									return
								}

								// Track which account this instance belongs to
								if pid > 0 {
									instance_account_tracker.TrackInstance(pid, account.ID)
									logger.LogInfo("Tracked instance PID %d with account %s", pid, account.Username)
								}

								preset_manager.UpdatePresetLastAccount(presetIndex, account.ID)

								customDialog.Hide()
								launchCallback()

								dialog.ShowInformation("Account Switched",
									fmt.Sprintf("Switched to %s and launching game!\n\nRoblox will open with this account.", account.Username),
									window)
							}
						}
					}, window)
			} else {
				// Vivaldi not running - get auth ticket directly
				// Note: Auth ticket is independent, no cache clearing needed
				// This allows multi-instance to work properly
				logger.LogInfo("Getting auth ticket for: %s", account.Username)

				// Get auth ticket directly from cookie
				authTicket, err := cookie_manager.GetAuthTicket(cookie.Value)
				if err != nil {
					logger.LogError("Failed to get auth ticket: %v", err)
					dialog.ShowError(fmt.Errorf("Failed to get auth ticket: %v\n\nThe cookie may have expired. Try recapturing it.", err), window)
					return
				}

				logger.LogInfo("Got auth ticket, launching as: %s", account.Username)

				// Step 4: Check if private server was selected
				usePrivateServer := serverTypeSelect != nil && serverTypeSelect.Selected == "Private Server"

				if usePrivateServer && preset.PrivateServerLinkCode != "" {
					// Check if browser is logged in as the correct account
					browserUsername, _ := cookie_manager.GetCurrentBrowserCookieUsername()

					if browserUsername != "" && !strings.EqualFold(browserUsername, account.Username) {
						// Browser is logged in as different account - offer to clear
						dialog.ShowConfirm("Account Mismatch",
							fmt.Sprintf("Browser is logged in as: %s\nYou selected: %s\n\nClear browser session to log in as %s?",
								browserUsername, account.Username, account.Username),
							func(clearSession bool) {
								if clearSession {
									// Save current browser cookie before clearing
									saveBrowserCookieBeforeClear()
									cookie_manager.ClearVivaldiRobloxCookies()
									logger.LogInfo("Cleared Vivaldi Roblox cookies for account switch")
								}

								shareURL := fmt.Sprintf("https://www.roblox.com/share?code=%s&type=Server", preset.PrivateServerLinkCode)
								exec.Command("open", shareURL).Start()

								preset_manager.UpdatePresetLastAccount(presetIndex, account.ID)
								customDialog.Hide()
								launchCallback()

								if clearSession {
									dialog.ShowInformation("Private Server",
										fmt.Sprintf("Browser session cleared!\n\nPlease log in as %s.", account.Username),
										window)
								}
							}, window)
						return
					}

					// Browser matches or no session
					shareURL := fmt.Sprintf("https://www.roblox.com/share?code=%s&type=Server", preset.PrivateServerLinkCode)
					exec.Command("open", shareURL).Start()

					preset_manager.UpdatePresetLastAccount(presetIndex, account.ID)
					customDialog.Hide()
					launchCallback()

					if browserUsername == "" {
						dialog.ShowInformation("Private Server",
							fmt.Sprintf("Opening private server!\n\nPlease log in as %s.", account.Username),
							window)
					} else {
						dialog.ShowInformation("Private Server",
							fmt.Sprintf("Opening private server as %s!", account.Username),
							window)
					}
				} else {
					// Launch public server directly with auth ticket
					tempPreset := preset
					tempPreset.PrivateServerLinkCode = "" // Force public server

					pid, err := preset_manager.LaunchPresetWithTicket(tempPreset, authTicket)
					if err != nil {
						logger.LogError("Failed to launch preset: %v", err)
						dialog.ShowError(fmt.Errorf("Failed to launch: %v", err), window)
						return
					}

					// Track which account this instance belongs to
					if pid > 0 {
						instance_account_tracker.TrackInstance(pid, account.ID)
						logger.LogInfo("Tracked instance PID %d with account %s", pid, account.Username)
					}

					preset_manager.UpdatePresetLastAccount(presetIndex, account.ID)

					customDialog.Hide()
					launchCallback()

					dialog.ShowInformation("Account Switched",
						fmt.Sprintf("Switched to %s and launching game!", account.Username),
						window)
				}
			}
		} else {
			dialog.ShowInformation("Select Account",
				"Please select an account from the dropdown first.",
				window)
		}
	})

	// Launch button - just launches with current session
	launchBtn := widget.NewButton("Launch Now", func() {
		logger.LogInfo("User confirmed launch")

		selectedIndex := -1
		for i, opt := range options {
			if opt == selectWidget.Selected {
				selectedIndex = i
				break
			}
		}

		logger.LogDebug("Selected index: %d, option: %s", selectedIndex, selectWidget.Selected)

		// Check if private server was selected
		usePrivateServer := serverTypeSelect != nil && serverTypeSelect.Selected == "Private Server"

		if usePrivateServer && preset.PrivateServerLinkCode != "" {
			// Open private server via browser
			shareURL := fmt.Sprintf("https://www.roblox.com/share?code=%s&type=Server", preset.PrivateServerLinkCode)
			logger.LogInfo("Opening private server via browser: %s", shareURL)
			exec.Command("open", shareURL).Start()

			customDialog.Hide()
			launchCallback()

			dialog.ShowInformation("Private Server",
				"Opening private server in browser!\n\nMake sure you're logged in to the correct account.",
				window)
			return
		}

		var selectedAccountID string
		if selectedIndex == 0 {
			logger.LogInfo("Launching without account (public server)")
			// Force public server by clearing private server code
			tempPreset := preset
			tempPreset.PrivateServerLinkCode = ""
			if err := preset_manager.LaunchPreset(tempPreset); err != nil {
				logger.LogError("Failed to launch preset: %v", err)
			}
		} else if selectedIndex > 0 {
			account := accounts[selectedIndex-1]
			selectedAccountID = account.ID

			logger.LogInfo("Launching with account: %s (ID: %s) - public server", account.Username, account.ID)

			// Force public server by clearing private server code
			tempPreset := preset
			tempPreset.PrivateServerLinkCode = ""
			if err := preset_manager.LaunchPreset(tempPreset); err != nil {
				logger.LogError("Failed to launch preset with account: %v", err)
			}

			if err := preset_manager.UpdatePresetLastAccount(presetIndex, selectedAccountID); err != nil {
				logger.LogError("Failed to update last used account: %v", err)
			} else {
				logger.LogDebug("Saved last used account for preset")
			}
		}

		customDialog.Hide()
		launchCallback()
	})

	cancelBtn := widget.NewButton("Cancel", func() {
		logger.LogInfo("User cancelled launch")
		customDialog.Hide()
	})

	// Add buttons to dialog
	buttonBox := container.NewHBox(
		cancelBtn,
		switchAccountBtn,
		launchBtn,
	)

	fullContent := container.NewVBox(
		content,
		widget.NewSeparator(),
		buttonBox,
	)

	customDialog = dialog.NewCustomWithoutButtons("Launch Preset", fullContent, window)
	customDialog.Show()
}

// isVivaldiRunning checks if Vivaldi browser is running
func isVivaldiRunning() bool {
	cmd := exec.Command("pgrep", "-x", "Vivaldi")
	err := cmd.Run()
	return err == nil
}

// showAccountSwitchWorkflow guides user through switching Roblox account
func showAccountSwitchWorkflow(window fyne.Window, account account_manager.Account, preset preset_manager.Preset, presetIndex int, launchCallback func()) {
	logger.LogInfo("Starting account switch workflow for: %s", account.Username)

	displayName := account.Username
	if account.Label != "" {
		displayName = fmt.Sprintf("%s (%s)", account.Label, account.Username)
	}

	// Step 1: Show instructions
	instructionsText := fmt.Sprintf(`🔄 Account Switch Required

To play as: %s

Steps:
1. Click 'Open Roblox Login' below
2. Log out of current account (if needed)
3. Log in as: %s
4. Come back here and click 'Launch Game'

⚠️ All Roblox instances share the same login.
Make sure you're logged into the correct account before launching.`, displayName, account.Username)

	instructionsLabel := widget.NewLabel(instructionsText)
	instructionsLabel.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(
		instructionsLabel,
	)

	customDialog := dialog.NewCustomWithoutButtons("Switch Account", content, window)

	// Open Login button
	openLoginBtn := widget.NewButton("Open Roblox Login", func() {
		logger.LogInfo("Opening Roblox login page for account switch")
		roblox_login.OpenRobloxLoginPage()
	})

	// Launch Game button
	launchGameBtn := widget.NewButton("Launch Game", func() {
		logger.LogInfo("Launching preset after account switch: %s with account %s", preset.Name, account.Username)

		if err := preset_manager.LaunchPreset(preset); err != nil {
			logger.LogError("Failed to launch preset: %v", err)
			dialog.ShowError(err, window)
		}

		// Save last used account
		if err := preset_manager.UpdatePresetLastAccount(presetIndex, account.ID); err != nil {
			logger.LogError("Failed to update last used account: %v", err)
		}

		customDialog.Hide()
		launchCallback()
	})

	cancelBtn := widget.NewButton("Cancel", func() {
		logger.LogInfo("User cancelled account switch")
		customDialog.Hide()
	})

	buttonBox := container.NewHBox(
		cancelBtn,
		openLoginBtn,
		launchGameBtn,
	)

	fullContent := container.NewVBox(
		content,
		widget.NewSeparator(),
		buttonBox,
	)

	customDialog = dialog.NewCustomWithoutButtons("Switch Account", fullContent, window)
	customDialog.Show()
}
