package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
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

	modeLog    = "log"
	modeStock  = "stock"
	modePrices = "prices"
	modeTags   = "tags"
)

func main() {
	var verbose bool
	var format string
	var mode string
	var md bool
	var nh bool
	var dbFile string

	conf := db.TableConfigDefault()
	flag.BoolVar(&verbose, "v", false, "Be verbose")
	flag.StringVar(&dbFile, "l", "", "Path to the log file [./rolls.log or $HOME/film-rolls.log]")
	flag.StringVar(&mode, "m", modeLog, fmt.Sprintf("Mode: %s, %s, %s or %s", modeLog, modeStock, modePrices, modeTags))
	flag.StringVar(&format, "f", formatPretty, fmt.Sprintf("Format: %s or %s", formatPlain, formatPretty))
	flag.StringVar(&conf.Separator, "s", conf.Separator, "Table column seperator")
	flag.BoolVar(&md, "md", false, fmt.Sprintf("Output markdown compatible table (implies -f %s, ignores -s)", formatPlain))
	flag.BoolVar(&nh, "nh", false, "Don't output header")

	flag.StringVar(&conf.Filter.ID, "id", "", "Filter by roll id")
	flag.StringVar(&conf.Filter.LID, "lid", "", "Filter by lab id")
	flag.StringVar(&conf.Filter.SID, "sid", "", "Filter by film stock id")
	flag.StringVar(&conf.Filter.CID, "cid", "", "Filter by camera id")
	flag.StringVar(&conf.Filter.Scan, "scan", "", "Filter by scan serial number")

	flag.BoolVar(&conf.Filter.StatusUndev, "undev", false, "Only show film rolls that have not yet been developed")
	flag.BoolVar(&conf.Filter.StatusDev, "dev", false, "Only show film rolls that have been developed")
	flag.BoolVar(&conf.Filter.StatusLab, "lab", false, "Only show film rolls that are at the lab")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s <flags>:\n", os.Args[0])
		fmt.Print(`  General:
    -m <mode>           one of log, stock, prices or tags (default "log")
    -l <logfile>        (default ./rolls.log or $HOME/film-rolls.log)
    -v                  be verbose

  Output:
    -md                 output markdown compatible table (implies -f plain, ignores -s)
    -nh                 don't output header
    -s  <separator>     (default " │ ")
    -f  <format>        Format: plain or pretty (default "pretty")

  Filter:
    -id   <roll-id>
    -cid  <camera-id>
    -lid  <lab-id>
    -sid  <stock-id>
    -scan <scan-number>
    -dev                show only developed rolls
    -undev              show only undeveloped rolls
    -lab                show only rolls at the lab
`)
	}
	flag.Parse()

	if dbFile == "" {
		dbFile = "rolls.log"
		if _, err := os.Stat(dbFile); os.IsNotExist(err) {
			h, err := os.UserHomeDir()
			if err != nil {
				exit(fmt.Errorf("could not find a log file: %w", err))
			}
			dbFile = filepath.Join(h, "film-rolls.log")
		}
	}

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

	var run func(db *db.DB)
	switch mode {
	case modeLog:
		conf.Width = termWidth()

		run = func(db *db.DB) {
			db.PrintTable(os.Stdout, conf)
		}

	case modeStock:
		conf.Width = termWidth()
		run = func(db *db.DB) {
			db.PrintStock(os.Stdout, conf)
		}

	case modePrices:
		conf.Width = termWidth()
		run = func(db *db.DB) {
			db.PrintPrices(os.Stdout, conf)
		}

	case modeTags:
		run = func(db *db.DB) {
			db.PrintTags(os.Stdout, conf.Filter)
		}

	default:
		fmt.Fprintf(os.Stderr, "invalid mode '%s'\n", mode)
		os.Exit(1)
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
