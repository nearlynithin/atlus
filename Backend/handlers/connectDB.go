package handlers

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/sceptix-club/atlus/Backend/globals"
)

func InitDB() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Failed to connect to Supabase connection pooler: %v", err)
	}

	var version string
	if err := pool.QueryRow(ctx, "SELECT version()").Scan(&version); err != nil {
		log.Fatalf("Query test failed: %v", err)
	}

	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		log.Fatalf("Unable to find schema.sql", err)
	}

	_, err = pool.Exec(ctx, string(schema))
	if err != nil {
		log.Fatalf("Unable to create tables", err)
	} else {
		fmt.Println("Tables created successfully")
	}

	log.Println("Connected via pooler to:", version)
	globals.DB = pool

	forkPtr := flag.Bool("dev", false, "DEV MODE : to truncate all db tables, on startup")
	flag.Parse()
	if *forkPtr {
		globals.DB.Exec(ctx, "TRUNCATE table users, sessions, submissions;")
		fmt.Println("truncated tables users and sessions!")
	}
}

func addUser(ctx context.Context, user globals.User) error {
	var inputID int

	err := globals.DB.QueryRow(ctx,
		`INSERT INTO users (github_id, username, github_url, avatar, email)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (github_id) DO UPDATE SET
			username = EXCLUDED.username,
  			github_url = EXCLUDED.github_url,
  			avatar = EXCLUDED.avatar,
  			email = EXCLUDED.email
		 RETURNING input_id`,
		user.Github_id, user.Username, user.Github_url, user.Avatar_url, user.Email,
	).Scan(&inputID)

	if err != nil {
		fmt.Printf("Error adding user: %s", err)
		return err
	}

	_, err = globals.DB.Exec(ctx,
		`INSERT INTO sessions (session_id, github_id, input_id, expires_at)
		 VALUES ($1, $2, $3, NOW() + INTERVAL '30 day')`,
		user.SessionToken, user.Github_id, inputID,
	)
	if err != nil {
		fmt.Printf("Error adding session: %s", err)
		return err
	}
	return nil
}

func fetchUser(ctx context.Context, sessionID string) (string, error) {
	var username string

	err := globals.DB.QueryRow(ctx,
		`SELECT username FROM users
		JOIN sessions s on s.github_id = users.github_id
		WHERE s.session_id = $1 AND s.expires_at > NOW()`, sessionID).Scan(&username)

	if err != nil {
		return "", err
	}

	return username, nil
}

func fetchUserByGithubID(ctx context.Context, githubID int64) (globals.User, error) {
	var user globals.User

	err := globals.DB.QueryRow(ctx, `
		SELECT github_id, username, github_url, avatar, email  FROM users where github_id = $1
	`, githubID).Scan(&user.Github_id, &user.Username, &user.Github_url, &user.Avatar_url, &user.Email)
	if err != nil {
		return user, err
	}

	// here, user does not have a session Token
	return user, nil
}

func updateSessionToken(ctx context.Context, githubID int64, newSessionID string) error {

	_, err := globals.DB.Exec(ctx, `
		INSERT INTO sessions (github_id, session_id, expires_at)
		VALUES ($1, $2, NOW() + INTERVAL '30 day')
		ON CONFLICT (github_id)
		DO UPDATE SET
		session_id = EXCLUDED.session_id,
		expires_at = EXCLUDED.expires_at
	`, githubID, newSessionID)

	if err != nil {
		log.Fatal(err)
		return err
	}

	return nil
}

func getSessionData(ctx context.Context, sessionID string) (globals.SessionData, error) {
	var sdata globals.SessionData

	err := globals.DB.QueryRow(ctx, `
		SELECT u.github_id, u.input_id, u.current_level FROM users u
		JOIN sessions s on s.github_id = u.github_id
		WHERE s.session_id = $1 AND s.expires_at > NOW()
		`, sessionID).Scan(&sdata.GithubID, &sdata.InputID, &sdata.CurrentLevel)

	if err != nil {
		return globals.SessionData{}, err
	}

	err = globals.DB.QueryRow(ctx, `
        SELECT level_id FROM levels
        WHERE release_time > NOW()
        ORDER BY release_time
        LIMIT 1;
        `).Scan(&sdata.NextReleaseLevel)

	if err != nil {
		if err == pgx.ErrNoRows {
			// TODO: this is means there is no new level scheduled to release => end of the event, need to handle it later
			sdata.NextReleaseLevel = 100
		} else {
			fmt.Println("But this happened, so sql.ErrNoRows didn't work")
			return globals.SessionData{}, err
		}
	}

	fmt.Printf("Next level to be released: %d", sdata.NextReleaseLevel)

	return sdata, nil
}

func deleteSessionToken(ctx context.Context, sessionID string) error {
	_, err := globals.DB.Exec(ctx, `
		DELETE FROM sessions
		WHERE session_id = $1
	`, sessionID)

	if err != nil {
		return err
	}

	fmt.Printf("Deleting session : %s\n", sessionID)
	return nil
}

func updateUserLevel(ctx context.Context, githubID int64, currentLevel int, level int, pass bool) error {

	if !pass {
		return updateSubmission(ctx, githubID, level, pass)
	}

	if level > currentLevel {
		return fmt.Errorf("Level %d not completed yet", currentLevel)
	}

	if level == currentLevel {
		// user passed the currentLevel which he is at
		// so update current_level in the db
		_, err := globals.DB.Exec(ctx, `
			UPDATE users set current_level = $1
			WHERE github_id = $2
		`, currentLevel+1, githubID)

		fmt.Printf("LEVEL UPDATED")

		if err != nil {
			log.Printf("Error advancing player level %s\n", err.Error())
			return err
		}
		updateSubmission(ctx, githubID, level, pass)
	}
	return nil
}

func updateSubmission(ctx context.Context, githubID int64, level int, pass bool) error {
	var attempts int

	err := globals.DB.QueryRow(ctx, `
            INSERT INTO submissions (github_id, level_id, last_submission, attempts)
            VALUES ($1, $2, NOW(), 1)
            ON CONFLICT (github_id, level_id)
            DO UPDATE SET
                last_submission = NOW(),
                attempts = submissions.attempts + 1
                WHERE submissions.passed = FALSE
            RETURNING attempts;
        `, githubID, level).Scan(&attempts)

	if err != nil {
		if err == pgx.ErrNoRows {
			fmt.Printf("level is already passed")
			return nil
		}
		return fmt.Errorf("Error inserting/updating submission for the user %d, %v", githubID, err)
	}

	if pass {
		// couting a streak if attempts = 1 and the user passed the level
		if attempts == 1 {
			_, err := globals.DB.Exec(ctx, `
                UPDATE users SET
                streak = streak + 1
                WHERE github_id = $1
                `, githubID)
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
	`, githubID, level)

		if err != nil {
			return fmt.Errorf("Error updating time_taken: %v", err)
		}

		rowsAffected := result.RowsAffected()

		if rowsAffected == 0 {
			log.Printf("User %d already has time_taken set for level %d — no update needed.\n", githubID, level)
			return nil
		}
	} else if attempts == 1 {
		// user had failed on first attempt, set the streak to zero
		_, err := globals.DB.Exec(ctx, `
                UPDATE users SET
                streak = 1
                WHERE github_id = $1
                `, githubID)
		if err != nil {
			return fmt.Errorf("Error setting streak to zero : %v", err)
		}
	}

	return nil
}
