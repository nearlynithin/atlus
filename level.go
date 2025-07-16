package main

import (
	"bytes"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/yuin/goldmark"
)



func levelHandler(tpl *template.Template) http.HandlerFunc {
	return func (w http.ResponseWriter, r* http.Request) {
		var loggedIn bool
		var user string

		if c, err := r.Cookie("session"); err == nil {
			user = sessions[c.Value]
			if user != "" {
				loggedIn = true
			}
		}

		if !loggedIn {
			http.Redirect(w,r,"/login", http.StatusSeeOther)
			return
		}

		slug := r.PathValue("slug")
		fileName := "./puzzles/"+slug+"/"+slug+".md"
		file , err := os.Open(fileName)
		if err != nil {
			log.Panic("file was not found",fileName)
		}
		defer file.Close()

		b, err := io.ReadAll(file)
		if err != nil {
			log.Panic("can't read the file")	
		}
		var buf bytes.Buffer
		if err := goldmark.Convert(b,&buf); err != nil {
			log.Panic("Cannot read markdown")
		}

		err = tpl.Execute(w, map[string]any{
			"LoggedIn" : loggedIn,
			"User": user,
			"Content": template.HTML(buf.String()),
		})
	}
}

func inputHandler(w http.ResponseWriter, r* http.Request) {
	var user string
	var loggedIn bool
	
	if c, err := r.Cookie("session"); err == nil {
		user = sessions[c.Value]
		if user != "" {
			loggedIn = true
		}
	}

	if !loggedIn {
		http.Error(w,"session invalid, please login with a github account as everyone gets a different input", 401)
		return
	}

	// we need to fetch the userNo from the db using sessionID
	userNo := 1

	// slug = level number
	slug := r.PathValue("slug")
	fileName := "./puzzles/"+slug+"/inputs/"+strconv.Itoa(userNo)+".txt"
	file , err := os.Open(fileName)
	if err != nil {
			log.Panic("file was not found",fileName)
	}

	defer file.Close()
	b, err := io.ReadAll(file)
	if err != nil {
		log.Panic("can't read the file")	
	}
	io.Copy(w, bytes.NewBuffer(b))
}