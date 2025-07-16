package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"github.com/joho/godotenv"
	"github.com/sceptix-club/atlus/Backend/handlers"
)


func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Unable to load .env")
	}


	mux := http.NewServeMux()
	tpl := template.Must(template.ParseFiles("static/index.html"))
	conf := handlers.InitOAuthConfig()
	lf := handlers.LoginFlow{Conf: conf}

	
	mux.HandleFunc("/",  handlers.RootHandler(tpl))
	mux.HandleFunc("/login/", lf.GithubLoginHandler)
	mux.HandleFunc("/github/callback/", lf.GithubCallbackHandler)
	mux.HandleFunc("/puzzles/{slug}",handlers.LevelHandler(tpl))
	mux.HandleFunc("/inputs/{slug}", handlers.InputHandler)

	addr := ":8000"
	fmt.Printf("Listening on localhost%s ...\n",addr)
	log.Panic(http.ListenAndServe(":8000", mux))
}
