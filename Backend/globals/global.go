package globals

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

var Hostname string
var Port string

type User struct {
	Github_id    int64  `json:"id"`
	Username     string `json:"login"`
	Github_url   string `json:"html_url"`
	Avatar_url   string `json:"avatar_url"`
	SessionToken string
	Email        string
}

type Email struct {
	Email      string `json:"email"`
	Primary    bool   `json:"primary"`
	Verified   bool   `json:"verified"`
	Visibility string `json:"visibility"`
}

type SessionData struct {
	GithubID         int64
	InputID          int
	CurrentLevel     int
	NextReleaseLevel int
}

var DB *pgxpool.Pool
