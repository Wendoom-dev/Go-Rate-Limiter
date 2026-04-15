package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	redisclient "github.com/Wendoom-dev/Go-Rate-Limiter"
	"github.com/Wendoom-dev/Go-Rate-Limiter/redisclient"
)

func checkHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "This is working")
}

func main() {
	ctx := context.Background()

	// Initialize Redis + load Lua script
	redisClient, err := redisclient.NewRedisClient(ctx)
	if err != nil {
		log.Fatal("Failed to initialize Redis:", err)
	}

	fmt.Println("Lua script loaded with SHA:", redisClient.ScriptSHA)

	// TODO: later we will pass redisClient into limiter + handler

	http.HandleFunc("/check", checkHandler)

	addr := ":8080"
	log.Println("Server Running on", addr)

	err = http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}
