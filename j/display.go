package main

import (
	"fmt"
	"strings"
)

// display formats an Array for REPL output, following J conventions.
func display(a *Array) string {
	if a == nil {
		return ""
	}
	return displayAt(a, 0)
}

func displayAt(a *Array, depth int) string {
	switch d := a.data.(type) {
	case []int64:
		return displayNums(a.shape, func(i int) string { return fmt.Sprintf("%d", d[i]) })
	case []float64:
		return displayNums(a.shape, func(i int) string { return formatFloat(d[i]) })
	case []bool:
		return displayNums(a.shape, func(i int) string {
			if d[i] {
				return "1"
			}
			return "0"
		})
	case []*Array:
		if a.rank() == 0 {
			return "+-\n|" + displayAt(d[0], depth+1) + "|\n+-"
		}
		parts := make([]string, len(d))
		for i, b := range d {
			parts[i] = displayAt(b, depth+1)
		}
		return strings.Join(parts, " ")
	}
	return ""
}

func displayNums(shape []int, atom func(int) string) string {
	n := 1
	for _, s := range shape {
		n *= s
	}
	if len(shape) == 0 {
		// scalar
		return atom(0)
	}
	if len(shape) == 1 {
		parts := make([]string, n)
		for i := range n {
			parts[i] = atom(i)
		}
		return strings.Join(parts, " ")
	}
	// matrix or higher: print row by row separated by newlines
	rows := shape[0]
	cols := n / rows
	lines := make([]string, rows)
	for r := range rows {
		parts := make([]string, cols)
		for c := range cols {
			parts[c] = atom(r*cols + c)
		}
		lines[r] = strings.Join(parts, " ")
	}
	return strings.Join(lines, "\n")
}

func formatFloat(f float64) string {
	s := fmt.Sprintf("%g", f)
	return s
}
