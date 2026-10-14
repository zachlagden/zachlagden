package main

import (
	"fmt"
	"strconv"
	"strings"
)

func renderBadge(count int64) string {
	digits := strconv.FormatInt(count, 10)
	var value strings.Builder
	for index, digit := range digits {
		if index > 0 && (len(digits)-index)%3 == 0 {
			value.WriteByte(',')
		}
		value.WriteRune(digit)
	}
	text := value.String()
	valueWidth := len(text)*7 + 14
	width := 40 + valueWidth
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20" role="img" aria-label="Profile views: %s">
<title>Profile views: %s</title>
<clipPath id="badge"><rect width="%d" height="20" rx="3"/></clipPath>
<g clip-path="url(#badge)"><rect width="40" height="20" fill="#555"/><rect x="40" width="%d" height="20" fill="#666"/></g>
<g fill="#fff" text-anchor="middle" font-family="Verdana,DejaVu Sans,sans-serif" font-size="11"><text x="20" y="14">views</text><text x="%d" y="14">%s</text></g>
</svg>`, width, text, text, width, valueWidth, 40+valueWidth/2, text)
}
