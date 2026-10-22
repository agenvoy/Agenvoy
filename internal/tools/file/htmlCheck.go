package file

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/js"
)

var (
	htmlScriptRegex   = regexp.MustCompile(`(?is)<script\b([^>]*)>(.*?)</script>`)
	htmlSrcAttrRegex  = regexp.MustCompile(`(?i)\bsrc\s*=`)
	htmlTypeAttrRegex = regexp.MustCompile(`(?i)\btype\s*=\s*["']?([^"'\s>]+)`)
	htmlQuickUIRegex  = regexp.MustCompile(`(?i)<script[^>]+src\s*=\s*["'][^"']*@pardnchiu/quickui@`)
	htmlTInitRegex    = regexp.MustCompile("T\\.init\\(\\s*['\"`]([^'\"`$]+)['\"`]\\s*,")
	htmlMarkupRegex   = regexp.MustCompile(`\.(innerHTML|outerHTML|insertAdjacentHTML|appendChild)\b|document\.(createElement|write)\b`)
	htmlIDAttrRegex   = regexp.MustCompile(`\bid\s*=\s*["']([^"']+)["']`)
	htmlMustacheRegex = regexp.MustCompile(`\{\{([^}]*)\}\}`)
	htmlExprRegex     = regexp.MustCompile(`&&|\|\||=>|\.toFixed\(|\?[^:]*:`)
	htmlIfRegex       = regexp.MustCompile(`:(?:if|else-if)\s*=\s*"([^"]*)"`)
)

func checkHTML(content string) ([]string, bool) {
	var problems []string
	list := func(items []string) string {
		return strings.Join(slices.Compact(slices.Sorted(slices.Values(items))), ", ")
	}

	quickUI := htmlQuickUIRegex.MatchString(content) && strings.Contains(content, "new QUI(")
	if !quickUI {
		problems = append(problems, "QuickUI is not mounted: load https://cdn.jsdelivr.net/npm/@pardnchiu/quickui@0/dist/QuickUI.js and call new QUI({ id, data, when }) at the end of <body>")
	}

	var inline strings.Builder
	for i, m := range htmlScriptRegex.FindAllStringSubmatch(content, -1) {
		attrs, body := m[1], m[2]
		if htmlSrcAttrRegex.MatchString(attrs) || strings.TrimSpace(body) == "" {
			continue
		}
		if t := htmlTypeAttrRegex.FindStringSubmatch(attrs); t != nil {
			switch strings.ToLower(t[1]) {
			case "text/javascript", "application/javascript", "module":
			default:
				continue
			}
		}
		if _, err := js.Parse(parse.NewInputString(body), js.Options{}); err != nil {
			problems = append(problems, fmt.Sprintf("<script> #%d does not parse, so nothing on the page renders: %v", i+1, err))
		}
		inline.WriteString(body)
		inline.WriteString("\n")
	}
	script := inline.String()

	if m := htmlMarkupRegex.FindAllString(script, -1); len(m) > 0 {
		problems = append(problems, fmt.Sprintf("script writes markup (%s): bind it in the QuickUI template and change it through APP.data; an element a library takes (e.g. a MapLibre marker) comes from QuickUI's _(tag, attrs, children)", list(m)))
	}

	markup := htmlScriptRegex.ReplaceAllString(content, "")
	ids := make(map[string]bool)
	for _, m := range htmlIDAttrRegex.FindAllStringSubmatch(markup, -1) {
		ids[m[1]] = true
	}
	var missing []string
	for _, m := range htmlTInitRegex.FindAllStringSubmatch(script, -1) {
		if !ids[m[1]] {
			missing = append(missing, m[1])
		}
	}
	if len(missing) > 0 {
		problems = append(problems, fmt.Sprintf("T.init targets with no container in the markup (the chart silently draws nothing): %s", list(missing)))
	}

	var exprs []string
	for _, m := range htmlMustacheRegex.FindAllStringSubmatch(markup, -1) {
		if htmlExprRegex.MatchString(m[1]) {
			exprs = append(exprs, "{{"+m[1]+"}}")
		}
	}
	if len(exprs) > 0 {
		problems = append(problems, fmt.Sprintf("{{ }} evaluates no expressions and renders these literally; precompute the value in data: %s", list(exprs)))
	}

	var conds []string
	for _, m := range htmlIfRegex.FindAllStringSubmatch(markup, -1) {
		if htmlExprRegex.MatchString(m[1]) {
			conds = append(conds, m[1])
		}
	}
	if len(conds) > 0 {
		problems = append(problems, fmt.Sprintf(":if takes one comparison, no && / || / ternary; precompute a flag in data: %s", list(conds)))
	}

	return problems, quickUI
}
