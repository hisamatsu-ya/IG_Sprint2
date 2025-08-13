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
	// env
	brokers := splitAndTrim(getenv("KAFKA_BROKERS", "kafka:9092"), ",")
	topics := map[string]string{
		"movie":   getenv("MOVIE_TOPIC", "movie-events"),
		"user":    getenv("USER_TOPIC", "user-events"),
		"payment": getenv("PAYMENT_TOPIC", "payment-events"),
	}
	port := getenv("PORT", "8082")
	httpAddr := ":" + port

	// producers
	writers := make(map[string]*kafka.Writer, len(topics))
	for _, topic := range topics {
		writers[topic] = &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.Hash{},
			// AllowAutoTopicCreation: true, // убрано — полагаемся на брокер
		}
	}

	s := &Server{
		topics:  topics,
		writers: writers,
		mux:     http.NewServeMux(),
		brokers: brokers,
	}
	s.routes()
	s.startConsumers() // фоновые читатели

	srv := &http.Server{
		Addr:              httpAddr,
		Handler:           logging(s.mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// graceful shutdown
	idle := make(chan struct{})
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		for _, w := range s.writers {
			_ = w.Close()
		}
		for _, r := range s.readers {
			_ = r.Close()
		}
		close(idle)
	}()

	log.Printf("events-service listening on %s; brokers=%v; topics=%v", httpAddr, brokers, topics)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}
	<-idle
}

func (s *Server) routes() {
	// health/ping
	s.mux.HandleFunc("/api/events/health", s.healthHandler)
	s.mux.HandleFunc("/healthz", s.healthHandler)
	s.mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("events-service OK"))
	})

	// события: принимаем любой непустой JSON
	s.mux.HandleFunc("/api/events/movie", s.wrapJSON("movie", s.acceptAny))
	s.mux.HandleFunc("/api/events/user", s.wrapJSON("user", s.acceptAny))
	s.mux.HandleFunc("/api/events/payment", s.wrapJSON("payment", s.acceptAny))
}

func (s *Server) healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": true})
}

// мягкая валидация для тестов
func (s *Server) acceptAny(_ context.Context, raw json.RawMessage) error {
	if len(raw) == 0 || string(raw) == "null" {
		return errors.New("body must be non-empty JSON")
	}
	return nil
}

// ---- consumers ----
func (s *Server) startConsumers() {
	for evType, topic := range s.topics {
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:     s.brokers,
			Topic:       topic,
			GroupID:     "events-service",
			StartOffset: kafka.LastOffset, // только новые
			// AllowAutoTopicCreation: true, // убрано — полагаемся на брокер
			MinBytes: 1,
			MaxBytes: 10e6,
		})
		s.readers = append(s.readers, r)

		go func(et, tp string, rd *kafka.Reader) {
			for {
				msg, err := rd.ReadMessage(context.Background())
				if err != nil {
					if errors.Is(err, context.Canceled) {
						return
					}
					log.Printf("consumer error topic=%s: %v", tp, err)
					continue
				}
				log.Printf(
					"Consumed event_type=%s topic=%s partition=%d offset=%d key=%s value=%s",
					et, tp, msg.Partition, msg.Offset, string(msg.Key), string(msg.Value),
				)
			}
		}(evType, topic, r)
	}
}

// ---- producer ----
func (s *Server) publish(ctx context.Context, eventType string, payload []byte) error {
	topic, ok := s.topics[eventType]
	if !ok {
		return fmt.Errorf("unknown event type: %s", eventType)
	}
	writer, ok := s.writers[topic]
	if !ok {
		return fmt.Errorf("no writer for topic: %s", topic)
	}
	msg := kafka.Message{
		Key:   []byte(eventType),
		Value: payload,
		Time:  time.Now(),
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(eventType)},
			{Key: "content_type", Value: []byte("application/json")},
		},
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := writer.WriteMessages(ctx, msg); err != nil {
		return err
	}
	log.Printf("Published event_type=%s topic=%s", eventType, topic)
	return nil
}

// ---- wrappers/helpers ----
func (s *Server) wrapJSON(eventType string, fn func(ctx context.Context, raw json.RawMessage) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if ct := strings.ToLower(r.Header.Get("Content-Type")); !strings.HasPrefix(ct, "application/json") {
			http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
			return
		}
		defer r.Body.Close()

		var raw json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if err := fn(r.Context(), raw); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.publish(r.Context(), eventType, raw); err != nil {
			http.Error(w, "failed to publish to kafka", http.StatusBadGateway)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"status": "success"})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
