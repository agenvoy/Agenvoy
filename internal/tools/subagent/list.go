package subagent

import (
	"fmt"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/session"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"
)

func listSessions(selfID string) string {
	keyword := strings.ToLower(strings.TrimSpace(selfID))

	var sb strings.Builder
	for _, s := range session.ListSessions() {
		if !strings.Contains(strings.ToLower(s.SelfID), keyword) {
			continue
		}
		role := strings.Join(strings.Fields(s.Role), " ")
		if role == "" {
			role = "(no role description)"
		}
		label := s.SelfID
		if s.Name != "" && s.Name != s.SelfID {
			label = fmt.Sprintf("%s (%s)", s.SelfID, s.Name)
		}
		fmt.Fprintf(&sb, "- %s — %s\n", label, go_pkg_utils.TruncateString(role, 256))
	}
	if sb.Len() == 0 {
		return fmt.Sprintf("no session whose self id contains %q — spawn a temp subagent (self_id empty)", selfID)
	}
	return strings.TrimRight(sb.String(), "\n")
}
