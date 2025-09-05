// File: internal/handlers/auth_handlers.go
package handlers

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/iyunix/go-internist/internal/domain"
	"github.com/iyunix/go-internist/internal/services"
)

var (
	usernameRegex     = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)
	phoneRegex        = regexp.MustCompile(`^\+?[0-9]{7,15}$`)
	passwordMinLength = 8
)

// AuthHandler holds the dependencies for authentication handlers.
type AuthHandler struct {
	UserService *services.UserService
	SMSService  *services.SMSService
}


// This is the correct version
func NewAuthHandler(userService *services.UserService, smsService *services.SMSService) *AuthHandler {
    return &AuthHandler{
        UserService: userService, // Use the userService that was passed in
        SMSService:  smsService,  // Use the smsService that was passed in
    }
}

// Register handles new user registrations by orchestrating validation, verification, and user creation.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	username, phone, password, errMsg := validateInput(
		r.FormValue("username"),
		r.FormValue("phone_number"),
		r.FormValue("password"),
	)
	if errMsg != "" {
		data := convertToInterfaceMap(map[string]string{"Error": errMsg})
		renderTemplate(w, "register.html", data)
		return
	}

	// Generate a cryptographically secure 6-digit code.
	code, err := generateSecureCode()
	if err != nil {
		log.Printf("Failed to generate secure code: %v", err)
		data := convertToInterfaceMap(map[string]string{"Error": "An internal error occurred. Please try again."})
		renderTemplate(w, "register.html", data)
		return
	}

	// Delegate temporary storage and logic to the service layer.
	// The service should handle hashing the password and storing user data in a "pending" state.
	pendingUser := &domain.User{Username: username, PhoneNumber: phone, Password: password}
	verificationTTL := 10 * time.Minute // Codes are valid for 10 minutes

	if err := h.UserService.InitiateVerification(r.Context(), pendingUser, code, verificationTTL); err != nil {
		log.Printf("Failed to initiate verification: %v", err)
		data := convertToInterfaceMap(map[string]string{"Error": err.Error()})
		renderTemplate(w, "register.html", data)
		return
	}

	// Send the SMS via the SMS service.
    if err := h.SMSService.SendVerificationCode(r.Context(), phone, code); err != nil {
		log.Printf("SMS send error: %v", err)
		data := convertToInterfaceMap(map[string]string{"Error": "Failed to send SMS. Please try again."})
		renderTemplate(w, "register.html", data)
		return
	}

	// Redirect to the verification page on success.
	data := map[string]interface{}{"PhoneNumber": phone}
	renderTemplate(w, "verify_sms.html", data)
}

// VerifySMS handles code verification and finalizes registration.
func (h *AuthHandler) VerifySMS(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}
	phone := r.FormValue("phone_number")
	code := r.FormValue("sms_code")

	// Delegate the core verification logic to the user service.
	// This service method should check the code, ensure it's not expired, and finalize the user.
	if _, err := h.UserService.FinalizeVerification(r.Context(), phone, code); err != nil {
		log.Printf("Verification error: %v", err)
		data := map[string]interface{}{"Error": err.Error(), "PhoneNumber": phone}
		renderTemplate(w, "verify_sms.html", data)
		return
	}

	// On successful verification, redirect to the login page.
	http.Redirect(w, r, "/login?verified=true", http.StatusSeeOther)
}

// ResendSMS handles resending the verification code.
func (h *AuthHandler) ResendSMS(w http.ResponseWriter, r *http.Request) {
	phone := r.URL.Query().Get("phone")
	if !phoneRegex.MatchString(phone) {
		data := map[string]interface{}{"Error": "Invalid phone number format.", "PhoneNumber": phone}
		renderTemplate(w, "verify_sms.html", data)
		return
	}

	// Generate a new secure code for the resend attempt.
	code, err := generateSecureCode()
	if err != nil {
		log.Printf("Failed to generate secure code for resend: %v", err)
		data := map[string]interface{}{"Error": "Failed to generate new code.", "PhoneNumber": phone}
		renderTemplate(w, "verify_sms.html", data)
		return
	}

	verificationTTL := 10 * time.Minute

	// Delegate the resend logic to the service.
	// The service should check if there is a pending user and if a resend is allowed (e.g., rate-limiting).
	if err := h.UserService.ResendVerificationCode(r.Context(), phone, code, verificationTTL); err != nil {
		log.Printf("SMS resend logic error: %v", err)
		data := map[string]interface{}{"Error": err.Error(), "PhoneNumber": phone}
		renderTemplate(w, "verify_sms.html", data)
		return
	}

	// Send the new code via the SMS service.
    if err := h.SMSService.SendVerificationCode(r.Context(), phone, code); err != nil {
		log.Printf("SMS resend send error: %v", err)
		data := map[string]interface{}{"Error": "Failed to resend SMS. Please try again.", "PhoneNumber": phone}
		renderTemplate(w, "verify_sms.html", data)
		return
	}

	// Let the user know the code has been successfully resent.
	data := map[string]interface{}{
		"PhoneNumber": phone,
		"Success":     "A new verification code has been sent.",
	}
	renderTemplate(w, "verify_sms.html", data)
}

// Login validates user credentials, sets auth cookies, and redirects to chat.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := strings.TrimSpace(r.FormValue("password"))
	if username == "" || password == "" {
		data := convertToInterfaceMap(map[string]string{"Error": "Username and password are required."})
		renderTemplate(w, "login.html", data)
		return
	}

	token, err := h.UserService.Login(r.Context(), username, password)
	if err != nil {
		log.Printf("Login error: %v", err)
		data := convertToInterfaceMap(map[string]string{"Error": "Invalid username or password."})
		renderTemplate(w, "login.html", data)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   true, // Set to false if not using HTTPS in local dev
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/chat", http.StatusSeeOther)
}

// Logout clears the auth_token cookie and redirects to login.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true, // Set to false if not using HTTPS in local dev
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// generateSecureCode creates a cryptographically secure 6-digit code.
func generateSecureCode() (string, error) {
	var b [6]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		return "", err
	}
	// A simple way to get a 6-digit number from random bytes.
	code := (int(b[0])<<40 + int(b[1])<<32 + int(b[2])<<24 + int(b[3])<<16 + int(b[4])<<8 + int(b[5])) % 1000000
	return fmt.Sprintf("%06d", code), nil
}

// validateInput ensures that username, phone, and password meet basic rules.
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

// convertToInterfaceMap transforms map[string]string into map[string]interface{}.
func convertToInterfaceMap(strMap map[string]string) map[string]interface{} {
	ifaceMap := make(map[string]interface{}, len(strMap))
	for k, v := range strMap {
		ifaceMap[k] = v
	}
	return ifaceMap
}