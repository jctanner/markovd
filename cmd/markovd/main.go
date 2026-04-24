package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jctanner/markovd/internal/api"
	"github.com/jctanner/markovd/internal/auth"
	"github.com/jctanner/markovd/internal/db"
	"github.com/jctanner/markovd/internal/runner"
)

func main() {
	port := envOr("MARKOVD_PORT", "8080")
	dbURL := envOr("MARKOVD_DB_URL", "postgres://markovd:markovd@localhost:5432/markovd?sslmode=disable")
	jwtSecret := envOr("MARKOVD_JWT_SECRET", "")
	markovBin := envOr("MARKOVD_MARKOV_BIN", "markov")
	callbackToken := envOr("MARKOVD_CALLBACK_TOKEN", "")
	callbackURL := envOr("MARKOVD_CALLBACK_URL", fmt.Sprintf("http://localhost:%s/api/v1/events", port))

	if jwtSecret == "" {
		jwtSecret = generateSecret()
		log.Printf("WARNING: No MARKOVD_JWT_SECRET set, using random secret (tokens will not survive restarts)")
	}
	if callbackToken == "" {
		callbackToken = generateSecret()
		log.Printf("Generated callback token: %s", callbackToken)
		log.Printf("Set MARKOVD_CALLBACK_TOKEN to persist across restarts")
	}

	database, err := db.New(dbURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer database.Close()
	log.Printf("Connected to database")

	authProvider := auth.NewLocalProvider(database)
	jwtMgr := auth.NewJWTManager(jwtSecret, 24*time.Hour)

	ensureAdminUser(authProvider, database)

	shellRunner := runner.NewShellRunner(markovBin)

	srv := api.NewServer(database, authProvider, jwtMgr, shellRunner, callbackToken, callbackURL)

	log.Printf("Starting markovd on :%s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), srv.Router()); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func ensureAdminUser(provider auth.Provider, database *db.DB) {
	count, err := database.CountUsers(context.Background())
	if err != nil {
		log.Printf("Warning: could not check user count: %v", err)
		return
	}
	if count > 0 {
		return
	}

	password := generateSecret()[:12]
	_, err = provider.CreateUser("admin", password)
	if err != nil {
		log.Printf("Warning: could not create admin user: %v", err)
		return
	}
	log.Printf("Created default admin user:")
	log.Printf("  Username: admin")
	log.Printf("  Password: %s", password)
	log.Printf("  (change this password after first login)")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func generateSecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
