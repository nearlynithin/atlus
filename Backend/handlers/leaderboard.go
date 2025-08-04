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
	UserStats    = "stats"
)

type leaderboardFunc func(ctx context.Context) (map[string]any, error)

func LeaderboardHandler(tpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ctx := r.Context()
		// sdata := ctx.Value("sessionData").(globals.SessionData)
		tpl.ExecuteTemplate(w, "leaderboard", map[string]any{
			"LoggedIn":    true,
			"Leaderboard": true,
		})
	}
}

func LeaderboardLiveHandler(tpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		username := r.URL.Query().Get("user")

		ctx := r.Context()
		ctx = context.WithValue(ctx, "user", username)

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
	case UserStats:
		return userStatsHandler, nil
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

func userStatsHandler(ctx context.Context) (map[string]any, error) {
	res := map[string]any{}
	username := ctx.Value("user")
	if username == "" {
		return res, fmt.Errorf("error reading user")
	}

	type Record struct {
		LevelId   string
		TimeTaken time.Duration
		Attempts  int
	}

	var data []Record

	rows, err := globals.DB.Query(ctx, `
	    SELECT level_id, time_taken, attempts FROM submissions
	    WHERE username = $1
	    AND passed = TRUE
	    ORDER BY time_taken ASC
	    `, username)

	if err != nil {
		log.Printf("error fetching the Champion leaderboard, %v", err)
		return res, err
	}
	defer rows.Close()

	for rows.Next() {
		var row Record
		err := rows.Scan(&row.LevelId, &row.TimeTaken, &row.Attempts)
		if err != nil {
			log.Printf("error scanning the row for stats, %v", err)
			return res, err
		}
		data = append(data, row)
	}

	res = map[string]any{
		"UserStats": data,
	}
	return res, nil
}
