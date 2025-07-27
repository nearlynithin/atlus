package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	pgx "github.com/jackc/pgx/v5"
	"github.com/sceptix-club/atlus/Backend/globals"
)

func SubmitAnswerHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	var submissionData globals.SubmissionData

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
		http.Error(w, "Invalid url request", http.StatusBadRequest)
		return
	}
	newSlug := fmt.Sprintf("level%d", level)

	// Get session info from DB
	sdata, err := getSessionData(ctx, sessionID)
	if err != nil {
		http.Error(w, "Invalid or expired session", http.StatusUnauthorized)
		return
	}

	if sdata.NextReleaseLevel <= level {
		http.Error(w, "Level is not released yet!", http.StatusForbidden)
		return
	}

	if level > sdata.CurrentLevel {
		http.Error(w, fmt.Sprintf("Level not unlocked yet, please complete level%d first", sdata.CurrentLevel), http.StatusForbidden)
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
	solution := strings.TrimSpace(string(correctBytes))

	submissionData.CurrentLevel = sdata.CurrentLevel
	submissionData.GithubID = sdata.GithubID
	submissionData.Username = sdata.Username
	submissionData.PuzzleLevel = level

	// Compare answers
	if answer == solution {
		submissionData.Pass = true
		err = updateUserLevel(ctx, submissionData)

		if err != nil {
			log.Println(err.Error())
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/puzzles/level%d", level+1), http.StatusSeeOther)

	} else {
		submissionData.Pass = false
		err = updateUserLevel(ctx, submissionData)

		if err != nil {
			log.Println(err.Error())
			return
		}

		fmt.Fprintf(w, "Incorrect answer. You are still on level %d. Try again!", sdata.CurrentLevel)
	}
}

func updateUserLevel(ctx context.Context, submissionData globals.SubmissionData) error {

	if !submissionData.Pass {
		return updateSubmission(ctx, submissionData)
	}

	if submissionData.PuzzleLevel > submissionData.CurrentLevel {
		return fmt.Errorf("Level %d not completed yet", submissionData.CurrentLevel)
	}

	if submissionData.PuzzleLevel == submissionData.CurrentLevel {
		// user passed the currentLevel which he is at
		// so update current_level in the db
		_, err := globals.DB.Exec(ctx, `
			UPDATE users set current_level = $1
			WHERE github_id = $2
		`, submissionData.CurrentLevel+1, submissionData.GithubID)

		fmt.Printf("LEVEL UPDATED")

		if err != nil {
			log.Printf("Error advancing player level %s\n", err.Error())
			return err
		}
		updateSubmission(ctx, submissionData)
	}
	return nil
}

func updateSubmission(ctx context.Context, submissionData globals.SubmissionData) error {
	var attempts int

	err := globals.DB.QueryRow(ctx, `
            INSERT INTO submissions (github_id, username, level_id, last_submission, attempts)
            VALUES ($1, $2, $3, NOW(), 1)
            ON CONFLICT (github_id, level_id)
            DO UPDATE SET
                last_submission = NOW(),
                attempts = submissions.attempts + 1
                WHERE submissions.passed = FALSE
            RETURNING attempts;
        `, submissionData.GithubID, submissionData.Username, submissionData.PuzzleLevel).Scan(&attempts)

	if err != nil {
		fmt.Printf("error inserting into submissions, %v", err)
		if err == pgx.ErrNoRows {
			fmt.Printf("level is already passed")
			return nil
		}
		return fmt.Errorf("Error inserting/updating submission for the user %s, %v", submissionData.Username, err)
	}

	if submissionData.Pass {
		// couting a streak if attempts = 1 and the user passed the level
		if attempts == 1 {
			_, err := globals.DB.Exec(ctx, `
                UPDATE users SET
                streak = streak + 1
                WHERE github_id = $1
                `, submissionData.GithubID)
			if err != nil {
				return fmt.Errorf("errror updating streak %v", err)
			}
		}

		result, err := globals.DB.Exec(ctx, `
		UPDATE submissions AS s
		SET time_taken = s.last_submission - l.release_time,
		passed = TRUE
		FROM levels l
		WHERE s.github_id = $1
		AND s.level_id = $2
		AND s.passed = FALSE
		AND s.level_id = l.level_id
	`, submissionData.GithubID, submissionData.PuzzleLevel)

		if err != nil {
			return fmt.Errorf("Error updating time_taken: %v", err)
		}

		rowsAffected := result.RowsAffected()

		if rowsAffected == 0 {
			log.Printf("User %s already has time_taken set for level %d — no update needed.\n", submissionData.Username, submissionData.PuzzleLevel)
			return nil
		}
	} else if attempts == 1 {
		// user had failed on first attempt, set the streak to zero
		_, err := globals.DB.Exec(ctx, `
                UPDATE users SET
                streak = 1
                WHERE github_id = $1
                `, submissionData.GithubID)
		if err != nil {
			return fmt.Errorf("Error setting streak to zero : %v", err)
		}
	}

	return nil
}
