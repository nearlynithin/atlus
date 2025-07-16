package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Unable to load .env")
	}

	conf := initOAuthConfig()
	lf := &loginFlow{
		conf: conf,
	}

	mux := http.NewServeMux()
	tpl := template.Must(template.ParseFiles("static/index.html"))
	
	mux.HandleFunc("/", rootHandler(tpl))
	mux.HandleFunc("/login/", lf.githubLoginHandler)
	mux.HandleFunc("/github/callback/", lf.githubCallbackHandler)
	mux.HandleFunc("/puzzles/{slug}",levelHandler(tpl))
	mux.HandleFunc("/inputs/{slug}", inputHandler)

	addr := ":8000"
	fmt.Printf("Listening on localhost%s ...\n",addr)
	log.Panic(http.ListenAndServe(":8000", mux))
}
