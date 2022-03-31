package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func main() {
	const title string = " Minecraft Ansible Config Builder "
	app := tview.NewApplication()
	pages := tview.NewPages()
	modal := tview.NewModal()
	services := tview.NewList()
	hideModal := func() {
		pages.RemovePage("modal")
		app.SetFocus(services)
	}
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			hideModal()
			return nil
		}
		return event
	})
	modal.
		SetText("Do you want to quit the application?").
		AddButtons([]string{"Quit", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Quit" {
				app.Stop()
			} else {
				hideModal()
			}
		})
	services.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			pages.AddAndSwitchToPage("modal", modal, true)
			return nil
		}
		return event
	})
	form := tview.NewForm()
	textView := tview.NewTextView()
	form.
		AddInputField("Name", "silvia", 0, nil, nil).
		AddInputField("Server JAR", "paper-1.18.2-271.jar", 0, nil, nil).
		AddInputField("Server Port", "35565", 0, nil, nil)
	form.Box.SetBorder(true).SetTitle(" Settings ")
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			app.SetFocus(services)
			return nil
		}
		return event
	})
	textView.
		SetText(`Lorem ipsum dolor sit amet, consectetur adipiscing elit. Quisque metus mi, ultrices quis rhoncus non, tempor in metus. Suspendisse eget fermentum mauris, quis viverra ipsum. Mauris a felis malesuada, euismod lorem in, dictum mauris. Nunc ultricies ornare orci, faucibus aliquam enim sollicitudin et. Integer non sapien ut justo accumsan mattis in quis eros. Vivamus dapibus est eu nisl hendrerit posuere. Vestibulum quis velit vitae tellus porttitor ornare ac sit amet nibh. Vivamus blandit euismod blandit. Maecenas tincidunt non nunc vel pulvinar. Nam rutrum lacus et orci facilisis tincidunt. Quisque aliquam lectus iaculis purus placerat, eu porttitor leo eleifend. Quisque finibus mauris ac ex dapibus vehicula. Nunc sodales tortor vitae ante dignissim placerat. `)
	textView.Box.SetBorder(true).SetTitle(" Info ")

	for i, name := range []string{"Eka", "Toka"} {
		key := i
		services.AddItem(name, "", rune(key+'1'), func() {
			// log.Printf("You selected %d", key)
			app.SetFocus(form)
		})
	}
	services.Box.SetBorder(true).SetTitle(" Services ")

	flex := tview.NewFlex().
		AddItem(services, 0, 1, false).
		AddItem(form, 0, 2, false).
		AddItem(textView, 0, 3, false)
	flex.Box.SetBorder(true).SetTitle(title)

	pages.AddPage("background", flex, true, true)

	if err := app.SetRoot(pages, true).SetFocus(services).Run(); err != nil {
		panic(err)
	}
}
