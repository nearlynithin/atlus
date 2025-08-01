package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/yuin/goldmark"
)

func LevelHandler(tpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		slug := r.PathValue("slug")
		level, err := getLevelParam(slug)
		if err != nil {
			http.Error(w, "Invalid url request", http.StatusBadRequest)
			return
		}

		c, err := r.Cookie("session")
		if err != nil {
			http.Redirect(w, r, "/login/", http.StatusSeeOther)
			log.Print("session invalid ", http.StatusUnauthorized)
			return
		}

		sdata, err := getSessionData(ctx, c.Value)
		if err != nil {
			http.Redirect(w, r, "/login/", http.StatusSeeOther)
			log.Print("Please login to play", http.StatusUnauthorized)
			return
		}

		loggedIn := true

		if sdata.NextReleaseLevel <= level {
			http.Error(w, "Level is not released yet!", http.StatusForbidden)
			return
		}

		if level > sdata.CurrentLevel {
			http.Error(w, fmt.Sprintf("Level not unlocked yet, please complete level%d first", sdata.CurrentLevel), http.StatusForbidden)
			return
		}

		newSlug := fmt.Sprintf("level%d", level)
		filePath := "./puzzles/" + newSlug + "/" + newSlug + ".md"
		file, err := os.Open(filePath)
		if err != nil {
			log.Printf("LevelHandler: failed to open file %s: %v", filePath, err)
			http.Error(w, "Puzzle file not found", http.StatusNotFound)
			return
		}
		defer file.Close()

		b, err := io.ReadAll(file)
		if err != nil {
			log.Panic("can't read the file")
		}
		var buf bytes.Buffer
		if err := goldmark.Convert(b, &buf); err != nil {
			log.Panic("Cannot read markdown")
		}

		tpl.ExecuteTemplate(w, "level", map[string]any{
			"Level":    true,
			"LoggedIn": loggedIn,
			"Slug":     newSlug,
			"Puzzle":   template.HTML(buf.String()),
		})
	}
}
