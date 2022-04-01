package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"gopkg.in/yaml.v3"
)

type ServerInstance struct {
	Name       string
	ServerJar  string `yaml:"server_jar"`
	ServerPort int64  `yaml:"server_port"`
	WorldName  string `yaml:"world_name"`
}

type Settings struct {
	ServerUser      string           `yaml:"server_user"`
	ServerGroup     string           `yaml:"server_group"`
	ServerInstances []ServerInstance `yaml:"server_instances"`
}

const (
	// Form items
	Name = iota
	WorldName
	ServerJar
	ServerPort
)

func usage() {
	println(`minecraft-builder 1.0.0

USAGE:
    minecraft-builder [path]

ARGS:
    <path>
        a path to Minecraft Ansible's 'group_vars/all'
`)
	os.Exit(0)
}

func init() {
	if len(os.Args) < 2 {
		usage()
	}
}

func loadSettings() *Settings {
	buf, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to read config file due to %v", err))
	}
	retVal := Settings{}
	err = yaml.Unmarshal(buf, &retVal)
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to unmarshal config from file due to %v", err))
	}
	return &retVal
}

func backUpSettings() {
	srcDir, srcFilename := path.Split(os.Args[1])
	basename := path.Base(srcFilename)
	destName := fmt.Sprintf("%s.%s", strings.TrimSuffix(basename, filepath.Ext(basename)), "bak")
	destPath := path.Join(srcDir, destName)

	var fin *os.File
	var err error
	if fin, err = os.Open(os.Args[1]); err != nil {
		log.Fatal(fmt.Sprintf("Failed to open '%s' due to %v", srcFilename, err))
	}
	defer fin.Close()

	var fout *os.File
	if fout, err = os.Create(destPath); err != nil {
		log.Fatal(fmt.Sprintf("Failed to create '%s' due to %v", destPath, err))
		log.Fatal(err)
	}
	defer fout.Close()

	if _, err = io.Copy(fout, fin); err != nil {
		log.Fatal(err)
	}
}

func saveSettings(settings *Settings) {
	backUpSettings()
	if data, err := yaml.Marshal(settings); err != nil {
		log.Fatal(fmt.Sprintf("Failed to marshal config due to %v", err))
	} else {
		if err = ioutil.WriteFile(os.Args[1], data, 0); err != nil {
			log.Fatal(fmt.Sprintf("Failed to save config due to %v", err))
		}
	}
}

func loadJars() []string {
	var jars []string
	if entries, err := ioutil.ReadDir("."); err != nil {
		log.Fatal(fmt.Sprintf("Failed to read the current dir due to %v", err))
	} else {
		for _, entry := range entries {
			if strings.ToLower(path.Ext(entry.Name())) == ".jar" {
				jars = append(jars, entry.Name())
			}
		}
	}
	return jars
}

func main() {
	const mainPage = "main"
	const modalPage = "modal"
	const title string = " Minecraft Ansible Config Builder "
	const servicesHelp = "This list contains the detected Minecraft servers from the input YAML. You can quit this application by pressing ESC while this panel is active"
	const settingsHelp = `Name: the name of the systemd service
World Name: the name of the Minecraft world
Server JAR: the JAR file to use for the service
Server Port: the port to use for the service`
	const servicesName = "services"
	const settingsName = "settings"
	const infoName = "info"
	const quitButton = "Quit"
	const saveAndQuitButton = "Save & Quit"
	const cancelButton = "Cancel"
	selectedServer := 0
	settings := loadSettings()
	helpTexts := make(map[string]string)
	helpTexts[servicesName] = servicesHelp
	helpTexts[settingsName] = settingsHelp
	jars := loadJars()

	app := tview.NewApplication()
	pages := tview.NewPages()
	modal := tview.NewModal()
	services := tview.NewList()
	form := tview.NewForm()
	textView := tview.NewTextView()

	populateForm := func(server ServerInstance) {
		form.GetFormItem(Name).(*tview.InputField).SetText(server.Name)
		form.GetFormItem(WorldName).(*tview.InputField).SetText(server.WorldName)
		selection := -1
		for i, jar := range jars {
			if jar == server.ServerJar {
				selection = i
			}
		}
		if selection < 0 {
			jars = append(loadJars(), fmt.Sprintf("!%s", server.ServerJar))
			selection = len(jars) - 1
		}
		form.GetFormItem(ServerJar).(*tview.DropDown).SetOptions(jars, nil)
		form.GetFormItem(ServerJar).(*tview.DropDown).SetCurrentOption(selection)
		form.GetFormItem(ServerPort).(*tview.InputField).SetText(strconv.FormatInt(server.ServerPort, 10))
	}

	saveForm := func() {
		server := settings.ServerInstances[selectedServer]
		server.Name = form.GetFormItem(Name).(*tview.InputField).GetText()
		server.WorldName = form.GetFormItem(WorldName).(*tview.InputField).GetText()
		_, jar := form.GetFormItem(ServerJar).(*tview.DropDown).GetCurrentOption()
		if jar[0] == '!' {
			jar = jar[1:]
		}
		server.ServerJar = jar
		port, _ := strconv.Atoi(form.GetFormItem(ServerPort).(*tview.InputField).GetText())
		server.ServerPort = int64(port)
		settings.ServerInstances[selectedServer] = server
	}

	hideModal := func() {
		pages.RemovePage(modalPage)
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
		AddButtons([]string{saveAndQuitButton, quitButton, cancelButton}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			switch buttonLabel {
			case saveAndQuitButton:
				saveSettings(settings)
				fallthrough
			case quitButton:
				app.Stop()
			default:
				hideModal()
			}
		})
	services.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			pages.AddAndSwitchToPage(modalPage, modal, true)
			return nil
		}
		return event
	})

	form.
		AddInputField("Name", "", 0, nil, nil).
		AddInputField("World Name", "", 0, nil, nil).
		AddDropDown("Server JAR", jars, 0, nil).
		AddInputField("Server Port", "", 0, nil, nil)
	form.Box.SetBorder(true).SetTitle(fmt.Sprintf(" %s ", strings.Title(settingsName)))
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			app.SetFocus(services)
			saveForm()
			return nil
		}
		return event
	})
	populateForm(settings.ServerInstances[selectedServer])

	textView.SetWrap(true).
		SetWordWrap(true).
		SetDynamicColors(true)
	textView.Box.SetBorder(true).SetTitle(fmt.Sprintf(" %s ", strings.Title(infoName)))

	for i, instance := range settings.ServerInstances {
		key := int(i)
		services.AddItem(instance.Name, "", rune(key+'1'), func() {
			helpText := helpTexts[settingsName]
			_, jar := form.GetFormItem(ServerJar).(*tview.DropDown).GetCurrentOption()
			if jar[0] == '!' {
				helpText = fmt.Sprintf("%s\n\n[red]'%s' not found in current directory!", helpText, jar[1:])
			}
			textView.SetText(helpText)
			selectedServer = key
			populateForm(settings.ServerInstances[selectedServer])
			app.SetFocus(form)
		})
	}
	services.Box.SetBorder(true).SetTitle(fmt.Sprintf(" %s ", strings.Title(servicesName))).
		SetFocusFunc(func() {
			textView.SetText(helpTexts[servicesName])
		})

	flex := tview.NewFlex().
		AddItem(services, 0, 1, false).
		AddItem(form, 0, 2, false).
		AddItem(textView, 0, 3, false)
	flex.Box.SetBorder(true).SetTitle(title)

	pages.AddPage(mainPage, flex, true, true)

	if err := app.SetRoot(pages, true).SetFocus(services).Run(); err != nil {
		panic(err)
	}
}
