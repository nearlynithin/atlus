package handlers

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	pgx "github.com/jackc/pgx/v5"
	"github.com/sceptix-club/atlus/Backend/globals"
)

type SubmissionStatus string

const (
	AlreadyPassed   SubmissionStatus = "already_passed"
	LevelIncomplete SubmissionStatus = "level_incomplete"
	LevelPassed     SubmissionStatus = "level_passed"
	LevelFailed     SubmissionStatus = "level_failed"
	Cooldown        SubmissionStatus = "cooldown"
	SubmissionError SubmissionStatus = "submission_error"
)

func SubmitAnswerHandler(tpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only allow POST requests
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx := r.Context()
		var submissionData globals.SubmissionData

		slug := r.PathValue("slug")
		level, err := getLevelParam(slug)
		if err != nil {
			http.Error(w, "Invalid url request", http.StatusBadRequest)
			return
		}
		newSlug := fmt.Sprintf("level%d", level)

		// Get session info from DB
		sdata := ctx.Value("sessionData").(globals.SessionData)

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
		} else {
			submissionData.Pass = false
		}
		status, err := updateUserAttempt(ctx, submissionData)
		if err != nil {
			log.Printf("unknown error: %v", err)
			fmt.Fprintf(w, "an error occured, please try again.")
			return
		} else {
			switch status {
			case Cooldown:
				tpl.ExecuteTemplate(w, "base", map[string]any{
					"LoggedIn": true,
					"Info":     true,
					"Cooldown": true,
				})
				return
			case AlreadyPassed:
				http.Redirect(w, r, fmt.Sprintf("/puzzles/level%d", submissionData.CurrentLevel), http.StatusSeeOther)
				return
			case LevelPassed:
				tpl.ExecuteTemplate(w, "base", map[string]any{
					"LoggedIn":  true,
					"Info":      true,
					"Passed":    true,
					"NextLevel": level + 1,
				})
				// http.Redirect(w, r, fmt.Sprintf("/puzzles/level%d", level+1), http.StatusSeeOther)
				return
			case LevelFailed:
				tpl.ExecuteTemplate(w, "base", map[string]any{
					"LoggedIn": true,
					"Info":     true,
					"Failed":   true,
					"Level":    level,
				})
				return
			default:
				log.Printf("unknown error: %v", err)
				fmt.Fprintf(w, "an error occured, please try again.")
				return
			}
		}
	}
}

func updateUserAttempt(ctx context.Context, submissionData globals.SubmissionData) (SubmissionStatus, error) {
	tx, err := globals.DB.Begin(ctx)
	if err != nil {
		return SubmissionError, fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	if submissionData.Pass {
		if submissionData.PuzzleLevel > submissionData.CurrentLevel {
			return LevelIncomplete, fmt.Errorf("Level %d not completed yet", submissionData.CurrentLevel)
		}
		if submissionData.PuzzleLevel == submissionData.CurrentLevel {
			// user passed the currentLevel which he is at
			// so update current_level in the db
			_, err := tx.Exec(ctx, `
			UPDATE users SET
			current_level = $1
			WHERE github_id = $2
		`, submissionData.CurrentLevel+1, submissionData.GithubID)

			if err != nil {
				return SubmissionError, fmt.Errorf("error advancing user level : %v", err)
			}
			log.Printf("Advanced user %s to level %d", submissionData.Username, submissionData.CurrentLevel+1)
		}
		return submissionTx(ctx, tx, submissionData)
	} else {
		return submissionTx(ctx, tx, submissionData)
	}
}

func submissionTx(ctx context.Context, tx pgx.Tx, submissionData globals.SubmissionData) (SubmissionStatus, error) {

	var existingAttempts int
	var hasPassed bool
	var cooldownEnd time.Time

	err := tx.QueryRow(ctx, `
        SELECT COALESCE(attempts, 0), COALESCE(passed, FALSE), COALESCE(cooldown, NOW())
        FROM submissions
        WHERE github_id = $1 AND level_id = $2
        `, submissionData.GithubID, submissionData.PuzzleLevel).Scan(&existingAttempts, &hasPassed, &cooldownEnd)

	if err != nil && err != pgx.ErrNoRows {
		return SubmissionError, fmt.Errorf("Error checking existing submission: %v", err)
	}

	if hasPassed {
		return AlreadyPassed, nil
	}

	// cooldown not complete yet
	if time.Now().UTC().Before(cooldownEnd) {
		return Cooldown, nil
	}

	var newAttempts int
	err = tx.QueryRow(ctx, `
            INSERT INTO submissions (github_id, username, level_id, last_submission, attempts)
            VALUES ($1, $2, $3, NOW(), 1)
            ON CONFLICT (github_id, level_id)
            DO UPDATE SET
                last_submission = NOW(),
                attempts = submissions.attempts + 1
            RETURNING attempts
        `, submissionData.GithubID, submissionData.Username, submissionData.PuzzleLevel).Scan(&newAttempts)
	if err != nil {
		return SubmissionError, fmt.Errorf("error inserting/updating submission: %v", err)
	}

	if newAttempts%3 == 0 {
		cooldownMinutes := (newAttempts / 3) * 15
		cooldownDuration := fmt.Sprintf("%d minutes", int(cooldownMinutes))

		_, err := tx.Exec(ctx, `
            UPDATE submissions SET
            cooldown = NOW() + $1
            WHERE github_id = $2 AND level_id = $3
            `, cooldownDuration, submissionData.GithubID, submissionData.PuzzleLevel)
		if err != nil {
			return SubmissionError, fmt.Errorf("error setting cooldown, %v", err)
		}
	}

	if submissionData.Pass {
		_, err := tx.Exec(ctx, `
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
			return SubmissionError, fmt.Errorf("error updating submission as passed, %v", err)
		}

		if newAttempts == 1 {
			_, err := tx.Exec(ctx, `
                UPDATE users SET
                streak = streak + 1
                WHERE github_id = $1 `, submissionData.GithubID)
			if err != nil {
				return SubmissionError, fmt.Errorf("errror updating streak %v", err)
			}
		}
		err = tx.Commit(ctx)
		if err != nil {
			return SubmissionError, fmt.Errorf("failed to commit transaction: %v", err)
		}
		return LevelPassed, nil
	} else {
		// user had failed on first attempt, set the streak to zero
		_, err := tx.Exec(ctx, `
                UPDATE users SET
                streak = 0
                WHERE github_id = $1
                `, submissionData.GithubID)
		if err != nil {
			return SubmissionError, fmt.Errorf("Error setting streak : %v", err)
		}
		err = tx.Commit(ctx)
		if err != nil {
			return SubmissionError, fmt.Errorf("failed to commit transaction: %v", err)
		}
		return LevelFailed, nil
	}
}
