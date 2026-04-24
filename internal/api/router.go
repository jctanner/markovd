package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jctanner/markovd/internal/auth"
	"github.com/jctanner/markovd/internal/db"
	"github.com/jctanner/markovd/internal/runner"
)

type Server struct {
	db            *db.DB
	auth          auth.Provider
	jwt           *auth.JWTManager
	runner        runner.Runner
	callbackToken string
	callbackURL   string
}

func NewServer(database *db.DB, authProvider auth.Provider, jwtMgr *auth.JWTManager, r runner.Runner, callbackToken, callbackURL string) *Server {
	return &Server{
		db:            database,
		auth:          authProvider,
		jwt:           jwtMgr,
		runner:        r,
		callbackToken: callbackToken,
		callbackURL:   callbackURL,
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/register", s.handleRegister)

		r.Post("/events", s.handleEvent)

		r.Group(func(r chi.Router) {
			r.Use(s.jwtMiddleware)

			r.Get("/runs", s.handleListRuns)
			r.Get("/runs/{runID}", s.handleGetRun)
			r.Post("/runs", s.handleCreateRun)

			r.Get("/workflows", s.handleListWorkflows)
			r.Get("/workflows/{name}", s.handleGetWorkflow)
			r.Post("/workflows", s.handleCreateWorkflow)
		})
	})

	return r
}

type contextKey string

const claimsKey contextKey = "claims"

func (s *Server) jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := s.jwt.Validate(tokenStr)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}
		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getClaims(r *http.Request) *auth.Claims {
	claims, _ := r.Context().Value(claimsKey).(*auth.Claims)
	return claims
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
