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

    // --- Initialize All Layers ---
    userRepo := repository.NewGormUserRepository(db)
    chatRepo := repository.NewChatRepository(db)
    messageRepo := repository.NewMessageRepository(db)

    aiService := services.NewAIService(cfg.AvalaiAPIKeyEmbedding, cfg.AvalaiAPIKeyLLM)

    userService := services.NewUserService(userRepo, cfg.JWTSecretKey)
    chatService := services.NewChatService(chatRepo, messageRepo, aiService)

    authHandler := handlers.NewAuthHandler(userService)
    chatHandler := handlers.NewChatHandler(userService, chatService)
    pageHandler := handlers.NewPageHandler()

    authMiddleware := middleware.NewJWTMiddleware(cfg.JWTSecretKey)

    r := mux.NewRouter()
    r.Use(middleware.LoggingMiddleware)

    // Health check endpoint
    r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    }).Methods("GET")

    // Static files
    fs := http.FileServer(http.Dir("./web/static/"))
    r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

    // Public routes
    r.HandleFunc("/", pageHandler.ShowLoginPage).Methods("GET")
    r.HandleFunc("/login", pageHandler.ShowLoginPage).Methods("GET")
    r.HandleFunc("/register", pageHandler.ShowRegisterPage).Methods("GET")
    r.HandleFunc("/login", authHandler.Login).Methods("POST")
    r.HandleFunc("/register", authHandler.Register).Methods("POST")

    // Protected routes
    protected := r.PathPrefix("/").Subrouter()
    protected.Use(authMiddleware)

    // Page-serving protected routes
    protected.HandleFunc("/chat", pageHandler.ShowChatPage).Methods("GET")

    // API protected routes
    api := protected.PathPrefix("/api").Subrouter()
    api.HandleFunc("/chats", chatHandler.GetUserChats).Methods("GET")
    api.HandleFunc("/chats", chatHandler.HandleChatMessage).Methods("POST")
    api.HandleFunc("/chats/{id:[0-9]+}/messages", chatHandler.GetChatMessages).Methods("GET")
    api.HandleFunc("/chats/{id:[0-9]+}/messages", chatHandler.HandleChatMessage).Methods("POST")
    api.HandleFunc("/chats/{id:[0-9]+}", chatHandler.DeleteChat).Methods("DELETE")

    srv := &http.Server{
        Addr:         ":" + cfg.ServerPort,
        Handler:      r,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Run server in a goroutine
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
        log.Fatalf("Server Shutdown Failed:%+v", err)
    }

    log.Println("Server stopped gracefully")
}
