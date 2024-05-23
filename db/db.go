package db

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"time"
)

type Store struct {
	ID       ID
	Name     string
	Products []Product
}

type Product struct {
	*Stock
	Amount    int
	Price     float64
	PerUnit   float64
	Precision int
}

type Company struct {
	ID   ID
	Name string
}

func (c *Company) IDString() string {
	if c == nil {
		return ""
	}
	return string(c.ID)
}

func (c *Company) Short() string {
	return c.Name
}

type StockType string

func (s StockType) BlackWhite() bool { return s == BWNegative || s == BWPositive }
func (s StockType) Color() bool      { return s == ColorNegative || s == ColorPositive }
func (s StockType) Neg() bool        { return s == BWNegative || s == ColorNegative }
func (s StockType) Pos() bool        { return s == BWPositive || s == ColorPositive }

func (s StockType) String() string {
	switch s {
	case ColorNegative:
		return "CLR-"
	case ColorPositive:
		return "CLR+"
	case BWNegative:
		return "B/W-"
	case BWPositive:
		return "B/W+"
	}

	return ""
}

const (
	ColorNegative = "C"
	ColorPositive = "CS"
	BWNegative    = "BW"
	BWPositive    = "BWS"
)

type Stock struct {
	ID      ID
	Company *Company
	Name    string
	ISO     ISO
	Format  string
	Type    StockType
	Rolls   int
}

func (s *Stock) IDString() string {
	if s == nil {
		return ""
	}
	return string(s.ID)
}

type ISO struct {
	Low, High uint32
}

func (iso ISO) String() string {
	if iso.Low == iso.High {
		return fmt.Sprintf("%d", iso.Low)
	}
	return fmt.Sprintf("%d-%d", iso.Low, iso.High)
}

type Lab struct {
	ID   ID
	Name string
}

func (l *Lab) IDString() string {
	if l == nil {
		return ""
	}
	return string(l.ID)
}

func LabNone() *Lab { return &Lab{ID0(), ""} }

func (l *Lab) None() bool {
	return l == nil || l.ID == ID0()
}

type ID string

func ID0() ID { return "" }

func (id ID) String() string { return string(id) }

type Camera struct {
	ID    ID
	Brand string
	Model string
}

func (c *Camera) IDString() string {
	if c == nil {
		return ""
	}
	return string(c.ID)
}

func (c *Camera) Short() string {
	return fmt.Sprintf("%s %s", c.Brand, c.Model)
}

type Entry struct {
	LoadDate   time.Time
	LabInDate  time.Time
	LabOutDate time.Time

	Stock  *Stock
	Camera *Camera
	Lab    *Lab

	Loaded bool

	File string
	Scan uint

	Line uint

	Note []string
}

func (e Entry) ID(i int) string {
	h := sha512.New()
	fmt.Fprintf(
		h,
		"%s\n[%s]\n[%s]\n",
		e.LoadDate.Format(dateFormat),
		string(e.Camera.ID),
		string(e.Stock.ID),
	)
	if i != 0 {
		fmt.Fprintln(h, i)
	}

	b := h.Sum(nil)
	return hex.EncodeToString(b)
}

func MkID(str string) (ID, error) {
	b := make([]byte, len(str))
	copy(b, str)
	return ID(b), nil
}

type Entries []Entry

func (e Entries) Len() int      { return len(e) }
func (e Entries) Swap(i, j int) { e[i], e[j] = e[j], e[i] }
func (e Entries) Less(i, j int) bool {
	a, b := e[i], e[j]
	if a.LoadDate.Before(b.LoadDate) {
		return true
	} else if b.LoadDate.Before(a.LoadDate) {
		return false
	}

	if a.File < b.File {
		return true
	} else if a.File > b.File {
		return false
	}

	return a.Line < b.Line
}

type DB struct {
	Entries Entries

	Companies map[ID]*Company
	Stocks    map[ID]*Stock
	Cameras   map[ID]*Camera
	Labs      map[ID]*Lab
	Stores    map[ID]*Store
}

func (db *DB) Row(filter Filter, row func(e Entry, id string)) {
	ids := make(map[string]struct{})
	loaded := make(map[ID]int)
	for i, e := range db.Entries {
		loaded[e.Camera.ID] = -1
		if e.Lab == nil {
			loaded[e.Camera.ID] = i
		}
	}

	for i, e := range db.Entries {
		var id string
		const n = 5
		try := 0
		for {
			id = e.ID(try)[:n]
			if _, ok := ids[id]; !ok {
				break
			}
			try++
		}

		ids[id] = struct{}{}

		e.Loaded = loaded[e.Camera.ID] == i
		if !filter.Match(id, e) {
			continue
		}

		row(e, id)
	}
}

func (db *DB) String() string {
	buf := bytes.NewBuffer(nil)
	db.PrintLogs(buf, defaultConf)
	return buf.String()
}
