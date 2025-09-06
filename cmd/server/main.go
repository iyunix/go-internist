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
	"github.com/iyunix/go-internist/internal/services/admin_services"
	"github.com/iyunix/go-internist/internal/services/user_services"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

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
	// MODIFIED: Now handling the error from NewAIService
	aiService, err := services.NewAIService(
		cfg.AvalaiAPIKeyEmbedding,
		cfg.JabirAPIKey,
		"https://api.avalai.ir/v1",
		"https://openai.jabirproject.org/v1",
		cfg.EmbeddingModelName,
	)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize AI Service: %v", err)
	}

	pineconeService, err := services.NewPineconeService(
		cfg.PineconeAPIKey,
		cfg.PineconeIndexHost,
		cfg.PineconeNamespace,
	)
	if err != nil {
		log.Fatalf("Failed to initialize Pinecone service: %v", err)
	}

	userService := user_services.NewUserService(userRepo, cfg.JWTSecretKey, cfg.AdminPhoneNumber)

	// MODIFIED: Now handling the error from NewChatService
	chatService, err := services.NewChatService(chatRepo, messageRepo, aiService, pineconeService, cfg.RetrievalTopK)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize Chat Service: %v", err)
	}
	
	smsService := services.NewSMSService()
	balanceService := user_services.NewBalanceService(userRepo)
	adminService := admin_services.NewAdminService(userRepo)

	// --- Handlers ---
	authHandler := handlers.NewAuthHandler(userService, smsService, balanceService)
	
	// MODIFIED: Now handling the error from NewChatHandler
	chatHandler, err := handlers.NewChatHandler(userService, chatService)
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize Chat Handler: %v", err)
	}

	pageHandler := handlers.NewPageHandler()
	adminHandler := handlers.NewAdminHandler(adminService)

	// --- Router Setup ---
	r := mux.NewRouter()
	authMiddleware := middleware.NewJWTMiddleware(cfg.JWTSecretKey)
	adminMiddleware := middleware.RequireAdmin(userRepo)

	r.Use(corsMiddleware)
	r.Use(middleware.RecoverPanic)
	r.Use(middleware.LoggingMiddleware)

	// --- Public Routes ---
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK); _, _ = w.Write([]byte("OK")) }).Methods("GET")
	r.HandleFunc("/api/log", handlers.LogFrontendEvent).Methods("POST")
	r.HandleFunc("/", pageHandler.ShowIndexPage).Methods("GET")
	r.HandleFunc("/login", pageHandler.ShowLoginPage).Methods("GET")
	r.HandleFunc("/register", pageHandler.ShowRegisterPage).Methods("GET")
	r.HandleFunc("/login", authHandler.Login).Methods("POST")
	r.HandleFunc("/register", authHandler.Register).Methods("POST")
	r.HandleFunc("/logout", authHandler.Logout).Methods("GET")
	r.HandleFunc("/verify-sms", authHandler.VerifySMS).Methods("POST")
	r.HandleFunc("/verify-sms", pageHandler.ShowVerifySMSPage).Methods("GET")
	r.HandleFunc("/resend-sms", authHandler.ResendSMS).Methods("GET")

	// --- Protected Routes (for regular users) ---
	protected := r.PathPrefix("/").Subrouter()
	protected.Use(authMiddleware)
	protected.HandleFunc("/chat", pageHandler.ShowChatPage).Methods("GET")

	api := protected.PathPrefix("/api").Subrouter()
	api.HandleFunc("/user/balance", authHandler.GetUserCreditHandler).Methods("GET")
	api.HandleFunc("/chats", chatHandler.GetUserChats).Methods("GET")
	api.HandleFunc("/chats", chatHandler.CreateChat).Methods("POST")
	api.HandleFunc("/chats/{id:[0-9]+}/messages", chatHandler.GetChatMessages).Methods("GET")
	api.HandleFunc("/chats/{id:[0-9]+}", chatHandler.DeleteChat).Methods("DELETE")
	api.HandleFunc("/chats/{id:[0-9]+}/stream", chatHandler.StreamChatSSE).Methods("GET")

	// --- DEFINE AND PROTECT ADMIN ROUTES ---
	adminPageRoutes := r.PathPrefix("/admin").Subrouter()
	adminPageRoutes.Use(authMiddleware)
	adminPageRoutes.Use(adminMiddleware)
	adminPageRoutes.HandleFunc("", pageHandler.ShowAdminPage).Methods("GET")

	adminApiRoutes := r.PathPrefix("/api/admin").Subrouter()
	adminApiRoutes.Use(authMiddleware)
	adminApiRoutes.Use(adminMiddleware)
	adminApiRoutes.HandleFunc("/users", adminHandler.GetAllUsersHandler).Methods("GET")
	adminApiRoutes.HandleFunc("/users/export", adminHandler.ExportUsersCSVHandler).Methods("GET")
	adminApiRoutes.HandleFunc("/users/plan", adminHandler.ChangePlanHandler).Methods("POST")
	adminApiRoutes.HandleFunc("/users/renew", adminHandler.RenewSubscriptionHandler).Methods("POST")
	adminApiRoutes.HandleFunc("/users/topup", adminHandler.TopUpBalanceHandler).Methods("POST")

	// --- Custom Error Handlers ---
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageHandler.ShowErrorPage(w, "404", "Page Not Found", "The page you are looking for does not exist.")
	})
	r.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageHandler.ShowErrorPage(w, "405", "Method Not Allowed", "The method is not allowed for this resource.")
	})

	// --- Server Configuration ---
	port := ":8081"
	if cfg.ServerPort != "" {
		port = ":" + cfg.ServerPort
	}
	srv := &http.Server{
		Addr:    port,
		Handler: r,
	}

	// --- Startup Logging ---
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("==================================================")
	log.Printf("🤖 Internist AI - Medical Chat Assistant")
	log.Printf("==================================================")
	log.Printf("🚀 Server starting on port %s", port)
	log.Printf("🌐 Local access: http://localhost%s", port)
	log.Printf("💬 Chat interface: http://localhost%s/chat", port)
	log.Printf("🔒 Admin panel: http://localhost%s/admin", port)
	log.Printf("🔄 Server ready to accept connections!")
	log.Printf("==================================================")

	// --- Start Server in Goroutine ---
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server startup failed: %v", err)
		}
	}()

	// --- Graceful Shutdown ---
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("🛑 Shutting down server gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("❌ Server shutdown failed: %v", err)
	}
	log.Println("✅ Server stopped gracefully")
}