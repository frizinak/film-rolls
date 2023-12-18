package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/containerd/console"
	"github.com/frizinak/film-rolls/db"
)

func exit(err error) {
	if err == nil {
		return
	}

	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func main() {
	var verbose bool
	var output string
	flag.BoolVar(&verbose, "v", false, "Be verbose.")
	flag.StringVar(&output, "f", "text", "Output format: text or html")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s <flags> [file]:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	var run func(db *db.DB)
	switch output {
	case "text":
		w := 0
		s, err := console.Current().Size()
		if err == nil {
			w = int(s.Width) - 5
		}
		if w < 80 && w != 0 {
			w = 80
		}

		run = func(db *db.DB) { db.PrintTable(os.Stdout, w) }

	case "html":
		run = func(db *db.DB) {
			fmt.Println(`<table class="film-rolls">`)
			db.PrintHTMLTable(os.Stdout)
			fmt.Println(`</table>`)
		}

	default:
		fmt.Fprintln(os.Stderr, "invalid output format")
		os.Exit(1)
	}

	dbFile := flag.Arg(0)
	if dbFile == "" {
		dbFile = "./rolls.log"
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Opening %s\n", dbFile)
	}

	bench := time.Now()
	f, err := os.Open(dbFile)
	exit(err)
	db, err := db.Parse(f)
	f.Close()
	exit(err)

	run(db)

	if verbose {
		fmt.Fprintln(os.Stderr, time.Since(bench))
	}
}
