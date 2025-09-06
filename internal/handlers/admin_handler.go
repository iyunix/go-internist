// File: internal/handlers/admin_handler.go
package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/iyunix/go-internist/internal/domain" // <-- ADDED THIS IMPORT
	"github.com/iyunix/go-internist/internal/services/admin_services"
)

// AdminHandler holds the dependencies for admin-related HTTP handlers.
type AdminHandler struct {
	adminService *admin_services.AdminService
}

// NewAdminHandler creates a new instance of AdminHandler.
func NewAdminHandler(adminService *admin_services.AdminService) *AdminHandler {
	return &AdminHandler{
		adminService: adminService,
	}
}

// GetAllUsersHandler handles the API request to fetch all users.
func (h *AdminHandler) GetAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := h.adminService.GetAllUsers(r.Context())
	if err != nil {
		log.Printf("[AdminHandler] Error getting all users: %v", err)
		http.Error(w, "Failed to retrieve users", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}


// --- THE OLD AddCreditsHandler IS GONE. IT IS REPLACED BY THE THREE HANDLERS BELOW. ---


// A struct for requests that only need a UserID.
type userActionRequest struct {
	UserID uint `json:"userID"`
}

// RenewSubscriptionHandler handles the API request to reset a user's balance to their plan's limit.
func (h *AdminHandler) RenewSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	var req userActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.adminService.RenewSubscription(r.Context(), req.UserID); err != nil {
		log.Printf("[AdminHandler] Error renewing subscription for user %d: %v", req.UserID, err)
		http.Error(w, "Failed to renew subscription", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Subscription renewed successfully"})
}

// A struct for requests to change a user's plan.
type changePlanRequest struct {
	UserID  uint                     `json:"userID"`
	NewPlan domain.SubscriptionPlan `json:"newPlan"`
}

// ChangePlanHandler handles the API request to change a user's subscription plan.
func (h *AdminHandler) ChangePlanHandler(w http.ResponseWriter, r *http.Request) {
	var req changePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.adminService.ChangeUserPlan(r.Context(), req.UserID, req.NewPlan); err != nil {
		log.Printf("[AdminHandler] Error changing plan for user %d: %v", req.UserID, err)
		http.Error(w, "Failed to change plan", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "User plan updated successfully"})
}

// A struct for requests to top up a user's balance.
type topUpRequest struct {
	UserID      uint `json:"userID"`
	AmountToAdd int  `json:"amountToAdd"`
}

// TopUpBalanceHandler handles the API request to add bonus credits to a user.
func (h *AdminHandler) TopUpBalanceHandler(w http.ResponseWriter, r *http.Request) {
	var req topUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.AmountToAdd <= 0 {
		http.Error(w, "Amount to add must be positive", http.StatusBadRequest)
		return
	}

	if err := h.adminService.TopUpBalance(r.Context(), req.UserID, req.AmountToAdd); err != nil {
		log.Printf("[AdminHandler] Error topping up balance for user %d: %v", req.UserID, err)
		http.Error(w, "Failed to top up balance", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Balance topped up successfully"})
}


// ExportUsersCSVHandler generates and serves a CSV file of all users.
func (h *AdminHandler) ExportUsersCSVHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Fetch all users from the service.
	users, err := h.adminService.GetAllUsers(r.Context())
	if err != nil {
		log.Printf("[AdminHandler] Error exporting users: %v", err)
		http.Error(w, "Failed to export users", http.StatusInternalServerError)
		return
	}

	// 2. Set headers to tell the browser to download the file.
	filename := fmt.Sprintf("users_export_%s.csv", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")

	// 3. Create a CSV writer that writes directly to the HTTP response.
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// 4. Write the header row.
	header := []string{"ID", "Username", "PhoneNumber", "CurrentBalance", "TotalBalance", "IsAdmin"}
	if err := csvWriter.Write(header); err != nil {
		log.Printf("[AdminHandler] Error writing CSV header: %v", err)
		return
	}

	// 5. Loop through users and write each one as a row in the CSV.
	for _, user := range users {
		record := []string{
			strconv.FormatUint(uint64(user.ID), 10),
			user.Username,
			user.PhoneNumber,
			strconv.Itoa(user.CharacterBalance),
			strconv.Itoa(user.TotalCharacterBalance),
			strconv.FormatBool(user.IsAdmin),
		}
		if err := csvWriter.Write(record); err != nil {
			log.Printf("[AdminHandler] Error writing CSV record for user %d: %v", user.ID, err)
			// Stop if we can no longer write to the connection.
			return
		}
	}
	log.Printf("[AdminHandler] Successfully exported %d users to CSV.", len(users))
}