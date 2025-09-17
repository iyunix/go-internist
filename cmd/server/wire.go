// G:\go_internist\cmd\server\wire.go
//go:build wireinject
// +build wireinject

// File: cmd/server/wire.go
package main

import (
    "strconv"
    "time"
    
    "github.com/google/wire"
    "gorm.io/gorm"
    
    "github.com/iyunix/go-internist/internal/config"
    "github.com/iyunix/go-internist/internal/handlers"
    "github.com/iyunix/go-internist/internal/repository/chat"
    "github.com/iyunix/go-internist/internal/repository/message"
    "github.com/iyunix/go-internist/internal/repository/user"
    "github.com/iyunix/go-internist/internal/repository/verification"
    "github.com/iyunix/go-internist/internal/services"
    "github.com/iyunix/go-internist/internal/services/admin_services"
    "github.com/iyunix/go-internist/internal/services/ai"
    "github.com/iyunix/go-internist/internal/services/sms"
    "github.com/iyunix/go-internist/internal/services/user_services"
)

// Application aggregates all services and handlers
type Application struct {
    Config             *config.Config
    Logger             services.Logger
    AuthHandler        *handlers.AuthHandler
    ChatHandler        *handlers.ChatHandler
    PageHandler        *handlers.PageHandler
    AdminHandler       *handlers.AdminHandler
    ChatService        *services.ChatService
    AIService          *services.AIService
    PineconeService    *services.PineconeService
    SMSService         *services.SMSService
    UserService        *user_services.UserService
    AuthService        *user_services.AuthService
    VerificationService *user_services.VerificationService
    BalanceService     *user_services.BalanceService
    AdminService       *admin_services.AdminService
    UserRepo           user.UserRepository
}

// Wrapper types to avoid string ambiguity
type JWTSecret string
type AdminPhone string

// Provider functions
func ProvideConfig() (*config.Config, error) {
    return config.New()
}

func ProvideLogger() services.Logger {
    return services.NewLogger("go_internist")
}

func ProvideJWTSecret(cfg *config.Config) JWTSecret {
    return JWTSecret(cfg.JWTSecretKey)
}

func ProvideAdminPhone(cfg *config.Config) AdminPhone {
    return AdminPhone(cfg.AdminPhoneNumber)
}

// ✅ NEW: Add missing providers
func ProvideRetrievalTopK(cfg *config.Config) int {
    return cfg.RetrievalTopK
}

func ProvideUserServicesLogger(logger services.Logger) user_services.Logger {
    return logger
}

func ProvideAdminServicesLogger(logger services.Logger) admin_services.Logger {
    return logger
}

// Wrapped constructors for user services
func NewUserServiceWrapped(repo user.UserRepository, jwtSecret JWTSecret, adminPhone AdminPhone, logger services.Logger) *user_services.UserService {
    return user_services.NewUserService(repo, string(jwtSecret), string(adminPhone), logger)
}

func NewAuthServiceWrapped(repo user.UserRepository, jwtSecret JWTSecret, adminPhone AdminPhone, logger services.Logger) *user_services.AuthService {
    return user_services.NewAuthService(repo, string(jwtSecret), string(adminPhone), logger)
}

func ProvideAIConfig(cfg *config.Config) *ai.Config {
    aiConfig := ai.DefaultConfig()
    aiConfig.EmbeddingKey = cfg.AvalaiAPIKeyEmbedding
    aiConfig.LLMKey = cfg.JabirAPIKey
    aiConfig.EmbeddingBaseURL = "https://api.avalai.ir/v1"
    aiConfig.LLMBaseURL = "https://openai.jabirproject.org/v1"
    aiConfig.EmbeddingModel = cfg.EmbeddingModelName
    return aiConfig
}

func ProvideSMSConfig(cfg *config.Config) *sms.Config {
    templateID, _ := strconv.Atoi(cfg.SMSTemplateID)
    return &sms.Config{
        AccessKey:  cfg.SMSAccessKey,    // ✅ From config
        TemplateID: templateID,
        APIURL:     cfg.SMSAPIURL,       // ✅ From config
        Timeout:    30 * time.Second,
    }
}


func ProvideAIProvider(aiConfig *ai.Config) ai.AIProvider {
    return ai.NewOpenAIProvider(aiConfig)
}

func ProvideSMSProvider(smsConfig *sms.Config) sms.Provider {
    return sms.NewSMSIRProvider(smsConfig)
}

func ProvidePineconeService(cfg *config.Config, logger services.Logger) (*services.PineconeService, error) {
    return services.NewPineconeService(
        cfg.PineconeAPIKey,
        cfg.PineconeIndexHost,
        cfg.PineconeNamespace,
        logger,
    )
}

func InitializeApplication(cfg *config.Config, logger services.Logger, db *gorm.DB) (*Application, error) {
    wire.Build(
        // Basic providers
        ProvideJWTSecret,
        ProvideAdminPhone,
        ProvideRetrievalTopK,
        ProvideUserServicesLogger,
        ProvideAdminServicesLogger,
        
        // AI Configuration
        ProvideAIConfig,
        ProvideAIProvider,
        
        // SMS Configuration  
        ProvideSMSConfig,
        ProvideSMSProvider,
        
        // Pinecone Service
        ProvidePineconeService,
        
        // Repositories
        user.NewGormUserRepository,
        verification.NewGormVerificationRepository,
        chat.NewChatRepository,
        message.NewMessageRepository,
        
        // Core Services
        services.NewAIService,
        services.NewSMSService,
        services.NewChatService,
        
        // User Services (wrapped)
        NewUserServiceWrapped,
        NewAuthServiceWrapped,
        user_services.NewVerificationService,
        user_services.NewBalanceService,
        
        // Admin Services
        admin_services.NewAdminService,
        
        // Handlers
        handlers.NewAuthHandler,
        handlers.NewChatHandler,
        handlers.NewPageHandler,
        handlers.NewAdminHandler,
        
        // Application constructor
        wire.Struct(new(Application), "*"),
    )
    return &Application{}, nil
}
