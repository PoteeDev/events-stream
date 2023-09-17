package models

type TeamsEvents struct {
	Round int     `json:"round"`
	Teams []Teams `json:"teams"`
}
type Checkers struct {
	Name      string `json:"name"`
	GetStatus int    `json:"get_status"`
	PutStatus int    `json:"put_status"`
}
type Exploits struct {
	Name   string `json:"name"`
	Status int    `json:"status"`
}
type TeamServices struct {
	Name       string     `json:"name"`
	PingStatus int        `json:"ping_status"`
	Checkers   []Checkers `json:"checkers"`
	Exploits   []Exploits `json:"exploits"`
}
type Teams struct {
	Name     string         `json:"name"`
	Services []TeamServices `json:"services"`
}
