package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/sceptix-club/atlus/Backend/globals"
	"github.com/sceptix-club/atlus/Backend/handlers"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Unable to load .env")
	}
	globals.Hostname = os.Getenv("HOSTNAME")
	globals.Port = os.Getenv("PORT")

	handlers.InitDB()
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	tpl := template.Must(template.ParseGlob("static/*.html"))
	conf := handlers.InitOAuthConfig()
	lf := handlers.LoginFlow{Conf: conf}

	mux.HandleFunc("/", handlers.RootHandler(tpl))
	mux.HandleFunc("/login/", lf.GithubLoginHandler)
	mux.HandleFunc("/logout/", handlers.GithubLogoutHandler)
	mux.HandleFunc("/github/callback/", lf.GithubCallbackHandler)
	mux.HandleFunc("/puzzles/{slug}", handlers.LevelHandler(tpl))
	mux.HandleFunc("/inputs/{slug}", handlers.InputHandler)
	mux.HandleFunc("/submitAnswer/{slug}", handlers.SubmitAnswerHandler(tpl))
	mux.HandleFunc("/leaderboard/", handlers.LeaderboardHandler(tpl))
	mux.HandleFunc("/leaderboard/live/{slug}", handlers.LeaderboardLiveHandler(tpl))
	mux.HandleFunc("/profile", handlers.ProfileHandler(tpl))

	fmt.Printf("Listening on %s%s ...\n", globals.Hostname, globals.Port)
	log.Panic(http.ListenAndServe(":"+globals.Port, mux))
}
