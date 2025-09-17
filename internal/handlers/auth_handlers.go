// G:\go_internist\internal\handlers\auth_handlers.go
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
	"fmt"
    "math/rand"
    "golang.org/x/crypto/bcrypt"
	"github.com/iyunix/go-internist/internal/services"
	"github.com/iyunix/go-internist/internal/services/user_services"
)

var (
	usernameRegex     = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)
	phoneRegex        = regexp.MustCompile(`^09\d{9}$`)
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


type PendingRegistration struct {
    Username string
    Phone    string
    Password string // store as hash!
    Code     string
    Expires  time.Time
}
var pendingRegistrations = make(map[string]*PendingRegistration)


func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()
    username := r.FormValue("username")
    phone := r.FormValue("phone_number")
    password := r.FormValue("password")
    _, validatedPhone, validPassword, errMsg := validateInput(username, phone, password)
    if errMsg != "" {
        data := map[string]interface{}{"Error": errMsg, "Username": username, "PhoneNumber": validatedPhone}
        RenderTemplate(w, "register.html", data)
        return
    }

		// Check if username is already taken!
	existingByUsername, err := h.UserService.GetUserByUsername(r.Context(), username)
	if err == nil && existingByUsername != nil {
		data := map[string]interface{}{
			"Error":      "Username is already taken.",
			"Username":   username,
			"PhoneNumber": validatedPhone,
		}
		RenderTemplate(w, "register.html", data)
		return
	}
    // Check if user already exists!
    existingUser, err := h.UserService.GetUserByPhone(r.Context(), validatedPhone)
    if err == nil && existingUser != nil {
        data := map[string]interface{}{"Error": "User already exists.", "Username": username, "PhoneNumber": validatedPhone}
        RenderTemplate(w, "register.html", data)
        return
    }


    hash, _ := bcrypt.GenerateFromPassword([]byte(validPassword), bcrypt.DefaultCost)
    code := generate6DigitCode()
    pendingRegistrations[validatedPhone] = &PendingRegistration{
        Username: username, Phone: validatedPhone, Password: string(hash),
        Code: code, Expires: time.Now().Add(15 * time.Minute),
    }
    h.SMSService.SendVerificationCode(r.Context(), validatedPhone, code)
    http.Redirect(w, r, "/verify-sms?phone="+validatedPhone, http.StatusSeeOther)
}

func generate6DigitCode() string {
    return fmt.Sprintf("%06d", rand.Intn(1000000))
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

func (h *AuthHandler) VerifySMS(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Invalid form data", http.StatusBadRequest)
        return
    }
    phone := r.FormValue("phone_number")
    code := r.FormValue("sms_code")
    action := r.FormValue("action")

    if action == "reset" {
        // Old password reset logic (as before)
        user, err := h.UserService.GetUserByPhone(r.Context(), phone)
        if err != nil {
            data := map[string]interface{}{"Error": "User not found or invalid phone number.", "PhoneNumber": phone, "Action": action}
            RenderTemplate(w, "verify_sms.html", data)
            return
        }
        verificationErr := h.VerificationService.VerifyPasswordResetCode(r.Context(), user.ID, code)
        if verificationErr != nil {
            // handle error
            data := map[string]interface{}{"Error": verificationErr.Error(), "PhoneNumber": phone, "Action": action}
            RenderTemplate(w, "verify_sms.html", data)
            return
        }
        http.Redirect(w, r, "/reset-password?phone="+phone+"&code="+code, http.StatusSeeOther)
        return
    }

    // ---- Registration flow: Only use pendingRegistrations, don't look up DB user ----
    pend, ok := pendingRegistrations[phone]
    if !ok || time.Now().After(pend.Expires) {
        data := map[string]interface{}{"Error": "Verification expired or not found.", "PhoneNumber": phone}
        RenderTemplate(w, "verify_sms.html", data)
        return
    }
    if pend.Code != code {
        data := map[string]interface{}{"Error": "Invalid verification code.", "PhoneNumber": phone}
        RenderTemplate(w, "verify_sms.html", data)
        return
    }
    // Actually create DB user
    user, err := h.AuthService.Register(r.Context(), pend.Username, pend.Phone, pend.Password)
    if err != nil {
        delete(pendingRegistrations, phone)
        data := map[string]interface{}{"Error": "Failed to create user.", "PhoneNumber": phone}
        RenderTemplate(w, "verify_sms.html", data)
        return
    }
    user.IsVerified = true
    user.Status = "active"
    now := time.Now()
    user.VerifiedAt = &now
    h.UserService.UpdateUser(r.Context(), user)
    delete(pendingRegistrations, phone)
    http.Redirect(w, r, "/login?verified=true", http.StatusSeeOther)
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
