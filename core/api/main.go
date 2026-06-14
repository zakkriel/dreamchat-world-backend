package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://postgres:postgres@localhost:5432/dreamchat?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	debug := os.Getenv("DREAMCHAT_MODE") == "debug"
	mux := http.NewServeMux()
	mux.Handle("/worlds/", NewActorPageHandler(pool, debug))

	addr := ":8080"
	log.Printf("dreamchat world backend (read-only projection API) on %s (debug=%v)", addr, debug)
	log.Fatal(http.ListenAndServe(addr, mux))
}
