package db

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
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

func (l *Lab) String() string {
	return fmt.Sprintf("[%s] %s", l.ID, l.Name)
}

type ID [3]byte

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

	Note string
}

func (e Entry) ID() string {
	h := sha512.New()
	fmt.Fprintln(h, e.LoadDate.Format(dateFormat))
	fmt.Fprintln(h, e.Camera.ID)
	fmt.Fprintln(h, e.Stock.ID)

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

func (db *DB) PrintHTMLTable(w io.Writer) {
	t := table.New()
	ids := make(map[string]struct{})
	loaded := make(map[ID]int)
	tr := table.ColFixed(table.Str("<tr>"))
	tre := table.ColFixed(table.Str("</tr>"))
	td := table.ColFixed(table.Str("<td>"))
	tde := table.ColFixed(table.Str("</td>"))
	addCol := func(col table.Col) {
		t.AddCol(td)
		t.AddCol(col)
		t.AddCol(tde)
	}

	for i, e := range db.Entries {
		if e.Lab == nil {
			loaded[e.Camera.ID] = i
		}
	}

	for i, e := range db.Entries {
		t.NewRow()
		t.AddCol(tr)

		addCol(table.ColFixed(table.Str(e.LoadDate.Format(dateFormat))))

		id := e.ID()
		var idstr string
		n := 4
		for {
			idstr = id[0:n]
			if _, ok := ids[idstr]; !ok {
				break
			}
			n++
		}
		ids[idstr] = struct{}{}
		addCol(table.ColFixed(table.Str(idstr)))

		addCol(table.ColFixed(table.Str(e.Camera.ID.String())))
		camPrefix, camSuffix := "", ""
		if loaded[e.Camera.ID] == i {
			camPrefix = `<span class="active">`
			camSuffix = `</span>`
		}
		addCol(table.ColFixed(table.ColPreSuf(table.Str(e.Camera.Brand), camPrefix, camSuffix)))
		addCol(table.ColFixed(table.ColPreSuf(table.Str(e.Camera.Model), camPrefix, camSuffix)))

		addCol(table.ColFixed(table.Str(e.Stock.ID.String())))
		addCol(table.ColFixed(table.Str(e.Stock.Company.Name)))
		addCol(table.ColFixed(table.Str(e.Stock.Name)))
		addCol(table.ColFixed(table.Str(e.Stock.ISO.String())))

		var labName, labInDate, labOutDate string
		labID := "[N/A]"
		if e.Lab != nil {
			labID = e.Lab.ID.String()
			labName = e.Lab.Name
			if e.LabInDate != (time.Time{}) {
				labInDate = e.LabInDate.Format(dateFormat)
			}
			if e.LabOutDate != (time.Time{}) {
				labOutDate = e.LabOutDate.Format(dateFormat)
			}
		}
		addCol(table.ColFixed(table.Str(labID)))
		addCol(table.ColFixed(table.Str(labName)))
		addCol(table.ColFixed(table.Str(labInDate)))
		addCol(table.ColFixed(table.Str(labOutDate)))

		scan := ""
		if e.Scan != 0 {
			scan = fmt.Sprintf("%04d", e.Scan)
		}
		addCol(table.ColFixed(table.TermStr(scan)))
		addCol(table.ColFixed(table.Str(e.Note)))

		t.AddCol(tre)
	}

	t.WriteTo(w, "")
}

func (db *DB) PrintTable(w io.Writer, width int) {
	t := table.New()
	space := table.TermStr(" ")
	line := table.TermStr(" \u2502 ")
	ids := make(map[string]struct{})
	loaded := make(map[ID]int)

	for i, e := range db.Entries {
		if e.Lab == nil {
			loaded[e.Camera.ID] = i
		}
	}

	for i, e := range db.Entries {
		t.NewRow()
		t.AddCol(table.ColFixed(table.TermStr(e.LoadDate.Format(dateFormat))))
		t.AddCol(table.ColFixed(line))

		id := e.ID()
		var idstr string
		n := 4
		for {
			idstr = id[0:n]
			if _, ok := ids[idstr]; !ok {
				break
			}
			n++
		}
		ids[idstr] = struct{}{}
		t.AddCol(table.ColFixed(table.TermStr(idstr)))
		t.AddCol(table.ColFixed(line))

		t.AddCol(table.ColFixed(table.ColPreSuf(table.TermStr(e.Camera.ID.String()), "\033[38;5;244m", "\033[0m")))
		t.AddCol(table.ColFixed(space))
		camPrefix, camSuffix := "", ""
		if loaded[e.Camera.ID] == i {
			camPrefix = "\033[31m"
			camSuffix = "\033[0m"
		}
		t.AddCol(table.ColFixed(table.ColPreSuf(table.TermStr(e.Camera.Brand), camPrefix, camSuffix)))
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColFixed(table.ColPreSuf(table.TermStr(e.Camera.Model), camPrefix, camSuffix)))
		t.AddCol(table.ColFixed(line))

		t.AddCol(table.ColFixed(table.ColPreSuf(table.TermStr(e.Stock.ID.String()), "\033[38;5;244m", "\033[0m")))
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColFixed(table.ColPreSuf(table.TermStr(e.Stock.Company.Name), "\033[32m", "\033[0m")))
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColFixed(table.ColPreSuf(table.TermStr(e.Stock.Name), "\033[32m", "\033[0m")))
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColAlignRight(table.ColFixed(table.TermStr(e.Stock.ISO.String()))))
		t.AddCol(table.ColFixed(line))

		var labName, labInDate, labOutDate string
		labID := "[N/A]"
		if e.Lab != nil {
			labID = e.Lab.ID.String()
			labName = e.Lab.Name
			if e.LabInDate != (time.Time{}) {
				labInDate = e.LabInDate.Format(dateFormat)
			}
			if e.LabOutDate != (time.Time{}) {
				labOutDate = e.LabOutDate.Format(dateFormat)
			}
		}
		t.AddCol(table.ColFixed(table.ColPreSuf(table.TermStr(labID), "\033[38;5;244m", "\033[0m")))
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColFixed(table.TermStr(labName)))
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColFixed(table.TermStr(labInDate)))
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColFixed(table.TermStr(labOutDate)))
		t.AddCol(table.ColFixed(line))

		scan := ""
		if e.Scan != 0 {
			scan = fmt.Sprintf("%04d", e.Scan)
		}
		t.AddCol(table.ColFixed(table.TermStr(scan)))
		t.AddCol(table.ColFixed(line))
		t.AddCol(table.TermStr(e.Note))
	}
	if width != 0 {
		t.SetFixedWidth(width)
	}
	t.WriteTo(w, "")
}

func (db *DB) String() string {
	buf := bytes.NewBuffer(nil)
	db.PrintTable(buf, 0)
	return buf.String()
}
