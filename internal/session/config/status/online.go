package configStatus

const (
	StatusOnline = "online"
	StatusIdle   = "idle"
)

type Status struct {
	State string `json:"state"`
	Count int    `json:"count"`
}
