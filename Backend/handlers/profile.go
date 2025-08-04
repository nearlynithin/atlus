package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/sceptix-club/atlus/Backend/globals"
)

func ProfileHandler(tpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		sdata := ctx.Value("sessionData").(globals.SessionData)

		created := time.Now().UTC().Sub(sdata.CreatedAt)
		joined := fmt.Sprintf("Joined %v ago", created)

		tpl.ExecuteTemplate(w, "base", map[string]any{
			"LoggedIn":     true,
			"Profile":      true,
			"Avatar":       sdata.Avatar,
			"Username":     sdata.Username,
			"GithubUrl":    sdata.GithubUrl,
			"CurrentLevel": sdata.CurrentLevel,
			"Streak":       sdata.Streak,
			"Joined":       joined,
		})
	}
}
