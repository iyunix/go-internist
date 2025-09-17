// File: internal/handlers/admin_handler.go
package handlers

import (
	"context"
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

// writeJSONError writes a structured JSON error response
func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// writeJSONSuccess writes a structured JSON success response
func writeJSONSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// GetAllUsersHandler handles the API request to fetch all users with pagination and search.
// 🚀 Route: GET /api/v1/admin/users?page=1&limit=10&search=john
func (h *AdminHandler) GetAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	page, err := strconv.Atoi(query.Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(query.Get("limit"))
	if err != nil || limit < 1 {
		limit = 10
	}

	search := query.Get("search")

	users, total, err := h.adminService.GetAllUsers(r.Context(), page, limit, search)
	if err != nil {
		log.Printf("[AdminHandler] Error getting all users: %v", err)
		writeJSONError(w, "Failed to retrieve users", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"users": users,
		"total": total,
		"page":  page,
		"limit": limit,
	}

	writeJSONSuccess(w, response)
}

// ————————————————————————————————————————————————————————————————————————————————

type userActionRequest struct {
	UserID uint `json:"userID"`
}

// RenewSubscriptionHandler renews a user’s subscription.
// 🚀 Route: POST /api/v1/admin/users/renew
func (h *AdminHandler) RenewSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	var req userActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.adminService.RenewSubscription(r.Context(), req.UserID); err != nil {
		log.Printf("[AdminHandler] Error renewing subscription for user %d: %v", req.UserID, err)
		writeJSONError(w, "Failed to renew subscription", http.StatusInternalServerError)
		return
	}

	writeJSONSuccess(w, map[string]string{"message": "Subscription renewed successfully"})
}

// ————————————————————————————————————————————————————————————————————————————————

type changePlanRequest struct {
	UserID  uint                  `json:"userID"`
	NewPlan domain.SubscriptionPlan `json:"newPlan"`
}

// ChangePlanHandler changes a user’s subscription plan.
// 🚀 Route: POST /api/v1/admin/users/change-plan
func (h *AdminHandler) ChangePlanHandler(w http.ResponseWriter, r *http.Request) {
	var req changePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.adminService.ChangeUserPlan(r.Context(), req.UserID, req.NewPlan); err != nil {
		log.Printf("[AdminHandler] Error changing plan for user %d: %v", req.UserID, err)
		writeJSONError(w, "Failed to change plan", http.StatusInternalServerError)
		return
	}

	writeJSONSuccess(w, map[string]string{"message": "User plan updated successfully"})
}

// ————————————————————————————————————————————————————————————————————————————————

type topUpRequest struct {
	UserID uint `json:"userID"`
	Amount int  `json:"amount"`
}

// TopUpBalanceHandler adds balance to a user’s account.
// 🚀 Route: POST /api/v1/admin/users/top-up
func (h *AdminHandler) TopUpBalanceHandler(w http.ResponseWriter, r *http.Request) {
	var req topUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Amount <= 0 {
		writeJSONError(w, "Amount to add must be positive", http.StatusBadRequest)
		return
	}

	if err := h.adminService.TopUpBalance(r.Context(), req.UserID, req.Amount); err != nil {
		log.Printf("[AdminHandler] Error topping up balance for user %d: %v", req.UserID, err)
		writeJSONError(w, "Failed to top up balance", http.StatusInternalServerError)
		return
	}

	writeJSONSuccess(w, map[string]string{"message": "Balance topped up successfully"})
}

// ————————————————————————————————————————————————————————————————————————————————

// ExportUsersCSVHandler exports all users as CSV (with optional pagination for large datasets).
// 🚀 Route: GET /api/v1/admin/users/export?chunk_size=5000&page=1
func (h *AdminHandler) ExportUsersCSVHandler(w http.ResponseWriter, r *http.Request) {
	// Optional: Add timeout for large exports
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	query := r.URL.Query()

	// Optional: Support chunked export for huge datasets
	page, _ := strconv.Atoi(query.Get("page"))
	limit, _ := strconv.Atoi(query.Get("chunk_size"))

	// If no pagination, export all (service should handle 0,0 as "all")
	users, _, err := h.adminService.GetAllUsers(ctx, page, limit, "")
	if err != nil {
		log.Printf("[AdminHandler] Error exporting users: %v", err)
		http.Error(w, "Failed to export users", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("users_export_%s.csv", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// ✅ FIELD WHITELISTING — Explicitly define columns to export
	// Safe against schema changes or accidental PII exposure
	header := []string{"ID", "Username", "PhoneNumber", "Status", "IsAdmin", "CurrentBalance", "TotalBalance", "Plan"}
	if err := csvWriter.Write(header); err != nil {
		log.Printf("[AdminHandler] Error writing CSV header: %v", err)
		return
	}

	for _, user := range users {
		record := []string{
			strconv.FormatUint(uint64(user.ID), 10), // adjust if ID is int64 → use FormatInt
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
		// Flush after each row for streaming safety — prevents memory buildup
		csvWriter.Flush()
	}

	log.Printf("[AdminHandler] Successfully exported %d users to CSV.", len(users))
}