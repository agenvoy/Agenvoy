package tool

import (
	"regexp"
	"strings"
)

type toolMeta struct {
	name    *regexp.Regexp
	hasArg  bool
	argGlob *regexp.Regexp
}

func parse(line string) (toolMeta, bool) {
	open := strings.IndexByte(line, '(')
	if open < 0 {
		name := strings.TrimSpace(line)
		if name == "" {
			return toolMeta{}, false
		}
		return toolMeta{name: formatRegex(name)}, true
	}
	if !strings.HasSuffix(line, ")") {
		return toolMeta{}, false
	}

	toolName := strings.TrimSpace(line[:open])
	pattern := line[open+1 : len(line)-1]
	if toolName == "" {
		return toolMeta{}, false
	}
	return toolMeta{
		name:    formatRegex(toolName),
		hasArg:  true,
		argGlob: formatRegex(pattern),
	}, true
}

func formatRegex(pattern string) *regexp.Regexp {
	var sb strings.Builder
	sb.WriteByte('^')
	i := 0
	for i < len(pattern) {
		c := pattern[i]
		switch c {
		case '\\':
			if i+1 < len(pattern) {
				next := pattern[i+1]
				if next == '*' || next == '?' || next == '\\' {
					sb.WriteByte('\\')
					sb.WriteByte(next)
					i += 2
					continue
				}
			}
			sb.WriteString(`\\`)
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				sb.WriteString(".*")
				i += 2
				continue
			}
			sb.WriteString("[^/]*")
		case '?':
			sb.WriteString("[^/]")
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '[', ']':
			sb.WriteByte('\\')
			sb.WriteByte(c)
		default:
			sb.WriteByte(c)
		}
		i++
	}
	sb.WriteByte('$')
	re, err := regexp.Compile(sb.String())
	if err != nil {
		return regexp.MustCompile("^$a")
	}
	return re
}
