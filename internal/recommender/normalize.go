package recommender

import (
	"strings"
	"unicode"
)

var editionSuffixes = []string{
	"special edition",
	"remastered",
	"remake",
	"enhanced edition",
	"game of the year edition",
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
	"hd edition",
	"classic edition",
}

func normalizeName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))

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

	for _, suffix := range editionSuffixes {
		s = strings.TrimSuffix(s, suffix)
	}

	s = strings.TrimPrefix(s, "the ")

	fields := strings.FieldsFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || r == ':' || r == '-' || r == '_'
	})
	s = strings.Join(fields, " ")

	return strings.TrimSpace(s)
}
