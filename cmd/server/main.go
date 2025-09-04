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

    "github.com/iyunix/go-internist/internal/config"
    "github.com/iyunix/go-internist/internal/domain"
    "github.com/iyunix/go-internist/internal/handlers"
    "github.com/iyunix/go-internist/internal/middleware"
    "github.com/iyunix/go-internist/internal/repository"
    "github.com/iyunix/go-internist/internal/services"

    "gorm.io/gorm"
)

func main() {
    cfg := config.Load()

    // --- Database Setup ---
    db, err := gorm.Open(sqlite.Open("notebook.db"), &gorm.Config{})
    if err != nil {
        log.Fatalf("DB Error: %v", err)
    }
    if err := db.AutoMigrate(&domain.User{}, &domain.Chat{}, &domain.Message{}); err != nil {
        log.Fatalf("DB Migration Error: %v", err)
    }

    // --- Initialize Repositories ---
    userRepo := repository.NewGormUserRepository(db)
    chatRepo := repository.NewChatRepository(db)
    messageRepo := repository.NewMessageRepository(db)

    // --- AI Services (Embeddings + Jabir LLM) ---
    aiService := services.NewAIService(
        cfg.AvalaiAPIKeyEmbedding,            // embeddings API key
        cfg.JabirAPIKey,                      // Jabir LLM API key
        "https://api.avalai.ir/v1",           // embeddings base URL
        "https://openai.jabirproject.org/v1", // Jabir LLM base URL
    )

    // --- Pinecone (Retrieval) ---
    pineconeService, err := services.NewPineconeService(
        cfg.PineconeAPIKey,
        cfg.PineconeIndexHost, // full host copied from Pinecone Console
        cfg.PineconeNamespace, // namespace (can be empty for default)
    )
    if err != nil {
        log.Fatalf("Failed to initialize Pinecone service: %v", err)
    }

    // --- Domain Services ---
    userService := services.NewUserService(userRepo, cfg.JWTSecretKey)
    // Updated to inject pineconeService for RAG
    chatService := services.NewChatService(chatRepo, messageRepo, aiService, pineconeService)

    // --- HTTP Handlers ---
    authHandler := handlers.NewAuthHandler(userService)
    chatHandler := handlers.NewChatHandler(userService, chatService)
    pageHandler := handlers.NewPageHandler()

    // --- Middleware (JWT) ---
    authMiddleware := middleware.NewJWTMiddleware(cfg.JWTSecretKey)

    // --- Router & Middlewares ---
    r := mux.NewRouter()                  // Gorilla Mux router
    r.Use(middleware.RecoverPanic)        // recover first
    r.Use(middleware.LoggingMiddleware)   // then request logging

    // Health check
    r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("OK"))
    }).Methods("GET")

    // Static files
    r.PathPrefix("/static/").
        Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

    // Public pages
    r.HandleFunc("/", pageHandler.ShowLoginPage).Methods("GET")
    r.HandleFunc("/login", pageHandler.ShowLoginPage).Methods("GET")
    r.HandleFunc("/register", pageHandler.ShowRegisterPage).Methods("GET")
    r.HandleFunc("/login", authHandler.Login).Methods("POST")
    r.HandleFunc("/register", authHandler.Register).Methods("POST")

    // Protected area
    protected := r.PathPrefix("/").Subrouter()
    protected.Use(authMiddleware)

    // Protected pages
    protected.HandleFunc("/chat", pageHandler.ShowChatPage).Methods("GET")

    // Protected API
    api := protected.PathPrefix("/api").Subrouter()
    api.HandleFunc("/chats", chatHandler.GetUserChats).Methods("GET")
    api.HandleFunc("/chats", chatHandler.HandleChatMessage).Methods("POST")
    api.HandleFunc("/chats/{id:[0-9]+}/messages", chatHandler.GetChatMessages).Methods("GET")
    api.HandleFunc("/chats/{id:[0-9]+}/messages", chatHandler.HandleChatMessage).Methods("POST")
    api.HandleFunc("/chats/{id:[0-9]+}", chatHandler.DeleteChat).Methods("DELETE")

    // --- HTTP Server ---
    srv := &http.Server{
        Addr:         ":" + cfg.ServerPort,
        Handler:      r,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Start server
    go func() {
        log.Printf("Server is starting on port %s...", cfg.ServerPort)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("ListenAndServe error: %v", err)
        }
    }()

    // Graceful shutdown
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
