package main

import (
	"fmt"
	"image/color"
	"insadem/multi_roblox_macos/internal/account_manager"
	"insadem/multi_roblox_macos/internal/close_all_app_instances"
	"insadem/multi_roblox_macos/internal/discord_link_parser"
	"insadem/multi_roblox_macos/internal/discord_redirect"
	"insadem/multi_roblox_macos/internal/instance_account_tracker"
	"insadem/multi_roblox_macos/internal/instance_manager"
	"insadem/multi_roblox_macos/internal/label_manager"
	"insadem/multi_roblox_macos/internal/preset_manager"
	"insadem/multi_roblox_macos/internal/resource_monitor"
	"insadem/multi_roblox_macos/internal/roblox_login"
	"insadem/multi_roblox_macos/internal/roblox_session"
	"insadem/multi_roblox_macos/internal/thumbnail_cache"
	"strconv"
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
	mainApp := app.New()
	window := mainApp.NewWindow("Multi Roblox Manager")
	window.Resize(fyne.NewSize(500, 600))

	// Create tabs
	tabs := container.NewAppTabs(
		container.NewTabItem("Instances", createInstancesTab(window)),
		container.NewTabItem("Presets", createPresetsTab(window)),
		container.NewTabItem("Accounts", createAccountsTab(window)),
		container.NewTabItem("About", createAboutTab(window)),
	)

	window.SetContent(tabs)
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
			labelBox := border.Objects[0].(*fyne.Container)        // Center
			colorIndicator := border.Objects[1].(*canvas.Rectangle) // Left
			buttonBox := border.Objects[2].(*fyne.Container)       // Right

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
					labelText += fmt.Sprintf(" - %s", accountLabel)
				}
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
			// Create card-style layout with thumbnail
			thumbnail := canvas.NewImageFromFile("")
			thumbnail.FillMode = canvas.ImageFillContain
			thumbnail.SetMinSize(fyne.NewSize(80, 80))

			nameLabel := widget.NewLabel("Game Name")
			nameLabel.TextStyle = fyne.TextStyle{Bold: true}

			urlLabel := widget.NewLabel("URL")
			urlLabel.TextStyle = fyne.TextStyle{Italic: true}

			buttonBox := container.NewHBox(
				widget.NewButton("Launch", nil),
				widget.NewButton("Delete", nil),
			)

			infoBox := container.NewVBox(
				nameLabel,
				urlLabel,
				buttonBox,
			)

			return container.NewBorder(
				nil, nil,
				thumbnail, // Left: thumbnail
				nil,
				infoBox, // Center: game info and buttons
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(presets) {
				return
			}

			preset := presets[id]
			border := obj.(*fyne.Container)

			thumbnail := border.Objects[2].(*canvas.Image)     // Left
			infoBox := border.Objects[4].(*fyne.Container)     // Center

			nameLabel := infoBox.Objects[0].(*widget.Label)
			urlLabel := infoBox.Objects[1].(*widget.Label)
			buttonBox := infoBox.Objects[2].(*fyne.Container)

			launchBtn := buttonBox.Objects[0].(*widget.Button)
			deleteBtn := buttonBox.Objects[1].(*widget.Button)

			// Set game name
			nameLabel.SetText(preset.Name)

			// Set URL (truncated)
			urlText := preset.URL
			if len(urlText) > 40 {
				urlText = urlText[:37] + "..."
			}
			urlLabel.SetText(urlText)

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

func createAboutTab(window fyne.Window) fyne.CanvasObject {
	title := widget.NewLabel("Multi Roblox Manager")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	version := widget.NewLabel("Version 2.2.0")
	version.Alignment = fyne.TextAlignCenter

	description := widget.NewLabel("Manage multiple Roblox instances with ease.\n\nFeatures:\n• Instance Counter & Manager with Account Tracking\n• Quick Launch Presets with Auto-fetch\n• Account Management with Keychain Security\n• Resource Monitor\n• Instance Labeling")
	description.Wrapping = fyne.TextWrapWord

	discordButton := widget.NewButtonWithIcon("Discord Server", resourceDiscordPng, func() {
		discord_redirect.RedirectToServer(discord_link_parser.DiscordLink())
	})

	return container.NewVBox(
		widget.NewSeparator(),
		title,
		version,
		widget.NewSeparator(),
		description,
		widget.NewSeparator(),
		discordButton,
	)
}

func createAccountsTab(window fyne.Window) fyne.CanvasObject {
	// Load accounts
	accounts, _ := account_manager.LoadAccounts()
	var accountList *widget.List

	// Detect current logged-in account
	currentUsername, _ := roblox_session.GetCurrentUsername()
	currentAccountLabel := widget.NewLabel("Current Account: Not detected")
	if currentUsername != "" {
		currentAccountLabel.SetText(fmt.Sprintf("Current Account: %s", currentUsername))
	}
	currentAccountLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Account list
	accountList = widget.NewList(
		func() int { return len(accounts) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Account"),
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
			editBtn := box.Objects[1].(*widget.Button)
			deleteBtn := box.Objects[2].(*widget.Button)

			displayText := account.Username
			if account.Label != "" {
				displayText = fmt.Sprintf("%s (%s)", account.Label, account.Username)
			}
			label.SetText(displayText)

			editBtn.OnTapped = func() {
				showEditAccountDialog(window, account.ID, func() {
					accounts, _ = account_manager.LoadAccounts()
					accountList.Refresh()
				})
			}

			deleteBtn.OnTapped = func() {
				dialog.ShowConfirm("Delete Account",
					fmt.Sprintf("Delete account '%s'?\n\nPassword will be removed from Keychain.", account.Username),
					func(yes bool) {
						if yes {
							account_manager.DeleteAccount(account.ID)
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
		passwordEntry.SetPlaceHolder("Roblox Password")

		labelEntry := widget.NewEntry()
		labelEntry.SetPlaceHolder("Label (e.g., Main Account, Alt 1)")

		formItems := []*widget.FormItem{
			widget.NewFormItem("Username", usernameEntry),
			widget.NewFormItem("Password", passwordEntry),
			widget.NewFormItem("Label", labelEntry),
		}

		dialog.ShowForm("Add Account", "Add", "Cancel", formItems, func(ok bool) {
			if ok && usernameEntry.Text != "" && passwordEntry.Text != "" {
				if err := account_manager.AddAccount(usernameEntry.Text, passwordEntry.Text, labelEntry.Text); err != nil {
					dialog.ShowError(fmt.Errorf("Failed to add account: %w", err), window)
				} else {
					accounts, _ = account_manager.LoadAccounts()
					accountList.Refresh()
				}
			}
		}, window)
	})

	infoLabel := widget.NewLabel("Accounts are stored securely in macOS Keychain.\nPasswords are never saved to disk.")
	infoLabel.Wrapping = fyne.TextWrapWord

	// Layout
	return container.NewBorder(
		container.NewVBox(
			currentAccountLabel,
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
	accounts, err := account_manager.LoadAccounts()
	if err != nil || len(accounts) == 0 {
		// No accounts configured, just launch without account
		roblox_login.LaunchWithoutAccount()
		launchCallback()
		return
	}

	// Create account selection options
	var options []string
	options = append(options, "Launch without account")
	for _, acc := range accounts {
		displayText := acc.Username
		if acc.Label != "" {
			displayText = fmt.Sprintf("%s (%s)", acc.Label, acc.Username)
		}
		options = append(options, displayText)
	}

	selectWidget := widget.NewSelect(options, nil)
	selectWidget.SetSelected(options[0])

	infoLabel := widget.NewLabel("Select an account to launch with, or launch without logging in.\n\nNote: Manual login required - automatic login coming in future update.")
	infoLabel.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(
		infoLabel,
		widget.NewSeparator(),
		widget.NewLabel("Account:"),
		selectWidget,
	)

	dialog.ShowCustomConfirm("Launch Roblox Instance", "Launch", "Cancel", content, func(launch bool) {
		if launch {
			selectedIndex := -1
			for i, opt := range options {
				if opt == selectWidget.Selected {
					selectedIndex = i
					break
				}
			}

			var selectedAccountID string
			if selectedIndex == 0 {
				// Launch without account
				roblox_login.LaunchWithoutAccount()
			} else if selectedIndex > 0 {
				// Launch with selected account
				account := accounts[selectedIndex-1]
				selectedAccountID = account.ID
				password, _ := account_manager.GetPassword(account.ID)
				roblox_login.LaunchWithAccount(account.Username, password)
			}

			// Track which account was used (will be associated with PID after launch)
			// Note: We can't get PID immediately, tracking will be done via cleanup
			if selectedAccountID != "" {
				// Instance will be tracked when it appears in next refresh
			}

			launchCallback()
		}
	}, window)
}

// showAccountSelectionForPreset shows account selection for preset launch
func showAccountSelectionForPreset(window fyne.Window, preset preset_manager.Preset, presetIndex int, launchCallback func()) {
	accounts, err := account_manager.LoadAccounts()
	if err != nil || len(accounts) == 0 {
		// No accounts, just launch preset
		preset_manager.LaunchPreset(preset)
		launchCallback()
		return
	}

	// Create account selection options with last used account pre-selected
	var options []string
	options = append(options, "Launch without account")
	defaultSelection := 0

	for i, acc := range accounts {
		displayText := acc.Username
		if acc.Label != "" {
			displayText = fmt.Sprintf("%s (%s)", acc.Label, acc.Username)
		}

		// Pre-select last used account for this preset
		if preset.LastAccountUsed == acc.ID {
			defaultSelection = i + 1
		}

		options = append(options, displayText)
	}

	selectWidget := widget.NewSelect(options, nil)
	selectWidget.SetSelected(options[defaultSelection])

	infoLabel := widget.NewLabel(fmt.Sprintf("Select account to launch '%s'", preset.Name))
	infoLabel.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(
		infoLabel,
		widget.NewSeparator(),
		widget.NewLabel("Account:"),
		selectWidget,
	)

	dialog.ShowCustomConfirm("Launch Preset", "Launch", "Cancel", content, func(launch bool) {
		if launch {
			selectedIndex := -1
			for i, opt := range options {
				if opt == selectWidget.Selected {
					selectedIndex = i
					break
				}
			}

			var selectedAccountID string
			if selectedIndex == 0 {
				// Launch without account
				preset_manager.LaunchPreset(preset)
			} else if selectedIndex > 0 {
				// Launch with selected account
				account := accounts[selectedIndex-1]
				selectedAccountID = account.ID
				password, _ := account_manager.GetPassword(account.ID)

				// Launch preset with account
				roblox_login.LaunchWithAccount(account.Username, password)
				time.Sleep(500 * time.Millisecond)
				preset_manager.LaunchPreset(preset)

				// Save last used account for this preset
				preset_manager.UpdatePresetLastAccount(presetIndex, selectedAccountID)
			}

			launchCallback()
		}
	}, window)
}
