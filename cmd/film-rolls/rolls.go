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

const (
	formatPlain  = "plain"
	formatPretty = "pretty"

	modeLog   = "log"
	modeStock = "stock"
	modeTags  = "tags"
)

func main() {
	var verbose bool
	var format string
	var mode string
	var id string
	var md bool
	var nh bool
	conf := db.TableConfigDefault()
	flag.BoolVar(&verbose, "v", false, "Be verbose.")
	flag.StringVar(&mode, "m", modeLog, fmt.Sprintf("Mode: %s, %s or %s", modeLog, modeStock, modeTags))
	flag.StringVar(&format, "f", formatPretty, fmt.Sprintf("Format: %s or %s", formatPlain, formatPretty))
	flag.StringVar(&conf.Separator, "s", conf.Separator, "Table column seperator")
	flag.BoolVar(&md, "md", false, fmt.Sprintf("Output markdown compatible table (implies -f %s, ignores -s)", formatPlain))
	flag.BoolVar(&nh, "nh", false, "Don't output header")
	flag.StringVar(&id, "id", "", "Only show film roll with the given id")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s <flags> [file]:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if format != formatPlain && format != formatPretty {
		fmt.Fprintf(os.Stderr, "invalid format '%s'\n", format)
		os.Exit(1)
	}

	termWidth := func() int {
		if md {
			return 0
		}
		s, err := console.Current().Size()
		if err != nil {
			return 0
		}
		w := int(s.Width) - 5
		if w < 80 && w != 0 {
			w = 80
		}

		return w
	}

	conf.Header = true
	if nh {
		conf.Header = false
	}

	if md {
		// conf.Header = true
		conf.HeaderSep = true
		conf.Separator = " | "
		conf.StartEndWithSeperator = true
		format = formatPlain
	}

	conf.Color = format == formatPretty
	conf.Pretty = conf.Color

	var run func(db *db.DB, id string)
	switch mode {
	case modeLog:
		conf.IDFilter = id
		conf.Width = termWidth()

		run = func(db *db.DB, id string) {
			db.PrintTable(os.Stdout, conf)
		}

	case modeStock:
		conf.Width = termWidth()
		run = func(db *db.DB, id string) {
			db.PrintStock(os.Stdout, conf)
		}

	case modeTags:
		run = func(db *db.DB, id string) {
			db.PrintTags(os.Stdout, id)
		}

	default:
		fmt.Fprintf(os.Stderr, "invalid mode '%s'\n", mode)
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

	run(db, id)

	if verbose {
		fmt.Fprintln(os.Stderr, time.Since(bench))
	}
}
