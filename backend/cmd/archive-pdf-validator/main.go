package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strconv"

	"github.com/ledongthuc/pdf"
)

func main() {
	// This helper is intentionally a separate process. A pathological PDF is
	// killed by the API deadline and parsing is serialized by the parent.
	debug.SetMemoryLimit(128 << 20)
	if len(os.Args) != 3 {
		os.Exit(2)
	}
	maxPages, err := strconv.Atoi(os.Args[2])
	if err != nil || maxPages < 1 {
		os.Exit(2)
	}
	pages, ok := validate(os.Args[1], maxPages)
	if !ok {
		os.Exit(1)
	}
	_, _ = fmt.Fprintln(os.Stdout, pages)
}

func validate(path string, maxPages int) (pages int, ok bool) {
	defer func() {
		if recover() != nil {
			pages, ok = 0, false
		}
	}()
	file, reader, err := pdf.Open(path)
	if err != nil {
		return 0, false
	}
	defer file.Close() //nolint:errcheck
	pages = reader.NumPage()
	return pages, pages >= 1 && pages <= maxPages
}
