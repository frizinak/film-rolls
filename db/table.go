package db

import "github.com/frizinak/film-rolls/table"

type TableConfig struct {
	Filter Filter

	Notes  bool
	Short  bool
	Color  bool
	Pretty bool

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

func rowStock(
	t *table.Table,
	conf TableConfig,
	space table.Col,
	stockID, stockName, stockFormat, stockType, stockISO, stockCompany string,
) {
	clr := func(seq string) string {
		if conf.Color {
			return seq
		}
		return ""
	}

	t.AddCol(table.ColFixed(table.ColPreSuf(
		table.TermStr(stockID),
		clr("\033[38;5;244m"),
		clr("\033[0m"),
	)))

	if !conf.Short {
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColAlignRight(table.ColFixed(table.TermStr(stockFormat))))
		t.AddCol(table.ColFixed(space))
		t.AddCol(table.ColAlignRight(table.ColFixed(table.TermStr(stockType))))
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
	}
}
