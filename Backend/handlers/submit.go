package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/sceptix-club/atlus/Backend/globals"
)

func SubmitAnswerHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract session cookie
	cookie, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "Session cookie not found", http.StatusUnauthorized)
		return
	}
	sessionID := cookie.Value

	// Get session info from DB
	ctx := context.Background()
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
	answerFile := fmt.Sprintf("data/%d/answers/%d.txt", sdata.CurrentLevel, sdata.InputID)
	correctBytes, err := os.ReadFile(answerFile)
	if err != nil {
		http.Error(w, "Correct answer file not found", http.StatusInternalServerError)
		return
	}
	correctAnswer := strings.TrimSpace(string(correctBytes))

	// Compare answers
	if answer == correctAnswer {
		_, err := globals.DB.Exec(ctx,
			`UPDATE users SET current_level = current_level + 1 WHERE input_id = $1`,
			sdata.InputID,
		)
		if err != nil {
			http.Error(w, "Failed to update level", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "Correct answer! You've progressed to level %d.", sdata.CurrentLevel+1)
	} else {
		fmt.Fprintf(w, "Incorrect answer. You are still on level %d. Try again!", sdata.CurrentLevel)
	}
}
