// File: internal/handlers/auth_handlers.go
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/iyunix/go-internist/internal/services"
	"github.com/iyunix/go-internist/internal/services/user_services"
)

var (
	usernameRegex   = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)
	phoneRegex      = regexp.MustCompile(`^\+?[0-9]{7,15}$`)
	passwordMinLength = 8
)

type AuthHandler struct {
	UserService         *user_services.UserService
	AuthService         *user_services.AuthService
	VerificationService *user_services.VerificationService
	SMSService          *services.SMSService
	BalanceService      *user_services.BalanceService
}

func NewAuthHandler(
	userService *user_services.UserService,
	authService *user_services.AuthService,
	verificationService *user_services.VerificationService,
	smsService *services.SMSService,
	balanceService *user_services.BalanceService,
) *AuthHandler {
	return &AuthHandler{
		UserService:         userService,
		AuthService:         authService,
		VerificationService: verificationService,
		SMSService:          smsService,
		BalanceService:      balanceService,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	phone := r.FormValue("phone_number")
	
	validatedUsername, validatedPhone, password, errMsg := validateInput(
		username,
		phone,
		r.FormValue("password"),
	)
	if errMsg != "" {
		// CHANGE: Pass back user input along with the error for a better UX.
		data := map[string]interface{}{
			"Error":       errMsg,
			"Username":    validatedUsername,
			"PhoneNumber": validatedPhone,
		}
		RenderTemplate(w, "register.html", data)
		return
	}

	user, err := h.AuthService.Register(r.Context(), validatedUsername, validatedPhone, password)
	if err != nil {
		log.Printf("Failed to register user: %v", err)
		data := map[string]interface{}{
			"Error":       err.Error(),
			"Username":    validatedUsername,
			"PhoneNumber": validatedPhone,
		}
		RenderTemplate(w, "register.html", data)
		return
	}

	if err := h.VerificationService.SendVerificationCode(r.Context(), user.ID); err != nil {
		log.Printf("Failed to send verification code: %v", err)
		data := map[string]interface{}{"Error": err.Error()}
		RenderTemplate(w, "register.html", data)
		return
	}

	// Redirect to the verification page with the phone number
	http.Redirect(w, r, "/verify-sms?phone="+validatedPhone, http.StatusSeeOther)
}

// ... (VerifySMS, ResendSMS, Login, Logout handlers remain unchanged) ...

func (h *AuthHandler) VerifySMS(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Invalid form data", http.StatusBadRequest)
        return
    }
    phone := r.FormValue("phone_number")
    code := r.FormValue("sms_code")

    user, err := h.UserService.GetUserByPhone(r.Context(), phone)
    if err != nil {
        log.Printf("User not found: %v", err)
        data := map[string]interface{}{"Error": "User not found or invalid phone number.", "PhoneNumber": phone}
        RenderTemplate(w, "verify_sms.html", data)
        return
    }

    if err := h.VerificationService.VerifyCode(r.Context(), user.ID, code); err != nil {
        log.Printf("Verification error: %v", err)
        data := map[string]interface{}{"Error": err.Error(), "PhoneNumber": phone}
        RenderTemplate(w, "verify_sms.html", data)
        return
    }

    http.Redirect(w, r, "/login?verified=true", http.StatusSeeOther)
}

func (h *AuthHandler) ResendSMS(w http.ResponseWriter, r *http.Request) {
    phone := r.URL.Query().Get("phone")
    if !phoneRegex.MatchString(phone) {
        data := map[string]interface{}{"Error": "Invalid phone number format.", "PhoneNumber": phone}
        RenderTemplate(w, "verify_sms.html", data)
        return
    }

    user, err := h.UserService.GetUserByPhone(r.Context(), phone)
    if err != nil {
        log.Printf("User not found for resend: %v", err)
        data := map[string]interface{}{"Error": "User not found.", "PhoneNumber": phone}
        RenderTemplate(w, "verify_sms.html", data)
        return
    }

    if err := h.VerificationService.ResendVerificationCode(r.Context(), user.ID); err != nil {
        log.Printf("SMS resend logic error: %v", err)
        data := map[string]interface{}{"Error": err.Error(), "PhoneNumber": phone}
        RenderTemplate(w, "verify_sms.html", data)
        return
    }

    data := map[string]interface{}{
        "PhoneNumber": phone,
        "Success":     "A new verification code has been sent.",
    }
    RenderTemplate(w, "verify_sms.html", data)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Invalid form data", http.StatusBadRequest)
        return
    }

    username := strings.TrimSpace(r.FormValue("username"))
    password := strings.TrimSpace(r.FormValue("password"))
    if username == "" || password == "" {
        data := map[string]interface{}{"Error": "Username and password are required."}
        RenderTemplate(w, "login.html", data)
        return
    }

    _, token, err := h.AuthService.Login(r.Context(), username, password)
    if err != nil {
        log.Printf("Login error: %v", err)
        data := map[string]interface{}{"Error": "Invalid username or password."}
        RenderTemplate(w, "login.html", data)
        return
    }

    http.SetCookie(w, &http.Cookie{
        Name:     "auth_token",
        Value:    token,
        Expires:  time.Now().Add(24 * time.Hour),
        HttpOnly: true,
        Secure:   r.TLS != nil, // Use secure cookies in production
        Path:     "/",
        SameSite: http.SameSiteLaxMode,
    })
    http.Redirect(w, r, "/chat", http.StatusSeeOther)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
    http.SetCookie(w, &http.Cookie{
        Name:     "auth_token",
        Value:    "",
        Expires:  time.Unix(0, 0),
        HttpOnly: true,
        Secure:   r.TLS != nil,
        Path:     "/",
        SameSite: http.SameSiteLaxMode,
    })
    http.Redirect(w, r, "/login", http.StatusSeeOther)
}


// --- NEW HANDLERS FOR PASSWORD RESET ---

// HandleForgotPassword initiates the password reset process.
func (h *AuthHandler) HandleForgotPassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}
	phone := r.FormValue("phone_number")
	if !phoneRegex.MatchString(phone) {
		data := map[string]interface{}{"Error": "Invalid phone number format."}
		RenderTemplate(w, "forgot_password.html", data)
		return
	}

	// This will require a new method in your VerificationService
	err := h.VerificationService.SendPasswordResetCode(r.Context(), phone)
	if err != nil {
		log.Printf("Error sending password reset code for phone %s: %v", phone, err)
		// Security: Show a generic message to prevent phone number enumeration
	}

	data := map[string]interface{}{"Success": "If an account with that phone number exists, a reset code has been sent."}
	RenderTemplate(w, "forgot_password.html", data)
}

// HandleResetPassword verifies the code and sets the new password.
func (h *AuthHandler) HandleResetPassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}
	phone := r.FormValue("phone_number")
	code := r.FormValue("reset_code")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	if password != confirmPassword {
		http.Redirect(w, r, "/reset-password?error=passwords_do_not_match&phone="+phone+"&code="+code, http.StatusSeeOther)
		return
	}
	if len(password) < passwordMinLength {
		http.Redirect(w, r, "/reset-password?error=password_too_short&phone="+phone+"&code="+code, http.StatusSeeOther)
		return
	}

	// This will require a new method in your VerificationService
	err := h.VerificationService.VerifyAndResetPassword(r.Context(), phone, code, password)
	if err != nil {
		log.Printf("Error resetting password for phone %s: %v", phone, err)
		http.Redirect(w, r, "/reset-password?error=invalid_code_or_user&phone="+phone+"&code="+code, http.StatusSeeOther)
		return
	}

	// Success! Redirect to login with a success message.
	http.Redirect(w, r, "/login?reset=success", http.StatusSeeOther)
}


// --- EXISTING HANDLERS AND HELPERS ---

func (h *AuthHandler) GetUserCreditHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(uint)
	if !ok {
		http.Error(w, "Authentication error: User ID not found in context", http.StatusUnauthorized)
		return
	}

	currentBalance, totalBalance, err := h.BalanceService.GetUserBalanceInfo(r.Context(), userID)
	if err != nil {
		log.Printf("Error getting user balance info for user %d: %v", userID, err)
		http.Error(w, "Failed to retrieve user credit", http.StatusInternalServerError)
		return
	}

	response := struct {
		Balance      int `json:"balance"`
		TotalCredits int `json:"totalCredits"`
	}{
		Balance:      currentBalance,
		TotalCredits: totalBalance,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func validateInput(username, phone, password string) (string, string, string, string) {
	username = strings.TrimSpace(username)
	phone = strings.TrimSpace(phone)
	password = strings.TrimSpace(password)

	var errMsg string
	switch {
	case !usernameRegex.MatchString(username):
		errMsg = "Username must be 3-20 characters, alphanumeric or underscore."
	case !phoneRegex.MatchString(phone):
		errMsg = "Phone number format invalid."
	case len(password) < passwordMinLength:
		errMsg = "Password must be at least 8 characters."
	}
	return username, phone, password, errMsg
}