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

type Filter struct {
	ID  string
	LID string
	SID string
	CID string

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
		if val == v {
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

	switch {
	case f.ID != "" && f.not(f.ID, id):
		return false
	case f.not(f.LID, f.gID(e.Lab)):
		return false
	case f.not(f.CID, f.gID(e.Camera)):
		return false
	case f.notl(scan, eScan):
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

type DB struct {
	Entries []Entry

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
		loaded bool,
		loadedString,
		id,
		date,
		cameraID, cameraBrand, cameraModel,
		stockID, stockName, stockFormat, stockType, stockISO, stockCompany,
		labID, labName, labInDate, labOutDate,
		scan, linenr,
		note string,
	) {
		t.NewRow()
		if conf.StartEndWithSeparator {
			t.AddCol(table.ColFixed(lline))
		}

		t.AddCol(table.ColFixed(table.TermStr(date)))
		t.AddCol(table.ColFixed(line))

		t.AddCol(table.ColFixed(table.TermStr(id)))
		t.AddCol(table.ColFixed(line))

		cidClr := "\033[38;5;244m"
		if conf.Short && loaded {
			cidClr = "\033[31m"
		}
		t.AddCol(table.ColFixed(table.ColPreSuf(
			table.TermStr(cameraID),
			clr(cidClr),
			clr("\033[0m"),
		)))

		if !conf.Short {
			t.AddCol(table.ColFixed(space))
			camPrefix, camSuffix := "", ""
			if loaded && conf.Color {
				camPrefix = "\033[31m"
				camSuffix = "\033[0m"
			}

			t.AddCol(table.ColFixed(table.ColPreSuf(table.TermStr(cameraBrand), camPrefix, camSuffix)))
			t.AddCol(table.ColFixed(space))
			t.AddCol(table.ColFixed(table.ColPreSuf(table.TermStr(cameraModel), camPrefix, camSuffix)))
		}

		t.AddCol(table.ColFixed(line))

		if !conf.Color {
			t.AddCol(table.ColFixed(table.TermStr(loadedString)))
			t.AddCol(table.ColFixed(line))
		}

		rowStock(t, conf, space, stockID, stockName, stockFormat, stockType, stockISO, stockCompany)
		t.AddCol(table.ColFixed(line))

		t.AddCol(table.ColFixed(table.ColPreSuf(
			table.TermStr(labID),
			clr("\033[38;5;244m"),
			clr("\033[0m"),
		)))

		if !conf.Short {
			t.AddCol(table.ColFixed(space))
			t.AddCol(table.ColFixed(table.TermStr(labName)))
			t.AddCol(table.ColFixed(space))
			t.AddCol(table.ColFixed(table.TermStr(labInDate)))
			t.AddCol(table.ColFixed(space))
			t.AddCol(table.ColFixed(table.TermStr(labOutDate)))
		}

		t.AddCol(table.ColFixed(line))

		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(scan))))
		t.AddCol(table.ColFixed(line))
		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(linenr))))

		if conf.Notes {
			t.AddCol(table.ColFixed(line))
			t.AddCol(table.TermStr(note))
		}

		if conf.StartEndWithSeparator {
			t.AddCol(table.ColFixed(rline))
		}
	}

	if conf.Header {
		row(
			false,
			"Loaded",
			"ID",
			"Date",
			"CID", "Brand", "Model",
			"SID", "Stock", "Format", "Type", "ISO", "Manufacturer",
			"LID", "Lab Name", "Lab in", "Lab out",
			"Scan", "Line",
			"Notes",
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
			hs, hs, hs, hs, hs, hs,
			hs, hs, hs, hs,
			hs, hs,
			hs,
		)
	}

	db.Row(conf.Filter, func(e Entry, id string) {
		var labName, labInDate, labOutDate string
		labID := "N/A"
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
			scan = fmt.Sprintf("%d", e.Scan)
		}
		loadedString := " "
		if e.Loaded {
			loadedString = "loaded"
		}

		var note1 string
		if conf.Notes && len(e.Note) != 0 {
			note1 = e.Note[0]
		}
		row(
			e.Loaded,
			loadedString,
			id,
			e.LoadDate.Format(dateFormat),
			e.Camera.ID.String(), e.Camera.Brand, e.Camera.Model,
			e.Stock.ID.String(), e.Stock.Name, e.Stock.Format, e.Stock.Type.String(), e.Stock.ISO.String(), e.Stock.Company.Name,
			labID, labName, labInDate, labOutDate,
			scan, fmt.Sprintf("%d", e.Line),
			note1,
		)

		if conf.Notes && len(e.Note) > 1 {
			notes := make([]string, len(e.Note))
			copy(notes, e.Note[1:])
			for _, note := range notes {
				row(
					false,
					"",
					"",
					"",
					"", "", "",
					"", "", "", "", "", "",
					"", "", "", "",
					"", "",
					note,
				)
			}
		}
	})
	if conf.Width != 0 {
		t.SetFixedWidth(conf.Width)
	}
	t.WriteTo(w, "")
}

func (db *DB) PrintTags(w io.Writer, filter Filter) {
	r := strings.NewReplacer(" ", "_")
	clean := func(str string) string {
		return strings.ToLower(r.Replace(str))
	}

	list := make([]string, 0, 6)
	db.Row(filter, func(e Entry, id string) {
		list = list[:0]
		list = append(list, fmt.Sprintf("id:%s", id))
		list = append(list, fmt.Sprintf("camera:%s-%s", clean(e.Camera.Brand), clean(e.Camera.Model)))
		list = append(list, fmt.Sprintf("film:%s-%s", clean(e.Stock.Company.Name), clean(e.Stock.Name)))
		list = append(list, fmt.Sprintf("iso:%s", clean(e.Stock.ISO.String())))
		list = append(list, fmt.Sprintf("format:%s", clean(e.Stock.Format)))
		list = append(list, fmt.Sprintf("type:%s", clean(e.Stock.Type.String())))
		if !e.Lab.None() {
			list = append(list, fmt.Sprintf("lab:%s", clean(e.Lab.Name)))
		}
		if e.Scan != 0 {
			list = append(list, fmt.Sprintf("scan:%04d", e.Scan))
		}

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
		stockID, stockName, stockFormat, stockType, stockISO, stockCompany string,
	) {
		t.NewRow()

		if conf.StartEndWithSeparator {
			t.AddCol(table.ColFixed(lline))
		}

		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(available))))
		t.AddCol(table.ColFixed(line))

		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(shot))))
		t.AddCol(table.ColFixed(line))

		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(total))))
		t.AddCol(table.ColFixed(line))

		rowStock(t, conf, space, stockID, stockName, stockFormat, stockType, stockISO, stockCompany)

		if conf.StartEndWithSeparator {
			t.AddCol(table.ColFixed(rline))
		}
	}

	if conf.Header {
		row("Avail", "Shot", "Total", "SID", "Stock", "Format", "Type", "ISO", "Manufacturer")
	}
	if conf.HeaderSep {
		hs := ":---"
		hsr := "---:"
		row(hsr, hsr, hsr, hs, hs, hsr, hsr, hs, hs)
	}

	type s struct {
		*Stock
		Rolls int
	}

	used := make(map[ID]struct{})
	sorted := make([]*s, 0, len(db.Stocks))
	{
		l := make(map[ID]*s, len(db.Stocks))
		for id, stock := range db.Stocks {
			l[id] = &s{stock, stock.Rolls}
		}

		db.Row(Filter{}, func(e Entry, id string) {
			used[e.Stock.ID] = struct{}{}
			l[e.Stock.ID].Rolls--
		})

		for _, stock := range l {
			sorted = append(sorted, stock)
		}

		slices.SortFunc(sorted, func(i, j *s) int {
			if c := cmp.Compare(i.Rolls, j.Rolls); c != 0 {
				return c
			}

			if c := cmp.Compare(i.Stock.Rolls, j.Stock.Rolls); c != 0 {
				return c
			}

			if c := cmp.Compare(i.Name, j.Name); c != 0 {
				return c
			}

			return 0
		})
	}

	for _, stock := range sorted {
		if !conf.Filter.MatchStock(stock.Stock) {
			continue
		}

		if _, ok := used[stock.Stock.ID]; !ok && stock.Stock.Rolls == 0 {
			continue
		}

		row(
			strconv.Itoa(stock.Rolls),
			strconv.Itoa(stock.Stock.Rolls-stock.Rolls),
			strconv.Itoa(stock.Stock.Rolls),
			stock.Stock.ID.String(),
			stock.Stock.Name,
			stock.Stock.Format,
			stock.Stock.Type.String(),
			stock.Stock.ISO.String(),
			stock.Stock.Company.Name,
		)
	}

	if conf.Width != 0 {
		t.SetFixedWidth(conf.Width)
	}
	t.WriteTo(w, "")
}

func (db *DB) PrintPrices(w io.Writer, conf TableConfig) {
	t := table.New()
	space := table.TermStr(" ")
	line := table.TermStr(conf.Separator)
	lline := table.TermStr(strings.TrimLeft(conf.Separator, " "))
	rline := table.TermStr(strings.TrimRight(conf.Separator, " "))

	if !conf.Pretty {
		space = line
	}

	row := func(
		bestPrice bool, bestPriceString string,
		perUnit, price, amount,
		store,
		stockID, stockName, stockFormat, stockType, stockISO, stockCompany string,
	) {
		t.NewRow()

		pricePrefix, priceSuffix := "", ""
		if !bestPrice && conf.Color {
			pricePrefix = "\033[31m"
			priceSuffix = "\033[0m"
		}

		if conf.StartEndWithSeparator {
			t.AddCol(table.ColFixed(lline))
		}

		t.AddCol(table.ColFixed(table.ColPreSuf(
			table.ColAlignRight(table.TermStr(perUnit)),
			pricePrefix,
			priceSuffix,
		)))
		t.AddCol(table.ColFixed(line))

		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(price))))
		t.AddCol(table.ColFixed(line))

		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(amount))))
		t.AddCol(table.ColFixed(line))

		rowStock(t, conf, space, stockID, stockName, stockFormat, stockType, stockISO, stockCompany)
		t.AddCol(table.ColFixed(line))

		t.AddCol(table.ColFixed(table.TermStr(store)))

		if !conf.Color {
			t.AddCol(table.ColFixed(line))
			t.AddCol(table.ColFixed(table.TermStr(bestPriceString)))
		}

		if conf.StartEndWithSeparator {
			t.AddCol(table.ColFixed(rline))
		}
	}

	if conf.Header {
		row(
			false, "Best Price",
			"Price", "Total", "Amount",
			"Store",
			"SID", "Stock", "Format", "Type", "ISO", "Manufacturer",
		)
	}
	if conf.HeaderSep {
		hs := ":---"
		hsr := "---:"
		row(
			false,
			hs,
			hsr, hsr, hsr,
			hs,
			hs, hs, hsr, hsr, hs, hs,
		)
	}

	type product struct {
		StoreID ID
		Product
	}

	cheapest := make(map[ID]float64)
	sorted := make([]*product, 0)
	for _, s := range db.Stores {
		for _, p := range s.Products {
			sorted = append(sorted, &product{StoreID: s.ID, Product: p})
		}
	}

	slices.SortFunc(sorted, func(i, j *product) int {
		c := cmp.Compare(i.PerUnit, j.PerUnit)
		if c != 0 {
			return c
		}

		return cmp.Compare(i.ID, j.ID)
	})

	for _, p := range sorted {
		if cheapest[p.ID] == 0 || p.PerUnit < cheapest[p.ID] {
			cheapest[p.ID] = p.PerUnit
		}
	}

	for _, p := range sorted {
		s := db.Stores[p.StoreID]
		best := cheapest[p.ID] == p.PerUnit
		row(
			best, "yes",
			strconv.FormatFloat(p.PerUnit, 'f', p.Precision, 64),
			strconv.FormatFloat(p.Price, 'f', p.Precision, 64),
			strconv.Itoa(p.Amount),
			fmt.Sprintf("%s %s", s.ID.String(), s.Name),
			p.ID.String(),
			p.Name,
			p.Format,
			p.Type.String(),
			p.ISO.String(),
			p.Company.Name,
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
