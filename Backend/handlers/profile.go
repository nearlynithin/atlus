package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"time"
)

func ProfileHandler(tpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		c, err := r.Cookie("session")
		if err != nil {
			fmt.Fprintf(w, "Invalid session, please login to continue : %v", err)
			return
		}
		sdata, err := getSessionData(ctx, c.Value)
		if err != nil {
			fmt.Fprintf(w, "Invalid session, please login to continue : %v", err)
			return
		}

		created := time.Now().UTC().Sub(sdata.CreatedAt)
		joined := fmt.Sprintf("Joined %v ago", created)

		tpl.ExecuteTemplate(w, "base", map[string]any{
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
