package main

import (
	"fmt"
	"insadem/multi_roblox_macos/internal/close_all_app_instances"
	"insadem/multi_roblox_macos/internal/discord_link_parser"
	"insadem/multi_roblox_macos/internal/discord_redirect"
	"insadem/multi_roblox_macos/internal/instance_manager"
	"insadem/multi_roblox_macos/internal/open_app"
	"insadem/multi_roblox_macos/internal/preset_manager"
	"insadem/multi_roblox_macos/internal/resource_monitor"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
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

	// Instance list with resource stats
	instanceList := widget.NewList(
		func() int { return 0 },
		func() fyne.CanvasObject {
			return container.NewVBox(
				widget.NewLabel("Instance"),
				widget.NewLabel("Resources"),
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
			box := obj.(*fyne.Container)
			instanceLabel := box.Objects[0].(*widget.Label)
			resourceLabel := box.Objects[1].(*widget.Label)

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

			instanceLabel.SetText(fmt.Sprintf("Instance %d (PID: %d)", id+1, instance.PID))
			resourceLabel.SetText(resourceInfo)
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
