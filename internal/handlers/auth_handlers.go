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
	usernameRegex     = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)
	phoneRegex        = regexp.MustCompile(`^\+?[0-9]{7,15}$`)
	passwordMinLength = 8
)

type AuthHandler struct {
	UserService         *user_services.UserService
	AuthService         *user_services.AuthService
	VerificationService *user_services.VerificationService
	SMSService          *services.SMSService
	BalanceService      *user_services.BalanceService
}

func NewAuthHandler(userService *user_services.UserService,
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

// Login authenticates the user (username OR phone) and sets the JWT cookie.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}
	identifier := strings.TrimSpace(r.FormValue("username"))
	password := strings.TrimSpace(r.FormValue("password"))

	if identifier == "" || password == "" {
		data := map[string]interface{}{"Error": "Username/phone and password are required."}
		RenderTemplate(w, "login.html", data)
		return
	}

	_, token, err := h.AuthService.Login(r.Context(), identifier, password)
	if err != nil {
		log.Printf("Login error: %v", err)
		data := map[string]interface{}{
			"Error":      "Invalid credentials.",
			"Identifier": identifier,
		}
		RenderTemplate(w, "login.html", data)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/chat", http.StatusSeeOther)
}

// Register new user
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.FormValue("username")
	phone := r.FormValue("phone_number")
	validatedUsername, validatedPhone, password, errMsg := validateInput(username, phone, r.FormValue("password"))
	if errMsg != "" {
		data := map[string]interface{}{"Error": errMsg, "Username": validatedUsername, "PhoneNumber": validatedPhone}
		RenderTemplate(w, "register.html", data)
		return
	}
	user, err := h.AuthService.Register(r.Context(), validatedUsername, validatedPhone, password)
	if err != nil {
		log.Printf("Failed to register user: %v", err)
		data := map[string]interface{}{"Error": err.Error(), "Username": validatedUsername, "PhoneNumber": validatedPhone}
		RenderTemplate(w, "register.html", data)
		return
	}
	if err := h.VerificationService.SendVerificationCode(r.Context(), user.ID); err != nil {
		log.Printf("Failed to send verification code: %v", err)
		data := map[string]interface{}{"Error": err.Error()}
		RenderTemplate(w, "register.html", data)
		return
	}
	http.Redirect(w, r, "/verify-sms?phone="+validatedPhone, http.StatusSeeOther)
}

// Forgot password: always redirect to verification
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
	_ = h.VerificationService.SendPasswordResetCode(r.Context(), phone)
	http.Redirect(w, r, "/verify-sms?phone="+phone+"&action=reset", http.StatusSeeOther)
}

// VerifySMS for either registration or password reset
func (h *AuthHandler) VerifySMS(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}
	phone := r.FormValue("phone_number")
	code := r.FormValue("sms_code")
	action := r.URL.Query().Get("action")

	user, err := h.UserService.GetUserByPhone(r.Context(), phone)
	if err != nil {
		data := map[string]interface{}{"Error": "User not found or invalid phone number.", "PhoneNumber": phone}
		RenderTemplate(w, "verify_sms.html", data)
		return
	}

	var verificationErr error
	if action == "reset" {
		verificationErr = h.VerificationService.VerifyPasswordResetCode(r.Context(), user.ID, code)
	} else {
		verificationErr = h.VerificationService.VerifyCode(r.Context(), user.ID, code)
	}
	if verificationErr != nil {
		log.Printf("Verification error for action '%s': %v", action, verificationErr)
		data := map[string]interface{}{"Error": verificationErr.Error(), "PhoneNumber": phone}
		RenderTemplate(w, "verify_sms.html", data)
		return
	}

	if action == "reset" {
		http.Redirect(w, r, "/reset-password?phone="+phone+"&code="+code, http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/login?verified=true", http.StatusSeeOther)
	}
}

// Reset password: final step
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
		http.Redirect(w, r, "/reset-password?error=Passwords do not match&phone="+phone+"&code="+code, http.StatusSeeOther)
		return
	}
	if len(password) < passwordMinLength {
		http.Redirect(w, r, "/reset-password?error=Password is too short&phone="+phone+"&code="+code, http.StatusSeeOther)
		return
	}
	if err := h.VerificationService.VerifyAndResetPassword(r.Context(), phone, code, password); err != nil {
		log.Printf("Error resetting password for phone %s: %v", phone, err)
		http.Redirect(w, r, "/reset-password?error=Invalid code or user. Please try again.&phone="+phone, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/login?reset=success", http.StatusSeeOther)
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
		data := map[string]interface{}{"Error": "User not found.", "PhoneNumber": phone}
		RenderTemplate(w, "verify_sms.html", data)
		return
	}
	if err := h.VerificationService.ResendVerificationCode(r.Context(), user.ID); err != nil {
		data := map[string]interface{}{"Error": err.Error(), "PhoneNumber": phone}
		RenderTemplate(w, "verify_sms.html", data)
		return
	}
	data := map[string]interface{}{"PhoneNumber": phone, "Success": "A new verification code has been sent."}
	RenderTemplate(w, "verify_sms.html", data)
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

func (h *AuthHandler) GetUserCreditHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(uint)
	if !ok {
		http.Error(w, "Authentication error", http.StatusUnauthorized)
		return
	}
	currentBalance, totalBalance, err := h.BalanceService.GetUserBalanceInfo(r.Context(), userID)
	if err != nil {
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
	json.NewEncoder(w).Encode(response)
}

// simple helper for registration input validation
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
