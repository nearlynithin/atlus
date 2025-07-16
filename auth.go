package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// temporary session ID storage in place of a db
var sessions = map[string]string{}

type loginFlow struct {
	conf *oauth2.Config
}

type User struct {
	Username string `json:"login"`
}

func initOAuthConfig() *oauth2.Config{
	
	var GithubClientID = os.Getenv("GITHUB_CLIENT_ID")
	var GithubClientSecret = os.Getenv("GITHUB_CLIENT_SECRET")
	
	if(len(GithubClientID) == 0 || len(GithubClientSecret) == 0){
		log.Fatal("client id and secret not initialized")
	}
	
	conf := &oauth2.Config{
		ClientID: GithubClientID,
		ClientSecret: GithubClientSecret,
		Scopes: []string{},
		Endpoint: github.Endpoint,
		RedirectURL: "http://localhost:8000/github/callback/",
	}
	
	return conf
}

func rootHandler(tpl * template.Template) http.HandlerFunc {
	return func (w http.ResponseWriter, r *http.Request) {
		var loggedIn bool
		var user string

		if c, err := r.Cookie("session"); err == nil {
			user = sessions[c.Value]
			if user != "" {
				loggedIn = true
			}
		}
		
		tpl.Execute(w, map[string]any{
			"LoggedIn": loggedIn,
			"User": user,
		})
	}
}

func (lf * loginFlow) githubLoginHandler(w http.ResponseWriter, r * http.Request){
	state := generateSessionID()
	c := &http.Cookie{
		Name: "state",
		Value: state,
		Path: "/",
		MaxAge: int(time.Hour.Seconds()),
		Secure: r.TLS != nil,
		HttpOnly: true,
	}
	http.SetCookie(w,c)

	redirectURL := lf.conf.AuthCodeURL(state, oauth2.AccessTypeOnline)
	http.Redirect(w, r, redirectURL, 301)
}


func (lf * loginFlow) githubCallbackHandler(w http.ResponseWriter, r * http.Request){
	state, err := r.Cookie("state")
	if err != nil {
		http.Error(w, "state not found", http.StatusBadRequest)
		return
	}
	
	if r.URL.Query().Get("state") != state.Value {
		http.Error(w, "state did not match", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	tok, err := lf.conf.Exchange(context.Background(), code)
	if err != nil{
		log.Fatal(err)
	}

	client := lf.conf.Client(context.Background(), tok)
	res, err := client.Get("https://api.github.com/user")
	if err != nil {
		panic(err)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal("Failed to read response body:", err)
	}
	fmt.Printf("%s\n", body) // or string(body)

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		log.Fatal("Failed to parse the login body", err)
	}

	sessionID := generateSessionID()
	c := &http.Cookie{
		Name: "session",
		Value: sessionID,
		Path: "/",
		MaxAge: 60 * 60 * 24 * 30, // 30 days
		HttpOnly: true,
	}
	http.SetCookie(w,c)
	sessions[sessionID] = user.Username
	http.Redirect(w,r,"/", http.StatusSeeOther)	
}

func getGithubUserInfo(accessToken string) string {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	resBody, _ := io.ReadAll(res.Body)
	return string(resBody)
}

func generateSessionID() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		log.Fatal("Failed to generate session ID")
	}
	return hex.EncodeToString(b)
}