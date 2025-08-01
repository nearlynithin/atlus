package handlers

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/sceptix-club/atlus/Backend/globals"
)

const (
	StreakRune   = "streak"
	FlashRune    = "flash"
	ChampionRune = "champion"
)

type leaderboardFunc func(ctx context.Context) (map[string]any, error)

func LeaderboardHandler(tpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		c, err := r.Cookie("session")
		if err != nil {
			http.Redirect(w, r, "/login/", http.StatusSeeOther)
			fmt.Fprintf(w, "Invalid session, please login to continue : %v", err)
			return
		}
		_, err = getSessionData(ctx, c.Value)
		if err != nil {
			http.Redirect(w, r, "/login/", http.StatusSeeOther)
			return
		}
		tpl.ExecuteTemplate(w, "leaderboard", map[string]any{
			"LoggedIn":    true,
			"Leaderboard": true,
		})
	}
}

func LeaderboardLiveHandler(tpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		slug := r.PathValue("slug")

		handler, err := runeHandler(slug)
		if err != nil {
			http.Error(w, "Unknown leaderboard", http.StatusNotFound)
			return
		}

		data, err := handler(ctx)
		err = tpl.ExecuteTemplate(w, "leaderboard-table", data)
		if err != nil {
			log.Printf("error executing the template, %v", err)
			return
		}
	}
}

func runeHandler(slug string) (leaderboardFunc, error) {
	switch slug {
	case StreakRune:
		return streakHandler, nil
	case FlashRune:
		return flashHandler, nil
	case ChampionRune:
		return championHandler, nil
	}
	return nil, nil
}

func streakHandler(ctx context.Context) (map[string]any, error) {

	res := map[string]any{}

	type Record struct {
		Username  string
		Streak    int
		GithubUrl string
	}

	var data []Record

	rows, err := globals.DB.Query(ctx, `
            SELECT username, streak, github_url FROM users
            ORDER BY streak DESC
            LIMIT 10
	    `)
	if err != nil {
		log.Printf("error fetching the streak leaderboard, %v", err)
		return res, err
	}
	defer rows.Close()

	for rows.Next() {
		var row Record
		err := rows.Scan(&row.Username, &row.Streak, &row.GithubUrl)
		if err != nil {
			log.Printf("error scanning the row for streaks, %v", err)
			return res, err
		}
		data = append(data, row)
	}

	if err := rows.Err(); err != nil {
		log.Printf("iteration error: %v", err)
		return nil, err
	}

	res = map[string]any{
		"StreakRune": data,
	}
	return res, nil
}

func flashHandler(ctx context.Context) (map[string]any, error) {
	res := map[string]any{}

	type Record struct {
		LevelId   string
		Username  string
		TimeTaken time.Duration
	}

	var data []Record

	rows, err := globals.DB.Query(ctx, `
	    SELECT DISTINCT ON (level_id) level_id, username, time_taken
	    FROM submissions
	    WHERE passed = TRUE
	    ORDER BY level_id, time_taken ASC
	    LIMIT 10
	    `)

	if err != nil {
		log.Printf("error fetching the Flash leaderboard, %v", err)
		return res, err
	}
	defer rows.Close()

	for rows.Next() {
		var row Record
		err := rows.Scan(&row.LevelId, &row.Username, &row.TimeTaken)
		if err != nil {
			log.Printf("error scanning the row for flash, %v", err)
			return res, err
		}
		data = append(data, row)
	}

	if err := rows.Err(); err != nil {
		log.Printf("iteration error: %v", err)
		return nil, err
	}

	res = map[string]any{
		"FlashRune": data,
	}
	return res, nil
}

func championHandler(ctx context.Context) (map[string]any, error) {
	res := map[string]any{}

	type Record struct {
		Username     string
		CurrentLevel int
		GithubUrl    string
	}

	var data []Record

	rows, err := globals.DB.Query(ctx, `
	    SELECT username, current_level, github_url FROM users
	    ORDER BY current_level DESC
	    LIMIT 10
	    `)

	if err != nil {
		log.Printf("error fetching the Champion leaderboard, %v", err)
		return res, err
	}
	defer rows.Close()

	for rows.Next() {
		var row Record
		err := rows.Scan(&row.Username, &row.CurrentLevel, &row.GithubUrl)
		if err != nil {
			log.Printf("error scanning the row for Champion, %v", err)
			return res, err
		}
		data = append(data, row)
	}

	if err := rows.Err(); err != nil {
		log.Printf("iteration error: %v", err)
		return nil, err
	}

	res = map[string]any{
		"ChampionRune": data,
	}
	return res, nil
}
