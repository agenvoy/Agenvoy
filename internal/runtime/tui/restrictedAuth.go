package tui

type RestrictedAuthDone struct {
	pendingID string
	cached    bool
	err       error
}
