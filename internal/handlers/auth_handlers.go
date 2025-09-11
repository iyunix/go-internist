// G:\go_internist\internal\handlers\auth_handlers.go
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

    // ✅ FIXED: Now passes username as first parameter
    user, err := h.AuthService.Register(r.Context(), username, phone, password)
    if err != nil {
        log.Printf("Failed to register user: %v", err)
        data := convertToInterfaceMap(map[string]string{"Error": err.Error()})
        renderTemplate(w, "register.html", data)
        return
    }

    // ✅ REMOVE: No longer needed since username is set during creation
    // user.Username = username
    // if err := h.UserService.UpdateUser(r.Context(), user); err != nil {
    //     log.Printf("Failed to update username: %v", err)
    //     data := convertToInterfaceMap(map[string]string{"Error": "Registration failed. Please try again."})
    //     renderTemplate(w, "register.html", data)
    //     return
    // }

    // Send verification code
    if err := h.VerificationService.SendVerificationCode(r.Context(), user.ID); err != nil {
        log.Printf("Failed to send verification code: %v", err)
        data := convertToInterfaceMap(map[string]string{"Error": err.Error()})
        renderTemplate(w, "register.html", data)
        return
    }

    data := map[string]interface{}{"PhoneNumber": phone}
    renderTemplate(w, "verify_sms.html", data)
}



// FIXED: VerifySMS method
func (h *AuthHandler) VerifySMS(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Invalid form data", http.StatusBadRequest)
        return
    }
    phone := r.FormValue("phone_number")
    code := r.FormValue("sms_code")

    // FIXED: Find user by phone first, then verify
    user, err := h.UserService.GetUserByPhone(r.Context(), phone)
    if err != nil {
        log.Printf("User not found: %v", err)
        data := map[string]interface{}{"Error": "User not found", "PhoneNumber": phone}
        renderTemplate(w, "verify_sms.html", data)
        return
    }

    // FIXED: Use correct method signature (returns only error)
    if err := h.VerificationService.VerifyCode(r.Context(), user.ID, code); err != nil {
        log.Printf("Verification error: %v", err)
        data := map[string]interface{}{"Error": err.Error(), "PhoneNumber": phone}
        renderTemplate(w, "verify_sms.html", data)
        return
    }

    http.Redirect(w, r, "/login?verified=true", http.StatusSeeOther)
}

// FIXED: ResendSMS method
func (h *AuthHandler) ResendSMS(w http.ResponseWriter, r *http.Request) {
    phone := r.URL.Query().Get("phone")
    if !phoneRegex.MatchString(phone) {
        data := map[string]interface{}{"Error": "Invalid phone number format.", "PhoneNumber": phone}
        renderTemplate(w, "verify_sms.html", data)
        return
    }

    // FIXED: Find user by phone first
    user, err := h.UserService.GetUserByPhone(r.Context(), phone)
    if err != nil {
        log.Printf("User not found for resend: %v", err)
        data := map[string]interface{}{"Error": "User not found", "PhoneNumber": phone}
        renderTemplate(w, "verify_sms.html", data)
        return
    }

    // FIXED: Use correct method signature with user ID
    if err := h.VerificationService.ResendVerificationCode(r.Context(), user.ID); err != nil {
        log.Printf("SMS resend logic error: %v", err)
        data := map[string]interface{}{"Error": err.Error(), "PhoneNumber": phone}
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

    // FIXED: Use blank identifier to ignore unused user return value
    _, token, err := h.AuthService.Login(r.Context(), username, password)
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
        Secure:   false,
        Path:     "/",
        SameSite: http.SameSiteLaxMode,
    })
    http.Redirect(w, r, "/chat", http.StatusSeeOther)
}


// ... rest of your methods remain the same ...

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
    http.SetCookie(w, &http.Cookie{
        Name:     "auth_token",
        Value:    "",
        Expires:  time.Unix(0, 0),
        HttpOnly: true,
        Secure:   false,
        Path:     "/",
        SameSite: http.SameSiteLaxMode,
    })
    http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *AuthHandler) GetUserCreditHandler(w http.ResponseWriter, r *http.Request) {
    userID, ok := r.Context().Value("userID").(uint)
    if !ok {
        http.Error(w, "Authentication error: User ID not found in context", http.StatusUnauthorized)
        return
    }

    // This will need to be fixed based on your BalanceService methods
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

// Helper functions remain unchanged
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
