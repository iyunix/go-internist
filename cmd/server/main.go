// G:\go_internist\cmd\server\main.go
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/mux"
	"gorm.io/gorm"

	"github.com/iyunix/go-internist/internal/config"
	"github.com/iyunix/go-internist/internal/domain"
	"github.com/iyunix/go-internist/internal/handlers"
	"github.com/iyunix/go-internist/internal/middleware"
	"github.com/iyunix/go-internist/internal/repository/chat"
	"github.com/iyunix/go-internist/internal/repository/message"
	"github.com/iyunix/go-internist/internal/repository/user"
	"github.com/iyunix/go-internist/internal/services"
	"github.com/iyunix/go-internist/internal/services/admin_services"
	"github.com/iyunix/go-internist/internal/services/ai"
	"github.com/iyunix/go-internist/internal/services/sms"
	"github.com/iyunix/go-internist/internal/services/user_services"
)

// CORS only (no CSP here)
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

// G:\go_internist\cmd\server\main.go

func cspMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // --- This is the new, corrected policy ---
        csp := "default-src 'self'; " +
               // Allow scripts from Tailwind's CDN
               "script-src 'self' 'unsafe-inline' https://cdn.tailwindcss.com; " +
               // Allow stylesheets from Google Fonts
               "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
               // Allow font files to be loaded from Google's static domain
               "font-src https://fonts.gstatic.com; " +
               // Allow images from self (your domain)
               "img-src 'self' data:; " +
               "connect-src 'self';"

        w.Header().Set("Content-Security-Policy", csp)
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        
        next.ServeHTTP(w, r)
    })
}
func main() {
	startTime := time.Now()

	// Logger
	logger := services.NewLogger("go_internist")
	logger.Info("🤖 Internist AI - Medical Chat Assistant starting",
		"version", "1.0.0",
		"go_env", os.Getenv("GO_ENV"),
		"log_level", os.Getenv("LOG_LEVEL"))

	// Config
	cfg := config.Load()
	logger.Info("configuration loaded successfully")

	// Database
	logger.Info("initializing database connection", "database", "notebook.db")
	db, err := gorm.Open(sqlite.Open("notebook.db"), &gorm.Config{})
	if err != nil {
		logger.Error("database connection failed", "error", err, "database", "notebook.db")
		os.Exit(1)
	}
	logger.Info("database connected successfully", "driver", "sqlite")

	// Migrations
	logger.Info("running database migrations")
	if err := db.AutoMigrate(&domain.User{}, &domain.Chat{}, &domain.Message{}); err != nil {
		logger.Error("database migration failed", "error", err,
			"tables", []string{"users", "chats", "messages"})
		os.Exit(1)
	}
	logger.Info("database migrations completed successfully")

	// Repositories
	logger.Info("initializing repositories")
	userRepo := user.NewGormUserRepository(db)
	chatRepo := chat.NewChatRepository(db)
	messageRepo := message.NewMessageRepository(db)
	logger.Info("repositories initialized successfully")

	// Services
	logger.Info("initializing services")

	// AI Service
	logger.Info("configuring AI service")
	aiConfig := ai.DefaultConfig()
	aiConfig.EmbeddingKey = cfg.AvalaiAPIKeyEmbedding
	aiConfig.LLMKey = cfg.JabirAPIKey
	aiConfig.EmbeddingBaseURL = "https://api.avalai.ir/v1"
	aiConfig.LLMBaseURL = "https://openai.jabirproject.org/v1"
	aiConfig.EmbeddingModel = cfg.EmbeddingModelName
	if err := aiConfig.Validate(); err != nil {
		logger.Error("AI configuration validation failed", "error", err)
		os.Exit(1)
	}
	aiProvider := ai.NewOpenAIProvider(aiConfig)
	aiService := services.NewAIService(aiProvider, logger)
	logger.Info("AI service initialized successfully",
		"embedding_model", cfg.EmbeddingModelName,
		"embedding_provider", "avalai",
		"llm_provider", "jabir")

	// Pinecone
	logger.Info("initializing Pinecone vector database service")
	pineconeService, err := services.NewPineconeService(
		cfg.PineconeAPIKey,
		cfg.PineconeIndexHost,
		cfg.PineconeNamespace,
	)
	if err != nil {
		logger.Error("Pinecone service initialization failed", "error", err,
			"index_host", cfg.PineconeIndexHost,
			"namespace", cfg.PineconeNamespace)
		os.Exit(1)
	}
	logger.Info("Pinecone service initialized successfully",
		"namespace", cfg.PineconeNamespace,
		"retry_config", "3 attempts with backoff")

	// SMS
	logger.Info("configuring SMS service")
	smsConfig := &sms.Config{
		AccessKey: os.Getenv("SMS_ACCESS_KEY"),
		TemplateID: func() int {
			id, _ := strconv.Atoi(os.Getenv("SMS_TEMPLATE_ID"))
			return id
		}(),
		APIURL:  os.Getenv("SMS_API_URL"),
		Timeout: 10 * time.Second,
	}
	if err := smsConfig.Validate(); err != nil {
		logger.Error("SMS configuration validation failed", "error", err)
		os.Exit(1)
	}
	smsProvider := sms.NewSMSIRProvider(smsConfig)
	smsService := services.NewSMSService(smsProvider, logger)
	logger.Info("SMS service initialized successfully",
		"provider", "sms.ir",
		"timeout", smsConfig.Timeout.String())

	// User services
	logger.Info("initializing user services")
	userService := user_services.NewUserService(userRepo, cfg.JWTSecretKey, cfg.AdminPhoneNumber, logger)
	authService := user_services.NewAuthService(userRepo, cfg.JWTSecretKey, cfg.AdminPhoneNumber, logger)
	balanceService := user_services.NewBalanceService(userRepo, logger)
	verificationService := user_services.NewVerificationService(userRepo, smsService, logger)
	logger.Info("user services initialized successfully",
		"services", []string{"user", "auth", "balance", "lockout", "verification"})

	// Chat service
	logger.Info("initializing medical chat service")
	chatService, err := services.NewChatService(chatRepo, messageRepo, aiService, pineconeService, cfg.RetrievalTopK)
	if err != nil {
		logger.Error("chat service initialization failed", "error", err,
			"retrieval_top_k", cfg.RetrievalTopK)
		os.Exit(1)
	}
	logger.Info("medical chat service initialized successfully",
		"retrieval_top_k", cfg.RetrievalTopK)

	// Admin service
	adminService := admin_services.NewAdminService(userRepo, logger)
	logger.Info("admin service initialized successfully")

	// Handlers
	logger.Info("initializing HTTP handlers")
	authHandler := handlers.NewAuthHandler(userService, authService, verificationService, smsService, balanceService)

	chatHandler, err := handlers.NewChatHandler(userService, chatService)
	if err != nil {
		logger.Error("chat handler initialization failed", "error", err)
		os.Exit(1)
	}

	pageHandler := handlers.NewPageHandler()
	adminHandler := handlers.NewAdminHandler(adminService)
	logger.Info("HTTP handlers initialized successfully")

	// Router
	logger.Info("configuring HTTP router and middleware")
	r := mux.NewRouter()
	authMW := middleware.NewJWTMiddleware(authService)
	adminMW := middleware.RequireAdmin(userRepo)

	r.Use(corsMiddleware)
	r.Use(cspMiddleware) // Set CSP only once here
	r.Use(middleware.RecoverPanic)
	r.Use(middleware.LoggingMiddleware)

	// Public routes
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}).Methods("GET")
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

	// Protected routes
	protected := r.PathPrefix("/").Subrouter()
	protected.Use(authMW)
	protected.HandleFunc("/chat", pageHandler.ShowChatPage).Methods("GET")

	api := protected.PathPrefix("/api").Subrouter()
	api.HandleFunc("/user/balance", authHandler.GetUserCreditHandler).Methods("GET")
	api.HandleFunc("/chats", chatHandler.GetUserChats).Methods("GET")
	api.HandleFunc("/chats", chatHandler.CreateChat).Methods("POST")
	api.HandleFunc("/chats/{id:[0-9]+}/messages", chatHandler.GetChatMessages).Methods("GET")
	api.HandleFunc("/chats/{id:[0-9]+}/messages", chatHandler.SendMessage).Methods("POST")
	api.HandleFunc("/chats/{id:[0-9]+}", chatHandler.DeleteChat).Methods("DELETE")
	api.HandleFunc("/chats/{id:[0-9]+}/stream", chatHandler.StreamChatSSE).Methods("GET")

	// Admin routes
	adminPage := r.PathPrefix("/admin").Subrouter()
	adminPage.Use(authMW)
	adminPage.Use(adminMW)
	adminPage.HandleFunc("", pageHandler.ShowAdminPage).Methods("GET")

	adminAPI := r.PathPrefix("/api/admin").Subrouter()
	adminAPI.Use(authMW)
	adminAPI.Use(adminMW)
	adminAPI.HandleFunc("/users", adminHandler.GetAllUsersHandler).Methods("GET")
	adminAPI.HandleFunc("/users/export", adminHandler.ExportUsersCSVHandler).Methods("GET")
	adminAPI.HandleFunc("/users/plan", adminHandler.ChangePlanHandler).Methods("POST")
	adminAPI.HandleFunc("/users/renew", adminHandler.RenewSubscriptionHandler).Methods("POST")
	adminAPI.HandleFunc("/users/topup", adminHandler.TopUpBalanceHandler).Methods("POST")

	// Custom error handlers
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageHandler.ShowErrorPage(w, "404", "Page Not Found", "The page you are looking for does not exist.")
	})
	r.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageHandler.ShowErrorPage(w, "405", "Method Not Allowed", "The method is not allowed for this resource.")
	})

	logger.Info("HTTP routes configured successfully")

	// Server
	port := ":8081"
	if cfg.ServerPort != "" {
		port = ":" + cfg.ServerPort
	}
	srv := &http.Server{
		Addr:         port,
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	initTime := time.Since(startTime)
	logger.Info("🚀 server initialization completed",
		"initialization_time", initTime.String(),
		"port", port,
		"read_timeout", "15s",
		"write_timeout", "15s",
		"idle_timeout", "60s")

	logger.Info("==================================================")
	logger.Info("🤖 Internist AI - Medical Chat Assistant", "status", "ready")
	logger.Info("🚀 server starting", "port", port)
	logger.Info("🌐 local access", "url", fmt.Sprintf("http://localhost%s", port))
	logger.Info("💬 chat interface", "url", fmt.Sprintf("http://localhost%s/chat", port))
	logger.Info("🔒 admin panel", "url", fmt.Sprintf("http://localhost%s/admin", port))
	logger.Info("🔄 server ready to accept connections")
	logger.Info("==================================================")

	// Start
	go func() {
		logger.Info("HTTP server starting", "address", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server startup failed", "error", err, "address", srv.Addr)
			os.Exit(1)
		}
		logger.Info("HTTP server stopped accepting new connections")
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	receivedSignal := <-stop

	logger.Info("🛑 shutdown signal received",
		"signal", receivedSignal.String(),
		"initiating_graceful_shutdown", true)

	shutdownStart := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced shutdown", "error", err, "timeout", "15s")
		os.Exit(1)
	}

	shutdownTime := time.Since(shutdownStart)
	totalUptime := time.Since(startTime)
	logger.Info("✅ server stopped gracefully",
		"shutdown_time", shutdownTime.String(),
		"total_uptime", totalUptime.String())
}
