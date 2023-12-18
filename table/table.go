package table

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

type Align uint8

const (
	AlignLeft Align = iota
	AlignRight
)

type Col interface {
	Width() int
	Align() Align
	Prefix() string
	String() string
	Suffix() string
	Fixed() bool
}

type fixed struct{ Col }

func (f fixed) Fixed() bool { return true }

type prefixed struct {
	Col
	prefix string
}

func (p prefixed) Prefix() string { return p.prefix }

type suffixed struct {
	Col
	suffix string
}

func (s suffixed) Suffix() string { return s.suffix }

type aligned struct {
	Col
	align Align
}

func (a aligned) Align() Align { return a.align }

func ColAlignLeft(col Col) Col               { return aligned{col, AlignLeft} }
func ColAlignRight(col Col) Col              { return aligned{col, AlignRight} }
func ColFixed(col Col) Col                   { return fixed{col} }
func ColPrefixed(col Col, prefix string) Col { return prefixed{col, prefix} }
func ColSuffixed(col Col, suffix string) Col { return suffixed{col, suffix} }
func ColPreSuf(col Col, prefix, suffix string) Col {
	return ColPrefixed(ColSuffixed(col, suffix), prefix)
}

type TermStr string

func (t TermStr) Width() int     { return runewidth.StringWidth(string(t)) }
func (t TermStr) Align() Align   { return AlignLeft }
func (t TermStr) String() string { return string(t) }
func (t TermStr) Prefix() string { return "" }
func (t TermStr) Suffix() string { return "" }
func (t TermStr) Fixed() bool    { return false }

type Str string

func (s Str) Width() int     { return utf8.RuneCountInString(string(s)) }
func (s Str) String() string { return string(s) }
func (s Str) Align() Align   { return AlignLeft }
func (s Str) Prefix() string { return "" }
func (s Str) Suffix() string { return "" }
func (s Str) Fixed() bool    { return false }

func ClrTermStr(clr string, str string) Col {
	ts := TermStr(str)
	if clr == "" {
		return ts
	}
	return ColSuffixed(ColPrefixed(ts, clr), "\033[0m")
}

func TermStrs(strs ...string) []Col {
	c := make([]Col, len(strs))
	for i := range strs {
		c[i] = TermStr(strs[i])
	}
	return c
}

func Strs(strs ...string) []Col {
	c := make([]Col, len(strs))
	for i := range strs {
		c[i] = Str(strs[i])
	}
	return c
}

type Table struct {
	width int
	head  []Col
	rows  [][]Col
}

func New() *Table {
	return &Table{0, make([]Col, 0), make([][]Col, 0)}
}

func (t *Table) SetFixedWidth(w int) {
	t.width = w
}

func (t *Table) NewRow() { t.rows = append(t.rows, make([]Col, 0)) }

func (t *Table) AddRow(cols ...Col) {
	t.NewRow()
	for _, col := range cols {
		t.AddCol(col)
	}
}

func (t *Table) AddCol(col Col) {
	if len(t.rows) == 0 {
		panic("can't add col without a row")
	}
	t.rows[len(t.rows)-1] = append(t.rows[len(t.rows)-1], col)
}

func (t *Table) AddHeadCol(value Col) { t.head = append(t.head, value) }

func (t *Table) WriteTo(wr io.Writer, sep string) {
	widest := len(t.head)
	for i := range t.rows {
		if len(t.rows[i]) > widest {
			widest = len(t.rows[i])
		}
	}

	fixed := make([]bool, widest)
	w := make([]int, widest)
	for i := range t.head {
		fixed[i] = fixed[i] || t.head[i].Fixed()
		w[i] = t.head[i].Width()
	}

	for _, row := range t.rows {
		for i, col := range row {
			fixed[i] = fixed[i] || col.Fixed()
			if wi := col.Width(); wi > w[i] {
				w[i] = wi
			}
		}
	}

	if t.width != 0 {
		sum := 0
		for i := range w {
			sum += w[i]
		}

		func() {
			if sum >= t.width {
				return
			}
			fix := 0
			for i := range fixed {
				if fixed[i] {
					fix++
				}
			}

			widen := (len(w) - fix)
			if widen <= 0 {
				return
			}
			per := (t.width - sum) / widen
			rem := (t.width - sum) - (per * widen)
			for i := range w {
				if !fixed[i] {
					w[i] += per + rem
					rem = 0
				}
			}
		}()
	}

	f := make([]string, 0, len(t.head))
	strs := make([]any, 0, len(t.head))
	for i, r := range t.head {
		f = append(f, "%s%-"+strconv.Itoa(w[i])+"s%s")
		strs = append(strs, r.Prefix(), r.String(), r.Suffix())
	}
	if len(strs) != 0 {
		fmt.Fprintf(wr, strings.Join(f, sep)+"\n", strs...)
	}

	for _, row := range t.rows {
		f = f[:0]
		strs = strs[:0]
		for i, col := range row {
			a := "-"
			if col.Align() == AlignRight {
				a = ""
			}
			f = append(f, "%s%"+a+strconv.Itoa(w[i])+"s%s")
			strs = append(strs, col.Prefix(), col.String(), col.Suffix())
		}
		fmt.Fprintf(wr, strings.Join(f, sep)+"\n", strs...)
	}
}
