package handlers

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sceptix-club/atlus/Backend/globals"
)

func InitDB() {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Failed to connect to Supabase connection pooler: %v", err)
	}

	var version string
	if err := pool.QueryRow(ctx, "SELECT version()").Scan(&version); err != nil {
		log.Fatalf("Query test failed: %v", err)
	}

	log.Println("Connected via pooler to:", version)
	globals.DB = pool

	forkPtr := flag.Bool("dev", false ,"DEV MODE : to truncate all db tables, on startup")
	flag.Parse()
	if *forkPtr {
		globals.DB.Exec(ctx, "TRUNCATE table users, sessions;")
		fmt.Println("truncated tables users and sessions!")
	}
}

func addUser(ctx context.Context, user globals.User) (error) {
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
	`, githubID).Scan(&user.Github_id, &user.Username, &user.Github_url,&user.Avatar_url, &user.Email )
	if err != nil {
		return user, err
	}

	// here, user does not have a session Token
	return user, nil
}

func updateSessionToken(ctx context.Context, githubID int64, newSessionID string) (error) {

	_, err := globals.DB.Exec(ctx, `
		INSERT INTO sessions (github_id, session_id, expires_at)
		VALUES ($1, $2, NOW() + INTERVAL '30 day')
		ON CONFLICT (github_id)
		DO UPDATE SET 
		session_id = EXCLUDED.session_id
		expires_at = EXCLUDED.expires_at
	`, githubID, newSessionID)

	if err != nil {
		return err
	}

	return nil
}