package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ericp/chronos-bot-reminder/internal/database/models"
	"github.com/ericp/chronos-bot-reminder/internal/database/repositories"
	"github.com/ericp/chronos-bot-reminder/internal/services"
	"github.com/google/uuid"
)

// PlannerHandler handles "Day Planner" HTTP requests. This is a web/mobile
// only feature — there is no Discord bot surface for it.
type PlannerHandler struct {
	itemRepo repositories.PlannerItemRepository
	dfmRepo  repositories.DFMItemRepository
}

// NewPlannerHandler creates a new planner handler
func NewPlannerHandler(
	itemRepo repositories.PlannerItemRepository,
	dfmRepo repositories.DFMItemRepository,
) *PlannerHandler {
	return &PlannerHandler{
		itemRepo: itemRepo,
		dfmRepo:  dfmRepo,
	}
}

// GetItems returns all of the account's planner items
func (h *PlannerHandler) GetItems(w http.ResponseWriter, r *http.Request) {
	accountID := r.Context().Value(AccountIDKey).(uuid.UUID)

	items, err := h.itemRepo.GetByAccountID(accountID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to fetch planner items")
		return
	}
	if items == nil {
		items = []models.PlannerItem{}
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

// AddItem creates a new planner item, optionally linked to an existing DFM item
func (h *PlannerHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	accountID := r.Context().Value(AccountIDKey).(uuid.UUID)

	var req struct {
		Content   string  `json:"content"`
		Period    string  `json:"period"`
		DFMItemID *string `json:"dfm_item_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		WriteError(w, http.StatusBadRequest, "Content is required")
		return
	}

	period := models.PlannerPeriod(strings.ToLower(strings.TrimSpace(req.Period)))
	if !period.IsValid() {
		WriteError(w, http.StatusBadRequest, "Period must be 'morning' or 'afternoon'")
		return
	}

	itemCount, err := h.itemRepo.CountByAccountID(accountID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to check item count")
		return
	}
	if itemCount >= services.MaxPlannerItemsPerAccount {
		WriteError(w, http.StatusBadRequest, fmt.Sprintf("You have reached the maximum of %d planner items", services.MaxPlannerItemsPerAccount))
		return
	}

	item := &models.PlannerItem{
		AccountID: accountID,
		Content:   req.Content,
		Period:    period,
	}

	// If linking to a DFM item, verify it belongs to this account and adopt
	// its current checked state so the link starts in sync.
	if req.DFMItemID != nil && strings.TrimSpace(*req.DFMItemID) != "" {
		dfmItemID, err := uuid.Parse(*req.DFMItemID)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "Invalid dfm_item_id")
			return
		}
		dfmItem, err := h.dfmRepo.GetByID(dfmItemID)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "Failed to fetch DFM item")
			return
		}
		if dfmItem == nil {
			WriteError(w, http.StatusNotFound, "DFM item not found")
			return
		}
		item.DFMItemID = &dfmItemID
		item.Checked = dfmItem.Checked
	}

	if err := h.itemRepo.Create(item); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to create planner item")
		return
	}

	WriteJSON(w, http.StatusCreated, item)
}

// getOwnedItem fetches a planner item and verifies it belongs to the account
func (h *PlannerHandler) getOwnedItem(accountID uuid.UUID, itemIDStr string) (*models.PlannerItem, int, string) {
	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		return nil, http.StatusBadRequest, "Invalid item ID"
	}

	item, err := h.itemRepo.GetByID(itemID)
	if err != nil {
		return nil, http.StatusInternalServerError, "Failed to fetch item"
	}
	if item == nil || item.AccountID != accountID {
		return nil, http.StatusNotFound, "Item not found"
	}

	return item, 0, ""
}

// UpdateItem updates a planner item's content, checked state and/or period.
// Checking/unchecking a linked item also syncs the linked DFM item.
func (h *PlannerHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	accountID := r.Context().Value(AccountIDKey).(uuid.UUID)

	item, status, msg := h.getOwnedItem(accountID, r.PathValue("id"))
	if item == nil {
		WriteError(w, status, msg)
		return
	}

	var req struct {
		Content *string `json:"content"`
		Checked *bool   `json:"checked"`
		Period  *string `json:"period"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Content != nil {
		content := strings.TrimSpace(*req.Content)
		if content == "" {
			WriteError(w, http.StatusBadRequest, "Content cannot be empty")
			return
		}
		item.Content = content
	}
	if req.Period != nil {
		period := models.PlannerPeriod(strings.ToLower(strings.TrimSpace(*req.Period)))
		if !period.IsValid() {
			WriteError(w, http.StatusBadRequest, "Period must be 'morning' or 'afternoon'")
			return
		}
		item.Period = period
	}

	checkedChanged := req.Checked != nil && *req.Checked != item.Checked
	if req.Checked != nil {
		item.Checked = *req.Checked
	}

	if err := h.itemRepo.Update(item); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to update item")
		return
	}

	// Keep the linked DFM item in sync when the checked state changed.
	if checkedChanged && item.DFMItemID != nil {
		dfmItem, err := h.dfmRepo.GetByID(*item.DFMItemID)
		if err == nil && dfmItem != nil && dfmItem.Checked != item.Checked {
			dfmItem.Checked = item.Checked
			_ = h.dfmRepo.Update(dfmItem)
		}
	}

	WriteJSON(w, http.StatusOK, item)
}

// DeleteItem removes a planner item. The link to a DFM item (if any) is not
// affected — only the planner entry itself is deleted.
func (h *PlannerHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	accountID := r.Context().Value(AccountIDKey).(uuid.UUID)

	item, status, msg := h.getOwnedItem(accountID, r.PathValue("id"))
	if item == nil {
		WriteError(w, status, msg)
		return
	}

	if err := h.itemRepo.Delete(item.ID); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to delete item")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "Item deleted successfully"})
}

// ClearAll deletes every planner item for the account, so the user can wipe
// their plan clean and start fresh.
func (h *PlannerHandler) ClearAll(w http.ResponseWriter, r *http.Request) {
	accountID := r.Context().Value(AccountIDKey).(uuid.UUID)

	if err := h.itemRepo.DeleteByAccountID(accountID); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to clear planner items")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"message": "Planner cleared successfully"})
}

// Reorder updates the position/period of multiple items at once, e.g. after
// a drag-and-drop reorder or a move between Morning/Afternoon.
func (h *PlannerHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	accountID := r.Context().Value(AccountIDKey).(uuid.UUID)

	var req struct {
		Items []struct {
			ID       string `json:"id"`
			Position int    `json:"position"`
			Period   string `json:"period"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if len(req.Items) == 0 {
		WriteError(w, http.StatusBadRequest, "No items provided")
		return
	}

	updates := make([]repositories.ReorderInput, 0, len(req.Items))
	for _, i := range req.Items {
		id, err := uuid.Parse(i.ID)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "Invalid item ID: "+i.ID)
			return
		}
		period := models.PlannerPeriod(strings.ToLower(strings.TrimSpace(i.Period)))
		if !period.IsValid() {
			WriteError(w, http.StatusBadRequest, "Period must be 'morning' or 'afternoon'")
			return
		}
		updates = append(updates, repositories.ReorderInput{
			ID:       id,
			Position: i.Position,
			Period:   period,
		})
	}

	if err := h.itemRepo.Reorder(accountID, updates); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to reorder items")
		return
	}

	items, err := h.itemRepo.GetByAccountID(accountID)
	if err != nil {
		WriteJSON(w, http.StatusOK, map[string]string{"message": "Items reordered successfully"})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]interface{}{"items": items})
}
