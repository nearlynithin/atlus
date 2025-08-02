package handlers

import (
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

	"github.com/sceptix-club/atlus/Backend/globals"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type LoginFlow struct {
	Conf *oauth2.Config
}

func InitOAuthConfig() *oauth2.Config {

	var GithubClientID = os.Getenv("GITHUB_CLIENT_ID")
	var GithubClientSecret = os.Getenv("GITHUB_CLIENT_SECRET")

	if len(GithubClientID) == 0 || len(GithubClientSecret) == 0 {
		log.Fatal("client id and secret not initialized")
	}

	Conf := &oauth2.Config{
		ClientID:     GithubClientID,
		ClientSecret: GithubClientSecret,
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
		RedirectURL:  fmt.Sprintf("http://%s:%s/github/callback/", globals.Hostname, globals.Port),
	}

	return Conf
}

func RootHandler(tpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var sdata globals.SessionData

		var loggedIn bool

		if c, err := r.Cookie("session"); err == nil {
			sdata, err = getSessionData(ctx, c.Value)
			if err != nil {
				loggedIn = false
			} else {
				fmt.Printf("\nFETCHED USER: %s\n", sdata.Username)
				loggedIn = true
			}
		}

		levels := make([]int, sdata.CurrentLevel)
		for i := range levels {
			levels[i] = i + 1
		}

		err := tpl.ExecuteTemplate(w, "base", map[string]any{
			"Home":     true,
			"LoggedIn": loggedIn,
			"Username": sdata.Username,
			"Levels":   levels,
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (Lf *LoginFlow) GithubLoginHandler(w http.ResponseWriter, r *http.Request) {
	state := GenerateSessionID()
	c := &http.Cookie{
		Name:     "state",
		Value:    state,
		Path:     "/",
		MaxAge:   int(time.Hour.Seconds()),
		Secure:   r.TLS != nil,
		HttpOnly: true,
	}
	http.SetCookie(w, c)
	fmt.Printf("State set %s", state)

	redirectURL := Lf.Conf.AuthCodeURL(state, oauth2.AccessTypeOnline)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (Lf *LoginFlow) GithubCallbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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
	tok, err := Lf.Conf.Exchange(ctx, code)
	if err != nil {
		fmt.Printf("failed to exchange code: %v", err)
		return
	}

	client := Lf.Conf.Client(ctx, tok)

	githubUser := GetGithubUserInfo(client)

	// check for existing user
	existingUser, err := fetchUserByGithubID(ctx, githubUser.Github_id)
	if err != nil {
		// user not found
		githubUser.SessionToken = GenerateSessionID()
		err = addUser(ctx, githubUser)
		if err != nil {
			http.Error(w, "Failed to add user", http.StatusInternalServerError)
			return
		}
	} else {
		// user was found, update a new session Token
		githubUser = existingUser
		githubUser.SessionToken = GenerateSessionID()
		err := updateSessionToken(ctx, githubUser.Github_id, githubUser.SessionToken)
		if err != nil {
			http.Error(w, "failed to update session token", http.StatusInternalServerError)
			return
		}
	}

	stateClear := &http.Cookie{
		Name:     "state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}
	http.SetCookie(w, stateClear)

	c := &http.Cookie{
		Name:     "session",
		Value:    githubUser.SessionToken,
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 30, // 30 days
		HttpOnly: true,
	}
	http.SetCookie(w, c)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func GithubLogoutHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	c, err := r.Cookie("session")
	if err != nil {
		log.Print(err.Error())
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	sessionID := c.Value

	err = deleteSessionToken(ctx, sessionID)
	if err != nil {
		log.Printf("Unable to delete session")
		log.Print(err.Error())
	}
	sessionClear := &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}
	http.SetCookie(w, sessionClear)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func GetGithubUserInfo(client *http.Client) globals.User {

	var user globals.User
	var emails []globals.Email

	res, err := client.Get("https://api.github.com/user")
	if err != nil {
		panic(err)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal("Failed to read response body:", err)
	}

	if err := json.Unmarshal(body, &user); err != nil {
		log.Fatal("Failed to parse the login body", err)
	}

	emailRes, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		log.Fatal("Failed to get emails ", err)
	}
	defer emailRes.Body.Close()

	emailBody, err := io.ReadAll(emailRes.Body)
	if err != nil {
		log.Fatal("Failed to read email body ", err)
	}

	if err := json.Unmarshal(emailBody, &emails); err != nil {
		log.Fatal("Failed to parse email list ", err)
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			user.Email = e.Email
			break
		}
	}
	user.SessionToken = GenerateSessionID()
	return user
}

func GenerateSessionID() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		log.Fatal("Failed to generate session ID")
	}
	return hex.EncodeToString(b)
}
