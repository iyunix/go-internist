// File: internal/handlers/auth_handlers.go
package handlers

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/iyunix/go-internist/internal/domain"
	"github.com/iyunix/go-internist/internal/services"
	"github.com/iyunix/go-internist/internal/services/user_services"
)

var (
	usernameRegex     = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)
	phoneRegex        = regexp.MustCompile(`^\+?[0-9]{7,15}$`)
	passwordMinLength = 8
)

// AuthHandler holds the dependencies for authentication handlers.
type AuthHandler struct {
	UserService    *user_services.UserService
	SMSService     *services.SMSService
	BalanceService *user_services.BalanceService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(
	userService *user_services.UserService,
	smsService *services.SMSService,
	balanceService *user_services.BalanceService,
) *AuthHandler {
	return &AuthHandler{
		UserService:    userService,
		SMSService:     smsService,
		BalanceService: balanceService,
	}
}

// --- All handlers from Register to Logout remain unchanged ---

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

	code, err := generateSecureCode()
	if err != nil {
		log.Printf("Failed to generate secure code: %v", err)
		data := convertToInterfaceMap(map[string]string{"Error": "An internal error occurred. Please try again."})
		renderTemplate(w, "register.html", data)
		return
	}

	pendingUser := &domain.User{Username: username, PhoneNumber: phone, Password: password}
	verificationTTL := 10 * time.Minute

	if err := h.UserService.InitiateVerification(r.Context(), pendingUser, code, verificationTTL); err != nil {
		log.Printf("Failed to initiate verification: %v", err)
		data := convertToInterfaceMap(map[string]string{"Error": err.Error()})
		renderTemplate(w, "register.html", data)
		return
	}

	if err := h.SMSService.SendVerificationCode(r.Context(), phone, code); err != nil {
		log.Printf("SMS send error: %v", err)
		data := convertToInterfaceMap(map[string]string{"Error": "Failed to send SMS. Please try again."})
		renderTemplate(w, "register.html", data)
		return
	}

	data := map[string]interface{}{"PhoneNumber": phone}
	renderTemplate(w, "verify_sms.html", data)
}

func (h *AuthHandler) VerifySMS(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}
	phone := r.FormValue("phone_number")
	code := r.FormValue("sms_code")

	if _, err := h.UserService.FinalizeVerification(r.Context(), phone, code); err != nil {
		log.Printf("Verification error: %v", err)
		data := map[string]interface{}{"Error": err.Error(), "PhoneNumber": phone}
		renderTemplate(w, "verify_sms.html", data)
		return
	}

	http.Redirect(w, r, "/login?verified=true", http.StatusSeeOther)
}

func (h *AuthHandler) ResendSMS(w http.ResponseWriter, r *http.Request) {
	phone := r.URL.Query().Get("phone")
	if !phoneRegex.MatchString(phone) {
		data := map[string]interface{}{"Error": "Invalid phone number format.", "PhoneNumber": phone}
		renderTemplate(w, "verify_sms.html", data)
		return
	}

	code, err := generateSecureCode()
	if err != nil {
		log.Printf("Failed to generate secure code for resend: %v", err)
		data := map[string]interface{}{"Error": "Failed to generate new code.", "PhoneNumber": phone}
		renderTemplate(w, "verify_sms.html", data)
		return
	}

	verificationTTL := 10 * time.Minute

	if err := h.UserService.ResendVerificationCode(r.Context(), phone, code, verificationTTL); err != nil {
		log.Printf("SMS resend logic error: %v", err)
		data := map[string]interface{}{"Error": err.Error(), "PhoneNumber": phone}
		renderTemplate(w, "verify_sms.html", data)
		return
	}

	if err := h.SMSService.SendVerificationCode(r.Context(), phone, code); err != nil {
		log.Printf("SMS resend send error: %v", err)
		data := map[string]interface{}{"Error": "Failed to resend SMS. Please try again.", "PhoneNumber": phone}
		renderTemplate(w, "verify_sms.html", data)
		return
	}

	data := map[string]interface{}{
		"PhoneNumber": phone,
		"Success":     "A new verification code has been sent.",
	}
	renderTemplate(w, "verify_sms.html", data)
}

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
		Secure:   false, // Set to false for local HTTP development
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
		Secure:   false, // Set to false for local HTTP development
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}


// --- THIS FUNCTION HAS BEEN UPDATED ---
// GetUserCreditHandler handles the API request for the user's credit balance.
func (h *AuthHandler) GetUserCreditHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(uint)
	if !ok {
		http.Error(w, "Authentication error: User ID not found in context", http.StatusUnauthorized)
		return
	}

	// UPDATED: Call the new service function to get both current and total balance.
	currentBalance, totalBalance, err := h.BalanceService.GetUserBalanceInfo(r.Context(), userID)
	if err != nil {
		log.Printf("Error getting user balance info for user %d: %v", userID, err)
		http.Error(w, "Failed to retrieve user credit", http.StatusInternalServerError)
		return
	}

	// UPDATED: The response now uses the dynamic values fetched from the database.
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

// --- Helper functions remain unchanged ---

func generateSecureCode() (string, error) {
	var b [6]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		return "", err
	}
	code := (int(b[0])<<40 + int(b[1])<<32 + int(b[2])<<24 + int(b[3])<<16 + int(b[4])<<8 + int(b[5])) % 1000000
	return fmt.Sprintf("%06d", code), nil
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

func convertToInterfaceMap(strMap map[string]string) map[string]interface{} {
	ifaceMap := make(map[string]interface{}, len(strMap))
	for k, v := range strMap {
		ifaceMap[k] = v
	}
	return ifaceMap
}