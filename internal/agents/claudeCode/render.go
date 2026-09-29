package claudeCode

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	provider "github.com/pardnchiu/go-llm-router/core"
)

type blockWriter struct {
	list []map[string]any
	text strings.Builder
}

func (w *blockWriter) writeText(s string) {
	w.text.WriteString(s)
}

func (w *blockWriter) writeImage(url string) {
	source := imageSource(url)
	if source == nil {
		w.text.WriteString("[image omitted: unsupported image URL]")
		return
	}
	w.flush()
	w.list = append(w.list, map[string]any{"type": "image", "source": source})
}

func (w *blockWriter) flush() {
	if w.text.Len() == 0 {
		return
	}
	w.list = append(w.list, map[string]any{"type": "text", "text": w.text.String()})
	w.text.Reset()
}

func (w *blockWriter) blocks() []map[string]any {
	w.flush()
	return w.list
}

func (w *blockWriter) writeContent(content any) {
	parts, ok := contentParts(content)
	if !ok {
		w.writeText(contentText(content))
		return
	}
	for i, p := range parts {
		if i > 0 {
			w.writeText("\n")
		}
		switch {
		case p.Type == "text":
			w.writeText(p.Text)
		case p.ImageURL != nil:
			w.writeImage(p.ImageURL.URL)
		}
	}
}

func imageSource(url string) map[string]any {
	if rest, ok := strings.CutPrefix(url, "data:"); ok {
		meta, data, found := strings.Cut(rest, ",")
		mediaType, isBase64 := strings.CutSuffix(meta, ";base64")
		if !found || !isBase64 || !strings.HasPrefix(mediaType, "image/") || data == "" {
			return nil
		}
		return map[string]any{"type": "base64", "media_type": mediaType, "data": data}
	}
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		return map[string]any{"type": "url", "url": url}
	}
	return nil
}

func splitSystem(messages []provider.Message) (string, []provider.Message) {
	list := []string{}
	i := 0
	for ; i < len(messages) && messages[i].Role == "system"; i++ {
		list = append(list, contentText(messages[i].Content))
	}
	return strings.Join(list, "\n\n"), messages[i:]
}

func specOf(system, effort string) string {
	sum := sha256.Sum256([]byte(effort + "\x00" + system))
	return hex.EncodeToString(sum[:])
}

func renderTools(toolDefs []provider.Tool) string {
	if len(toolDefs) == 0 {
		return ""
	}
	names := make([]string, 0, len(toolDefs))
	finder := ""
	for _, t := range toolDefs {
		names = append(names, t.Function.Name)
		if t.Function.Name == "find_tools" {
			raw, _ := json.Marshal(t.Function)
			finder = string(raw)
		}
	}
	text := "<tools>\nnames: " + strings.Join(names, ", ")
	if finder != "" {
		text += "\nfind_tools: " + finder
	}
	return text + "\n</tools>"
}

func fingerprints(messages []provider.Message) []string {
	list := make([]string, 0, len(messages))
	for _, m := range messages {
		list = append(list, fingerprint(m))
	}
	return list
}

func fingerprint(m provider.Message) string {
	var sb strings.Builder
	sb.WriteString(m.Role)
	sb.WriteByte(0)
	if m.Role == "assistant" {
		for _, c := range m.ToolCalls {
			sb.WriteString(c.ID)
			sb.WriteByte(0)
			sb.WriteString(c.Function.Name)
			sb.WriteByte(0)
		}
		return sb.String()
	}
	sb.WriteString(m.ToolCallID)
	sb.WriteByte(0)
	sb.WriteString(contentText(m.Content))
	return sb.String()
}

func toolNames(messages []provider.Message) map[string]string {
	dic := map[string]string{}
	for _, m := range messages {
		for _, c := range m.ToolCalls {
			dic[c.ID] = c.Function.Name
		}
	}
	return dic
}

func renderInitial(system string, toolDefs []provider.Tool, messages []provider.Message) []map[string]any {
	w := &blockWriter{}
	fmt.Fprintf(&w.text, "<system_prompt>\n%s\n</system_prompt>\n\n", system)
	if tools := renderTools(toolDefs); tools != "" {
		w.writeText(tools + "\n\n")
	}
	writeMessages(w, messages, toolNames(messages))
	return w.blocks()
}

func renderMessages(messages []provider.Message, idName map[string]string) []map[string]any {
	w := &blockWriter{}
	writeMessages(w, messages, idName)
	return w.blocks()
}

func writeMessages(w *blockWriter, messages []provider.Message, idName map[string]string) {
	for i, m := range messages {
		if i > 0 {
			w.writeText("\n\n")
		}
		switch m.Role {
		case "tool":
			fmt.Fprintf(&w.text, "<tool_result name=%q>\n", idName[m.ToolCallID])
			w.writeContent(m.Content)
			w.writeText("\n</tool_result>")
		case "assistant":
			w.writeText("<assistant>\n")
			if text := contentText(m.Content); text != "" {
				w.writeText(text + "\n")
			}
			for _, c := range m.ToolCalls {
				fmt.Fprintf(&w.text, "<tool_call name=%q>%s</tool_call>\n", c.Function.Name, c.Function.Arguments)
			}
			w.writeText("</assistant>")
		default:
			fmt.Fprintf(&w.text, "<%s>\n", m.Role)
			w.writeContent(m.Content)
			fmt.Fprintf(&w.text, "\n</%s>", m.Role)
		}
	}
}

func contentParts(content any) ([]provider.ContentPart, bool) {
	switch v := content.(type) {
	case nil, string:
		return nil, false
	case []provider.ContentPart:
		return v, true
	}
	raw, err := json.Marshal(content)
	if err != nil {
		return nil, false
	}
	var parts []provider.ContentPart
	if json.Unmarshal(raw, &parts) != nil {
		return nil, false
	}
	return parts, true
}

func contentText(content any) string {
	if s, ok := content.(string); ok {
		return s
	}
	if content == nil {
		return ""
	}
	parts, ok := contentParts(content)
	if !ok {
		raw, err := json.Marshal(content)
		if err != nil {
			return fmt.Sprint(content)
		}
		return string(raw)
	}
	list := make([]string, 0, len(parts))
	for _, p := range parts {
		switch {
		case p.Type == "text":
			list = append(list, p.Text)
		case p.ImageURL != nil:
			sum := sha256.Sum256([]byte(p.ImageURL.URL))
			list = append(list, "[image "+hex.EncodeToString(sum[:8])+"]")
		}
	}
	return strings.Join(list, "\n")
}
