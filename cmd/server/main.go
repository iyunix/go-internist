// G:\go_internist\cmd\server\main.go
package main

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "strconv"
    "strings"
    "syscall"
    "time"
    "gorm.io/driver/postgres"
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

func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        
        // Get allowed origins from environment or use defaults
        allowedOriginsStr := os.Getenv("ALLOWED_ORIGINS")
        var allowedOrigins []string
        
        if allowedOriginsStr != "" {
            allowedOrigins = strings.Split(allowedOriginsStr, ",")
        } else {
            // Default allowed origins for development/production
            allowedOrigins = []string{
                "http://localhost:8080",
                "http://localhost:8081",
                "https://yourdomain.com", // Update this with your actual domain
            }
        }
        
        // Check if origin is allowed
        originAllowed := false
        for _, allowedOrigin := range allowedOrigins {
            if origin == strings.TrimSpace(allowedOrigin) {
                originAllowed = true
                break
            }
        }
        
        if originAllowed {
            w.Header().Set("Access-Control-Allow-Origin", origin)
        }
        
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")
        w.Header().Set("Access-Control-Allow-Credentials", "true")
        w.Header().Set("Access-Control-Max-Age", "86400")

        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}

func securityHeadersMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Content Security Policy - Enhanced for production
        directives := []string{
            "default-src 'self'",
            "base-uri 'self'",
            "object-src 'none'",
            "frame-ancestors 'none'",
            "img-src 'self' data:",
            "style-src 'self' 'unsafe-inline'", // TODO: Remove 'unsafe-inline' in future iteration
            "font-src 'self'",
            "script-src 'self' 'unsafe-inline'", // TODO: Remove 'unsafe-inline' in future iteration
            "connect-src 'self'",
            "frame-src 'none'",
            "media-src 'self'",
            "worker-src 'self'",
        }
        csp := strings.Join(directives, "; ")

        // Modern security headers (removed deprecated X-XSS-Protection)
        w.Header().Set("Content-Security-Policy", csp)
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
        
        // HSTS for HTTPS (only add if using HTTPS)
        if r.TLS != nil {
            w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        }
        
        next.ServeHTTP(w, r)
    })
}

func staticFileMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Set cache headers for static files
        // NOTE: In production, consider serving static files via Nginx/CDN for better performance
        if strings.HasPrefix(r.URL.Path, "/static/") {
            // Cache static files for 7 days
            w.Header().Set("Cache-Control", "public, max-age=604800")
            w.Header().Set("Expires", time.Now().Add(7*24*time.Hour).Format(http.TimeFormat))
            
            // Set content type based on file extension
            if strings.HasSuffix(r.URL.Path, ".css") {
                w.Header().Set("Content-Type", "text/css")
            } else if strings.HasSuffix(r.URL.Path, ".js") {
                w.Header().Set("Content-Type", "application/javascript")
            } else if strings.HasSuffix(r.URL.Path, ".woff2") {
                w.Header().Set("Content-Type", "font/woff2")
            }
        }
        
        next.ServeHTTP(w, r)
    })
}

func main() {
    startTime := time.Now()

    // Logger must be initialized first to be used by the config package
    logger := services.NewLogger("go_internist")
    logger.Info("🤖 Internist AI - Medical Chat Assistant starting")

    // === REPLACE THIS ENTIRE BLOCK ===
    // Load and validate config in one clean step
    cfg, err := config.New()
    if err != nil {
        logger.Error("FATAL: Configuration error", "error", err)
        os.Exit(1)
    }
    logger.Info("configuration loaded successfully", "environment", cfg.Environment)
    // === END REPLACEMENT ===

    // Database Connection
    logger.Info("initializing PostgreSQL database connection")
    db, err := gorm.Open(postgres.Open(cfg.GetDatabaseDSN()), &gorm.Config{
        NowFunc: func() time.Time {
            return time.Now().UTC()
        },
    })
    if err != nil {
        logger.Error("PostgreSQL connection failed", "error", err,
            "host", cfg.DBHost, "port", cfg.DBPort, "database", cfg.DBName)
        os.Exit(1)
    }

    // Configure connection pool
    sqlDB, err := db.DB()
    if err != nil {
        logger.Error("failed to get underlying sql.DB", "error", err)
        os.Exit(1)
    }
    
    // Connection pool settings
    sqlDB.SetMaxIdleConns(10)
    sqlDB.SetMaxOpenConns(100)
    sqlDB.SetConnMaxLifetime(time.Hour)
    
    logger.Info("PostgreSQL connected successfully", 
        "host", cfg.DBHost, "port", cfg.DBPort, "database", cfg.DBName,  // ✅ Use config fields
        "max_idle_conns", 10, "max_open_conns", 100)

    // Database Migrations
    // NOTE: In production, consider using golang-migrate/migrate for versioned migrations
    // Run migrations as: ./migrate -database "postgres://..." -path ./migrations up
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
    verificationService := user_services.NewVerificationService(userRepo, smsService, authService, logger)
    logger.Info("user services initialized successfully",
        "services", []string{"user", "auth", "balance", "lockout", "verification"})

    // Chat service
    logger.Info("initializing medical chat service")
    chatService, err := services.NewChatService(chatRepo, messageRepo, aiService, pineconeService, cfg.RetrievalTopK, cfg)
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

    pageHandler := handlers.NewPageHandler(userService, chatService, adminService)
    adminHandler := handlers.NewAdminHandler(adminService)
    logger.Info("HTTP handlers initialized successfully")

    // Router
    logger.Info("configuring HTTP router and middleware")
    r := mux.NewRouter()
    authMW := middleware.NewJWTMiddleware(authService, userService, cfg.AdminPhoneNumber)
    adminMW := middleware.RequireAdmin(userRepo)

    // Global middleware
    r.Use(corsMiddleware)
    r.Use(securityHeadersMiddleware)
    r.Use(staticFileMiddleware)
    r.Use(middleware.RecoverPanic)
    r.Use(middleware.LoggingMiddleware)

    // Static files configuration
    // PRODUCTION NOTE: Consider serving static files via Nginx reverse proxy or CDN
    // for better performance and reduced Go application load
    staticDir := os.Getenv("STATIC_DIR")
    if staticDir == "" {
        staticDir = "web/static"
    }
    r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

    // Health check endpoint
    r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte(`{"status":"ok","service":"go_internist","timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `"}`))
    }).Methods("GET")

    // Public routes
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

    // Password reset routes
    r.HandleFunc("/forgot-password", pageHandler.ShowForgotPasswordPage).Methods("GET")
    r.HandleFunc("/forgot-password", authHandler.HandleForgotPassword).Methods("POST")
    r.HandleFunc("/reset-password", pageHandler.ShowResetPasswordPage).Methods("GET")
    r.HandleFunc("/reset-password", authHandler.HandleResetPassword).Methods("POST")

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

    // Server configuration with environment variable support
    port := os.Getenv("PORT")
    if port == "" {
        port = cfg.ServerPort
    }
    if port == "" {
        port = "8080"  // Standard HTTP port for production
    }
    if !strings.HasPrefix(port, ":") {
        port = ":" + port
    }

    srv := &http.Server{
        Addr:         port,
        Handler:      r,
        ReadTimeout:  60 * time.Second,
        WriteTimeout: 120 * time.Second,  // Increased for chat streaming
        IdleTimeout:  120 * time.Second,
        MaxHeaderBytes: 1 << 20, // 1 MB
    }

    initTime := time.Since(startTime)
    logger.Info("🚀 server initialization completed",
        "initialization_time", initTime.String(),
        "port", port)
        
    logger.Info("==================================================")
    logger.Info("🤖 Internist AI - Medical Chat Assistant", "status", "ready")
    logger.Info("🚀 server starting", "port", port)
    logger.Info("🌐 local access", "url", fmt.Sprintf("http://localhost%s", port))
    logger.Info("💬 chat interface", "url", fmt.Sprintf("http://localhost%s/chat", port))
    logger.Info("🔒 admin panel", "url", fmt.Sprintf("http://localhost%s/admin", port))
    logger.Info("🔄 server ready to accept connections")
    logger.Info("==================================================")

    // Start server
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

    // Close database connections
    if sqlDB != nil {
        sqlDB.Close()
        logger.Info("database connections closed")
    }

    shutdownTime := time.Since(shutdownStart)
    totalUptime := time.Since(startTime)
    logger.Info("✅ server stopped gracefully",
        "shutdown_time", shutdownTime.String(),
        "total_uptime", totalUptime.String())
}
