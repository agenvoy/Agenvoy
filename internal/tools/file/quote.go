package file

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	leftSingleQuote  = '‘'
	rightSingleQuote = '’'
	leftDoubleQuote  = '“'
	rightDoubleQuote = '”'
)

func straightQuote(r rune) rune {
	switch r {
	case leftSingleQuote, rightSingleQuote:
		return '\''
	case leftDoubleQuote, rightDoubleQuote:
		return '"'
	}
	return r
}

func straightenQuotes(s string) (string, []int) {
	var sb strings.Builder
	sb.Grow(len(s))
	offsets := make([]int, 0, len(s)+1)
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if q := straightQuote(r); q != r {
			sb.WriteByte(byte(q))
			offsets = append(offsets, i)
		} else {
			sb.WriteString(s[i : i+size])
			for k := range size {
				offsets = append(offsets, i+k)
			}
		}
		i += size
	}
	offsets = append(offsets, len(s))
	return sb.String(), offsets
}

func findCurlyAnchor(content, old string) (string, bool) {
	straightOld, _ := straightenQuotes(old)
	straightContent, offsets := straightenQuotes(content)
	idx := strings.Index(straightContent, straightOld)
	if idx < 0 {
		return "", false
	}
	return content[offsets[idx]:offsets[idx+len(straightOld)]], true
}

func curlQuotes(anchor, text string) string {
	hasDouble := strings.ContainsAny(anchor, string([]rune{leftDoubleQuote, rightDoubleQuote}))
	hasSingle := strings.ContainsAny(anchor, string([]rune{leftSingleQuote, rightSingleQuote}))
	if !hasDouble && !hasSingle {
		return text
	}

	runes := []rune(text)
	for i, r := range runes {
		switch {
		case r == '"' && hasDouble:
			runes[i] = rightDoubleQuote
			if isOpeningQuote(runes, i) {
				runes[i] = leftDoubleQuote
			}
		case r == '\'' && hasSingle:
			runes[i] = rightSingleQuote
			if !isContraction(runes, i) && isOpeningQuote(runes, i) {
				runes[i] = leftSingleQuote
			}
		}
	}
	return string(runes)
}

func isOpeningQuote(runes []rune, i int) bool {
	if i == 0 {
		return true
	}
	switch runes[i-1] {
	case ' ', '\t', '\n', '\r', '(', '[', '{', '—', '–':
		return true
	}
	return false
}

func isContraction(runes []rune, i int) bool {
	return i > 0 && i < len(runes)-1 && unicode.IsLetter(runes[i-1]) && unicode.IsLetter(runes[i+1])
}
