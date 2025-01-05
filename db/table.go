package db

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/frizinak/film-rolls/table"
)

type Sort uint8

const (
	SortDefault Sort = iota
	SortScan
)

type FieldFunc func(header, start, row bool) string

type TableConfig struct {
	Filter Filter

	Notes  bool
	Labels bool
	Short  bool
	Color  bool
	Pretty bool
	Zebra  bool

	Sort Sort

	Header    bool
	HeaderSep bool

	Separator     string
	SeparatorFunc FieldFunc
	Escape        func(string) string

	Width int

	StockTotals bool
}

var defaultConf = TableConfig{
	Escape: func(s string) string { return s },
	SeparatorFunc: func(header, start, row bool) string {
		if row || start {
			return ""
		}

		return " \u2502 "
	},
}

func TableConfigDefault() TableConfig { return defaultConf }

const mdHeaderSep = ":---"

func mdHeaderRightSep(headerSep string) string {
	if headerSep == mdHeaderSep {
		return "---:"
	}

	return headerSep
}

func rowStock(
	header, start, row bool,
	t *table.Table,
	conf TableConfig,
	clrReset func() string,
	stockID, stockName, stockFormat, stockType, stockISO, stockCompany string,
) {
	clr := func(seq string) string {
		if conf.Color {
			return seq
		}
		return ""
	}

	if clrReset == nil {
		clrReset = func() string {
			return clr("\033[0m")
		}
	}

	space := table.ColFixed(table.TermStr(" "))
	sep := func(header, start, row bool) {
		if sep := conf.SeparatorFunc(header, start, row); sep != "" {
			t.AddCol(table.ColFixed(table.TermStr(sep)))
		}
	}
	sepSpace := func(header, start, row bool) {
		if conf.Pretty {
			if start {
				t.AddCol(space)
			}
			return
		}
		if sep := conf.SeparatorFunc(header, start, row); sep != "" {
			t.AddCol(table.ColFixed(table.TermStr(sep)))
		}
	}

	sep(header, true, start && row)
	t.AddCol(table.ColFixed(table.ColPreSuf(
		table.TermStr(stockID),
		clr("\033[38;5;244m"),
		clrReset(),
	)))

	if !conf.Short {
		sepSpace(header, false, false)

		sepSpace(header, true, false)
		t.AddCol(table.ColAlignRight(table.ColFixed(table.TermStr(mdHeaderRightSep(stockFormat)))))
		sepSpace(header, false, false)

		sepSpace(header, true, false)
		t.AddCol(table.ColAlignRight(table.ColFixed(table.TermStr(mdHeaderRightSep(stockType)))))
		sepSpace(header, false, false)

		sepSpace(header, true, false)
		t.AddCol(table.ColFixed(table.ColPreSuf(
			table.TermStr(stockCompany),
			clr("\033[32m"),
			clrReset(),
		)))
		sepSpace(header, false, false)

		sepSpace(header, true, false)
		t.AddCol(table.ColFixed(table.ColPreSuf(
			table.TermStr(stockName),
			clr("\033[32m"),
			clrReset(),
		)))
		sepSpace(header, false, false)

		sepSpace(header, true, false)
		t.AddCol(table.ColAlignRight(table.ColFixed(table.TermStr(mdHeaderRightSep(stockISO)))))
	}

	sep(header, false, !start && row)
}

func (db *DB) PrintRolls(w io.Writer, conf TableConfig) {
	t := table.New()

	zebra := true
	bgclr := func(prefix ...string) string {
		if !conf.Zebra || !conf.Color {
			return ""
		}
		z := "\033[48;5;235m\033[38;5;15m"
		if zebra {
			z = "\033[48;5;237m\033[38;5;15m"
		}
		if len(prefix) == 0 {
			return z
		}
		return z + strings.Join(prefix, "")
	}

	clr := func(seq string) string {
		if conf.Color {
			return seq
		}
		return ""
	}

	clrReset := func() string {
		return clr("\033[0m") + bgclr()
	}

	space := table.ColFixed(table.TermStr(" "))
	sep := func(header, start, row bool) {
		if sep := conf.SeparatorFunc(header, start, row); sep != "" {
			t.AddCol(table.ColFixed(table.TermStr(sep)))
		}
	}
	sepSpace := func(header, start, row bool) {
		if conf.Pretty {
			if start {
				t.AddCol(space)
			}
			return
		}
		if sep := conf.SeparatorFunc(header, start, row); sep != "" {
			t.AddCol(table.ColFixed(table.TermStr(sep)))
		}
	}

	row := func(
		header bool,
		loaded bool,
		loadedString,
		id,
		date,
		cameraID, cameraBrand, cameraModel,
		stockID, stockName, stockFormat, stockType, ISO, stockCompany,
		labID, labName, labInDate, labOutDate,
		file, scan, linenr,
		note string,
	) {
		t.NewRow()

		t.AddCol(table.ColFixed(table.Str(bgclr(" "))))
		sep(header, true, true)

		t.AddCol(table.ColFixed(table.TermStr(date)))
		sep(header, false, false)

		sep(header, true, false)
		t.AddCol(table.ColFixed(table.TermStr(id)))
		sep(header, false, false)

		sep(header, true, false)
		cidClr := "\033[38;5;244m"
		if conf.Short && loaded {
			cidClr = "\033[31m"
		}
		t.AddCol(table.ColFixed(table.ColPreSuf(
			table.TermStr(cameraID),
			clr(cidClr),
			clrReset(),
		)))

		if !conf.Short {
			camPrefix, camSuffix := "", ""
			if loaded && conf.Color {
				camPrefix = "\033[31m"
				camSuffix = clrReset()
			}
			sepSpace(header, false, false)

			sepSpace(header, true, false)
			t.AddCol(table.ColFixed(table.ColPreSuf(table.TermStr(cameraBrand), camPrefix, camSuffix)))
			sepSpace(header, false, false)

			sepSpace(header, true, false)
			t.AddCol(table.ColFixed(table.ColPreSuf(table.TermStr(cameraModel), camPrefix, camSuffix)))
		}
		sep(header, false, false)

		if !conf.Color {
			sep(header, true, false)
			t.AddCol(table.ColFixed(table.TermStr(loadedString)))
			sep(header, false, false)
		}

		rowStock(
			header, false, false,
			t,
			conf,
			clrReset,
			stockID, stockName, stockFormat, stockType, ISO, stockCompany,
		)

		sep(header, true, false)
		t.AddCol(table.ColFixed(table.ColPreSuf(
			table.TermStr(labID),
			clr("\033[38;5;244m"),
			clrReset(),
		)))

		if !conf.Short {
			sepSpace(header, false, false)

			sepSpace(header, true, false)
			t.AddCol(table.ColFixed(table.TermStr(labName)))
			sepSpace(header, false, false)

			sepSpace(header, true, false)
			t.AddCol(table.ColFixed(table.TermStr(labInDate)))
			sepSpace(header, false, false)

			sepSpace(header, true, false)
			t.AddCol(table.ColFixed(table.TermStr(labOutDate)))
		}
		sep(header, false, false)

		sep(header, true, false)
		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(mdHeaderRightSep(file)))))
		sep(header, false, false)

		sep(header, true, false)
		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(mdHeaderRightSep(scan)))))
		sep(header, false, false)

		sep(header, true, false)
		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(mdHeaderRightSep(linenr)))))
		sep(header, false, !conf.Notes && !conf.Labels)

		if conf.Notes || conf.Labels {
			sep(header, true, false)
			t.AddCol(table.TermStr(note))
			sep(header, false, true)
		}

		t.AddCol(table.ColFixed(table.Str(clr(" \033[0m"))))
	}

	if conf.Header {
		row(
			true,
			false,
			"Loaded",
			"ID",
			"Date",
			"CID", "Brand", "Model",
			"SID", "Stock", "Format", "Type", "EI", "Manufacturer",
			"LID", "Lab Name", "Lab in", "Lab out",
			"File", "Scan", "Line",
			"Notes",
		)
	}

	if conf.HeaderSep {
		hs := mdHeaderSep
		row(
			true,
			false,
			hs,
			hs,
			hs,
			hs, hs, hs,
			hs, hs, hs, hs, hs, hs,
			hs, hs, hs, hs,
			hs, hs, hs,
			hs,
		)
	}

	rows := func(row func(e Entry)) {
		db.Row(conf.Filter, row)
	}

	if conf.Sort == SortScan {
		rows = func(row func(e Entry)) {
			list := make(Entries, 0, len(db.Entries))
			db.Row(conf.Filter, func(e Entry) {
				list = append(list, e)
			})

			slices.SortFunc(list, func(a, b Entry) int {
				return cmp.Compare(a.Scan, b.Scan)
			})

			for _, e := range list {
				row(e)
			}
		}
	}

	rows(func(e Entry) {
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
		if e.State.Loaded {
			loadedString = "loaded"
		}

		var notes []string
		{
			var list []string
			if conf.Notes {
				list = e.Note
			}

			var labels []string
			if conf.Labels {
				labels = e.Labels.Values()
			}

			notes = make([]string, len(list)+len(labels))
			copy(notes, list)
			copy(notes[len(list):], labels)
		}

		var note1 string
		if len(notes) != 0 {
			note1 = notes[0]
		}

		zebra = !zebra
		esc := conf.Escape
		row(
			false,
			e.State.Loaded,
			loadedString,
			esc(e.State.ID),
			e.LoadDate.Format(dateFormat),
			esc(e.Camera.ID.String()), esc(e.Camera.Brand), esc(e.Camera.Model),
			esc(e.Stock.ID.String()), esc(e.Stock.Name), esc(e.Stock.Format), esc(e.Stock.Type.String()), esc(e.EIString()), esc(e.Stock.Company.Name),
			esc(labID), esc(labName), esc(labInDate), esc(labOutDate),
			esc(e.File), esc(scan), fmt.Sprintf("%d", e.Line),
			esc(note1),
		)

		if len(notes) > 1 {
			for _, note := range notes[1:] {
				row(
					false,
					false,
					"",
					"",
					"",
					"", "", "",
					"", "", "", "", "", "",
					"", "", "", "",
					"", "", "",
					esc(note),
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
	db.Row(filter, func(e Entry) {
		list = list[:0]
		list = append(list, fmt.Sprintf("id:%s", e.State.ID))
		list = append(list, fmt.Sprintf("camera:%s-%s", clean(e.Camera.Brand), clean(e.Camera.Model)))
		list = append(list, fmt.Sprintf("film:%s-%s", clean(e.Stock.Company.Name), clean(e.Stock.Name)))
		list = append(list, fmt.Sprintf("iso:%s", clean(e.ISOString())))
		list = append(list, fmt.Sprintf("ei:%s", clean(e.EIString())))
		list = append(list, fmt.Sprintf("format:%s", clean(e.Stock.Format)))
		list = append(list, fmt.Sprintf("type:%s", clean(e.Stock.Type.String())))
		if !e.Lab.None() {
			list = append(list, fmt.Sprintf("lab:%s", clean(e.Lab.Name)))
		}
		if e.Scan != 0 {
			list = append(list, fmt.Sprintf("scan:%04d", e.Scan))
		}

		tags := make(map[string]struct{}, 0)
		for _, t := range e.Labels["tags"] {
			l := strings.FieldsFunc(t, func(r rune) bool {
				return r == ' ' || r == ','
			})
			for _, v := range l {
				tags[v] = struct{}{}
			}
		}

		taglist := make([]string, 0, len(tags))
		for k := range tags {
			taglist = append(taglist, k)
		}
		sort.Strings(taglist)
		list = append(list, taglist...)

		fmt.Fprintln(w, strings.Join(list, " "))
	})
}

func (db *DB) PrintStocks(w io.Writer, conf TableConfig) {
	t := table.New()

	sep := func(header, start, row bool) {
		if sep := conf.SeparatorFunc(header, start, row); sep != "" {
			t.AddCol(table.ColFixed(table.TermStr(sep)))
		}
	}

	row := func(
		header bool,
		available, shot, total,
		stockID, stockName, stockFormat, stockType, stockISO, stockCompany string,
	) {
		t.NewRow()

		sep(header, true, true)

		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(mdHeaderRightSep(available)))))
		sep(header, false, false)

		sep(header, true, false)
		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(mdHeaderRightSep(shot)))))
		sep(header, false, false)

		sep(header, true, false)
		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(mdHeaderRightSep(total)))))
		sep(header, false, false)

		rowStock(
			header, false, true,
			t,
			conf,
			nil,
			stockID, stockName, stockFormat, stockType, stockISO, stockCompany,
		)
	}

	if conf.Header {
		row(true, "Avail", "Shot", "Total", "SID", "Stock", "Format", "Type", "ISO", "Manufacturer")
	}
	if conf.HeaderSep {
		hs := mdHeaderSep
		row(true, hs, hs, hs, hs, hs, hs, hs, hs, hs)
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

		db.Row(Filter{All: true}, func(e Entry) {
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

	esc := conf.Escape
	var totals, amount [3]int
	for _, stock := range sorted {
		if !conf.Filter.MatchStock(stock.Stock) {
			continue
		}

		if _, ok := used[stock.Stock.ID]; !ok && stock.Stock.Rolls == 0 {
			continue
		}

		if conf.Filter.StockAvailable && stock.Rolls == 0 {
			continue
		}

		amount[0] = stock.Rolls
		amount[1] = stock.Stock.Rolls - stock.Rolls
		amount[2] = stock.Stock.Rolls
		for i := range totals {
			totals[i] += amount[i]
		}
		row(
			false,
			strconv.Itoa(amount[0]),
			strconv.Itoa(amount[1]),
			strconv.Itoa(amount[2]),
			esc(stock.Stock.ID.String()),
			esc(stock.Stock.Name),
			esc(stock.Stock.Format),
			esc(stock.Stock.Type.String()),
			esc(stock.Stock.ISO.String()),
			esc(stock.Stock.Company.Name),
		)
	}

	if conf.StockTotals {
		row(
			false,
			strconv.Itoa(totals[0]),
			strconv.Itoa(totals[1]),
			strconv.Itoa(totals[2]),
			"",
			"",
			"",
			"",
			"",
			"",
		)
	}

	if conf.Width != 0 {
		t.SetFixedWidth(conf.Width)
	}
	t.WriteTo(w, "")
}

func (db *DB) PrintCameras(w io.Writer, conf TableConfig) {
	t := table.New()
	space := table.ColFixed(table.TermStr(" "))
	sep := func(header, start, row bool) {
		if sep := conf.SeparatorFunc(header, start, row); sep != "" {
			t.AddCol(table.ColFixed(table.TermStr(sep)))
		}
	}
	sepSpace := func(header, start, row bool) {
		if conf.Pretty {
			if start {
				t.AddCol(space)
			}
			return
		}
		if sep := conf.SeparatorFunc(header, start, row); sep != "" {
			t.AddCol(table.ColFixed(table.TermStr(sep)))
		}
	}

	clr := func(seq string) string {
		if conf.Color {
			return seq
		}
		return ""
	}

	row := func(
		header bool,
		cameraID, cameraBrand, cameraModel, ei,
		stockID, stockName, stockFormat, stockType, stockISO, stockCompany string,
	) {
		t.NewRow()

		sep(header, true, true)
		t.AddCol(table.ColFixed(table.ColPreSuf(
			table.TermStr(cameraID),
			clr("\033[38;5;244m"),
			clr("\033[0m"),
		)))
		if !conf.Short {
			sepSpace(header, false, false)

			sepSpace(header, true, false)
			t.AddCol(table.ColFixed(table.TermStr(cameraBrand)))
			sepSpace(header, false, false)

			sepSpace(header, true, false)
			t.AddCol(table.ColFixed(table.TermStr(cameraModel)))
		}
		sep(header, false, false)

		rowStock(
			header, false, false,
			t,
			conf,
			nil,
			stockID, stockName, stockFormat, stockType, stockISO, stockCompany,
		)

		sep(header, true, false)
		t.AddCol(table.ColFixed(table.TermStr(ei)))
		sep(header, false, true)
	}

	if conf.Header {
		row(true, "CID", "Brand", "Model", "EI", "SID", "Stock", "Format", "Type", "ISO", "Manufacturer")
	}
	if conf.HeaderSep {
		hs := mdHeaderSep
		row(true, hs, hs, hs, hs, hs, hs, hs, hs, hs, hs)
	}

	type c struct {
		*Camera
		Entry  Entry
		loaded time.Time
	}

	cams := make(map[ID]*c, len(db.Cameras))
	for k, v := range db.Cameras {
		cams[k] = &c{Camera: v}
	}

	db.Row(Filter{All: true}, func(e Entry) {
		if !e.State.Loaded {
			return
		}

		c := cams[e.Camera.ID]
		c.Entry = e
		c.loaded = e.LoadDate
	})

	sorted := make([]*c, 0, len(db.Cameras))
	for _, cam := range cams {
		sorted = append(sorted, cam)
	}

	slices.SortFunc(sorted, func(i, j *c) int {
		li, lj := i.Entry.Stock == nil, j.Entry.Stock == nil
		switch {
		case li && !lj:
			return -1
		case !li && lj:
			return 1
		case i.loaded.Before(j.loaded):
			return -1
		case j.loaded.Before(i.loaded):
			return 1
		}

		return cmp.Compare(i.Camera.ID, j.Camera.ID)
	})

	for _, cam := range sorted {
		if !conf.Filter.MatchCamera(cam.Camera) {
			continue
		}
		if conf.Filter.StatusLoaded && cam.Entry.Stock == nil {
			continue
		}
		if conf.Filter.StatusUnloaded && cam.Entry.Stock != nil {
			continue
		}

		ei := cam.Entry.EI()
		eiStr := ""
		if ei != 0 {
			eiStr = strconv.Itoa(ei)
		}

		var stockID, stockName, stockFormat, stockType, stockISO, stockCompany string
		if cam.Entry.Stock != nil {
			stockID = cam.Entry.Stock.ID.String()
			stockName = cam.Entry.Stock.Name
			stockFormat = cam.Entry.Stock.Format
			stockType = cam.Entry.Stock.Type.String()
			stockISO = cam.Entry.Stock.ISO.String()
			stockCompany = cam.Entry.Stock.Company.Name
		}

		row(
			false,
			cam.Camera.ID.String(), cam.Camera.Brand, cam.Camera.Model, eiStr,
			stockID, stockName, stockFormat, stockType, stockISO, stockCompany,
		)
	}

	if conf.Width != 0 {
		t.SetFixedWidth(conf.Width)
	}
	t.WriteTo(w, "")
}

func (db *DB) PrintPrices(w io.Writer, conf TableConfig) {
	t := table.New()
	sep := func(header, start, row bool) {
		if sep := conf.SeparatorFunc(header, start, row); sep != "" {
			t.AddCol(table.ColFixed(table.TermStr(sep)))
		}
	}

	row := func(
		header bool,
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

		sep(header, true, true)

		t.AddCol(table.ColFixed(table.ColPreSuf(
			table.ColAlignRight(table.TermStr(mdHeaderRightSep(perUnit))),
			pricePrefix,
			priceSuffix,
		)))
		sep(header, false, false)

		sep(header, true, false)
		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(mdHeaderRightSep(price)))))
		sep(header, false, false)

		sep(header, true, false)
		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(mdHeaderRightSep(amount)))))
		sep(header, false, false)

		rowStock(header, false, false,
			t,
			conf,
			nil,
			stockID, stockName, stockFormat, stockType, stockISO, stockCompany,
		)

		sep(header, true, false)
		t.AddCol(table.ColFixed(table.TermStr(store)))
		sep(header, false, conf.Color)

		if !conf.Color {
			sep(header, true, false)
			t.AddCol(table.ColFixed(table.TermStr(bestPriceString)))
			sep(header, false, true)
		}
	}

	if conf.Header {
		row(
			true,
			false, "Best Price",
			"Price", "Total", "Amount",
			"Store",
			"SID", "Stock", "Format", "Type", "ISO", "Manufacturer",
		)
	}
	if conf.HeaderSep {
		hs := mdHeaderSep
		row(
			true,
			false,
			hs,
			hs, hs, hs,
			hs,
			hs, hs, hs, hs, hs, hs,
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
		if !conf.Filter.MatchStock(p.Stock) {
			continue
		}

		s := db.Stores[p.StoreID]
		best := cheapest[p.ID] == p.PerUnit
		bestPriceString := "no"
		if best {
			bestPriceString = "yes"
		}
		row(
			false,
			best, bestPriceString,
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
