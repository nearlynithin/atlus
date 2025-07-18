package handlers

import (
	"context"
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
}

func addUser(ctx context.Context, user globals.User) {
	var inputID int
	err := globals.DB.QueryRow(ctx,
		`INSERT INTO users (github_id, username, github_url, avatar, email)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING input_id`,
		user.Github_id, user.Username, user.Github_url, user.Avatar_url, user.Email,
	).Scan(&inputID)
	if err != nil {
		log.Fatal("Error inserting the user:", err)
	}

	_, err = globals.DB.Exec(ctx,
		`INSERT INTO sessions (session_id, github_id, input_id, expires_at)
		 VALUES ($1, $2, $3, NOW() + INTERVAL '30 day')`,
		user.SessionToken, user.Github_id, inputID,
	)
	if err != nil {
		log.Fatal("Error inserting session:", err)
	}
}

func fetchUser(ctx context.Context, sessionID string) (string, error) {
	var username string

	err := globals.DB.QueryRow(ctx,
		`SELECT username FROM users
		JOIN sessions s on s.github_id = users.github_id
		WHERE s.session_id = $1`, sessionID).Scan(&username)

	if err != nil {
		return "", err
	}

	return username, nil
}