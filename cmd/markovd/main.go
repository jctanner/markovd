package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
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
	adminPassword := envOr("MARKOVD_ADMIN_PASSWORD", "")
	runnerType := envOr("MARKOVD_RUNNER", "shell")
	markovImage := envOr("MARKOVD_MARKOV_IMAGE", "")
	jobNamespace := envOr("MARKOVD_JOB_NAMESPACE", "")
	jobSA := envOr("MARKOVD_JOB_SERVICE_ACCOUNT", "pipeline-agent")
	jobImagePullPolicy := envOr("MARKOVD_JOB_IMAGE_PULL_POLICY", "")
	jobSecrets := envOr("MARKOVD_JOB_SECRETS", "")

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

	ensureAdminUser(authProvider, database, adminPassword)

	var r runner.Runner
	switch runnerType {
	case "shell":
		r = runner.NewShellRunner(markovBin)
	case "kubernetes":
		if markovImage == "" {
			log.Fatalf("MARKOVD_MARKOV_IMAGE is required when MARKOVD_RUNNER=kubernetes")
		}
		if jobNamespace == "" {
			if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
				jobNamespace = strings.TrimSpace(string(data))
			} else {
				log.Fatalf("MARKOVD_JOB_NAMESPACE is required (not running in a pod)")
			}
		}
		var err error
		r, err = runner.NewKubernetesRunner(markovImage, jobImagePullPolicy, jobNamespace, jobSA, runner.ParseSecrets(jobSecrets))
		if err != nil {
			log.Fatalf("Failed to create kubernetes runner: %v", err)
		}
		log.Printf("Using kubernetes runner (image=%s, namespace=%s, sa=%s)", markovImage, jobNamespace, jobSA)
	default:
		log.Fatalf("Unknown MARKOVD_RUNNER value: %q (expected 'shell' or 'kubernetes')", runnerType)
	}

	srv := api.NewServer(database, authProvider, jwtMgr, r, callbackToken, callbackURL)

	log.Printf("Starting markovd on :%s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), srv.Router()); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func ensureAdminUser(provider auth.Provider, database *db.DB, adminPassword string) {
	count, err := database.CountUsers(context.Background())
	if err != nil {
		log.Printf("Warning: could not check user count: %v", err)
		return
	}
	if count > 0 {
		return
	}

	password := adminPassword
	source := "environment"

	if password == "" {
		if path := os.Getenv("MARKOVD_ADMIN_PASSWORD_FILE"); path != "" {
			data, err := os.ReadFile(path)
			if err != nil {
				log.Printf("Warning: could not read admin password file %s: %v", path, err)
				return
			}
			password = strings.TrimSpace(string(data))
			source = "file"
		}
	}

	if password == "" {
		password = generateSecret()[:12]
		source = ""
	}

	_, err = provider.CreateUser("admin", password)
	if err != nil {
		log.Printf("Warning: could not create admin user: %v", err)
		return
	}

	if source == "" {
		log.Printf("Created default admin user:")
		log.Printf("  Username: admin")
		log.Printf("  Password: %s", password)
		log.Printf("  (change this password after first login)")
	} else {
		log.Printf("Created admin user with password from %s", source)
	}
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
