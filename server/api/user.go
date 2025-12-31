package api

import (
	"encoding/json"
	"net/http"

	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/auth"
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

	// Decrypt password from transmission
	password, err := auth.DecryptTransmission(req.Password)
	if err != nil {
		// Fallback to plain text if decryption fails (e.g. old client or already plain)
		password = req.Password
	}

	// Try to decrypt stored password
	passwordSecret := auth.ResolvePasswordSecret(ctx)
	decryptedPassword, err := auth.Decrypt(user.Password, passwordSecret)
	if err != nil {
		// Fallback to plain text for existing users
		decryptedPassword = user.Password
	}

	if decryptedPassword != password {
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

// handleGetMe returns the current user's profile
func (h *Handler) handleGetMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	username := di.MustInvoke[models.Username](ctx)
	db := di.MustInvoke[*gorm.DB](ctx)

	var user models.User
	if err := db.Where("username = ?", string(username)).First(&user).Error; err != nil {
		JSON(w, http.StatusNotFound, models.ErrorResponse{Error: "User not found"})
		return
	}

	JSON(w, http.StatusOK, user)
}

type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

// handleChangePassword updates the user's password
func (h *Handler) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	ctx := r.Context()
	username := di.MustInvoke[models.Username](ctx)
	db := di.MustInvoke[*gorm.DB](ctx)

	var user models.User
	if err := db.Where("username = ?", string(username)).First(&user).Error; err != nil {
		JSON(w, http.StatusNotFound, models.ErrorResponse{Error: "User not found"})
		return
	}

	// Decrypt passwords from transmission
	oldPassword, err := auth.DecryptTransmission(req.OldPassword)
	if err != nil {
		oldPassword = req.OldPassword
	}
	newPassword, err := auth.DecryptTransmission(req.NewPassword)
	if err != nil {
		newPassword = req.NewPassword
	}

	// Try to decrypt stored password
	passwordSecret := auth.ResolvePasswordSecret(ctx)
	decryptedPassword, err := auth.Decrypt(user.Password, passwordSecret)
	if err != nil {
		// Fallback to plain text for existing users
		decryptedPassword = user.Password
	}

	if decryptedPassword != oldPassword {
		JSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "Invalid old password"})
		return
	}

	encryptedPassword, err := auth.Encrypt(newPassword, passwordSecret)
	if err != nil {
		JSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to encrypt password"})
		return
	}

	if err := db.Model(&user).Update("password", encryptedPassword).Error; err != nil {
		JSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to update password"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
