package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/sceptix-club/atlus/Backend/globals"
)

func InputHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Cache-Control", "public, max-age=3600")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	slug := r.PathValue("slug")
	level, err := getLevelParam(slug)
	if err != nil {
		http.Error(w, "Invalid url request", http.StatusBadRequest)
		return
	}

	sdata := ctx.Value("sessionData").(globals.SessionData)

	if sdata.NextReleaseLevel <= level {
		http.Error(w, "Level is not released yet!", http.StatusForbidden)
		return
	}

	if level > sdata.CurrentLevel {
		http.Error(w, fmt.Sprintf("Level not unlocked yet, please complete level%d first", sdata.CurrentLevel), http.StatusForbidden)
		return
	}

	fileName := "./puzzles/" + slug + "/inputs/" + strconv.Itoa(sdata.InputID) + ".txt"
	file, err := os.Open(fileName)
	if err != nil {
		log.Panic("file was not found", fileName)
	}

	defer file.Close()
	b, err := io.ReadAll(file)
	if err != nil {
		log.Panic("can't read the file")
	}
	io.Copy(w, bytes.NewBuffer(b))
}

func getLevelParam(slug string) (int, error) {
	if !strings.HasPrefix(slug, "level") {
		return 0, errors.New("Invalid prefix on the slug")
	}

	levelStr := strings.TrimPrefix(slug, "level")
	level, err := strconv.Atoi(levelStr)
	if err != nil {
		return 0, err
	}

	return level, nil
}
