package db

import (
	"flag"
	"strconv"
	"strings"
	"time"
)

func wcGen(str string) []string {
	return strings.Split(str, "*")
}

func wcMatch(query []string, target string) bool {
	if len(query) == 1 {
		return target == query[0]
	}

	for i, p := range query {
		method := strings.Contains
		if i == 0 {
			method = strings.HasPrefix
		}
		if i == len(query)-1 {
			method = strings.HasSuffix
		}

		if p == "" {
			continue
		}

		if !method(target, p) {
			return false
		}
	}
	return true
}

func intRange(str string) ([]int, bool) {
	comma := strings.FieldsFunc(str, func(r rune) bool {
		return r == ',' || r == ' '
	})
	r := make([]int, 0, len(comma))
	for _, n := range comma {
		dash := strings.SplitN(n, "-", 2)
		v, err := strconv.Atoi(strings.TrimSpace(dash[0]))
		if err != nil {
			return r, false
		}
		if len(dash) != 2 {
			r = append(r, v)
			continue
		}

		if len(dash) == 2 {
			v2, err := strconv.Atoi(strings.TrimSpace(dash[1]))
			if err != nil {
				return r, false
			}
			if v2 < v {
				return r, false
			}
			for i := v; i <= v2; i++ {
				r = append(r, i)
			}
		}
	}

	return r, len(r) > 0
}

type Filter struct {
	ID  string
	LID string
	SID string
	CID string

	File string
	Scan string

	StatusUndev     bool
	StatusDev       bool
	StatusLab       bool
	StatusScanned   bool
	StatusUnscanned bool
	StatusLoaded    bool
	StatusUnloaded  bool

	StockFormat    string
	StockColor     bool
	StockBW        bool
	StockNeg       bool
	StockPos       bool
	StockAvailable bool

	StockISO string
	EI       string

	Since, Until             string
	SinceLabIn, UntilLabIn   string
	SinceLabOut, UntilLabOut string

	All bool
}

func DefineFilterFlags(f *Filter, fl *flag.FlagSet) {
	fl.StringVar(&f.ID, "id", "", "")
	fl.StringVar(&f.LID, "lid", "", "")
	fl.StringVar(&f.SID, "sid", "", "")
	fl.StringVar(&f.CID, "cid", "", "")
	fl.StringVar(&f.File, "file", "", "")
	fl.StringVar(&f.Scan, "scan", "", "")
	fl.StringVar(&f.StockFormat, "format", "", "")

	fl.StringVar(&f.StockISO, "iso", "", "")
	fl.StringVar(&f.EI, "ei", "", "")

	fl.BoolVar(&f.All, "a", false, "")
	fl.BoolVar(&f.StatusUndev, "undev", false, "")
	fl.BoolVar(&f.StatusDev, "dev", false, "")
	fl.BoolVar(&f.StatusLab, "lab", false, "")
	fl.BoolVar(&f.StatusScanned, "scanned", false, "")
	fl.BoolVar(&f.StatusUnscanned, "unscanned", false, "")
	fl.BoolVar(&f.StatusLoaded, "loaded", false, "")
	fl.BoolVar(&f.StatusUnloaded, "unloaded", false, "")
	fl.BoolVar(&f.StockColor, "color", false, "")
	fl.BoolVar(&f.StockBW, "bw", false, "")
	fl.BoolVar(&f.StockPos, "pos", false, "")
	fl.BoolVar(&f.StockNeg, "neg", false, "")
	fl.BoolVar(&f.StockAvailable, "available", false, "")

	fl.StringVar(&f.Since, "since", "", "")
	fl.StringVar(&f.Until, "until", "", "")

	fl.StringVar(&f.SinceLabIn, "since-lab-in", "", "")
	fl.StringVar(&f.UntilLabIn, "until-lab-in", "", "")

	fl.StringVar(&f.SinceLabOut, "since-lab-out", "", "")
	fl.StringVar(&f.UntilLabOut, "until-lab-out", "", "")
}

type idable interface {
	IDString() string
}

func (f Filter) gID(i idable) string {
	if i == nil {
		return ""
	}
	return i.IDString()
}

func (f Filter) notl(s []string, val string) bool {
	if len(s) == 0 || (len(s) == 1 && s[0] == "") {
		return false
	}
	if val == "" {
		return true
	}

	for _, v := range s {
		if wcMatch(wcGen(v), val) {
			return false
		}
	}
	return true
}

func (f Filter) not(s, val string) bool {
	return f.notl(strings.Split(s, ","), val)
}

func (f Filter) notinrange(rng string, valMin, valMax int) bool {
	options, ok := intRange(rng)
	if !ok {
		return false
	}
	for _, option := range options {
		if option >= valMin && option <= valMax {
			return false
		}
	}

	return true
}

func (f Filter) MatchStock(s *Stock) bool {
	switch {
	case f.not(f.SID, f.gID(s)):
		return false
	case f.not(f.StockFormat, s.Format):
		return false
	case f.StockColor && !s.Type.Color():
		return false
	case f.StockBW && !s.Type.BlackWhite():
		return false
	case f.StockPos && !s.Type.Pos():
		return false
	case f.StockNeg && !s.Type.Neg():
		return false
	case f.notinrange(f.StockISO, s.ISO.Low, s.ISO.High):
		return false
	}

	return true
}

func (f Filter) MatchCamera(c *Camera) bool {
	switch {
	case f.not(f.CID, f.gID(c)):
		return false
	}

	return true
}

func (f Filter) Match(e Entry) bool {
	if !f.All && e.Hide {
		return false
	}

	scan := make([]string, 0, 1)
	for _, v := range strings.Split(f.Scan, ",") {
		scan = append(scan, strings.TrimLeft(v, "0"))
	}

	eScan := ""
	if e.Scan != 0 {
		eScan = strconv.FormatUint(uint64(e.Scan), 10)
	}

	if !f.MatchStock(e.Stock) {
		return false
	}

	if !f.MatchCamera(e.Camera) {
		return false
	}

	ndate := func(s string, d time.Time, cmp func(time.Time) bool) bool {
		if s == "" {
			return false
		}

		if d == (time.Time{}) {
			return true
		}

		t, err := time.Parse(dateFormat, s)
		if err != nil {
			return true
		}

		return cmp(t)
	}

	ei := e.EI()

	switch {
	case f.not(f.ID, e.State.ID):
		return false
	case f.not(f.LID, f.gID(e.Lab)):
		return false
	case f.notl(scan, eScan):
		return false
	case f.not(f.File, e.File):
		return false
	case f.not(f.File, e.File):
		return false
	case f.StatusUndev && !e.Lab.None():
		return false
	case f.StatusDev && (e.Lab.None() || e.LabOutDate == (time.Time{})):
		return false
	case f.StatusLab && (e.Lab.None() || e.LabOutDate != (time.Time{})):
		return false
	case f.StatusScanned && e.Scan == 0:
		return false
	case f.StatusUnscanned && e.Scan != 0:
		return false
	case f.StatusLoaded && !e.State.Loaded:
		return false
	case f.StatusUnloaded && e.State.Loaded:
		return false
	case ndate(f.Since, e.LoadDate, e.LoadDate.Before):
		return false
	case ndate(f.Until, e.LoadDate, e.LoadDate.After):
		return false
	case ndate(f.SinceLabIn, e.LabInDate, e.LabInDate.Before):
		return false
	case ndate(f.UntilLabIn, e.LabInDate, e.LabInDate.After):
		return false
	case ndate(f.SinceLabOut, e.LabOutDate, e.LabOutDate.Before):
		return false
	case ndate(f.UntilLabOut, e.LabOutDate, e.LabOutDate.After):
		return false
	case f.notinrange(f.EI, ei, ei):
		return false
	}

	return true
}
