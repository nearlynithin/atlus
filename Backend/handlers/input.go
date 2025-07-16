package handlers

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"github.com/sceptix-club/atlus/Backend/globals"
)

func InputHandler(w http.ResponseWriter, r *http.Request) {
	var user string
	var loggedIn bool

	if c, err := r.Cookie("session"); err == nil {
		user =globals.Sessions[c.Value]
		if user != "" {
			loggedIn = true
		}
	}

	if !loggedIn {
		http.Error(w, "session invalid, please login with a github account as everyone gets a different input", 401)
		return
	}

	// we need to fetch the userNo from the db using sessionID
	userNo := 1

	// slug = level number
	slug := r.PathValue("slug")
	fileName := "./puzzles/" + slug + "/inputs/" + strconv.Itoa(userNo) + ".txt"
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
