package handlers

import (
	"html/template"
	"net/http"
)


func LeaderboardHandler(tpl * template.Template) http.HandlerFunc{

	return func(w http.ResponseWriter, r* http.Request) {
		tpl.ExecuteTemplate(w, "leaderboard", map[string]any{
			"Leaderboard": true,
		})
	}
}
