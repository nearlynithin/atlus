package handlers

import (
	"html/template"
	"math/rand/v2"
	"net/http"
)

func LeaderboardHandler(tpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tpl.ExecuteTemplate(w, "leaderboard", map[string]any{
			"Leaderboard": true,
		})
	}
}

func LeaderboardLiveHandler(tpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		data := map[string]any{
			"Entries": rand.Int32(),
		}

		// Execute only the leaderboard-table template, not the full page
		err := tpl.ExecuteTemplate(w, "leaderboard-table", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
