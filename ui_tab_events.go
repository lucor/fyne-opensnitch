package main

import (
	"sort"

	"fyne.io/fyne"
	"github.com/evilsocket/opensnitch/daemon/ui/protocol"
)

const eventsTabRows = 20

func makeEventsTab() fyne.Widget {
	headers := []string{"Time", "Action", "Process", "Destination", "Protocol", "Rule"}
	tbl := newTableWithHeaders(eventsTabRows, len(headers))
	tbl.SetHeaders(headers)
	return tbl
}

func eventsTabData(stats *protocol.Statistics) [][]string {
	events := stats.GetEvents()
	l := len(events)
	rows := eventsTabRows
	if l < eventsTabRows {
		rows = l
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].GetTime() > events[j].GetTime()
	})

	var data [][]string
	for i := 0; i < rows; i++ {
		data = append(data, []string{
			events[i].GetTime(),
			events[i].GetRule().GetAction(),
			events[i].GetConnection().GetProcessPath(),
			events[i].GetConnection().GetDstHost(),
			events[i].GetConnection().GetProtocol(),
			events[i].GetRule().GetName(),
		})
	}
	return data
}
