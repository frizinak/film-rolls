package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/frizinak/film-rolls/db"
	"github.com/sergi/go-diff/diffmatchpatch"
)

var linesRE = regexp.MustCompile(`\r\n|\n\r|\r`)

func tmpFile(file string) string {
	stamp := strconv.FormatInt(time.Now().UnixNano(), 36)
	rnd := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, rnd)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf(
		"%s.%s-%s.tmp",
		file,
		stamp,
		base64.RawURLEncoding.EncodeToString(rnd),
	)
}

type htmlWriter struct {
	w io.Writer
}

var htmlEsc = map[byte][]byte{
	'&':  []byte("&amp;"),
	'\'': []byte("&#39;"),
	'<':  []byte("&lt;"),
	'>':  []byte("&gt;"),
	'"':  []byte("&#34;"),
}

func (w *htmlWriter) Write(b []byte) (int, error) {
	olb := len(b)
	for i := 0; i < len(b); i++ {
		if v, ok := htmlEsc[b[i]]; ok {
			n := make([]byte, len(v)+len(b)-i-1)
			copy(n, v)
			copy(n[len(v):], b[i+1:])
			b = append(b[:i], n...)
			i += len(v) - 1
		}
	}

	n, err := w.w.Write(b)
	if n < len(b) {
		olb--
	}
	if olb < 0 {
		olb = 0
	}

	return olb, err
}

func web(flags *flag.FlagSet, addr, dbDir string, mode *string, conf *db.TableConfig) error {
	confCLI := *conf

	header := func(w http.ResponseWriter, title string) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, `<!DOCTYPE html><html lang="en-US"><head>`)
		t := "Film Rolls"
		h1 := t
		if title != "" {
			t = fmt.Sprintf("Film Rolls - %s", title)
			h1 = title
		}
		fmt.Fprintf(w, `<title>%s</title>`, t)
		fmt.Fprintln(w, `<style>
html, body { width: 100%; min-height: 100%; background-color: #333; color: #ccc }
.diff-del { background-color: #c33; }
.diff-ins { background-color: #373; }
.diff-eli { background-color: #555; }
form [name=confirmed] { display: none; }
a { color: #ddd; }
a:hover { color: #fff; }
</style>`)
		fmt.Fprintf(w, `</head><body><main><h1>%s</h1>`, h1)
	}
	footer := func(w io.Writer) {
		fmt.Fprintln(w, `</main></body></html>`)
	}

	wErr := func(w http.ResponseWriter, err error, status ...int) {
		if err != nil {
			fmt.Println(err)
		}

		s := 500
		if len(status) != 0 {
			s = status[0]
		}
		w.WriteHeader(s)
		header(w, http.StatusText(s))
		footer(w)
	}

	bools := make(map[string]struct{}, 0)
	block := map[string]struct{}{"l": {}}
	flags.VisitAll(func(f *flag.Flag) {
		str := f.Value.String()
		if str == "true" || str == "false" {
			bools[f.Name] = struct{}{}
		}
	})

	var dbase *db.DB
	reload := func() error {
		var err error
		dbase, err = db.ParseDir(dbDir, nil)
		return err
	}

	if err := reload(); err != nil {
		return err
	}

	files := make([]string, 0)
	filesAllowed := make(map[string]struct{}, 0)
	_, err := db.ParseDir(dbDir, func(_ *db.DB, file string) (bool, error) {
		files = append(files, file)
		filesAllowed[file] = struct{}{}
		return false, nil
	})
	if err != nil {
		return err
	}

	var l sync.Mutex
	return http.ListenAndServe(addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l.Lock()
		defer l.Unlock()

		var baseStr string
		{
			base := &url.URL{
				Scheme: r.URL.Scheme,
				Host:   r.Host,
			}

			if b := r.Header.Get("X-Basepath"); b != "" {
				base.Path = b
			}

			baseStr = base.String()
		}

		path := strings.Trim(r.URL.Path, "/")
		pathp := strings.Split(path, "/")
		if len(pathp) == 1 && pathp[0] == "" {
			pathp = nil
		}

		if path == "favicon.ico" {
			wErr(w, nil, http.StatusNotFound)
			return
		}

		if path == "edit" {
			header(w, "Files")
			for _, file := range files {
				fmt.Fprintf(w, "<a href=\"%s/edit/%s\">%s</a><br/>\n", baseStr, file, file)
			}
			footer(w)
			return
		}

		if len(pathp) > 1 && pathp[0] == "edit" {
			fn := filepath.Join(pathp[1:]...)
			if _, ok := filesAllowed[fn]; !ok {
				wErr(w, nil, http.StatusNotFound)
				return
			}

			file := filepath.Join(dbDir, fn)
			current, err := os.ReadFile(file)
			if err != nil {
				wErr(w, err)
				return
			}

			if r.Method == "POST" {
				if err := r.ParseForm(); err != nil {
					wErr(w, err)
					return
				}

				key := "input"
				if _, ok := r.Form["confirmed"]; ok {
					key = "confirmed"
				}

				input := linesRE.ReplaceAllString(r.Form.Get(key), "\n")
				if key == "confirmed" {
					tmp := tmpFile(file)
					f, err := os.Create(tmp)
					if err != nil {
						wErr(w, err)
						return
					}

					_, err = f.WriteString(input)
					f.Close()
					if err != nil {
						os.Remove(tmp)
						wErr(w, err)
						return
					}

					if err = os.Rename(tmp, file); err != nil {
						wErr(w, err)
						return
					}

					http.Redirect(w, r, baseStr, http.StatusFound)
					return
				}

				_, err = db.ParseDir(dbDir, func(db *db.DB, file string) (bool, error) {
					if file == fn {
						err := db.Parse(file, strings.NewReader(input))
						return false, err
					}

					return true, nil
				})

				if err != nil {
					wErr(w, err, http.StatusNotAcceptable)
					return
				}

				dmp := diffmatchpatch.New()
				diff := dmp.DiffMain(string(current), input, true)
				header(w, fn)
				fmt.Fprintln(w, "<pre>")
				for _, d := range diff {
					switch d.Type {
					case diffmatchpatch.DiffDelete:
						fmt.Fprintf(w, "<span class=\"diff-del\">%s</span>", html.EscapeString(d.Text))
					case diffmatchpatch.DiffInsert:
						fmt.Fprintf(w, "<span class=\"diff-ins\">%s</span>", html.EscapeString(d.Text))
					case diffmatchpatch.DiffEqual:
						text := html.EscapeString(d.Text)
						min := 6
						if lines := strings.Split(d.Text, "\n"); len(lines) > 2*min {
							end := lines[len(lines)-min:]
							lines = append(lines[:min], "<span class=\"diff-eli\">...</span>")
							lines = append(lines, end...)
							text = strings.Join(lines, "\n")
						}
						fmt.Fprintf(w, "<span class=\"diff-eql\">%s</span>", text)
					}
				}
				fmt.Fprintln(w, "</pre>")

				fmt.Fprintln(w, `<form method="POST">`)
				fmt.Fprintln(w, `<textarea name="confirmed" readonly="readonly" rows="50" cols="80">`)
				fmt.Fprint(w, input)
				fmt.Fprintln(w, `</textarea><br/>`)
				fmt.Fprintln(w, `<input type="submit" value="submit"/>`)
				fmt.Fprintln(w, `</form>`)

				footer(w)

				return
			}

			header(w, fn)
			fmt.Fprintln(w, `<form method="POST">`)
			fmt.Fprintln(w, `<textarea name="input" rows="50" cols="80">`)
			w.Write(current)
			fmt.Fprintln(w, `</textarea><br/>`)
			fmt.Fprintln(w, `<input type="submit" value="submit"/>`)
			fmt.Fprintln(w, `</form>`)
			footer(w)
			return
		}

		*conf = confCLI
		conf.Color = false
		conf.Pretty = false
		*mode = modeLog

		for i := 0; i < len(pathp); i++ {
			p := pathp[i]
			if _, isBool := bools[p]; isBool {
				if err := flag.CommandLine.Set(p, "true"); err != nil {
					wErr(w, err, http.StatusNotAcceptable)
					return
				}
				continue
			}

			if p == "h" || p == "help" {
				header(w, "Help")
				fmt.Fprint(w, "<pre>")
				usage(&htmlWriter{w})
				fmt.Fprint(w, "</pre>")
				footer(w)
				return
			}

			val := ""
			if i < len(pathp)-1 {
				val = pathp[i+1]
				i++
			}
			if _, blocked := block[p]; !blocked {
				if err := flag.CommandLine.Set(p, val); err != nil {
					wErr(w, err, http.StatusNotAcceptable)
					return
				}
				continue
			}

			wErr(w, nil, http.StatusNotAcceptable)
			return
		}

		if err = reload(); err != nil {
			wErr(w, err)
			return
		}

		header(w, "")
		fmt.Fprint(w, "<pre>")
		htmlw := &htmlWriter{w}
		switch *mode {
		case modeLog:
			dbase.PrintLogs(htmlw, *conf)
		case modeStock:
			dbase.PrintStocks(htmlw, *conf)
		case modeCameras:
			dbase.PrintCameras(htmlw, *conf)
		case modePrices:
			dbase.PrintPrices(htmlw, *conf)
		default:
			wErr(w, nil, http.StatusNotFound)
		}
		fmt.Fprint(w, "</pre>")
		footer(w)
	}))
}
