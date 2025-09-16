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

	"github.com/iyunix/go-internist/internal/domain"
	"github.com/iyunix/go-internist/internal/services/admin_services"
)

type AdminHandler struct {
	adminService *admin_services.AdminService
}

func NewAdminHandler(adminService *admin_services.AdminService) *AdminHandler {
	return &AdminHandler{
		adminService: adminService,
	}
}

// GetAllUsersHandler handles the API request to fetch all users with pagination and search.
func (h *AdminHandler) GetAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	// CHANGE: Parse page, limit, and search query parameters from the request.
	query := r.URL.Query()
	
	page, err := strconv.Atoi(query.Get("page"))
	if err != nil || page < 1 {
		page = 1 // Default to page 1
	}

	limit, err := strconv.Atoi(query.Get("limit"))
	if err != nil || limit < 1 {
		limit = 10 // Default to 10 results per page
	}

	search := query.Get("search")

	// This assumes your AdminService's GetAllUsers is updated to accept these parameters.
	users, total, err := h.adminService.GetAllUsers(r.Context(), page, limit, search)
	if err != nil {
		log.Printf("[AdminHandler] Error getting all users: %v", err)
		http.Error(w, "Failed to retrieve users", http.StatusInternalServerError)
		return
	}

	// CHANGE: Return a structured response with pagination data.
	response := map[string]interface{}{
		"users": users,
		"total": total,
		"page":  page,
		"limit": limit,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}


type userActionRequest struct {
	UserID uint `json:"userID"`
}

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

type changePlanRequest struct {
	UserID  uint                  `json:"userID"`
	NewPlan domain.SubscriptionPlan `json:"newPlan"`
}

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

// CHANGE: Updated the JSON tags to match what the frontend will send.
type topUpRequest struct {
	UserID uint `json:"userID"`
	Amount int  `json:"amount"`
}

func (h *AdminHandler) TopUpBalanceHandler(w http.ResponseWriter, r *http.Request) {
	var req topUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Amount <= 0 {
		http.Error(w, "Amount to add must be positive", http.StatusBadRequest)
		return
	}

	if err := h.adminService.TopUpBalance(r.Context(), req.UserID, req.Amount); err != nil {
		log.Printf("[AdminHandler] Error topping up balance for user %d: %v", req.UserID, err)
		http.Error(w, "Failed to top up balance", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Balance topped up successfully"})
}

func (h *AdminHandler) ExportUsersCSVHandler(w http.ResponseWriter, r *http.Request) {
	// This assumes GetAllUsers can be called without pagination/search params for a full export.
	users, _, err := h.adminService.GetAllUsers(r.Context(), 0, 0, "") // page=0, limit=0 means "all"
	if err != nil {
		log.Printf("[AdminHandler] Error exporting users: %v", err)
		http.Error(w, "Failed to export users", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("users_export_%s.csv", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	header := []string{"ID", "Username", "PhoneNumber", "Status", "IsAdmin", "CurrentBalance", "TotalBalance", "Plan"}
	if err := csvWriter.Write(header); err != nil {
		log.Printf("[AdminHandler] Error writing CSV header: %v", err)
		return
	}

	for _, user := range users {
		record := []string{
			strconv.FormatUint(uint64(user.ID), 10),
			user.Username,
			user.PhoneNumber,
			string(user.Status),
			strconv.FormatBool(user.IsAdmin),
			strconv.Itoa(user.CharacterBalance),
			strconv.Itoa(user.TotalCharacterBalance),
			string(user.SubscriptionPlan),
		}
		if err := csvWriter.Write(record); err != nil {
			log.Printf("[AdminHandler] Error writing CSV record for user %d: %v", user.ID, err)
			return
		}
	}
	log.Printf("[AdminHandler] Successfully exported %d users to CSV.", len(users))
}
