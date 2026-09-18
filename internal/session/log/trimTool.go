package sessionLog

import "strings"

func TrimSettledTools(content string) string {
	if content == "" {
		return content
	}

	settled := map[string]bool{}
	lastText := map[string]int{}
	index := 0
	for line := range strings.SplitSeq(content, "\n") {
		m := lineRegex.FindStringSubmatch(line)
		if len(m) < 6 || m[4] == "" {
			index++
			continue
		}
		switch m[3] {
		case "done", "canceled", "error":
			settled[m[4]] = true
		case "assistant":
			lastText[m[4]] = index
		}
		index++
	}
	if len(settled) == 0 {
		return content
	}

	list := make([]string, 0, index)
	index = 0
	for line := range strings.SplitSeq(content, "\n") {
		m := lineRegex.FindStringSubmatch(line)
		if len(m) >= 6 && settled[m[4]] {
			switch {
			case strings.HasPrefix(m[3], "tool_"), m[3] == "thinking":
				index++
				continue
			case m[3] == "assistant" && lastText[m[4]] != index:
				index++
				continue
			}
		}
		list = append(list, line)
		index++
	}
	return strings.Join(list, "\n")
}
