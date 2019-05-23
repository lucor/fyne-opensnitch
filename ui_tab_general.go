package main

import (
	"fmt"
	"time"

	"fyne.io/fyne"
	"github.com/evilsocket/opensnitch/daemon/ui/protocol"
)

var generalStatsDefaultValues = [][]string{[]string{"-", "waiting for connection", "0", "0", "0", "0"}}

func makeGeneralTab() fyne.Widget {
	headers := []string{"Version", "Daemon Status", "Uptime", "Rules", "Connections", "Dropped"}
	tbl := newTableWithHeaders(1, len(headers))
	tbl.SetHeaders(headers)
	tbl.SetData(generalStatsDefaultValues)
	return tbl
}

func generalTabData(stats *protocol.Statistics) [][]string {
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
