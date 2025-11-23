package main

import (
    "log"
    "net/http"
    "os"

    "github.com/go-chi/chi/v5"
    "github.com/jmoiron/sqlx"
    _ "github.com/jackc/pgx/v5/stdlib"
    
    "pr-review-assigner/internal/handlers"
    "pr-review-assigner/internal/repo"
    "pr-review-assigner/internal/service"
)

func main() {
    dsn := os.Getenv("DATABASE_URL")
    if dsn == "" {
        dsn = "postgres://user:password@db:5432/db?sslmode=disable"
    }
    
    db, err := sqlx.Connect("pgx", dsn)
    if err != nil {
        log.Fatalf("db connect: %v", err)
    }
    defer db.Close()

    // Initialize dependencies
    repository := repo.New(db)
    svc := service.New(repository)  // repo.Repo реализует repo.RepoInterface
    handler := handlers.NewHandler(svc)

    // Setup router
    r := chi.NewRouter()
    handler.RegisterRoutes(r)

    // Start server
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    
    log.Printf("Server starting on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, r))
}