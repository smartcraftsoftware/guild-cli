package output

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// PrintTable prints data in an aligned table format.
func PrintTable(w io.Writer, headers []string, rows [][]string) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	// Header
	fmt.Fprintln(tw, strings.Join(headers, "\t"))
	// Separator
	seps := make([]string, len(headers))
	for i, h := range headers {
		seps[i] = strings.Repeat("─", len(h))
	}
	fmt.Fprintln(tw, strings.Join(seps, "\t"))

	// Rows
	for _, row := range rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}

	tw.Flush()
}

// PrintDetail prints key-value pairs in a detail view.
func PrintDetail(w io.Writer, fields [][]string) {
	maxKeyLen := 0
	for _, f := range fields {
		if len(f[0]) > maxKeyLen {
			maxKeyLen = len(f[0])
		}
	}

	for _, f := range fields {
		key, value := f[0], f[1]
		fmt.Fprintf(w, "%-*s  %s\n", maxKeyLen, key+":", value)
	}
}
