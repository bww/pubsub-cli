package main

import (
	"fmt"
	"os"
	"strings"
)

func stringer(v interface{}) string {
	switch c := v.(type) {
	case nil:
		return ""
	case string:
		return c
	case *string:
		return *c
	default:
		return fmt.Sprint(v)
	}
}

func dumpAttrs(a map[string]string) string {
	b := &strings.Builder{}
	var n int
	for k, v := range a {
		if n > 0 {
			b.WriteString(", ")
		}
		b.WriteString(k)
		b.WriteString(": ")
		b.WriteString(v)
		n++
	}
	return b.String()
}

func logf(f string, a ...any) {
	fmt.Fprintf(os.Stderr, f, a...)
}

func log(m ...any) {
	fmt.Fprint(os.Stderr, m...)
}

func logln(m ...any) {
	fmt.Fprintln(os.Stderr, m...)
}
