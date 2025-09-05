// File: cmd/server/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/mux"
	"gorm.io/gorm"

	"github.com/iyunix/go-internist/internal/config"
	"github.com/iyunix/go-internist/internal/domain"
	"github.com/iyunix/go-internist/internal/handlers"
	"github.com/iyunix/go-internist/internal/middleware"
	"github.com/iyunix/go-internist/internal/repository"
	"github.com/iyunix/go-internist/internal/services"
)

func main() {
	cfg := config.Load()

	db, err := gorm.Open(sqlite.Open("notebook.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("DB Error: %v", err)
	}
	if err := db.AutoMigrate(&domain.User{}, &domain.Chat{}, &domain.Message{}); err != nil {
		log.Fatalf("DB Migration Error: %v", err)
	}

	// --- Repositories ---
	userRepo := repository.NewGormUserRepository(db)
	chatRepo := repository.NewChatRepository(db)
	messageRepo := repository.NewMessageRepository(db)

	// --- Services ---
	aiService := services.NewAIService(
		cfg.AvalaiAPIKeyEmbedding,
		cfg.JabirAPIKey,
		"https://api.avalai.ir/v1",
		"https://openai.jabirproject.org/v1",
		cfg.EmbeddingModelName,
	)

	pineconeService, err := services.NewPineconeService(
		cfg.PineconeAPIKey,
		cfg.PineconeIndexHost,
		cfg.PineconeNamespace,
	)
	if err != nil {
		log.Fatalf("Failed to initialize Pinecone service: %v", err)
	}

	userService := services.NewUserService(userRepo, cfg.JWTSecretKey)
	chatService := services.NewChatService(chatRepo, messageRepo, aiService, pineconeService, cfg.RetrievalTopK)
	
	// --- Handlers ---
	authHandler := handlers.NewAuthHandler(userService)
	chatHandler := handlers.NewChatHandler(userService, chatService)
	pageHandler := handlers.NewPageHandler()

	// --- Router Setup ---
	authMiddleware := middleware.NewJWTMiddleware(cfg.JWTSecretKey)
	r := mux.NewRouter()
	r.Use(middleware.RecoverPanic)
	r.Use(middleware.LoggingMiddleware)

	// --- Public Routes ---
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}).Methods("GET")
	
	// CORRECTED AND MOVED HERE: Public endpoint for frontend logging
	r.HandleFunc("/api/log", handlers.LogFrontendEvent).Methods("POST")

	r.HandleFunc("/", pageHandler.ShowLoginPage).Methods("GET")
	r.HandleFunc("/login", pageHandler.ShowLoginPage).Methods("GET")
	r.HandleFunc("/register", pageHandler.ShowRegisterPage).Methods("GET")
	r.HandleFunc("/login", authHandler.Login).Methods("POST")
	r.HandleFunc("/register", authHandler.Register).Methods("POST")

	// --- Protected Routes ---
	protected := r.PathPrefix("/").Subrouter()
	protected.Use(authMiddleware)
	protected.HandleFunc("/chat", pageHandler.ShowChatPage).Methods("GET")
	
	api := protected.PathPrefix("/api").Subrouter()
	api.HandleFunc("/chats", chatHandler.GetUserChats).Methods("GET")
	api.HandleFunc("/chats", chatHandler.CreateChat).Methods("POST")
	api.HandleFunc("/chats/{id:[0-9]+}/messages", chatHandler.GetChatMessages).Methods("GET")
	api.HandleFunc("/chats/{id:[0-9]+}", chatHandler.DeleteChat).Methods("DELETE")
	api.HandleFunc("/chats/{id:[0-9]+}/stream", chatHandler.StreamChatSSE).Methods("GET")

	// --- Server Start and Graceful Shutdown ---
	srv := &http.Server{
	Addr:         ":8081",
	Handler:      r,              // your mux
	ReadTimeout:  0,                   // or generous, SSE reads little after headers
	WriteTimeout: 0,                   // critical: disable or set very large for SSE
	IdleTimeout:  0,                   // optional: disable for long-lived
	}
	log.Fatal(srv.ListenAndServe())

	go func() {
		log.Printf("Server is starting on port %s...", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed: %+v", err)
	}
	log.Println("Server stopped gracefully")
}