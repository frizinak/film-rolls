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
	modeIDs    = "ids"
)

func main() {
	var verbose bool
	var format string
	var mode string
	var md bool
	var nh bool
	var dbFile string

	conf := db.TableConfigDefault()
	flag.BoolVar(&verbose, "v", false, "")
	flag.StringVar(&dbFile, "l", "", "")
	flag.StringVar(&mode, "m", modeLog, "")
	flag.StringVar(&format, "f", formatPretty, "")
	flag.StringVar(&conf.Separator, "s", conf.Separator, "")
	flag.BoolVar(&md, "md", false, "")
	flag.BoolVar(&nh, "nh", false, "")
	flag.BoolVar(&conf.Short, "short", false, "")
	flag.BoolVar(&conf.Notes, "notes", false, "")

	flag.StringVar(&conf.Filter.ID, "id", "", "")
	flag.StringVar(&conf.Filter.LID, "lid", "", "")
	flag.StringVar(&conf.Filter.SID, "sid", "", "")
	flag.StringVar(&conf.Filter.CID, "cid", "", "")
	flag.StringVar(&conf.Filter.Scan, "scan", "", "")
	flag.StringVar(&conf.Filter.StockFormat, "format", "", "")

	flag.BoolVar(&conf.Filter.StatusUndev, "undev", false, "")
	flag.BoolVar(&conf.Filter.StatusDev, "dev", false, "")
	flag.BoolVar(&conf.Filter.StatusLab, "lab", false, "")
	flag.BoolVar(&conf.Filter.StatusScanned, "scanned", false, "")
	flag.BoolVar(&conf.Filter.StatusUnscanned, "unscanned", false, "")
	flag.BoolVar(&conf.Filter.StatusLoaded, "loaded", false, "")
	flag.BoolVar(&conf.Filter.StatusUnloaded, "unloaded", false, "")
	flag.BoolVar(&conf.Filter.StockColor, "color", false, "")
	flag.BoolVar(&conf.Filter.StockBW, "bw", false, "")
	flag.BoolVar(&conf.Filter.StockPos, "pos", false, "")
	flag.BoolVar(&conf.Filter.StockNeg, "neg", false, "")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s <flags>:\n", os.Args[0])
		fmt.Print(`  General:
    -m <mode>               one of log, stock, prices or tags (default "log")
    -l <logfile>            (default ./rolls.log or $HOME/film-rolls.log)
    -v                      be verbose

  Output:
    -md                     output markdown compatible table (implies -f plain, ignores -s)
    -nh                     don't output header
    -s  <separator>         (default " │ ")
    -f  <format>            format: plain or pretty (default "pretty")
    -short                  shorter output
    -notes                  show notes

  Filter:
    -id     <roll-ids>      comma separated list of ids to filter on
    -cid    <camera-ids>    comma separated list of ids to filter on
    -lid    <lab-ids>       comma separated list of ids to filter on
    -sid    <stock-ids>     comma separated list of ids to filter on
    -scan   <scan-numbers>  comma separated list of ids to filter on
    -format <stock-formats> comma separated list of ids to filter on
    -dev                    show only developed rolls
    -undev                  show only undeveloped rolls
    -lab                    show only rolls at the lab
    -scanned                show only scanned rolls
    -unscanned              show only unscanned rolls
    -loaded                 show only loaded rolls
    -unloaded               show only unloaded rolls
    -color                  show only color rolls
    -bw                     show only b/w rolls
    -pos                    show only slides
    -neg                    show only negatives
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
		conf.StartEndWithSeparator = true
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

	case modeIDs:
		run = func(d *db.DB) {
			d.Row(db.Filter{}, func(e db.Entry, id string) {
				fmt.Println("id", id)
			})
			for id := range d.Cameras {
				fmt.Println("cid", id)
			}
			for id := range d.Stocks {
				fmt.Println("sid", id)
			}
			for id := range d.Labs {
				fmt.Println("lid", id)
			}
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
