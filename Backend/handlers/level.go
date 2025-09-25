package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/sceptix-club/atlus/Backend/globals"
	"github.com/yuin/goldmark"
)

func LevelHandler(tpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		slug := r.PathValue("slug")
		level, err := getLevelParam(slug)
		if err != nil {
			globals.RenderInfoPage(tpl, w, true, map[string]any{
				"InvalidRequest": true,
			})
			return
		}

		sdata := ctx.Value("sessionData").(globals.SessionData)

		if sdata.NextReleaseLevel <= level {
			globals.RenderInfoPage(tpl, w, true, map[string]any{
				"NotReleased": true,
				"NextLevel":   sdata.NextReleaseLevel,
			})
			return
		}

		if level > sdata.CurrentLevel {
			globals.RenderInfoPage(tpl, w, true, map[string]any{
				"Locked":       true,
				"CurrentLevel": sdata.CurrentLevel,
			})
			return
		}

		newSlug := fmt.Sprintf("level%d", level)
		filePath := "./puzzles/" + newSlug + "/" + newSlug + ".md"
		file, err := os.Open(filePath)
		if err != nil {
			globals.RenderInfoPage(tpl, w, true, map[string]any{
				"Unexpected": true,
			})
			log.Printf("failed to open puzzle file %s: %v", filePath, err)
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
			"LoggedIn": true,
			"Slug":     newSlug,
			"Puzzle":   template.HTML(buf.String()),
		})
	}
}
