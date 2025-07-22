package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

func SubmitAnswerHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Extract session cookie
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "Session cookie not found", http.StatusUnauthorized)
		return
	}
	sessionID := cookie.Value

	slug := r.PathValue("slug")
	level, err := getLevelParam(slug)
	if err != nil {
		http.Error(w,"Invalid url request", http.StatusBadRequest)
		return
	}
	newSlug := fmt.Sprintf("level%d",level)

	// Get session info from DB
	sdata, err := getSessionData(ctx, sessionID)
	if err != nil {
		http.Error(w, "Invalid or expired session", http.StatusUnauthorized)
		return
	}

	// Get submitted answer
	answer := strings.TrimSpace(r.FormValue("answer"))
	if answer == "" {
		http.Error(w, "Answer cannot be empty", http.StatusBadRequest)
		return
	}

	// Construct path: data/{level}/answers/{input_id}.txt
	answerFile := fmt.Sprintf("./puzzles/%s/outputs/%d.txt", newSlug, sdata.InputID)
	correctBytes, err := os.ReadFile(answerFile)
	if err != nil {
		http.Error(w, "Correct answer file not found", http.StatusInternalServerError)
		return
	}
	solution  := strings.TrimSpace(string(correctBytes))

	// Compare answers
	if answer == solution {
		err = updateUserLevel(ctx, sessionID, level, true)
		
		if err != nil {
			log.Println(err.Error())
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/puzzles/level%d", level+1), http.StatusPermanentRedirect)

	} else {
		fmt.Fprintf(w, "Incorrect answer. You are still on level %d. Try again!", sdata.CurrentLevel)
	}
}
