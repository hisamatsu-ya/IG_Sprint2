package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

type Server struct {
	topics  map[string]string        // eventType -> topic
	writers map[string]*kafka.Writer // topic -> writer
	readers []*kafka.Reader          // consumers
	mux     *http.ServeMux
	brokers []string
}

func main() {
	// --- env ---
	brokers := splitAndTrim(getenv("KAFKA_BROKERS", "kafka:9092"), ",")
	topics := map[string]string{
		"movie":   getenv("MOVIE_TOPIC", "movie-events"),
		"user":    getenv("USER_TOPIC", "user-events"),
		"payment": getenv("PAYMENT_TOPIC", "payment-events"),
	}
	port := getenv("PORT", "8082")
	httpAddr := ":" + port

	// --- producers (writers) ---
	writers := make(map[string]*kafka.Writer, len(topics))
	for _, topic := range topics {
		writers[topic] = &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  topic,
			Balancer:               &kafka.Hash{},
			AllowAutoTopicCreation: true,
		}
	}

	s := &Server{
		topics:  topics,
		writers: writers,
		mux:     http.NewServeMux(),
		brokers: brokers,
	}
	s.routes()
	s.startConsumers() // поднимаем фоновых читателей

	srv := &http.Server{
		Addr:              httpAddr,
		Handler:           logging(s.mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// --- graceful shutdown ---
	idle := make(chan struct{})
	go func() {
		c := make(chan os.Signal, 1)
