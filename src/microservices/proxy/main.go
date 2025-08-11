package main

import (
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func newProxyTarget(target string) *httputil.ReverseProxy {
	parsedURL, err := url.Parse(target)
	if err != nil {
		log.Fatalf("Invalid target URL %s: %v", target, err)
	}
	return httputil.NewSingleHostReverseProxy(parsedURL)
}

func chooseTarget(monolith, movies string, migrationPercent int, gradual bool, r *http.Request) string {
	if !gradual {
		return monolith
	}

	clientID := r.Header.Get("X-Forwarded-For")
	if clientID == "" {
		clientID = r.RemoteAddr
	}

	rand.Seed(int64(len(clientID) + time.Now().UnixNano()/int64(time.Minute)))
	if rand.Intn(100) < migrationPercent {
		return movies
	}
	return monolith
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	monolithURL := os.Getenv("MONOLITH_URL")
	moviesURL := os.Getenv("MOVIES_SERVICE_URL")

	gradual := os.Getenv("GRADUAL_MIGRATION") == "true"
	migrationPercent, _ := strconv.Atoi(os.Getenv("MOVIES_MIGRATION_PERCENT"))

	monolithProxy := newProxyTarget(monolithURL)
	moviesProxy := newProxyTarget(moviesURL)

	http.HandleFunc("/api/movies", func(w http.ResponseWriter, r *http.Request) {
		target := chooseTarget(monolithURL, moviesURL, migrationPercent, gradual, r)
		w.Header().Set("X-Proxy-Target", target)

		if target == moviesURL {
			moviesProxy.ServeHTTP(w, r)
		} else {
			monolithProxy.ServeHTTP(w, r)
		}
	})

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	log.Printf("Proxy service listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

