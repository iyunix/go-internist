// File: cmd/server/main.go
package main

import (
    "context"
    "database/sql"
    "encoding/json"  // ✅ Added for health check JSON encoding
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "strings"
    "syscall"
    "time"

    "github.com/gorilla/mux"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"

    "github.com/iyunix/go-internist/internal/config"
    "github.com/iyunix/go-internist/internal/domain"
    "github.com/iyunix/go-internist/internal/handlers"
    "github.com/iyunix/go-internist/internal/middleware"
    "github.com/iyunix/go-internist/internal/services"
)

//go:generate wire

func corsMiddleware(allowedOrigins []string) mux.MiddlewareFunc {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")
            
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

func healthCheckHandler(app *Application, sqlDB *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
        defer cancel()
        
        health := map[string]interface{}{
            "service":   "go_internist",
            "timestamp": time.Now().UTC().Format(time.RFC3339),
            "status":    "healthy",
            "checks":    make(map[string]interface{}),
        }
        
        allHealthy := true
        
        // Database Health Check
        if err := sqlDB.PingContext(ctx); err != nil {
            health["checks"].(map[string]interface{})["database"] = map[string]interface{}{
                "status": "unhealthy",
                "error":  err.Error(),
            }
            allHealthy = false
        } else {
            health["checks"].(map[string]interface{})["database"] = map[string]interface{}{
                "status": "healthy",
            }
        }

        aiStatus := app.AIService.GetProviderStatus()
        if !aiStatus.IsHealthy {
            health["checks"].(map[string]interface{})["ai_provider"] = map[string]interface{}{
                "status":  "unhealthy",
                "message": aiStatus.Message,
                "embedding_healthy": aiStatus.EmbeddingHealthy,
                "llm_healthy":      aiStatus.LLMHealthy,
            }
            allHealthy = false
        } else {
            health["checks"].(map[string]interface{})["ai_provider"] = map[string]interface{}{
                "status": "healthy",
                "embedding_healthy": aiStatus.EmbeddingHealthy,
                "llm_healthy":      aiStatus.LLMHealthy,
            }
        }
        

        // Pinecone Health Check (lightweight)
        if err := app.PineconeService.HealthCheck(ctx); err != nil {
            health["checks"].(map[string]interface{})["pinecone"] = map[string]interface{}{
                "status": "unhealthy",
                "error":  err.Error(),
            }
            allHealthy = false
        } else {
            health["checks"].(map[string]interface{})["pinecone"] = map[string]interface{}{
                "status": "healthy",
            }
        }
        
        // Set overall status
        if !allHealthy {
            health["status"] = "unhealthy"
            w.WriteHeader(http.StatusServiceUnavailable)
        } else {
            w.WriteHeader(http.StatusOK)
        }
        
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(health)
    }
}

// ✅ FIXED: Accept cfg parameter
func setupGlobalMiddleware(r *mux.Router, cfg *config.Config) {
    r.Use(corsMiddleware(cfg.AllowedOrigins))  // ✅ cfg now available
    r.Use(securityHeadersMiddleware)
    r.Use(staticFileMiddleware)
    r.Use(middleware.RecoverPanic)
    r.Use(middleware.LoggingMiddleware)
}

// ✅ FIXED: Accept cfg parameter
func setupStaticFiles(r *mux.Router, cfg *config.Config) {
    staticDir := cfg.StaticDir  // ✅ From config
    if staticDir == "" {
        staticDir = "web/static"
    }
    r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
}

// ✅ FIXED: Accept sqlDB parameter
func setupPublicRoutes(r *mux.Router, app *Application, sqlDB *sql.DB) {
    // Enhanced health check endpoint
    r.HandleFunc("/health", healthCheckHandler(app, sqlDB)).Methods("GET")  // ✅ sqlDB now available

    // Frontend logging
    r.HandleFunc("/api/log", handlers.LogFrontendEvent).Methods("POST")

    // Public routes
    r.HandleFunc("/", app.PageHandler.ShowIndexPage).Methods("GET")
    r.HandleFunc("/login", app.PageHandler.ShowLoginPage).Methods("GET")
    r.HandleFunc("/register", app.PageHandler.ShowRegisterPage).Methods("GET")
    r.HandleFunc("/login", app.AuthHandler.Login).Methods("POST")
    r.HandleFunc("/register", app.AuthHandler.Register).Methods("POST")
    r.HandleFunc("/logout", app.AuthHandler.Logout).Methods("GET")
    r.HandleFunc("/verify-sms", app.AuthHandler.VerifySMS).Methods("POST")
    r.HandleFunc("/verify-sms", app.PageHandler.ShowVerifySMSPage).Methods("GET")
    r.HandleFunc("/resend-sms", app.AuthHandler.ResendSMS).Methods("GET")

    // Password reset routes
    r.HandleFunc("/forgot-password", app.PageHandler.ShowForgotPasswordPage).Methods("GET")
    r.HandleFunc("/forgot-password", app.AuthHandler.HandleForgotPassword).Methods("POST")
    r.HandleFunc("/reset-password", app.PageHandler.ShowResetPasswordPage).Methods("GET")
    r.HandleFunc("/reset-password", app.AuthHandler.HandleResetPassword).Methods("POST")
}

func setupProtectedRoutes(r *mux.Router, app *Application, authMW mux.MiddlewareFunc) {
    // Protected routes
    protected := r.PathPrefix("/").Subrouter()
    protected.Use(authMW)
    protected.HandleFunc("/chat", app.PageHandler.ShowChatPage).Methods("GET")

    // API routes
    api := protected.PathPrefix("/api").Subrouter()
    api.HandleFunc("/user/balance", app.AuthHandler.GetUserCreditHandler).Methods("GET")
    api.HandleFunc("/chats", app.ChatHandler.GetUserChats).Methods("GET")
    api.HandleFunc("/chats", app.ChatHandler.CreateChat).Methods("POST")
    api.HandleFunc("/chats/{id:[0-9]+}/messages", app.ChatHandler.GetChatMessages).Methods("GET")
    api.HandleFunc("/chats/{id:[0-9]+}/messages", app.ChatHandler.SendMessage).Methods("POST")
    api.HandleFunc("/chats/{id:[0-9]+}", app.ChatHandler.DeleteChat).Methods("DELETE")
    api.HandleFunc("/chats/{id:[0-9]+}/stream", app.ChatHandler.StreamChatSSE).Methods("GET")
}

func setupAdminRoutes(r *mux.Router, app *Application, authMW, adminMW mux.MiddlewareFunc) {
    // Admin routes
    adminPage := r.PathPrefix("/admin").Subrouter()
    adminPage.Use(authMW)
    adminPage.Use(adminMW)
    adminPage.HandleFunc("", app.PageHandler.ShowAdminPage).Methods("GET")

    adminAPI := r.PathPrefix("/api/admin").Subrouter()
    adminAPI.Use(authMW)
    adminAPI.Use(adminMW)
    adminAPI.HandleFunc("/users", app.AdminHandler.GetAllUsersHandler).Methods("GET")
    adminAPI.HandleFunc("/users/export", app.AdminHandler.ExportUsersCSVHandler).Methods("GET")
    adminAPI.HandleFunc("/users/plan", app.AdminHandler.ChangePlanHandler).Methods("POST")
    adminAPI.HandleFunc("/users/renew", app.AdminHandler.RenewSubscriptionHandler).Methods("POST")
    adminAPI.HandleFunc("/users/topup", app.AdminHandler.TopUpBalanceHandler).Methods("POST")
}

func setupErrorHandlers(r *mux.Router, pageHandler *handlers.PageHandler) {
    // Custom error handlers
    r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        pageHandler.ShowErrorPage(w, "404", "Page Not Found", "The page you are looking for does not exist.")
    })
    r.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        pageHandler.ShowErrorPage(w, "405", "Method Not Allowed", "The method is not allowed for this resource.")
    })
}

// ✅ FIXED: Read from config instead of os.Getenv
func getServerPort(cfg *config.Config) string {
    port := cfg.Port  // ✅ From config
    if port == "" {
        port = cfg.ServerPort
    }
    if port == "" {
        port = "8080"  // Standard HTTP port for production
    }
    if !strings.HasPrefix(port, ":") {
        port = ":" + port
    }
    return port
}

func configureDatabaseConnection(db *gorm.DB) (*sql.DB, error) {
    sqlDB, err := db.DB()
    if err != nil {
        return nil, err
    }
    
    // Connection pool settings
    sqlDB.SetMaxIdleConns(10)
    sqlDB.SetMaxOpenConns(100)
    sqlDB.SetConnMaxLifetime(time.Hour)
    
    return sqlDB, nil
}

func runDatabaseMigrations(db *gorm.DB, logger services.Logger) error {
    logger.Info("running database migrations")
    if err := db.AutoMigrate(&domain.User{}, &domain.Chat{}, &domain.Message{}, &domain.VerificationCode{}); err != nil {
        logger.Error("database migration failed", "error", err,
            "tables", []string{"users", "chats", "messages", "verification_codes"})
        return err
    }
    logger.Info("database migrations completed successfully")
    return nil
}

func startServer(srv *http.Server, logger services.Logger) {
    go func() {
        logger.Info("HTTP server starting", "address", srv.Addr)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Error("server startup failed", "error", err, "address", srv.Addr)
            os.Exit(1)
        }
        logger.Info("HTTP server stopped accepting new connections")
    }()
}

func gracefulShutdown(srv *http.Server, sqlDB *sql.DB, logger services.Logger, startTime time.Time) {
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

func main() {
    startTime := time.Now()

    // Initialize logger first (still manual since it's used everywhere)
    logger := services.NewLogger("go_internist")
    logger.Info("🤖 Internist AI - Medical Chat Assistant starting")

    // Load and validate config in one clean step
    cfg, err := config.New()
    if err != nil {
        logger.Error("FATAL: Configuration error", "error", err)
        os.Exit(1)
    }
    logger.Info("configuration loaded successfully", "environment", cfg.Environment)

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
    sqlDB, err := configureDatabaseConnection(db)
    if err != nil {
        logger.Error("failed to configure database connection", "error", err)
        os.Exit(1)
    }
    
    logger.Info("PostgreSQL connected successfully", 
        "host", cfg.DBHost, "port", cfg.DBPort, "database", cfg.DBName,
        "max_idle_conns", 10, "max_open_conns", 100)

    // Database Migrations
    if err := runDatabaseMigrations(db, logger); err != nil {
        os.Exit(1)
    }

    // 🎯 WIRE MAGIC - Replace 50+ lines of manual DI with this single call!
    logger.Info("initializing application with Wire dependency injection")
    app, err := InitializeApplication(db)
    if err != nil {
        logger.Error("application initialization failed", "error", err)
        os.Exit(1)
    }
    logger.Info("🚀 application initialized successfully via Wire DI")

    // Router setup
    logger.Info("configuring HTTP router and middleware")
    r := mux.NewRouter()
    
    // Create middleware instances
    authMW := middleware.NewJWTMiddleware(app.AuthService, app.UserService, cfg.AdminPhoneNumber)
    adminMW := middleware.RequireAdmin(app.UserRepo)
    
    // ✅ CORRECTED: Pass all required parameters
    setupGlobalMiddleware(r, cfg)          // ✅ Pass cfg
    setupStaticFiles(r, cfg)               // ✅ Pass cfg  
    setupPublicRoutes(r, app, sqlDB)       // ✅ Pass sqlDB
    setupProtectedRoutes(r, app, authMW)
    setupAdminRoutes(r, app, authMW, adminMW)
    setupErrorHandlers(r, app.PageHandler)
    
    logger.Info("HTTP routes configured successfully")

    // Server configuration
    port := getServerPort(cfg)
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

    // Start server and handle graceful shutdown
    startServer(srv, logger)
    gracefulShutdown(srv, sqlDB, logger, startTime)
}
