package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Table struct {
	Header []string
	Rows   [][]interface{}
}

func NewTable(header []string) *Table {
	return &Table{
		Header: header,
		Rows:   make([][]interface{}, 0),
	}
}

func (t *Table) AddRow(row []interface{}) {
	t.Rows = append(t.Rows, row)
}

func (t *Table) Write() {
	rows := t.Render()
	widths := t.ColumnWidths(rows)

	for i, label := range t.Header {
		if i > 0 {
			fmt.Fprintf(os.Stderr, "  ")
		}

		label = fmt.Sprintf("%-*s", widths[i], strings.ToUpper(label))
		fmt.Fprint(os.Stderr, Colorize(ColorYellow, label))
	}

	fmt.Fprintln(os.Stderr, "")

	for _, row := range rows {
		for j, s := range row {
			if j > 0 {
				fmt.Printf("  ")
			}

			fmt.Printf("%-*s", widths[j], s)
		}

		fmt.Println("")
	}
}

func (t *Table) Render() [][]string {
	rows := make([][]string, len(t.Rows))

	for i, row := range t.Rows {
		rows[i] = make([]string, len(row))

		for j, value := range row {
			rows[i][j] = t.RenderValue(value)
		}
	}

	return rows
}

func (t *Table) RenderValue(value interface{}) string {
	switch v := value.(type) {
	case time.Time:
		return v.Format(time.RFC3339)
	case *time.Time:
		if v == nil {
			return ""
		} else {
			return v.Format(time.RFC3339)
		}
	case *time.Duration:
		if v == nil {
			return ""
		} else {
			return FormatDuration(*v)
		}
	}

	return fmt.Sprintf("%v", value)
}

func (t *Table) ColumnWidths(rows [][]string) []int {
	widths := make([]int, len(t.Header))

	for i, label := range t.Header {
		widths[i] = len(label)
	}

	for _, row := range rows {
		for j, value := range row {
			if len(value) > widths[j] {
				widths[j] = len(value)
			}
		}
	}

	return widths
}

func FormatDuration(d time.Duration) string {
	s := int(d.Seconds())

	if s == 0 {
		return ""
	}

	h := s / 3600
	s = s - h*3600

	m := s / 60
	s = s - m*60

	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}
