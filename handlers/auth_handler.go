package handlers

import (
	"context"
	"encoding/json"
	"go-server/middleware"
	"go-server/services"
	"go-server/utils/errors"
	"net/http"
)

type AuthHandler struct {
	userService *services.UserService
	jwtSecret   string
}

func NewAuthHandler(userService *services.UserService, jwtSecret string) *AuthHandler {
	return &AuthHandler{userService: userService, jwtSecret: jwtSecret}
}

func (h *AuthHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		middleware.WriteError(w, errors.ErrInvalidInput)
		return
	}

	userID, err := h.userService.Register(context.Background(), input.Username, input.Email, input.Password)
	if err != nil {
		middleware.WriteError(w, errors.Wrap(err, "REGISTRATION_ERROR", "Failed to register user", http.StatusInternalServerError))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"userID": userID})
}

func (h *AuthHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		middleware.WriteError(w, errors.ErrInvalidInput)
		return
	}
	token, err := h.userService.Login(context.Background(), input.Username, input.Password)
	if err != nil {
		middleware.WriteError(w, errors.Wrap(err, "LOGIN_ERROR", "Failed to login user", http.StatusUnauthorized))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}
