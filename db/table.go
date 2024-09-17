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

type TableConfig struct {
	Filter Filter

	Notes  bool
	Labels bool
	Short  bool
	Color  bool
	Pretty bool
	Zebra  bool

	Sort Sort

	Header                bool
	HeaderSep             bool
	Separator             string
	StartEndWithSeparator bool

	Width int
}

var defaultConf = TableConfig{
	Separator: " \u2502 ",
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
	t *table.Table,
	conf TableConfig,
	space table.Col,
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

	t.AddCol(table.ColFixed(table.ColPreSuf(
		table.TermStr(stockID),
		clr("\033[38;5;244m"),
		clrReset(),
	)))

	if !conf.Short {
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColAlignRight(table.ColFixed(table.TermStr(mdHeaderRightSep(stockFormat)))))
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColAlignRight(table.ColFixed(table.TermStr(mdHeaderRightSep(stockType)))))
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColFixed(table.ColPreSuf(
			table.TermStr(stockCompany),
			clr("\033[32m"),
			clrReset(),
		)))
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColFixed(table.ColPreSuf(
			table.TermStr(stockName),
			clr("\033[32m"),
			clrReset(),
		)))
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColAlignRight(table.ColFixed(table.TermStr(mdHeaderRightSep(stockISO)))))
	}
}

func (db *DB) PrintLogs(w io.Writer, conf TableConfig) {
	t := table.New()
	space := table.TermStr(" ")
	line := table.TermStr(conf.Separator)
	lline := table.TermStr(strings.TrimLeft(conf.Separator, " "))
	rline := table.TermStr(strings.TrimRight(conf.Separator, " "))
	if !conf.Pretty {
		space = line
	}

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

	row := func(
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
			clrReset(),
		)))

		if !conf.Short {
			t.AddCol(table.ColFixed(space))
			camPrefix, camSuffix := "", ""
			if loaded && conf.Color {
				camPrefix = "\033[31m"
				camSuffix = clrReset()
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

		rowStock(t, conf, space, clrReset, stockID, stockName, stockFormat, stockType, ISO, stockCompany)
		t.AddCol(table.ColFixed(line))

		t.AddCol(table.ColFixed(table.ColPreSuf(
			table.TermStr(labID),
			clr("\033[38;5;244m"),
			clrReset(),
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

		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(mdHeaderRightSep(file)))))
		t.AddCol(table.ColFixed(line))
		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(mdHeaderRightSep(scan)))))
		t.AddCol(table.ColFixed(line))
		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(mdHeaderRightSep(linenr)))))

		if conf.Notes || conf.Labels {
			t.AddCol(table.ColFixed(line))
			t.AddCol(table.TermStr(note))
		}

		if conf.StartEndWithSeparator {
			t.AddCol(table.ColFixed(rline))
		}

		t.AddCol(table.ColFixed(table.Str(clr(" \033[0m"))))
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
			"File", "Scan", "Line",
			"Notes",
		)
	}

	if conf.HeaderSep {
		hs := mdHeaderSep
		row(
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
		row(
			e.State.Loaded,
			loadedString,
			e.State.ID,
			e.LoadDate.Format(dateFormat),
			e.Camera.ID.String(), e.Camera.Brand, e.Camera.Model,
			e.Stock.ID.String(), e.Stock.Name, e.Stock.Format, e.Stock.Type.String(), e.ISOString(), e.Stock.Company.Name,
			labID, labName, labInDate, labOutDate,
			e.File, scan, fmt.Sprintf("%d", e.Line),
			note1,
		)

		if len(notes) > 1 {
			for _, note := range notes[1:] {
				row(
					false,
					"",
					"",
					"",
					"", "", "",
					"", "", "", "", "", "",
					"", "", "", "",
					"", "", "",
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
	db.Row(filter, func(e Entry) {
		list = list[:0]
		list = append(list, fmt.Sprintf("id:%s", e.State.ID))
		list = append(list, fmt.Sprintf("camera:%s-%s", clean(e.Camera.Brand), clean(e.Camera.Model)))
		list = append(list, fmt.Sprintf("film:%s-%s", clean(e.Stock.Company.Name), clean(e.Stock.Name)))
		list = append(list, fmt.Sprintf("iso:%s", clean(e.ISOString())))
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

		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(mdHeaderRightSep(available)))))
		t.AddCol(table.ColFixed(line))

		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(mdHeaderRightSep(shot)))))
		t.AddCol(table.ColFixed(line))

		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(mdHeaderRightSep(total)))))
		t.AddCol(table.ColFixed(line))

		rowStock(t, conf, space, nil, stockID, stockName, stockFormat, stockType, stockISO, stockCompany)

		if conf.StartEndWithSeparator {
			t.AddCol(table.ColFixed(rline))
		}
	}

	if conf.Header {
		row("Avail", "Shot", "Total", "SID", "Stock", "Format", "Type", "ISO", "Manufacturer")
	}
	if conf.HeaderSep {
		hs := mdHeaderSep
		row(hs, hs, hs, hs, hs, hs, hs, hs, hs)
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

func (db *DB) PrintCameras(w io.Writer, conf TableConfig) {
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
		cameraID, cameraBrand, cameraModel,
		stockID, stockName, stockFormat, stockType, stockISO, stockCompany string,
	) {
		t.NewRow()

		if conf.StartEndWithSeparator {
			t.AddCol(table.ColFixed(lline))
		}

		t.AddCol(table.ColFixed(table.ColPreSuf(
			table.TermStr(cameraID),
			clr("\033[38;5;244m"),
			clr("\033[0m"),
		)))
		if !conf.Short {
			t.AddCol(table.ColFixed(space))
			t.AddCol(table.ColFixed(table.TermStr(cameraBrand)))
			t.AddCol(table.ColFixed(space))
			t.AddCol(table.ColFixed(table.TermStr(cameraModel)))
		}
		t.AddCol(table.ColFixed(line))

		rowStock(t, conf, space, nil, stockID, stockName, stockFormat, stockType, stockISO, stockCompany)

		if conf.StartEndWithSeparator {
			t.AddCol(table.ColFixed(rline))
		}
	}

	if conf.Header {
		row("CID", "Brand", "Model", "SID", "Stock", "Format", "Type", "ISO", "Manufacturer")
	}
	if conf.HeaderSep {
		hs := mdHeaderSep
		row(hs, hs, hs, hs, hs, hs, hs, hs, hs)
	}

	type c struct {
		*Camera
		*Stock
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
		c.Stock = e.Stock
		c.loaded = e.LoadDate
	})

	sorted := make([]*c, 0, len(db.Cameras))
	for _, cam := range cams {
		sorted = append(sorted, cam)
	}

	slices.SortFunc(sorted, func(i, j *c) int {
		li, lj := i.Stock == nil, j.Stock == nil
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
		if conf.Filter.StatusLoaded && cam.Stock == nil {
			continue
		}
		if conf.Filter.StatusUnloaded && cam.Stock != nil {
			continue
		}

		var stockID, stockName, stockFormat, stockType, stockISO, stockCompany string
		if cam.Stock != nil {
			stockID = cam.Stock.ID.String()
			stockName = cam.Stock.Name
			stockFormat = cam.Stock.Format
			stockType = cam.Stock.Type.String()
			stockISO = cam.Stock.ISO.String()
			stockCompany = cam.Stock.Company.Name
		}

		row(
			cam.Camera.ID.String(), cam.Camera.Brand, cam.Camera.Model,
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
			table.ColAlignRight(table.TermStr(mdHeaderRightSep(perUnit))),
			pricePrefix,
			priceSuffix,
		)))
		t.AddCol(table.ColFixed(line))

		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(mdHeaderRightSep(price)))))
		t.AddCol(table.ColFixed(line))

		t.AddCol(table.ColFixed(table.ColAlignRight(table.TermStr(mdHeaderRightSep(amount)))))
		t.AddCol(table.ColFixed(line))

		rowStock(t, conf, space, nil, stockID, stockName, stockFormat, stockType, stockISO, stockCompany)
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
		hs := mdHeaderSep
		row(
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
