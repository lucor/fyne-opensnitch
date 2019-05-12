// This file defines a simple table widget for Fyne
package main

import (
	"fyne.io/fyne"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/widget"
)

// table represents a simple table widget for Fyne
type table struct {
	*widget.Box
	headers int
}

// SetContent refreshes the table content
func (t *table) SetContent(data [][]string) {
	var values []string
	for _, row := range data {
		values = append(values, row...)
	}

	container := t.Box.Children[0].(*fyne.Container)
	for i, label := range container.Objects {
		if i < t.headers {
			continue
		}
		label.(*widget.Label).SetText(values[i-t.headers])
	}
	widget.Refresh(t.Box)
}

// newTable returns a table
func newTable(headers []string, data [][]string) *table {

	container := fyne.NewContainerWithLayout(layout.NewGridLayout(len(headers)))

	for _, header := range headers {
		container.AddObject(
			widget.NewLabelWithStyle(header, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		)
	}

	for _, rows := range data {
		for _, d := range rows {
			container.AddObject(widget.NewLabel(d))
		}
	}
	obj := widget.NewVBox(container)

	return &table{
		headers: len(headers),
		Box:     obj,
	}
}
