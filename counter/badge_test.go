package main

import (
	"encoding/xml"
	"io"
	"math"
	"strings"
	"testing"
)

func TestBadgeIsValidSVG(t *testing.T) {
	cases := []struct {
		count int64
		text  string
	}{
		{0, "0"},
		{42, "42"},
		{123456, "123,456"},
		{math.MaxInt64, "9,223,372,036,854,775,807"},
	}
	for _, test := range cases {
		badge := renderBadge(test.count)
		if !strings.Contains(badge, "Profile views: "+test.text) {
			t.Fatalf("missing accessible count for %d", test.count)
		}
		decoder := xml.NewDecoder(strings.NewReader(badge))
		for {
			_, err := decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}
