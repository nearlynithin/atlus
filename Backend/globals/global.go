package globals

import (
	"time"

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
	GithubUrl        string
	Username         string
	Avatar           string
	Email            string
	InputID          int
	CurrentLevel     int
	Streak           int
	NextReleaseLevel int
	CreatedAt        time.Time
}

type SubmissionData struct {
	GithubID     int64
	Username     string
	CurrentLevel int
	PuzzleLevel  int
	Pass         bool
}

var DB *pgxpool.Pool
