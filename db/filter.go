package db

import (
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

	StockFormat string
	StockColor  bool
	StockBW     bool
	StockNeg    bool
	StockPos    bool
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

func (f Filter) Match(id string, e Entry) bool {
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

	switch {
	case f.not(f.ID, id):
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
	case f.StatusLoaded && !e.Loaded:
		return false
	case f.StatusUnloaded && e.Loaded:
		return false
	}

	return true
}
