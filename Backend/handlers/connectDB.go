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
		log.Fatalf("❌ Failed to connect to Supabase connection pooler: %v", err)
	}

	var version string
	if err := pool.QueryRow(ctx, "SELECT version()").Scan(&version); err != nil {
		log.Fatalf("❌ Query test failed: %v", err)
	}

	log.Println("✅ Connected via pooler to:", version)
	globals.DB = pool
}
