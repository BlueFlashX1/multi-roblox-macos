package main

import (
	"fmt"
	"image/color"
	"insadem/multi_roblox_macos/internal/close_all_app_instances"
	"insadem/multi_roblox_macos/internal/discord_link_parser"
	"insadem/multi_roblox_macos/internal/discord_redirect"
	"insadem/multi_roblox_macos/internal/instance_manager"
	"insadem/multi_roblox_macos/internal/label_manager"
	"insadem/multi_roblox_macos/internal/open_app"
	"insadem/multi_roblox_macos/internal/preset_manager"
	"insadem/multi_roblox_macos/internal/resource_monitor"
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

			// Set instance label text
			labelText := ""
			if instance.Label != "" {
				labelText = fmt.Sprintf("%s (PID: %d)", instance.Label, instance.PID)
			} else {
				labelText = fmt.Sprintf("Instance %d (PID: %d)", id+1, instance.PID)
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
		open_app.Open("/Applications/Roblox.app")
		time.Sleep(500 * time.Millisecond)
		updateInstances()
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

	// Preset list
	presetList = widget.NewList(
		func() int { return len(presets) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Preset"),
				widget.NewButton("Launch", nil),
				widget.NewButton("Delete", nil),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(presets) {
				return
			}

			preset := presets[id]
			box := obj.(*fyne.Container)
			label := box.Objects[0].(*widget.Label)
			launchBtn := box.Objects[1].(*widget.Button)
			deleteBtn := box.Objects[2].(*widget.Button)

			label.SetText(preset.Name)

			launchBtn.OnTapped = func() {
				preset_manager.LaunchPreset(preset)
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
		nameEntry.SetPlaceHolder("Preset Name (e.g., My Favorite Game)")

		urlEntry := widget.NewEntry()
		urlEntry.SetPlaceHolder("Roblox URL (e.g., roblox://placeId=123456)")

		formItems := []*widget.FormItem{
			widget.NewFormItem("Name", nameEntry),
			widget.NewFormItem("URL", urlEntry),
		}

		dialog.ShowForm("Add Preset", "Add", "Cancel", formItems, func(ok bool) {
			if ok && nameEntry.Text != "" && urlEntry.Text != "" {
				preset_manager.AddPreset(nameEntry.Text, urlEntry.Text)
				presets, _ = preset_manager.LoadPresets()
				presetList.Refresh()
			}
		}, window)
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

	version := widget.NewLabel("Version 2.0.0")
	version.Alignment = fyne.TextAlignCenter

	description := widget.NewLabel("Manage multiple Roblox instances with ease.\n\nFeatures:\n• Instance Counter & Manager\n• Quick Launch Presets\n• Auto-refresh instance list")
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
