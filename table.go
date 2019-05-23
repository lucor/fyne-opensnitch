// This file defines a simple table widget for Fyne
package main

import (
	"fmt"

	"fyne.io/fyne"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/widget"
)

// table represents a simple table widget for Fyne
type table struct {
	*widget.Box
	table *fyne.Container
	rows  int
	cols  int
}

func (t *table) SetHeaders(headers []string) error {
	if len(headers) != t.cols {
		return fmt.Errorf("Expected headers to be %d, got %d", len(headers), t.cols)
	}
	labels := t.table.Objects[0:len(headers)]
	for i, label := range labels {
		label.(*widget.Label).SetText(headers[i])
	}
	widget.Refresh(t)
	return nil
}

// SetContent refreshes the table content
func (t *table) SetData(data [][]string) error {
	var values []string
	for _, row := range data {
		values = append(values, row...)
	}

	if len(values) != (t.cols * t.rows) {
		return fmt.Errorf(
			"Expected table to be %d elements. %d x %d (rows x cols), got %d",
			(t.cols * t.rows), len(values), t.rows, t.cols,
		)
	}

	for i, label := range t.table.Objects {
		if i < t.cols {
			continue
		}
		label.(*widget.Label).SetText(values[i-t.cols])
	}
	widget.Refresh(t)
	return nil
}

// newTable returns a table of rows x cols plus an header row
func newTableWithHeaders(rows int, cols int) *table {

	tbl := fyne.NewContainerWithLayout(layout.NewGridLayout(cols))

	for c := 0; c < cols; c++ {
		tbl.AddObject(
			widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		)
	}

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			tbl.AddObject(
				widget.NewLabel(""),
			)
		}
	}

	return &table{
		Box:   widget.NewVBox(tbl),
		table: tbl,
		rows:  rows,
		cols:  cols,
	}
}
