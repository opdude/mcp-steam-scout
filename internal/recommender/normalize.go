package recommender

import (
	"strings"
	"unicode"
)

var editionSuffixes = []string{
	"special edition",
	"remastered",
	"remake",
	"warmastered edition",
	"warmastered",
	"enhanced edition",
	"enhanced",
	"game of the year edition",
	"game of the year",
	"goty edition",
	"goty",
	"definitive edition",
	"anniversary edition",
	"complete edition",
	"deluxe edition",
	"premium edition",
	"ultimate edition",
	"legendary edition",
	"director's cut",
	"directors cut",
	"hd edition",
	"classic edition",
	"survivor edition",
	"expanded edition",
	"extended edition",
	"collector's edition",
	"collectors edition",
	"digital edition",
	"limited edition",
}

var trademarkChars = []string{
	"\u2122", // ™
	"\u00AE", // ®
	"\u00A9", // ©
}

func normalizeName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))

	for _, ch := range trademarkChars {
		s = strings.ReplaceAll(s, ch, "")
	}

	for {
		start := strings.IndexRune(s, '(')
		if start == -1 {
			break
		}
		end := strings.IndexRune(s[start:], ')')
		if end == -1 {
			break
		}
		s = strings.TrimSpace(s[:start] + s[start+end+1:])
	}

	for {
		start := strings.IndexRune(s, '[')
		if start == -1 {
			break
		}
		end := strings.IndexRune(s[start:], ']')
		if end == -1 {
			break
		}
		s = strings.TrimSpace(s[:start] + s[start+end+1:])
	}

	prev := ""
	for s != prev {
		prev = s
		for _, suffix := range editionSuffixes {
			s = strings.TrimSuffix(s, suffix)
		}
		s = strings.TrimSpace(s)
	}

	s = strings.TrimPrefix(s, "the ")

	fields := strings.FieldsFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || r == ':' || r == '-' || r == '_'
	})
	s = strings.Join(fields, " ")

	return strings.TrimSpace(s)
}
