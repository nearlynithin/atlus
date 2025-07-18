package globals

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	Github_id int64 `json:"id"`
	Username string `json:"login"`
	Github_url string `json:"html_url"`
	Avatar_url string `json:"avatar_url"`
	SessionToken string
	Email string
}

type Email struct {
	Email string `json:"email"`
	Primary bool `json:"primary"`
	Verified bool `json:"verified"`
	Visibility string `json:"visibility"`
}

var Sessions = map[string]string{}

var DB *pgxpool.Pool
