// This file defines the Fyne application
package main

import (
	"fmt"
	"image/color"
	"os"
	"regexp"
	"strconv"
	"strings"
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

const osAppHealtCheck = 15 * time.Second

// osApp represents the Fyne OpenSnitch application
type osApp struct {
	fyneApp     fyne.App
	mainWin     fyne.Window
	chClose     chan os.Signal
	defaultRule *protocol.Rule
	askTimeout  time.Duration
	lastPing    uint64
}

// ShowAndRun show and run the application
func (a *osApp) ShowAndRun() {
	log.Info("starting Fyne OpenSnitch application")
	healtChecker := time.NewTicker(osAppHealtCheck)
	go func() {
		var last uint64
		var seen time.Time
		for range healtChecker.C {
			log.Info("old %d, new %d", last, a.lastPing)
			if a.lastPing > last {
				last = a.lastPing
				seen = time.Now()
				continue
			}

			lastSeenMsg := ""
			if last != 0 {
				lastSeenMsg = fmt.Sprintf(
					" Last ping received %s ago.",
					time.Since(seen).Truncate(time.Second),
				)
			}
			log.Error("Daemon not available.%s", lastSeenMsg)
			a.RefreshStats(&protocol.Statistics{})
		}
	}()
	a.mainWin.ShowAndRun()
}

func (a *osApp) RefreshStats(st *protocol.Statistics) {
	tabContainer := a.mainWin.Content().(*widget.TabContainer)
	selectedTab := tabContainer.CurrentTab()

	// Store last ping to report daemon availability
	a.lastPing = st.GetUptime()

	table := selectedTab.Content.(*table)
	table.SetContent(generalStatsData(st))
	a.mainWin.SetFixedSize(false)
	a.mainWin.Resize(tabContainer.MinSize())
	a.mainWin.SetFixedSize(true)
}

// Ask asks client a rule for the con connection
func (a *osApp) AskRule(con *protocol.Connection) (*protocol.Rule, bool) {
	win := a.fyneApp.NewWindow("OpenSnitch")
	win.SetFixedSize(true)
	win.CenterOnScreen()

	log.Info("con: %#v", con)
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

	// create a ruleCh where will wait an for user action or timeout
	ruleCh := make(chan *protocol.Rule, 1)

	// uiDefaultRule is the default rule to apply for the current process on
	// timeout or window close
	var uiDefaultRule *protocol.Rule
	var userRule *protocol.Rule

	type data struct {
		Action      string
		Operand     string
		OperandData string
		Duration    string
	}

	d := data{}
	// Select widget for the action
	action := widget.NewSelect([]string{"Allow Connections", "Block connections"}, func(s string) {
		d.Action = string(rule.Allow)
		if s == "Block connections" {
			d.Action = string(rule.Deny)
		}
	})
	if a.defaultRule.GetAction() == string(rule.Allow) {
		action.SetSelected("Allow Connections")
	} else {
		action.SetSelected("Block Connections")
	}

	// Map select with label with key
	operators := map[string]string{
		"from this process":                 string(rule.OpProcessPath),
		fmt.Sprintf("from user %d", uid):    string(rule.OpUserId),
		fmt.Sprintf("to port %d", destPort): string(rule.OpDstPort),
		fmt.Sprintf("to %s", destIP):        string(rule.OpDstIP),
		fmt.Sprintf("to %s", destHost):      string(rule.OpDstHost),
	}

	// Extract keys to pass to widget
	operatorOpts := []string{}
	for k := range operators {
		operatorOpts = append(operatorOpts, k)
	}

	// Select widget for the operator
	operator := widget.NewSelect(operatorOpts, func(s string) {
		op := operators[s]
		d.Operand = op
		switch op {
		case string(rule.OpProcessPath):
			d.OperandData = processPath
		case string(rule.OpUserId):
			d.OperandData = strconv.Itoa(int(uid))
		case string(rule.OpDstPort):
			d.OperandData = strconv.Itoa(int(destPort))
		case string(rule.OpDstIP):
			d.OperandData = destIP
		case string(rule.OpDstHost):
			d.OperandData = destHost
		}
	})

	log.Info("get operand %s", a.defaultRule.Operator.GetOperand())
	for k, v := range operators {
		if v == a.defaultRule.Operator.GetOperand() {
			operator.SetSelected(k)
			break
		}
	}

	// Select widget for the duration
	duration := widget.NewSelect([]string{"once", "for this session", "forever"}, func(s string) {
		switch s {
		case "once":
			d.Duration = string(rule.Once)
		case "for this session":
			d.Duration = string(rule.Restart)
		case "forever":
			d.Duration = string(rule.Always)
		}
	})
	switch a.defaultRule.GetDuration() {
	case string(rule.Once):
		duration.SetSelected("once")
	case string(rule.Restart):
		duration.SetSelected("for this session")
	case string(rule.Always):
		duration.SetSelected("forever")
	}

	// Apply button
	applyBtn := widget.NewButton("Apply", func() {
		userRule = &protocol.Rule{
			Action:   d.Action,
			Duration: d.Duration,
			Operator: &protocol.Operator{
				Type:    string(rule.Simple),
				Operand: d.Operand,
				Data:    d.OperandData,
			},
		}
		win.Close()
	})
	applyBtn.Style = widget.PrimaryButton

	win.SetContent(
		widget.NewVBox(
			widget.NewHBox(
				icon,
				widget.NewVBox(
					widget.NewLabelWithStyle(processName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
					widget.NewLabel(processPath),
					layout.NewSpacer(),
				),
				layout.NewSpacer(),
			),

			makeLine(),

			widget.NewLabel(fmt.Sprintf("%s is connecting to %s on %s port %d", processName, destHost, proto, destPort)),

			fyne.NewContainerWithLayout(layout.NewGridLayout(4),
				action,
				operator,
				duration,
				widget.NewVBox(applyBtn),
			),

			makeLine(),

			widget.NewHBox(
				widget.NewLabelWithStyle("Process", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel(strings.Join(processArgs, "\n")),
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

				// widget.NewLabelWithStyle("Process arguments", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				// widget.NewLabel(fmt.Sprintf("%v", processArgs)),
				// layout.NewSpacer(),
			),
		),
	)

	uiDefaultRule = &protocol.Rule{
		Action:   d.Action,
		Duration: d.Duration,
		Operator: &protocol.Operator{
			Type:    string(rule.Simple),
			Operand: d.Operand,
			Data:    d.OperandData,
		},
	}

	win.SetOnClosed(func() {
		if userRule != nil {
			ruleCh <- userRule
			return
		}
		ruleCh <- uiDefaultRule
	})

	win.Show()

	var pr *protocol.Rule
	select {
	case r := <-ruleCh:
		pr = r
	case <-time.After(a.askTimeout):
		log.Info("Timeout reached. Applying default rule")
		pr = uiDefaultRule
		win.Close()
	}

	makeRuleName(pr)
	return pr, true
}

func makeRuleName(r *protocol.Rule) {
	reg, _ := regexp.Compile("[^a-zA-Z0-9]+")
	op := r.GetOperator()
	name := fmt.Sprintf("%s-%s-%s", r.GetAction(), op.GetType(), op.GetData())
	r.Name = reg.ReplaceAllString(name, "-")
}

// newApp returns a new fyne opensnitch application
func newApp(sigChan chan os.Signal, cfg *uiConfig) *osApp {

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
		defaultRule: &protocol.Rule{
			Name:     "ui.default",
			Action:   string(cfg.DefaultAction),
			Duration: string(cfg.DefaultDuration),
			Operator: &protocol.Operator{
				Type:    string(rule.Simple),
				Operand: string(cfg.DefaultOperator),
			},
		},
		askTimeout: time.Duration(cfg.DefaultTimeout) * time.Second,
	}
}

func generalStatsData(stats *protocol.Statistics) [][]string {
	status := "running"
	if stats.GetUptime() == 0 {
		status = "waiting for connection"
	}
	uptime := time.Duration(int(stats.GetUptime())) * time.Second

	return [][]string{
		{
			stats.GetDaemonVersion(),
			status,
			uptime.String(),
			fmt.Sprintf("%d", stats.GetRules()),
			fmt.Sprintf("%d", stats.GetConnections()),
			fmt.Sprintf("%d", stats.GetDropped()),
		},
	}
}

var generalStatsDefaultValues = [][]string{[]string{"-", "waiting for connection", "0", "0", "0", "0"}}

func generalStats() *table {
	headers := []string{"Version", "Daemon Status", "Uptime", "Rules", "Connections", "Dropped"}
	data := generalStatsDefaultValues
	return newTable(headers, data)
}

func makeLine() fyne.CanvasObject {
	rect := canvas.NewRectangle(&color.RGBA{128, 128, 128, 255})
	rect.SetMinSize(fyne.NewSize(30, 2))
	return rect
}
