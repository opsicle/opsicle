package cli

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"golang.org/x/term"
)

type TableRowDataInsertor func(*Table) error

type NewTableOpts struct {
	Headers     []string
	Rows        TableRowDataInsertor
	IsFullWidth bool
}

func NewTable(opts NewTableOpts) *Table {
	table := &Table{
		Rows:        opts.Rows,
		isFullWidth: opts.IsFullWidth,
	}
	return table.Init(opts.Headers)
}

type Table struct {
	data  bytes.Buffer
	table *tablewriter.Table

	Rows TableRowDataInsertor

	isFullWidth bool
}

func (t *Table) Init(headers []string) *Table {
	t.table = tablewriter.NewWriter(&t.data)
	t.table.Options(tablewriter.WithHeaderAlignment(tw.AlignLeft))
	t.table.Configure(func(cfg *tablewriter.Config) {
		width, _, _ := term.GetSize(int(os.Stdout.Fd()))
		if t.isFullWidth {
			cfg.Widths.Global = width
		} else {
			cfg.MaxWidth = width
		}
		cfg.Row.Padding.Global.Top = " "
		cfg.Row.Padding.Global.Bottom = " "
	})
	t.table.Header(headers)
	return t
}

func (t *Table) Render() *Table {
	t.Rows(t)
	return t
}

func (t *Table) NewRow(values ...any) error {
	row := []string{}
	for _, value := range values {
		var valueAsString string
		switch v := value.(type) {
		case int, int8, int16, int32, int64, float32, float64:
			valueAsString = fmt.Sprintf("%v", v)
		case bool:
			valueAsString = "✅"
			if !v {
				valueAsString = "❌"
			}
		case string:
			valueAsString = v
		case []string:
			valueAsString = fmt.Sprintf(`["%s"]`, strings.Join(v, `", "`))
		case []byte:
			valueAsString = string(v)
		}
		row = append(row, valueAsString)
	}
	return t.table.Append(row)
}

func (t *Table) GetString() string {
	t.table.Render()
	return t.data.String()
}
