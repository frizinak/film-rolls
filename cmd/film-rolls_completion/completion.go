package main

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
)

func filter(opts []string, comp string) []string {
	l := make([]string, 0, len(opts))
	for _, o := range opts {
		if !strings.HasPrefix(o, comp) {
			continue
		}
		l = append(l, o)
	}
	return l
}

func main() {
	if len(os.Args) < 4 {
		os.Exit(0)
	}

	comp := os.Args[2]
	prev := os.Args[3]
	fl := ""
	if strings.HasPrefix(prev, "-") {
		fl = prev[1:]
	}

	var file func() []string
	noFile := func() []string { return nil }

	type flag struct {
		name    string
		options func() []string
	}

	var get func(typ string) []string
	{
		load := func() map[string][]string {
			l := make(map[string][]string)
			cmd := exec.Command("film-rolls", "-m", "ids")
			out, err := cmd.StdoutPipe()
			if err != nil {
				return l
			}
			err = cmd.Start()
			if err != nil {
				return l
			}

			scan := bufio.NewScanner(out)
			scan.Split(bufio.ScanLines)
			for scan.Scan() {
				t := strings.Fields(scan.Text())
				if len(t) != 2 {
					continue
				}
				typ := t[0]
				val := t[1]
				if _, ok := l[typ]; !ok {
					l[typ] = make([]string, 0, 1)
				}

				l[typ] = append(l[typ], val)
			}

			_ = cmd.Wait()
			return l
		}

		var data map[string][]string
		get = func(typ string) []string {
			if data == nil {
				data = load()
			}

			vals := data[typ]
			if vals == nil {
				vals = []string{}
			}
			return vals
		}
	}

	date := func() []string {
		return get("date")
	}

	allFlags := []flag{
		{name: "m", options: func() []string { return []string{"log", "stocks", "prices", "cameras", "tags", "web"} }},
		{name: "l", options: file},
		{name: "v", options: noFile},
		{name: "addr", options: noFile},
		{name: "md", options: noFile},
		{name: "nh", options: noFile},
		{name: "s", options: noFile},
		{name: "f", options: func() []string { return []string{"plain", "pretty"} }},
		{name: "short", options: noFile},
		{name: "notes", options: noFile},
		{name: "sort", options: func() []string { return []string{"date", "scan"} }},
		{name: "id", options: func() []string { return get("id") }},
		{name: "cid", options: func() []string { return get("cid") }},
		{name: "lid", options: func() []string { return get("lid") }},
		{name: "sid", options: func() []string { return get("sid") }},
		{name: "file", options: func() []string { return get("file") }},
		{name: "scan", options: noFile},
		{name: "format", options: func() []string { return get("format") }},
		{name: "dev", options: noFile},
		{name: "undev", options: noFile},
		{name: "lab", options: noFile},
		{name: "scanned", options: noFile},
		{name: "unscanned", options: noFile},
		{name: "loaded", options: noFile},
		{name: "unloaded", options: noFile},
		{name: "color", options: noFile},
		{name: "bw", options: noFile},
		{name: "pos", options: noFile},
		{name: "neg", options: noFile},
		{name: "available", options: noFile},
		{name: "since", options: date},
		{name: "until", options: date},
		{name: "since-lab-in", options: date},
		{name: "until-lab-in", options: date},
		{name: "since-lab-out", options: date},
		{name: "until-lab-out", options: date},
	}

	var opts []string
	for _, f := range allFlags {
		if f.name == fl && f.options != nil {
			opts = f.options()
			if opts == nil {
				fl = ""
			}
		}
	}

	if fl == "" {
		for _, f := range allFlags {
			opts = append(opts, "-"+f.name)
		}
	}

	for _, n := range filter(opts, comp) {
		os.Stdout.WriteString(n)
		os.Stdout.WriteString("\n")
	}
}
