package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/lookatitude/beluga-ai/pkg/messaging"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Configure voice backend
	voiceConfig := &vbiface.Config{
		Provider:     "twilio",
		PipelineType: vbiface.PipelineTypeSTTTTS,
		STTProvider:  "openai",
		TTSProvider:  "openai",
		ProviderConfig: map[string]any{
			"account_sid":  os.Getenv("TWILIO_ACCOUNT_SID"),
			"auth_token":   os.Getenv("TWILIO_AUTH_TOKEN"),
			"phone_number": os.Getenv("TWILIO_PHONE_NUMBER"),
		},
	}

	voiceBackend, err := backend.NewBackend(ctx, "twilio", voiceConfig)
	if err != nil {
		log.Fatalf("Failed to create voice backend: %v", err)
	}

	if err := voiceBackend.Start(ctx); err != nil {
		log.Fatalf("Failed to start voice backend: %v", err)
	}
	defer voiceBackend.Stop(ctx)

	// Configure messaging backend
	messagingConfig := &messaging.Config{
		Provider: "twilio",
		ProviderSpecific: map[string]any{
			"account_sid": os.Getenv("TWILIO_ACCOUNT_SID"),
			"auth_token":  os.Getenv("TWILIO_AUTH_TOKEN"),
		},
	}

	messagingBackend, err := messaging.NewBackend(ctx, "twilio", messagingConfig)
	if err != nil {
		log.Fatalf("Failed to create messaging backend: %v", err)
	}

	if err := messagingBackend.Start(ctx); err != nil {
		log.Fatalf("Failed to start messaging backend: %v", err)
	}
	defer messagingBackend.Stop(ctx)

	// Create HTTP router
	router := mux.NewRouter()

	// Voice webhook endpoint
	router.HandleFunc("/webhook/voice", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse webhook data
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		webhookData := make(map[string]string)
		for key, values := range r.PostForm {
			if len(values) > 0 {
				webhookData[key] = values[0]
			}
		}

		// Add URL for signature validation
		webhookData["_url"] = fmt.Sprintf("https://%s%s", r.Host, r.URL.Path)

		// Handle webhook
		// Note: In a full implementation, this would cast to TwilioBackend and call HandleWebhook
		// For now, this is a placeholder structure
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("POST")

	// Messaging webhook endpoint
	router.HandleFunc("/webhook/messaging", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse webhook data
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		webhookData := make(map[string]string)
		for key, values := range r.PostForm {
			if len(values) > 0 {
				webhookData[key] = values[0]
			}
		}

		// Add URL for signature validation
		webhookData["_url"] = fmt.Sprintf("https://%s%s", r.Host, r.URL.Path)

		// Handle webhook
		// Note: In a full implementation, this would cast to TwilioProvider and call HandleWebhook
		// For now, this is a placeholder structure
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("POST")

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
		})
	}).Methods("GET")

	// Start HTTP server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Webhook server starting on port %s", port)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("Shutting down webhook server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
}
