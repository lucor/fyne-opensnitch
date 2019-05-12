// This file defines the Fyne application
package main

import (
	"fmt"
	"image/color"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/widget"

	"github.com/evilsocket/opensnitch/daemon/log"
	"github.com/evilsocket/opensnitch/daemon/rule"
	"github.com/evilsocket/opensnitch/daemon/ui/protocol"
)

// osApp represents the Fyne OpenSnitch application
type osApp struct {
	fyneApp fyne.App
	mainWin fyne.Window
	chClose chan os.Signal
}

// ShowAndRun show and run the application
func (a *osApp) ShowAndRun() {
	log.Info("starting Fyne OpenSnitch application")
	a.mainWin.ShowAndRun()
}

func (a *osApp) RefreshStats(st *protocol.Statistics) {
	tabContainer := a.mainWin.Content().(*widget.TabContainer)
	selectedTab := tabContainer.CurrentTab()
	table := selectedTab.Content.(*table)
	table.SetContent(generalStatsData(st))
	a.mainWin.SetFixedSize(false)
	a.mainWin.Resize(tabContainer.MinSize())
	a.mainWin.SetFixedSize(true)
}

// Ask asks client a rule for the con connection
func (a *osApp) AskRule(con *protocol.Connection) (*protocol.Rule, bool) {

	var wg sync.WaitGroup

	win := a.fyneApp.NewWindow("OpenSnitch")
	win.SetFixedSize(true)
	win.CenterOnScreen()

	processPath := con.GetProcessPath()
	appInfo, ok := desktopApps[processPath]
	if !ok {
		appInfo = desktopApp{
			Name: processPath,
		}
	}

	processName := appInfo.Name
	proto := con.GetProtocol()
	destPort := con.GetDstPort()
	srcIP := con.GetSrcIp()
	destIP := con.GetDstIp()
	destHost := con.GetDstHost()
	uid := con.GetUserId()
	pid := con.GetProcessId()
	processArgs := con.GetProcessArgs()

	var icon *canvas.Image
	if appInfo.Icon == "" {
		resource := fyne.NewStaticResource("default_icon", defaultIcon)
		icon = canvas.NewImageFromResource(resource)
	} else {
		icon = canvas.NewImageFromFile(appInfo.Icon)
	}
	icon.FillMode = canvas.ImageFillOriginal
	icon.SetMinSize(fyne.NewSize(48, 48))

	// Create the rule
	// Default to accept connection once for the current process if not action is specified
	// Note: rule is applied also on window close
	t := time.Now()
	ruleName := fmt.Sprintf("test-%d%d%d", t.Hour(), t.Minute(), t.Second())
	r := &protocol.Rule{
		Name:     ruleName,
		Action:   string(rule.Allow),
		Duration: string(rule.Once),
		Operator: &protocol.Operator{
			Type:    string(rule.Simple),
			Operand: string(rule.OpProcessPath),
			Data:    con.GetProcessPath(),
		},
	}

	// Select widget for the action
	action := widget.NewSelect([]string{"Allow Connections", "Block connections"}, func(s string) {
		r.Action = string(rule.Allow)
		if s == "Block connections" {
			r.Action = string(rule.Deny)
		}
	})
	action.SetSelected("Allow Connections")

	// Map select with label with key
	operators := map[string]string{
		"from this process":                 "process",
		fmt.Sprintf("from user %d", uid):    "user",
		fmt.Sprintf("to port %d", destPort): "port",
		fmt.Sprintf("to %s", destIP):        "ip",
		fmt.Sprintf("to %s", destHost):      "host",
	}

	// Extract keys to pass to widget
	operatorOpts := []string{}
	for k := range operators {
		operatorOpts = append(operatorOpts, k)
	}

	// Select widget for the operator
	operator := widget.NewSelect(operatorOpts, func(s string) {
		simple := string(rule.Simple)
		switch operators[s] {
		case "process":
			r.Operator = &protocol.Operator{Type: simple, Operand: string(rule.OpProcessPath), Data: processPath}
		case "user":
			r.Operator = &protocol.Operator{Type: simple, Operand: string(rule.OpUserId), Data: strconv.Itoa(int(uid))}
		case "port":
			r.Operator = &protocol.Operator{Type: simple, Operand: string(rule.OpDstPort), Data: strconv.Itoa(int(destPort))}
		case "ip":
			r.Operator = &protocol.Operator{Type: simple, Operand: string(rule.OpDstIP), Data: destIP}
		case "host":
			r.Operator = &protocol.Operator{Type: simple, Operand: string(rule.OpDstHost), Data: destHost}
		}
	})
	operator.SetSelected("from this process")

	// Select widget for the duration
	duration := widget.NewSelect([]string{"once", "for this session", "forever"}, func(s string) {
		switch s {
		case "once":
			r.Duration = string(rule.Once)
		case "for this session":
			r.Duration = string(rule.Restart)
		case "forever":
			r.Duration = string(rule.Always)
		}
	})
	duration.SetSelected("once")

	// Apply button
	applyBtn := widget.NewButton("Apply", func() {
		win.Close()
	})
	applyBtn.Style = widget.PrimaryButton

	win.SetContent(
		widget.NewVBox(
			widget.NewHBox(
				icon,
				widget.NewVBox(
					layout.NewSpacer(),
					widget.NewLabelWithStyle(processName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
					layout.NewSpacer(),
				),
				layout.NewSpacer(),
			),

			makeLine(),

			widget.NewLabel(fmt.Sprintf("%s is connecting to %s on %s port %d", processPath, destHost, proto, destPort)),

			fyne.NewContainerWithLayout(layout.NewGridLayout(4),
				action,
				operator,
				duration,
				widget.NewVBox(applyBtn),
			),

			fyne.NewContainerWithLayout(layout.NewGridLayout(3),
				widget.NewLabelWithStyle("Source IP", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel(srcIP),
				layout.NewSpacer(),

				widget.NewLabelWithStyle("Destination IP", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel(destIP),
				layout.NewSpacer(),

				widget.NewLabelWithStyle("Destination Port", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel(fmt.Sprintf("%d", destPort)),
				layout.NewSpacer(),

				widget.NewLabelWithStyle("Destination Host", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel(destHost),
				layout.NewSpacer(),

				widget.NewLabelWithStyle("User ID", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel(fmt.Sprintf("%d", uid)),
				layout.NewSpacer(),

				widget.NewLabelWithStyle("Process ID", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel(fmt.Sprintf("%d", pid)),
				layout.NewSpacer(),

				widget.NewLabelWithStyle("Process arguments", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel(fmt.Sprintf("%v", processArgs)),
				layout.NewSpacer(),
			),
		),
	)
	win.SetOnClosed(func() {
		wg.Done()
	})

	wg.Add(1)
	win.Show()
	wg.Wait()

	return r, true
}

// newApp returns a new fyne opensnitch application
func newApp(sigChan chan os.Signal) *osApp {

	fyneApp := app.New()

	//mainWin := statsWindow(fyneApp, stats)
	mainWin := fyneApp.NewWindow("OpenSnitch Network Statistics")
	mainWin.SetFixedSize(true)

	content := widget.NewTabContainer(
		widget.NewTabItem("General", generalStats()),
	)

	mainWin.SetContent(content)

	mainWin.SetOnClosed(func() {
		log.Important("Received close on main app")
		sigChan <- syscall.SIGQUIT
		fyneApp.Quit()
	})

	return &osApp{
		fyneApp: fyneApp,
		mainWin: mainWin,
	}
}

func generalStatsData(stats *protocol.Statistics) [][]string {
	return [][]string{
		{
			stats.GetDaemonVersion(),
			fmt.Sprintf("%d", stats.GetUptime()),
			fmt.Sprintf("%d", stats.GetRules()),
			fmt.Sprintf("%d", stats.GetConnections()),
			fmt.Sprintf("%d", stats.GetDropped()),
		},
	}
}

func generalStats() *table {
	headers := []string{"Version", "Uptime", "Rules", "Connections", "Dropped"}
	data := [][]string{[]string{"-", "0", "0", "0", "0"}}
	return newTable(headers, data)
}

func makeLine() fyne.CanvasObject {
	rect := canvas.NewRectangle(&color.RGBA{128, 128, 128, 255})
	rect.SetMinSize(fyne.NewSize(30, 2))
	return rect
}
