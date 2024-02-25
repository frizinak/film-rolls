package db

import (
	"bytes"
	"cmp"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/frizinak/film-rolls/table"
)

type Company struct {
	ID   ID
	Name string
}

func (c *Company) String() string {
	return fmt.Sprintf("[%s] %s", c.ID, c.Name)
}

func (c *Company) Short() string {
	return c.Name
}

type Stock struct {
	ID      ID
	Company *Company
	Name    string
	ISO     ISO
	Rolls   int
}

func (s *Stock) String() string {
	return fmt.Sprintf("[%s] %s - %s %s", s.ID, s.Company.Short(), s.Name, s.ISO)
}

func (s *Stock) Short() string {
	return fmt.Sprintf("%s - %s %s", s.Company.Short(), s.Name, s.ISO)
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

func LabNone() *Lab { return &Lab{ID0(), ""} }

func (l *Lab) String() string {
	if l.None() {
		return "[N/A]"
	}

	return fmt.Sprintf("[%s] %s", l.ID, l.Name)
}

func (l *Lab) None() bool {
	return l == nil || l.ID == ID0()
}

type ID [3]byte

func ID0() ID { return ID{0, 0, 0} }

func (id ID) String() string { return fmt.Sprintf("[%s]", id[:]) }

type Camera struct {
	ID    ID
	Brand string
	Model string
}

func (c *Camera) String() string {
	return fmt.Sprintf("[%s] %s %s", c.ID, c.Brand, c.Model)
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

	Scan uint

	Line uint

	Note string
}

func (e Entry) ID(i int) string {
	h := sha512.New()
	fmt.Fprintln(h, e.LoadDate.Format(dateFormat))
	fmt.Fprintln(h, e.Camera.ID)
	fmt.Fprintln(h, e.Stock.ID)
	if i != 0 {
		fmt.Fprintln(h, i)
	}

	b := h.Sum(nil)
	return hex.EncodeToString(b)
}

func MkID(str string) (id ID, err error) {
	if len(str) != 3 {
		err = fmt.Errorf("invalid id: '%s'", str)
		return
	}

	for i := range str {
		id[i] = str[i]
	}

	return
}

type DB struct {
	Entries []Entry

	Companies map[ID]*Company
	Stocks    map[ID]*Stock
	Cameras   map[ID]*Camera
	Labs      map[ID]*Lab
}

func (db *DB) row(idFilter string, row func(e Entry, id string, active bool)) {
	ids := make(map[string]struct{})
	loaded := make(map[ID]int)
	for i, e := range db.Entries {
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

		if idFilter != "" && id != idFilter {
			continue
		}

		row(e, id, loaded[e.Camera.ID] == i)
	}
}

type TableConfig struct {
	IDFilter string

	Color  bool
	Pretty bool

	Header                bool
	HeaderSep             bool
	Separator             string
	StartEndWithSeperator bool

	Width int
}

var defaultConf = TableConfig{
	Separator: " \u2502 ",
}

func TableConfigDefault() TableConfig { return defaultConf }

func (db *DB) PrintTable(w io.Writer, conf TableConfig) {
	t := table.New()
	space := table.TermStr(" ")
	line := table.TermStr(conf.Separator)
	lline := table.TermStr(strings.TrimLeft(conf.Separator, " "))
	rline := table.TermStr(strings.TrimRight(conf.Separator, " "))

	if !conf.Pretty {
		space = line
	}

	clr := func(seq string) string {
		if conf.Color {
			return seq
		}
		return ""
	}

	row := func(
		active bool,
		activeString,
		id,
		date,
		cameraID, cameraBrand, cameraModel,
		stockID, stockName, stockISO, stockCompany,
		labID, labName, labInDate, labOutDate,
		scan, note, linenr string,
	) {
		t.NewRow()
		if conf.StartEndWithSeperator {
			t.AddCol(table.ColFixed(lline))
		}

		t.AddCol(table.ColFixed(table.TermStr(date)))
		t.AddCol(table.ColFixed(line))

		t.AddCol(table.ColFixed(table.TermStr(id)))
		t.AddCol(table.ColFixed(line))

		t.AddCol(table.ColFixed(table.ColPreSuf(
			table.TermStr(cameraID),
			clr("\033[38;5;244m"),
			clr("\033[0m"),
		)))

		t.AddCol(table.ColFixed(space))
		camPrefix, camSuffix := "", ""
		if active && conf.Color {
			camPrefix = "\033[31m"
			camSuffix = "\033[0m"
		}

		t.AddCol(table.ColFixed(table.ColPreSuf(table.TermStr(cameraBrand), camPrefix, camSuffix)))
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColFixed(table.ColPreSuf(table.TermStr(cameraModel), camPrefix, camSuffix)))
		t.AddCol(table.ColFixed(line))

		if !conf.Color {
			t.AddCol(table.ColFixed(table.TermStr(activeString)))
			t.AddCol(table.ColFixed(line))
		}

		t.AddCol(table.ColFixed(table.ColPreSuf(
			table.TermStr(stockID),
			clr("\033[38;5;244m"),
			clr("\033[0m"),
		)))
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColFixed(table.ColPreSuf(
			table.TermStr(stockCompany),
			clr("\033[32m"),
			clr("\033[0m"),
		)))
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColFixed(table.ColPreSuf(
			table.TermStr(stockName),
			clr("\033[32m"),
			clr("\033[0m"),
		)))
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColAlignRight(table.ColFixed(table.TermStr(stockISO))))
		t.AddCol(table.ColFixed(line))

		t.AddCol(table.ColFixed(table.ColPreSuf(
			table.TermStr(labID),
			clr("\033[38;5;244m"),
			clr("\033[0m"),
		)))
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColFixed(table.TermStr(labName)))
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColFixed(table.TermStr(labInDate)))
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColFixed(table.TermStr(labOutDate)))
		t.AddCol(table.ColFixed(line))

		t.AddCol(table.ColFixed(table.TermStr(scan)))
		t.AddCol(table.ColFixed(line))
		t.AddCol(table.ColFixed(table.TermStr(linenr)))
		t.AddCol(table.ColFixed(line))
		t.AddCol(table.TermStr(note))

		if conf.StartEndWithSeperator {
			t.AddCol(table.ColFixed(rline))
		}
	}

	if conf.Header {
		row(
			false,
			"Active",
			"ID",
			"Date",
			"[CID]", "Brand", "Model",
			"[SID]", "Stock", "ISO", "Manufacturer",
			"[LID]", "Lab Name", "Lab in", "Lab out",
			"Scan", "Note", "Line",
		)
	}

	if conf.HeaderSep {
		hs := ":---"
		row(
			false,
			hs,
			hs,
			hs,
			hs, hs, hs,
			hs, hs, hs, hs,
			hs, hs, hs, hs,
			hs, hs, hs,
		)
	}

	db.row(conf.IDFilter, func(e Entry, id string, active bool) {
		var labName, labInDate, labOutDate string
		labID := "[N/A]"
		if !e.Lab.None() {
			labID = e.Lab.ID.String()
			labName = e.Lab.Name
			if e.LabInDate != (time.Time{}) {
				labInDate = e.LabInDate.Format(dateFormat)
			}
			if e.LabOutDate != (time.Time{}) {
				labOutDate = e.LabOutDate.Format(dateFormat)
			}
		}
		scan := ""
		if e.Scan != 0 {
			scan = fmt.Sprintf("%04d", e.Scan)
		}
		activeString := " "
		if active {
			activeString = "loaded"
		}
		row(
			active,
			activeString,
			id,
			e.LoadDate.Format(dateFormat),
			e.Camera.ID.String(), e.Camera.Brand, e.Camera.Model,
			e.Stock.ID.String(), e.Stock.Name, e.Stock.ISO.String(), e.Stock.Company.Name,
			labID, labName, labInDate, labOutDate,
			scan, e.Note, fmt.Sprintf("%d", e.Line),
		)
	})
	if conf.Width != 0 {
		t.SetFixedWidth(conf.Width)
	}
	t.WriteTo(w, "")
}

func (db *DB) PrintTags(w io.Writer, idFilter string) {
	r := strings.NewReplacer(" ", "_")
	clean := func(str string) string {
		return strings.ToLower(r.Replace(str))
	}

	list := make([]string, 0, 6)
	db.row(idFilter, func(e Entry, id string, active bool) {
		list = list[:0]
		list = append(list, fmt.Sprintf("id:%s", id))
		list = append(list, fmt.Sprintf("camera:%s-%s", clean(e.Camera.Brand), clean(e.Camera.Model)))
		list = append(list, fmt.Sprintf("film:%s-%s", clean(e.Stock.Company.Name), clean(e.Stock.Name)))
		list = append(list, fmt.Sprintf("iso:%s", clean(e.Stock.ISO.String())))
		if !e.Lab.None() {
			list = append(list, fmt.Sprintf("lab:%s", clean(e.Lab.Name)))
		}
		if e.Scan != 0 {
			list = append(list, fmt.Sprintf("scan:%04d", e.Scan))
		}
		list = append(list, fmt.Sprintf("line:%d", e.Line))

		fmt.Fprintln(w, strings.Join(list, " "))
	})
}

func (db *DB) PrintStock(w io.Writer, conf TableConfig) {
	t := table.New()
	space := table.TermStr(" ")
	line := table.TermStr(conf.Separator)
	lline := table.TermStr(strings.TrimLeft(conf.Separator, " "))
	rline := table.TermStr(strings.TrimRight(conf.Separator, " "))

	if !conf.Pretty {
		space = line
	}

	row := func(
		available, shot, total,
		stockID, stockName, stockISO, stockCompany,
		camera string,
	) {
		t.NewRow()

		if conf.StartEndWithSeperator {
			t.AddCol(table.ColFixed(lline))
		}

		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(available))))
		t.AddCol(table.ColFixed(line))

		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(shot))))
		t.AddCol(table.ColFixed(line))

		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(total))))
		t.AddCol(table.ColFixed(line))

		t.AddCol(table.ColFixed(table.TermStr(stockID)))
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColFixed(table.TermStr(stockCompany)))
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColFixed(table.TermStr(stockName)))
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColFixed(table.TermStr(stockISO)))
		t.AddCol(table.ColFixed(line))
		t.AddCol(table.ColFixed(table.TermStr(camera)))

		if conf.StartEndWithSeperator {
			t.AddCol(table.ColFixed(rline))
		}
	}

	if conf.Header {
		row("Avail", "Shot", "Total", "SID", "Stock", "ISO", "Manufacturer", "Camera")
	}
	if conf.HeaderSep {
		hs := ":---"
		hsr := "---:"
		row(hsr, hsr, hsr, hs, hs, hs, hs, hs)
	}

	type s struct {
		*Stock
		*Camera
		Rolls int
	}

	sorted := make([]*s, 0, len(db.Stocks))
	{
		l := make(map[ID]*s, len(db.Stocks))
		for id, stock := range db.Stocks {
			l[id] = &s{stock, nil, stock.Rolls}
		}

		db.row("", func(e Entry, id string, active bool) {
			l[e.Stock.ID].Rolls--
			if active {
				l[e.Stock.ID].Camera = e.Camera
			}
		})

		for _, stock := range l {
			sorted = append(sorted, stock)
		}

		slices.SortFunc(sorted, func(i, j *s) int {
			return cmp.Compare(i.Name, j.Name)
		})
	}

	for _, stock := range sorted {
		var cam string
		if stock.Camera != nil {
			cam = fmt.Sprintf("%s %s %s", stock.Camera.ID.String(), stock.Camera.Brand, stock.Camera.Model)
		}
		row(
			strconv.Itoa(stock.Rolls),
			strconv.Itoa(stock.Stock.Rolls-stock.Rolls),
			strconv.Itoa(stock.Stock.Rolls),
			stock.Stock.ID.String(),
			stock.Stock.Name,
			stock.Stock.ISO.String(),
			stock.Stock.Company.Name,
			cam,
		)
	}

	if conf.Width != 0 {
		t.SetFixedWidth(conf.Width)
	}
	t.WriteTo(w, "")
}

func (db *DB) String() string {
	buf := bytes.NewBuffer(nil)
	db.PrintTable(buf, defaultConf)
	return buf.String()
}
