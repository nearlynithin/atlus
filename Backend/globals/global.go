package globals

import (
	"html/template"
	"net/http"
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

type ProblemSet struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

type SubmissionStatus int

const (
	AlreadyPassed SubmissionStatus = iota
	LevelIncomplete
	LevelPassed
	LevelFailed
	Cooldown
	SubmissionError
)

func RenderInfoPage(tpl *template.Template, w http.ResponseWriter, loggedIn bool, data map[string]any) {
	data["LoggedIn"] = loggedIn
	data["Info"] = true
	err := tpl.ExecuteTemplate(w, "base", data)
	if err != nil {
		http.Error(w, "Invalid url request", http.StatusBadRequest)
	}
}

var DB *pgxpool.Pool
