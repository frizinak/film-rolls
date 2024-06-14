package db_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/frizinak/film-rolls/db"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestEnc(t *testing.T) {
	str := `
Camera CAM
	Some
	Camera

Company COMP
	Some Company

Lab LAB
	Laboratory

Stock STOCK
	120 C
	Film stock
	COMP
	400

2024-05-01 STOCK CAM LAB 2024-05-02 2024-05-03 0060

2024-05-01 STOCK CAM LAB 2024-05-02 2024-05-03
	text

2024-05-01 STOCK CAM LAB 2024-05-02
	text
	some more text

2024-05-01 STOCK CAM -
	text
	+tags:pacific-ocean,under-the-sea
	some more text

2024-05-01 STOCK CAM
	text
	+tags:pacific-ocean,under-the-sea
	some more text
	+id:123
`
	r := strings.NewReader(str)
	data, err := db.Parse(r)
	if err != nil {
		t.Fatal(err)
	}

	buf := bytes.NewBuffer(nil)
	if err := data.Write(buf); err != nil {
		t.Fatal(err)
	}

	res := strings.TrimSpace(buf.String())
	exp := strings.TrimSpace(str)

	dmp := diffmatchpatch.New()
	diff := dmp.DiffMain(res, exp, true)
	if res != exp {
		t.Error(dmp.DiffPrettyText(diff))
	}
}
