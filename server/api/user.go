package api

import (
	"encoding/json"
	"net/http"

	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/di"
	"gorm.io/gorm"
)

// handleLogin authenticates a user and returns a JWT token
func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	ctx := r.Context()
	db := di.MustInvoke[*gorm.DB](ctx)
	var user models.User
	if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		JSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "Invalid username or password"})
		return
	}

	if user.Password != req.Password {
		JSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "Invalid username or password"})
		return
	}

	token, err := h.GenerateToken(r.Context(), user.Username)
	if err != nil {
		JSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to generate token"})
		return
	}

	JSON(w, http.StatusOK, models.LoginResponse{Token: token})
}
