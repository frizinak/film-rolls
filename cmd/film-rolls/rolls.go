package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/containerd/console"
	"github.com/frizinak/film-rolls/db"
)

const dateFormat = "2006-01-02"

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

	modeLog     = "log"
	modeStock   = "stocks"
	modePrices  = "prices"
	modeCameras = "cameras"
	modeTags    = "tags"
	modeIDs     = "ids"
)

func usage(w io.Writer) {
	fmt.Fprint(w, `film-rolls <flags>:
  General:
    -m <mode>               one of log, stocks, cameras, prices or tags (default "log")
    -l <logdirectory>       directory containing your .log and .def files
                            which are read in alphabetical order (first .def then .log).
                            (default $HOME/film-rolls)
    -v                      be verbose

  Output:
    -md                     output markdown compatible table (implies -f plain, ignores -s)
    -nh                     don't output header
    -s  <separator>         (default " │ ")
    -f  <format>            format: plain or pretty (default "pretty")
    -short                  shorter output
    -notes                  show notes
    -sort <sort-mode>       sort by either date or scan (default "date")

  Query Filters:
    -id     <QUERY>         filter by id
    -cid    <QUERY>         filter by camera id
    -lid    <QUERY>         filter by lab id
    -sid    <QUERY>         filter by stock id
    -file   <QUERY>         filter by filename
    -scan   <QUERY>         filter by scan
    -format <QUERY>         filter by film format
             QUERY:         comma separated list of individual queries, supports * wildcards
                            e.g.: -id 9f*,da930

  Date Filters:
    -since         <DATE>   only show rolls loaded in a camera starting from this date
    -until         <DATE>   only show rolls loaded in a camera until this date (inclusive)
    -since-lab-in  <DATE>   only show rolls delivered to a lab starting from this date
    -until-lab-in  <DATE>   only show rolls delivered to a lab until this date (inclusive)
    -since-lab-out <DATE>   only show rolls retrieved from a lab starting from this date
    -until-lab-out <DATE>   only show rolls retrieved from a lab until this date (inclusive)
                    DATE:   YYYY-MM-DD

  Boolean Filters:
    -dev                    only show developed rolls
    -undev                  only show undeveloped rolls
    -lab                    only show rolls at the lab
    -scanned                only show scanned rolls
    -unscanned              only show unscanned rolls
    -loaded                 only show loaded rolls
    -unloaded               only show unloaded rolls
    -color                  only show color rolls
    -bw                     only show b/w rolls
    -pos                    only show slides
    -neg                    only show negatives
    -available              only show rolls we have [-m stocks]
`)
}

type SortVar struct {
	*db.Sort
}

func (s *SortVar) String() string {
	switch *s.Sort {
	case db.SortDefault:
		return "date"
	case db.SortScan:
		return "scan"
	}
	return "<NOT IMPLEMENTED>"
}

func (s *SortVar) Set(i string) error {
	switch i {
	case "date":
		*s.Sort = db.SortDefault
	case "scan":
		*s.Sort = db.SortScan
	default:
		return fmt.Errorf("'%s' is not a valid sort option", i)
	}

	return nil
}

func main() {
	var verbose bool
	var format string
	var mode string
	var md bool
	var nh bool
	var dbDir string

	conf := db.TableConfigDefault()
	flag.BoolVar(&verbose, "v", false, "")
	flag.StringVar(&dbDir, "l", "", "")
	flag.StringVar(&mode, "m", modeLog, "")
	flag.StringVar(&format, "f", formatPretty, "")
	flag.StringVar(&conf.Separator, "s", conf.Separator, "")
	flag.BoolVar(&md, "md", false, "")
	flag.BoolVar(&nh, "nh", false, "")
	flag.BoolVar(&conf.Short, "short", false, "")
	flag.BoolVar(&conf.Notes, "notes", false, "")

	var sort = SortVar{&conf.Sort}
	flag.Var(&sort, "sort", "")

	flag.StringVar(&conf.Filter.ID, "id", "", "")
	flag.StringVar(&conf.Filter.LID, "lid", "", "")
	flag.StringVar(&conf.Filter.SID, "sid", "", "")
	flag.StringVar(&conf.Filter.CID, "cid", "", "")
	flag.StringVar(&conf.Filter.File, "file", "", "")
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
	flag.BoolVar(&conf.Filter.StockAvailable, "available", false, "")

	flag.StringVar(&conf.Filter.Since, "since", "", "")
	flag.StringVar(&conf.Filter.Until, "until", "", "")

	flag.StringVar(&conf.Filter.SinceLabIn, "since-lab-in", "", "")
	flag.StringVar(&conf.Filter.UntilLabIn, "until-lab-in", "", "")

	flag.StringVar(&conf.Filter.SinceLabOut, "since-lab-out", "", "")
	flag.StringVar(&conf.Filter.UntilLabOut, "until-lab-out", "", "")

	flag.Usage = func() { usage(os.Stderr) }
	flag.Parse()

	if dbDir == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			exit(fmt.Errorf("could not find a log file: %w", err))
		}
		dbDir = filepath.Join(h, "film-rolls")
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
			db.PrintLogs(os.Stdout, conf)
		}

	case modeStock:
		conf.Width = termWidth()
		run = func(db *db.DB) {
			db.PrintStocks(os.Stdout, conf)
		}

	case modeCameras:
		conf.Width = termWidth()
		run = func(db *db.DB) {
			db.PrintCameras(os.Stdout, conf)
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
		nilTime := time.Time{}
		run = func(d *db.DB) {
			fileUniq := make(map[string]struct{}, 0)
			formatUniq := make(map[string]struct{}, 0)
			dateUniq := make(map[string]struct{}, 0)

			date := func(d time.Time) {
				if d == nilTime {
					return
				}
				s := d.Format(dateFormat)
				if _, ok := dateUniq[s]; !ok {
					fmt.Println("date", s)
				}
			}

			d.Row(db.Filter{}, func(e db.Entry) {
				fmt.Println("id", e.State.ID)
				date(e.LoadDate)
				date(e.LabInDate)
				date(e.LabOutDate)

				func() {
					if e.File == "" {
						return
					}
					if _, ok := fileUniq[e.File]; ok {
						return
					}
					fileUniq[e.File] = struct{}{}
					fmt.Println("file", e.File)
				}()
			})
			for id := range d.Cameras {
				fmt.Println("cid", id)
			}
			for id, s := range d.Stocks {
				fmt.Println("sid", id)

				func() {
					if _, ok := formatUniq[s.Format]; ok {
						return
					}
					formatUniq[s.Format] = struct{}{}
					fmt.Println("format", s.Format)
				}()
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
		fmt.Fprintf(os.Stderr, "Opening %s\n", dbDir)
	}

	bench := time.Now()
	db, err := db.ParseDir(dbDir)
	exit(err)

	run(db)

	if verbose {
		fmt.Fprintln(os.Stderr, time.Since(bench))
	}
}
