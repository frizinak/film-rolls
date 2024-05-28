package db

import (
	"bufio"
	"cmp"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

const dateFormat = "2006-01-02"

func mk() *DB {
	return &DB{
		Entries: make([]Entry, 0),

		Companies: make(map[ID]*Company, 0),
		Stocks:    make(map[ID]*Stock, 0),
		Cameras:   make(map[ID]*Camera, 0),
		Labs:      make(map[ID]*Lab, 0),
		Stores:    make(map[ID]*Store, 0),
	}
}

func ParseDir(dir string) (*DB, error) {
	d, err := os.Open(dir)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, 1)
	for {
		e, err := d.Readdirnames(10)
		for _, n := range e {
			if !strings.HasSuffix(n, ".log") && !strings.HasSuffix(n, ".def") {
				continue
			}
			files = append(files, n)
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}
	}

	slices.SortFunc(files, func(a, b string) int {
		sa, sb := a[len(a)-3:], b[len(b)-3:]
		if c := cmp.Compare(sa, sb); c != 0 {
			return c
		}
		return cmp.Compare(a, b)
	})
	db := mk()
	for _, file := range files {
		f, err := os.Open(filepath.Join(dir, file))
		if err != nil {
			return nil, err
		}
		if err = parse(db, file[:len(file)-4], f); err != nil {
			return db, err
		}
	}

	if db == nil {
		return db, errors.New("no log files found")
	}

	sort.Sort(db.Entries)

	return db, nil
}

func Parse(r io.Reader) (*DB, error) {
	db := mk()
	err := parse(db, "", r)
	sort.Sort(db.Entries)
	return db, err
}

func parse(db *DB, file string, r io.Reader) error {
	var lastID ID
	scans := make(map[uint]struct{})

	const (
		keywordNone    = ""
		keywordCompany = "Company"
		keywordStock   = "Stock"
		keywordCamera  = "Camera"
		keywordLab     = "Lab"
		keywordNote    = "Note"
		keywordStore   = "Store"
	)

	s := bufio.NewScanner(r)
	s.Split(bufio.ScanLines)
	var keyword string
	var line uint
	var noteIndent int
	for s.Scan() {
		line++
		raw := s.Text()
		indent := 0
		for _, c := range raw {
			if c != ' ' {
				break
			}
			indent++
		}
		t := strings.TrimSpace(raw)
		if t == "" {
			keyword = keywordNone
			continue
		}
		if t[0] == '#' {
			continue
		}

		switch keyword {
		case keywordCompany:
			c, ok := db.Companies[lastID]
			if !ok {
				return fmt.Errorf("no company with id %s", lastID)
			}
			c.Name = t
			keyword = keywordNone
			continue
		case keywordStock:
			s, ok := db.Stocks[lastID]
			if !ok {
				return fmt.Errorf("no stock with id %s", lastID)
			}
			if s.Format == "" {
				f := strings.Fields(t)
				s.Format = f[0]
				s.Type = ColorNegative

				if len(f) == 2 {
					tp := StockType(strings.ToUpper(f[1]))
					if tp != ColorNegative && tp != ColorPositive && tp != BWNegative && tp != BWPositive {
						return fmt.Errorf("invalid stock type '%s' on line %d", f[1], line)
					}

					s.Type = tp
				} else if len(f) > 2 {
					return fmt.Errorf("invalid format on line %d", line)
				}
			} else if s.Name == "" {
				s.Name = t
			} else if s.Company == nil {
				cid, err := MkID(t)
				if err != nil {
					return err
				}
				s.Company = db.Companies[cid]
				if s.Company == nil {
					return fmt.Errorf("no company by id '%s'", t)
				}
			} else if s.ISO.Low == 0 {
				p := strings.FieldsFunc(t, func(r rune) bool {
					return r == ' ' || r == '-'
				})
				if len(p) > 2 {
					return fmt.Errorf("invalid ISO line %d: '%s'", line, t)
				}

				for i := range p {
					v, err := strconv.ParseUint(p[i], 10, 32)
					if err != nil {
						return fmt.Errorf("invalid integers in ISO line %d: '%s'", line, t)
					}
					switch i {
					case 0:
						s.ISO.Low = uint32(v)
					case 1:
						s.ISO.High = uint32(v)
					}
				}
				if s.ISO.High == 0 {
					s.ISO.High = s.ISO.Low
				}
				if s.ISO.High < s.ISO.Low {
					return fmt.Errorf("invalid ISO range in line %d: '%s'", line, t)
				}
			} else if s.Rolls == 0 {
				l := strings.FieldsFunc(t, func(r rune) bool {
					return r == ' ' || r == '+'
				})

				n := 0
				for _, s := range l {
					val, err := strconv.Atoi(s)
					if err != nil {
						return fmt.Errorf("invalid number on line %d: %s", line, s)
					}
					n += val
				}

				s.Rolls = n
				keyword = keywordNone
			}

			continue
		case keywordCamera:
			c, ok := db.Cameras[lastID]
			if !ok {
				return fmt.Errorf("no camera with id %s", lastID)
			}

			if c.Brand == "" {
				c.Brand = t
			} else if c.Model == "" {
				c.Model = t
				keyword = keywordNone
			}
			continue
		case keywordLab:
			l, ok := db.Labs[lastID]
			if !ok {
				return fmt.Errorf("no lab with id %s", lastID)
			}
			l.Name = t
			keyword = keywordNone
			continue
		case keywordNote:
			if indent < 2 {
				break
			}

			i := len(db.Entries) - 1
			spaces := indent - noteIndent
			if len(db.Entries[i].Note) == 0 {
				noteIndent = indent
				spaces = 0
			}
			if spaces < 0 {
				spaces = 0
			}

			str := make([]byte, len(t)+spaces)
			{
				i := 0
				for ; i < spaces; i++ {
					str[i] = ' '
				}
				for n := 0; n < len(t); n++ {
					str[i+n] = t[n]
				}
			}

			db.Entries[i].Note = append(db.Entries[i].Note, string(str))
			continue
		case keywordStore:
			s, ok := db.Stores[lastID]
			if !ok {
				return fmt.Errorf("no store with id %s", lastID)
			}

			if s.Name == "" {
				s.Name = t
				continue
			}

			f := strings.Fields(t)
			if len(f) != 3 {
				return fmt.Errorf("invalid product on line %d: %s", line, t)
			}

			sid, err := MkID(f[0])
			if err != nil {
				return err
			}
			stock, ok := db.Stocks[sid]
			if !ok {
				return fmt.Errorf("no stock with id %s", sid)
			}

			amount, err := strconv.Atoi(f[1])
			if err != nil {
				return fmt.Errorf("invalid amount on line %d", line)
			}

			price, err := strconv.ParseFloat(f[2], 64)
			if err != nil {
				return fmt.Errorf("invalid price on line %d", line)
			}

			precision := 2
			ix := strings.Index(f[2], ".")
			if ix != -1 {
				precision = len(f[2]) - ix - 1
			}

			s.Products = append(
				s.Products,
				Product{
					Stock:     stock,
					Amount:    amount,
					Price:     price,
					PerUnit:   price / float64(amount),
					Precision: precision,
				},
			)

			continue
		}

		p := strings.Fields(t)

		// UTC!
		if d, err := time.Parse(dateFormat, p[0]); err == nil {
			e, err := db.mkEntry(d, p, scans)
			if err != nil {
				return fmt.Errorf("%w: line %d: '%s'", err, line, t)
			}

			e.File = file
			e.Line = line
			db.Entries = append(db.Entries, e)
			keyword = keywordNote
			noteIndent = 0
			continue
		}

		if len(p) != 2 {
			return fmt.Errorf("invalid line %d: '%s'", line, t)
		}

		keyword = p[0]
		id, err := MkID(p[1])
		if err != nil {
			return err
		}

		lastID = id
		switch keyword {
		case keywordCompany:
			if _, ok := db.Companies[id]; ok {
				return fmt.Errorf("duplicate company id '%s'", id.String())
			}
			db.Companies[id] = &Company{ID: id}
		case keywordStock:
			if _, ok := db.Stocks[id]; ok {
				return fmt.Errorf("duplicate stock id '%s'", id.String())
			}
			db.Stocks[id] = &Stock{ID: id}
		case keywordCamera:
			if _, ok := db.Cameras[id]; ok {
				return fmt.Errorf("duplicate camera id '%s'", id.String())
			}
			db.Cameras[id] = &Camera{ID: id}
		case keywordLab:
			if _, ok := db.Labs[id]; ok {
				return fmt.Errorf("duplicate lab id '%s'", id.String())
			}
			db.Labs[id] = &Lab{ID: id}
		case keywordStore:
			if _, ok := db.Stores[id]; ok {
				return fmt.Errorf("duplicate store id '%s'", id.String())
			}
			db.Stores[id] = &Store{ID: id}
		default:
			return fmt.Errorf("invalid keyword: '%s'", keyword)
		}
	}

	return s.Err()
}

func (db *DB) mkEntry(d time.Time, p []string, scans map[uint]struct{}) (Entry, error) {
	e := Entry{LoadDate: d}
	if len(p) < 3 {
		return e, errors.New("invalid entry")
	}
	sid, err := MkID(p[1])
	if err != nil {
		return e, err
	}
	var ok bool
	e.Stock, ok = db.Stocks[sid]
	if !ok {
		return e, fmt.Errorf("no stock with id %s", sid)
	}

	cid, err := MkID(p[2])
	if err != nil {
		return e, err
	}
	e.Camera, ok = db.Cameras[cid]
	if !ok {
		return e, fmt.Errorf("no camera with id %s", cid)
	}

	if len(p) > 3 {
		if p[3] == "-" || p[3] == "--" || p[3] == "---" {
			e.Lab = LabNone()
			return e, nil
		}

		if len(p) < 5 {
			return e, errors.New("entry should contain lab-in-date when lab is specified")
		}
		lid, err := MkID(p[3])
		if err != nil {
			return e, err
		}
		e.Lab, ok = db.Labs[lid]
		if !ok {
			return e, fmt.Errorf("no lab with id %s", lid)
		}

		labin, err := time.Parse(dateFormat, p[4])
		if err != nil {
			return e, fmt.Errorf("error in lab-in-date: %w", err)
		}

		e.LabInDate = labin

	}

	if len(p) > 5 {
		labout, err := time.Parse(dateFormat, p[5])
		if err != nil {
			return e, fmt.Errorf("error in lab-out-date: %w", err)
		}

		e.LabOutDate = labout
	}

	if len(p) > 6 {
		_s, err := strconv.ParseUint(p[6], 10, 32)
		if err != nil {
			return e, fmt.Errorf("invalid scan page: %w", err)
		}
		s := uint(_s)
		if s != 0 {
			if _, ok := scans[s]; ok {
				return e, fmt.Errorf("duplicate scan page: %d", s)
			}
			scans[s] = struct{}{}
			e.Scan = s
		}
	}

	return e, nil
}
