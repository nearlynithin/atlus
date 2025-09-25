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

	forkPtr := flag.Bool("dev", false, "DEV MODE : to truncate all db tables, on startup")
	flag.Parse()
	if *forkPtr {
		pool.Exec(ctx, "drop table users, sessions, submissions, levels;")
		fmt.Println("dropped all tables!")
		schema, err := os.ReadFile("schema.sql")
		if err != nil {
			log.Fatalf("Unable to find schema.sql, %v", err)
		}

		_, err = pool.Exec(ctx, string(schema))
		if err != nil {
			log.Fatalf("Unable to create tables, %v", err)
		} else {
			fmt.Println("Tables created successfully")
		}
		setupQueries, err := os.ReadFile(".setup")
		_, err = pool.Exec(ctx, string(setupQueries))
		if err != nil {
			log.Fatalf("Unable to exec db setup queries, %v", err)
		} else {
			fmt.Println("Tables populated successfully")
		}
	} else {

		schema, err := os.ReadFile("schema.sql")
		if err != nil {
			log.Fatalf("Unable to find schema.sql, %v", err)
		}

		_, err = pool.Exec(ctx, string(schema))
		if err != nil {
			log.Fatalf("Unable to create tables, %v", err)
		} else {
			fmt.Println("Tables created successfully")
		}

	}
	log.Println("Connected via pooler to:", version)
	globals.DB = pool
}

func addUser(ctx context.Context, user globals.User) error {
	tx, err := globals.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("Error creating a transaction for adding user, %v", err)
	}
	defer tx.Rollback(ctx)

	var inputID int

	err = tx.QueryRow(ctx,
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

	_, err = tx.Exec(ctx,
		`INSERT INTO sessions (session_id, github_id, input_id, expires_at)
		 VALUES ($1, $2, $3, NOW() + INTERVAL '30 day')`,
		user.SessionToken, user.Github_id, inputID,
	)
	if err != nil {
		fmt.Printf("Error adding session: %s", err)
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("Error commiting a transaction on sessions, %v", err)
	}
	return nil
}

func fetchUserByGithubID(ctx context.Context, githubID int64) (globals.User, error) {
	var user globals.User

	err := globals.DB.QueryRow(ctx, `
		SELECT github_id, username, github_url, avatar, email  FROM users where github_id = $1
	`, githubID).Scan(&user.Github_id, &user.Username, &user.Github_url, &user.Avatar_url, &user.Email)
	if err != nil {
		log.Printf("error fetching user by githubID, %v", err)
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

	var fetchErr error

	done := make(chan error, 1)

	go func() {
		err := globals.DB.QueryRow(ctx, `
		    SELECT u.github_id, u.input_id, u.current_level, u.username, u.github_url, u.avatar, u.email, u.streak, u.created_at
		    FROM users u
		    JOIN sessions s on s.github_id = u.github_id
		    WHERE s.session_id = $1 AND s.expires_at > NOW()
		    `, sessionID).Scan(&sdata.GithubID, &sdata.InputID, &sdata.CurrentLevel, &sdata.Username,
			&sdata.GithubUrl, &sdata.Avatar, &sdata.Email, &sdata.Streak, &sdata.CreatedAt)
		done <- err
	}()

	err := globals.DB.QueryRow(ctx, `
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
			log.Printf("Error fetching next release level data :%v\n", err)
			return globals.SessionData{}, err
		}
	}

	fetchErr = <-done
	if fetchErr != nil {
		log.Printf("error fetching session data, %v", fetchErr)
		return globals.SessionData{}, fetchErr
	}

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
