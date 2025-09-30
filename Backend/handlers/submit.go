package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	pgx "github.com/jackc/pgx/v5"
	"github.com/sceptix-club/atlus/Backend/globals"
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
			http.Redirect(w, r, fmt.Sprintf("/puzzles/level%d", level), http.StatusSeeOther)
			return
		}

		if level > sdata.CurrentLevel {
			http.Redirect(w, r, fmt.Sprintf("/puzzles/level%d", level), http.StatusSeeOther)
			return
		}

		// Get submitted answer
		answer := strings.TrimSpace(r.FormValue("answer"))
		if answer == "" {
			globals.RenderInfoPage(tpl, w, true, map[string]any{
				"Unexpected": true,
			})
			log.Printf("Answer cannot be empty: %v\n", err)
			return
		}

		answerFile := fmt.Sprintf("./puzzles/%s/problem_set/%d.json", newSlug, sdata.InputID)
		b, err := os.ReadFile(answerFile)
		if err != nil {
			globals.RenderInfoPage(tpl, w, true, map[string]any{
				"Unexpected": true,
			})
			log.Printf("Correct answer file not found! %v\n", err)
			return
		}
		var problemSet globals.ProblemSet
		json.Unmarshal(b, &problemSet)

		submissionData.CurrentLevel = sdata.CurrentLevel
		submissionData.GithubID = sdata.GithubID
		submissionData.Username = sdata.Username
		submissionData.PuzzleLevel = level

		// Compare answers
		if answer == strings.TrimSpace(problemSet.Output) {
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
			case globals.Cooldown:
				globals.RenderInfoPage(tpl, w, true, map[string]any{
					"Cooldown": true,
				})
				return
			case globals.AlreadyPassed:
				http.Redirect(w, r, fmt.Sprintf("/puzzles/level%d", submissionData.CurrentLevel), http.StatusSeeOther)
				return
			case globals.LevelPassed:
				globals.RenderInfoPage(tpl, w, true, map[string]any{
					"Passed":    true,
					"NextLevel": level + 1,
				})
				return
			case globals.LevelFailed:
				globals.RenderInfoPage(tpl, w, true, map[string]any{
					"Failed": true,
					"Level":  level,
				})
				return
			case globals.SubmissionError:
				globals.RenderInfoPage(tpl, w, true, map[string]any{
					"Unexpected": true,
				})
			default:
				log.Printf("unknown error: %v", err)
				fmt.Fprintf(w, "an error occured, please try again.")
				return
			}
		}
	}
}

func updateUserAttempt(ctx context.Context, submissionData globals.SubmissionData) (globals.SubmissionStatus, error) {
	tx, err := globals.DB.Begin(ctx)
	if err != nil {
		return globals.SubmissionError, fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	if submissionData.Pass {
		if submissionData.PuzzleLevel > submissionData.CurrentLevel {
			return globals.LevelIncomplete, fmt.Errorf("Level %d not completed yet", submissionData.CurrentLevel)
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
				return globals.SubmissionError, fmt.Errorf("error advancing user level : %v", err)
			}
			log.Printf("Advanced user %s to level %d", submissionData.Username, submissionData.CurrentLevel+1)
		}
		return submissionTx(ctx, tx, submissionData)
	} else {
		return submissionTx(ctx, tx, submissionData)
	}
}

func submissionTx(ctx context.Context, tx pgx.Tx, submissionData globals.SubmissionData) (globals.SubmissionStatus, error) {

	var hasPassed bool
	var cooldown bool

	cooldownSteps := 15 // in minutes

	err := tx.QueryRow(ctx, `
        SELECT
            COALESCE(passed, FALSE),
            (cooldown IS NOT NULL AND cooldown >= NOW()) AS cooldown_active
        FROM submissions
        WHERE github_id = $1 AND level_id = $2
        `, submissionData.GithubID, submissionData.PuzzleLevel).Scan(&hasPassed, &cooldown)

	if err != nil && err != pgx.ErrNoRows {
		return globals.SubmissionError, fmt.Errorf("Error checking existing submission: %v", err)
	}

	if hasPassed {
		return globals.AlreadyPassed, nil
	}

	// cooldown not complete yet
	if cooldown {
		return globals.Cooldown, nil
	}

	var attempts int
	err = tx.QueryRow(ctx, `
            INSERT INTO submissions (github_id, username, level_id, last_submission, attempts, cooldown)
            VALUES ($1, $2, $3, NOW(), 1, NOW())
            ON CONFLICT (github_id, level_id)
            DO UPDATE SET
                last_submission = NOW(),
                attempts = submissions.attempts + 1,
                cooldown = NOW() + (FLOOR((submissions.attempts + 1) / 3.0) * $4 * INTERVAL '1 minute')
            RETURNING attempts
        `, submissionData.GithubID, submissionData.Username, submissionData.PuzzleLevel, cooldownSteps).Scan(&attempts)
	if err != nil {
		return globals.SubmissionError, fmt.Errorf("error inserting/updating submission: %v", err)
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
			return globals.SubmissionError, fmt.Errorf("error updating submission as passed, %v", err)
		}

		if attempts == 1 {
			_, err := tx.Exec(ctx, `
                UPDATE users SET
                streak = streak + 1
                WHERE github_id = $1 `, submissionData.GithubID)
			if err != nil {
				return globals.SubmissionError, fmt.Errorf("errror updating streak %v", err)
			}
		}
		err = tx.Commit(ctx)
		if err != nil {
			return globals.SubmissionError, fmt.Errorf("failed to commit transaction: %v", err)
		}
		return globals.LevelPassed, nil
	} else {
		// user had failed on first attempt, set the streak to zero
		_, err := tx.Exec(ctx, `
                UPDATE users SET
                streak = 0
                WHERE github_id = $1
                `, submissionData.GithubID)
		if err != nil {
			return globals.SubmissionError, fmt.Errorf("Error setting streak : %v", err)
		}
		err = tx.Commit(ctx)
		if err != nil {
			return globals.SubmissionError, fmt.Errorf("failed to commit transaction: %v", err)
		}
		return globals.LevelFailed, nil
	}
}
