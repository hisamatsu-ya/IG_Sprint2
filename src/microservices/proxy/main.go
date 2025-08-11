package main

import (
	"hash/fnv"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"time"
)

func newProxyTarget(target string) *httputil.ReverseProxy {
	u, err := url.Parse(target)
	if err != nil {
		log.Fatalf("invalid target URL %s: %v", target, err)
	}
	return httputil.NewSingleHostReverseProxy(u)
}

func chooseTarget(monolith, movies string, migrationPercent int, gradual bool, r *http.Request) string {
	if !gradual {
		return monolith
	}
	clientID := r.Header.Get("X-Forwarded-For")
	if clientID == "" {
		clientID = r.RemoteAddr
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(clientID))
	seed := int64(h.Sum32())
	rnd := rand.New(rand.NewSource(seed))
	if rnd.Intn(100) < migrationPercent {
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
	if monolithURL == "" {
		monolithURL = "http://monolith:8080"
	}
	moviesURL := os.Getenv("MOVIES_SERVICE_URL")
	if moviesURL == "" {
		moviesURL = "http://movies-service:8081"
	}
	gradual := false
	if v := os.Getenv("GRADUAL_MIGRATION"); v == "true" || v == "1" {
		gradual = true
	}
	migrationPercent := 50
	if v := os.Getenv("MOVIES_MIGRATION_PERCENT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 && n <= 100 {
			migrationPercent = n
		}
	}

	monolithProxy := newProxyTarget(monolithURL)
	moviesProxy := newProxyTarget(moviesURL)

	http.HandleFunc("/api/movies", func(w http.ResponseWriter, r *http.Request) {
		target := chooseTarget(monolithURL, moviesURL, migrationPercent, gradual, r)
		w.Header().Set("X-Feature-Flag-Gradual", strconv.FormatBool(gradual))
		w.Header().Set("X-Movies-Migration-Percent", strconv.Itoa(migrationPercent))
		if target == moviesURL {
			w.Header().Set("X-Proxy-Target", "movies")
			moviesProxy.ServeHTTP(w, r)
			return
		}
		w.Header().Set("X-Proxy-Target", "monolith")
		monolithProxy.ServeHTTP(w, r)
	})

	http.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Proxy-Target", "monolith")
		monolithProxy.ServeHTTP(w, r)
	})

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("CinemaAbyss Proxy is up. Try /api/movies\n"))
	})

	srv := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	log.Printf("proxy listening on :%s (gradual=%v, percent=%d, monolith=%s, movies=%s)",
		port, gradual, migrationPercent, monolithURL, moviesURL)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
