package main

import (
	"fmt"
	"flag"
	"log"
	"net/http"
)

func main() {
	port := flag.Int("port", 9001, "Port to listen on")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, "hello from backend: %d\n",*port)

	})
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), mux))
}