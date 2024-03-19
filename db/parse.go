package db

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

const dateFormat = "2006-01-02"

func Parse(r io.Reader) (*DB, error) {
	db := &DB{
		Entries: make([]Entry, 0),

		Companies: make(map[ID]*Company, 0),
		Stocks:    make(map[ID]*Stock, 0),
		Cameras:   make(map[ID]*Camera, 0),
		Labs:      make(map[ID]*Lab, 0),
	}

	var lastID ID
	scans := make(map[uint]struct{})

	const (
		keywordNone    = ""
		keywordCompany = "Company"
		keywordStock   = "Stock"
		keywordCamera  = "Camera"
		keywordLab     = "Lab"
		keywordEntry   = "Entry"
	)

	s := bufio.NewScanner(r)
	s.Split(bufio.ScanLines)
	var keyword string
	var line uint
	for s.Scan() {
		line++
		t := strings.TrimSpace(s.Text())
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
				return db, fmt.Errorf("no company with id %s", lastID)
			}
			c.Name = t
			keyword = keywordNone
			continue
		case keywordStock:
			s, ok := db.Stocks[lastID]
			if !ok {
				return db, fmt.Errorf("no stock with id %s", lastID)
			}
			if s.Format == "" {
				s.Format = t
			} else if s.Name == "" {
				s.Name = t
			} else if s.Company == nil {
				cid, err := MkID(t)
				if err != nil {
					return db, err
				}
				s.Company = db.Companies[cid]
				if s.Company == nil {
					return db, fmt.Errorf("no company by id '%s'", t)
				}
			} else if s.ISO.Low == 0 {
				p := strings.FieldsFunc(t, func(r rune) bool {
					return r == ' ' || r == '-'
				})
				if len(p) > 2 {
					return db, fmt.Errorf("invalid ISO line %d: '%s'", line, t)
				}

				for i := range p {
					v, err := strconv.ParseUint(p[i], 10, 32)
					if err != nil {
						return db, fmt.Errorf("invalid integers in ISO line %d: '%s'", line, t)
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
					return db, fmt.Errorf("invalid ISO range in line %d: '%s'", line, t)
				}
			} else if s.Rolls == 0 {
				l := strings.FieldsFunc(t, func(r rune) bool {
					return r == ' ' || r == '+'
				})

				n := 0
				for _, s := range l {
					val, err := strconv.Atoi(s)
					if err != nil {
						return db, fmt.Errorf("invalid number on line %d: %s", line, s)
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
				return db, fmt.Errorf("no camera with id %s", lastID)
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
				return db, fmt.Errorf("no lab with id %s", lastID)
			}
			l.Name = t
			keyword = keywordNone
			continue
		case keywordEntry:
			db.Entries[len(db.Entries)-1].Note = t
			keyword = keywordNone
			continue
		}

		p := strings.Fields(t)

		// UTC!
		if d, err := time.Parse(dateFormat, p[0]); err == nil {
			e, err := db.mkEntry(d, p, scans)
			if err != nil {
				return db, fmt.Errorf("%w: line %d: '%s'", err, line, t)
			}

			e.Line = line
			db.Entries = append(db.Entries, e)
			keyword = keywordEntry
			continue
		}

		if len(p) != 2 {
			return db, fmt.Errorf("invalid line %d: '%s'", line, t)
		}

		keyword = p[0]
		id, err := MkID(p[1])
		if err != nil {
			return db, err
		}

		lastID = id
		switch keyword {
		case keywordCompany:
			if _, ok := db.Companies[id]; ok {
				return db, fmt.Errorf("duplicate company id '%s'", id.String())
			}
			db.Companies[id] = &Company{ID: id}
		case keywordStock:
			if _, ok := db.Stocks[id]; ok {
				return db, fmt.Errorf("duplicate stock id '%s'", id.String())
			}
			db.Stocks[id] = &Stock{ID: id}
		case keywordCamera:
			if _, ok := db.Cameras[id]; ok {
				return db, fmt.Errorf("duplicate camera id '%s'", id.String())
			}
			db.Cameras[id] = &Camera{ID: id}
		case keywordLab:
			if _, ok := db.Labs[id]; ok {
				return db, fmt.Errorf("duplicate lab id '%s'", id.String())
			}
			db.Labs[id] = &Lab{ID: id}
		default:
			return db, fmt.Errorf("invalid keyword: '%s'", keyword)
		}
	}

	if err := s.Err(); err != nil {
		return db, err
	}
	return db, nil
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
