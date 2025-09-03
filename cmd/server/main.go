// File: cmd/server/main.go
package main

import (
	"log"
	"net/http"

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
	if err != nil { log.Fatal("DB Error: ", err) }
	db.AutoMigrate(&domain.User{}, &domain.Chat{}, &domain.Message{})

	// --- Initialize All Layers ---
	userRepo := repository.NewGormUserRepository(db)
	chatRepo := repository.NewChatRepository(db)
	messageRepo := repository.NewMessageRepository(db)

	// Create the new AI service
	aiService := services.NewAIService(cfg.AI_API_KEY) 

	// Pass the dependencies to each service
	userService := services.NewUserService(userRepo, cfg.JWTSecretKey)
	chatService := services.NewChatService(chatRepo, messageRepo, aiService) // <-- Corrected line

	// Create the handlers
	authHandler := handlers.NewAuthHandler(userService)
	chatHandler := handlers.NewChatHandler(userService, chatService)
	pageHandler := handlers.NewPageHandler()

	authMiddleware := middleware.NewJWTMiddleware(cfg.JWTSecretKey)

	// --- Setup Router ---
	r := mux.NewRouter()
	r.Use(middleware.LoggingMiddleware)


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
	api.HandleFunc("/chats", chatHandler.HandleChatMessage).Methods("POST") // Create new chat
	api.HandleFunc("/chats/{id:[0-9]+}/messages", chatHandler.GetChatMessages).Methods("GET")
	api.HandleFunc("/chats/{id:[0-9]+}/messages", chatHandler.HandleChatMessage).Methods("POST") // Add message to existing chat
	api.HandleFunc("/chats/{id:[0-9]+}", chatHandler.DeleteChat).Methods("DELETE")



	log.Printf("Server is starting on port %s...", cfg.ServerPort)
	log.Fatal(http.ListenAndServe(":"+cfg.ServerPort, r))
}