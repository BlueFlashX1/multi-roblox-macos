package main

import (
	"fmt"
	"insadem/multi_roblox_macos/internal/close_all_app_instances"
	"insadem/multi_roblox_macos/internal/discord_link_parser"
	"insadem/multi_roblox_macos/internal/discord_redirect"
	"insadem/multi_roblox_macos/internal/instance_manager"
	"insadem/multi_roblox_macos/internal/open_app"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

//go:generate fyne bundle -o bundled.go ./resources/discord.png
//go:generate fyne bundle -o bundled.go -append ./resources/more.png
//go:generate fyne bundle -o bundled.go -append ./resources/mop.png

func main() {
	mainApp := app.New()
	window := mainApp.NewWindow("Multi Roblox Manager")
	window.Resize(fyne.NewSize(400, 500))

	// Instance counter label
	counterLabel := widget.NewLabel("Running Instances: 0")
	counterLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Instance list
	instanceList := widget.NewList(
		func() int { return 0 },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Instance"),
				widget.NewButton("Close", nil),
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

		instanceList.Length = func() int {
			return len(currentInstances)
		}

		instanceList.UpdateItem = func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(currentInstances) {
				return
			}

			instance := currentInstances[id]
			box := obj.(*fyne.Container)
			label := box.Objects[0].(*widget.Label)
			button := box.Objects[1].(*widget.Button)

			label.SetText(fmt.Sprintf("Instance %d (PID: %d)", id+1, instance.PID))

			button.OnTapped = func() {
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
	discordButton := widget.NewButtonWithIcon("Discord Server", resourceDiscordPng, func() {
		discord_redirect.RedirectToServer(discord_link_parser.DiscordLink())
	})

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
	content := container.NewBorder(
		container.NewVBox(
			counterLabel,
			widget.NewSeparator(),
		),
		container.NewVBox(
			widget.NewSeparator(),
			newInstanceButton,
			closeAllButton,
			discordButton,
		),
		nil,
		nil,
		instanceList,
	)

	window.SetContent(content)
	window.ShowAndRun()
}
