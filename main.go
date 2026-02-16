package main

import (
	"fmt"
	"log"
	"net/http"
)

func checkHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "This is working")
}

func main() {
	http.HandleFunc("/check", checkHandler)

	addr := ":8080"
	log.Println("Server Running on ", addr)
	err := http.ListenAndServe(addr, nil)

	if err != nil {
		log.Fatal(err)
	}
}
