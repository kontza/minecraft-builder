package builder_application

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
	ServerPort int    `yaml:"server_port"`
	WorldName  string `yaml:"world_name"`
}

type Settings struct {
	ServerUser      string           `yaml:"server_user"`
	ServerGroup     string           `yaml:"server_group"`
	ServerInstances []ServerInstance `yaml:"server_instances"`
}

type void struct{}

const (
	// Form items
	Name = iota
	WorldName
	ServerJar
	ServerPort
)

type ApplicationBuilder interface {
	RunApplication()
}

type BuilderApplication struct {
	*tview.Application
	mainPage          string
	modalPage         string
	appTitle          string
	servicesHelp      string
	settingsHelp      string
	servicesName      string
	settingsName      string
	infoName          string
	logName           string
	quitButton        string
	saveAndQuitButton string
	cancelButton      string
	selectedServer    int64
	settings          *Settings
	configFilePath    string
	jars              []string
	pages             *tview.Pages
	modal             *tview.Modal
	services          *tview.List
	form              *tview.Form
	textView          *tview.TextView
	flex              *tview.Flex
	topFlex           *tview.Flex
	log               *tview.TextView
}

func NewApplicationBuilder() ApplicationBuilder {
	ba := &BuilderApplication{
		Application:  tview.NewApplication(),
		mainPage:     "main",
		modalPage:    "modal",
		appTitle:     "Minecraft Ansible Config Builder",
		servicesHelp: "This list contains the detected Minecraft servers from the input YAML. You can quit this application by pressing ESC while this panel is active",
		settingsHelp: `[::b]Name       [::-] the name of the systemd service
[::b]World Name [::-] the name of the Minecraft world
[::b]Server JAR [::-] the JAR file to use for the service
[::b]Server Port[::-] the port to use for the service`,
		servicesName:      "services",
		settingsName:      "settings",
		infoName:          "info",
		logName:           "log",
		quitButton:        "Quit",
		saveAndQuitButton: "Save & Quit",
		cancelButton:      "Cancel",
		selectedServer:    0,
		configFilePath:    os.Args[1],
	}
	return ba
}

func (ba *BuilderApplication) makeTitleString(input string) string {
	return fmt.Sprintf(" %s ", strings.Title(input))
}

func (ba *BuilderApplication) loadSettings() *BuilderApplication {
	buf, err := ioutil.ReadFile(ba.configFilePath)
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to read config file due to %v", err))
	}
	ba.settings = &Settings{}
	err = yaml.Unmarshal(buf, ba.settings)
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to unmarshal config from file due to %v", err))
	}
	return ba
}

func (ba *BuilderApplication) saveSettings() {
	ba.backUpSettings()
	if data, err := yaml.Marshal(ba.settings); err != nil {
		log.Fatal(fmt.Sprintf("Failed to marshal config due to %v", err))
	} else {
		if err = ioutil.WriteFile(ba.configFilePath, data, 0); err != nil {
			log.Fatal(fmt.Sprintf("Failed to save config due to %v", err))
		}
	}
}

func (ba *BuilderApplication) backUpSettings() {
	srcDir, srcFilename := path.Split(ba.configFilePath)
	basename := path.Base(srcFilename)
	destName := fmt.Sprintf("%s.%s", strings.TrimSuffix(basename, filepath.Ext(basename)), "bak")
	destPath := path.Join(srcDir, destName)

	var fin *os.File
	var err error
	if fin, err = os.Open(ba.configFilePath); err != nil {
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

func (ba *BuilderApplication) loadJars() *BuilderApplication {
	if entries, err := ioutil.ReadDir("."); err != nil {
		log.Fatal(fmt.Sprintf("Failed to read the current dir due to %v", err))
	} else {
		ba.jars = []string{}
		for _, entry := range entries {
			if strings.ToLower(path.Ext(entry.Name())) == ".jar" {
				ba.jars = append(ba.jars, entry.Name())
			}
		}
	}
	return ba
}

func (ba *BuilderApplication) hideModal() {
	ba.pages.RemovePage(ba.modalPage)
	ba.SetFocus(ba.services)
}

func (ba *BuilderApplication) populateForm(server ServerInstance) {
	ba.form.GetFormItem(Name).(*tview.InputField).SetText(server.Name)
	ba.form.GetFormItem(WorldName).(*tview.InputField).SetText(server.WorldName)
	selection := -1
	for i, jar := range ba.jars {
		if jar == server.ServerJar {
			selection = i
		}
	}
	if selection < 0 {
		exist := false
		for i, jar := range ba.jars {
			if jar[1:] == server.ServerJar {
				exist = true
				selection = i
			}
		}
		if !exist {
			ba.jars = append(ba.jars, fmt.Sprintf("!%s", server.ServerJar))
			selection = len(ba.jars) - 1
		}
	}
	ba.form.GetFormItem(ServerJar).(*tview.DropDown).
		SetOptions(ba.jars, nil).
		SetCurrentOption(selection)
	ba.form.GetFormItem(ServerPort).(*tview.InputField).SetText(strconv.FormatInt(int64(server.ServerPort), 10))
}

func (ba *BuilderApplication) checkPorts(port int) {
	var member void
	ports := make(map[int]void)
	ports[port] = member
	count := 1
	currentServer := ba.settings.ServerInstances[ba.selectedServer]
	for i, server := range ba.settings.ServerInstances {
		if i == int(ba.selectedServer) {
			continue
		}
		if server.ServerPort == port {
			log.Printf("[red]'%s' port %d clashes with '%s'![-]", currentServer.Name, port, server.Name)
			count++
		}
	}
}

func (ba *BuilderApplication) saveForm() {
	server := ba.settings.ServerInstances[ba.selectedServer]
	server.Name = ba.form.GetFormItem(Name).(*tview.InputField).GetText()
	server.WorldName = ba.form.GetFormItem(WorldName).(*tview.InputField).GetText()
	_, jar := ba.form.GetFormItem(ServerJar).(*tview.DropDown).GetCurrentOption()
	if jar[0] == '!' {
		jar = jar[1:]
	}
	server.ServerJar = jar
	port, _ := strconv.Atoi(ba.form.GetFormItem(ServerPort).(*tview.InputField).GetText())
	ba.checkPorts(port)
	server.ServerPort = port
	ba.settings.ServerInstances[ba.selectedServer] = server
}

func (ba *BuilderApplication) initModal() *BuilderApplication {
	ba.modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			ba.hideModal()
			return nil
		}
		return event
	})
	ba.modal.
		SetText("Do you want to quit the application?").
		AddButtons([]string{ba.saveAndQuitButton, ba.quitButton, ba.cancelButton}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			switch buttonLabel {
			case ba.saveAndQuitButton:
				ba.saveSettings()
				fallthrough
			case ba.quitButton:
				ba.Stop()
			default:
				ba.hideModal()
			}
		})
	return ba
}

func (ba *BuilderApplication) initForm() *BuilderApplication {
	ba.form.
		AddInputField("Name", "", 0, nil, nil).
		AddInputField("World Name", "", 0, nil, nil).
		AddDropDown("Server JAR", ba.jars, 0, nil).
		AddInputField("Server Port", "", 0, nil, nil)
	ba.form.Box.SetBorder(true).SetTitle(ba.makeTitleString(ba.settingsName))
	ba.form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			ba.SetFocus(ba.services)
			ba.saveForm()
			return nil
		}
		return event
	})
	ba.populateForm(ba.settings.ServerInstances[ba.selectedServer])
	return ba
}

func (ba *BuilderApplication) initServices() *BuilderApplication {
	ba.services.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			ba.pages.AddAndSwitchToPage(ba.modalPage, ba.modal, true)
			return nil
		}
		return event
	})
	for i, instance := range ba.settings.ServerInstances {
		key := int64(i)
		ba.services.AddItem(instance.Name, "", rune(key+'1'), func() {
			_, jar := ba.form.GetFormItem(ServerJar).(*tview.DropDown).GetCurrentOption()
			if jar[0] == '!' {
				log.Printf("[red]'%s' not found in current directory![-]\n", jar[1:])
			}
			ba.selectedServer = key
			ba.populateForm(ba.settings.ServerInstances[ba.selectedServer])
			ba.SetFocus(ba.form)
			ba.textView.SetText(ba.settingsHelp)
		})
	}
	ba.services.Box.
		SetBorder(true).
		SetTitle(ba.makeTitleString(ba.servicesName)).
		SetFocusFunc(func() {
			ba.textView.SetText(ba.servicesHelp)
		})
	return ba
}

func (ba *BuilderApplication) initTextView() *BuilderApplication {
	ba.textView.
		SetWordWrap(true).
		SetDynamicColors(true)
	ba.textView.Box.
		SetBorder(true).
		SetTitle(ba.makeTitleString(ba.infoName))
	ba.log.
		SetWordWrap(true).
		SetDynamicColors(true)
	ba.log.Box.
		SetBorder(true).
		SetTitle(ba.makeTitleString(ba.logName))
	return ba
}

func (ba *BuilderApplication) initFlex() *BuilderApplication {
	ba.topFlex.
		SetDirection(tview.FlexRow).
		AddItem(ba.flex, 0, 4, false).
		AddItem(ba.log, 0, 1, false)
	ba.topFlex.Box.
		SetBorder(true).
		SetTitle(ba.appTitle)
	ba.flex.
		AddItem(ba.services, 0, 1, false).
		AddItem(ba.form, 0, 2, false).
		AddItem(ba.textView, 0, 3, false)
	return ba
}

func (ba *BuilderApplication) initPages() *BuilderApplication {
	ba.pages.AddPage(ba.mainPage, ba.topFlex, true, true)
	return ba
}

func (ba *BuilderApplication) initialize() *BuilderApplication {
	ba.textView = tview.NewTextView()
	ba.log = tview.NewTextView()
	log.SetOutput(ba.log)
	ba.pages = tview.NewPages()
	ba.modal = tview.NewModal()
	ba.services = tview.NewList()
	ba.form = tview.NewForm()
	ba.flex = tview.NewFlex()
	ba.topFlex = tview.NewFlex()
	return ba.loadSettings().
		loadJars().
		initModal().
		initServices().
		initForm().
		initTextView().
		initFlex().
		initPages()
}

func (ba *BuilderApplication) RunApplication() {
	ba.initialize()
	if err := ba.
		SetRoot(ba.pages, true).
		SetFocus(ba.services).
		Run(); err != nil {
		panic(err)
	}
}
