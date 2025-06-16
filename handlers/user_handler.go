package handlers

import (
	"encoding/json"
	"go-server/middleware"
	"go-server/services"
	"go-server/utils/errors"
	"net/http"
	"strconv"
)

type UserHandler struct {
	userService *services.UserService
	jwtSecret   string
}

type NearbyFriendsResponse struct {
	NearbyFriends []services.NearbyUsers `json:"nearby_friends"`
	Count         int                    `json:"count"`
	Lat           float64                `json:"lat"`
	Lon           float64                `json:"lon"`
	Radius        float64                `json:"radius"`
}

type NearbyUsersResponse struct {
	NearbyUsers []services.NearbyUsers `json:"nearby_users"`
	Count       int                    `json:"count"`
	Lat         float64                `json:"lat"`
	Lon         float64                `json:"lon"`
	Radius      float64                `json:"radius"`
}

func NewUserHandler(userService *services.UserService, jwtSecret string) *UserHandler {
	return &UserHandler{
		userService: userService,
		jwtSecret:   jwtSecret,
	}
}

func (h *UserHandler) PingLocation(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		middleware.WriteError(w, errors.ErrUnauthorized)
		return
	}

	// Parse GPS coordinates
	lat, err := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	if err != nil {
		middleware.WriteError(w, errors.ErrInvalidInput)
		return
	}
	lon, err := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
	if err != nil {
		middleware.WriteError(w, errors.ErrInvalidInput)
		return
	}

	err = h.userService.UserLocationPing(r.Context(), lat, lon)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Location updated", "user_id": userID})
}

func (h *UserHandler) GetNearbyUsers(w http.ResponseWriter, r *http.Request) {
	// Parse GPS coordinates
	lat, err := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	if err != nil {
		middleware.WriteError(w, errors.ErrInvalidInput)
		return
	}
	lon, err := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
	if err != nil {
		middleware.WriteError(w, errors.ErrInvalidInput)
		return
	}
	radius, err := strconv.ParseFloat(r.URL.Query().Get("radius"), 64)
	if err != nil {
		middleware.WriteError(w, errors.ErrInvalidInput)
		return
	}
	if radius <= 0 {
		radius = 3000 // Default radius in meters
	}

	users, err := h.userService.GetNearbyUsers(r.Context(), lat, lon, radius)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	response := NearbyUsersResponse{
		NearbyUsers: users,
		Count:       len(users),
		Lat:         lat,
		Lon:         lon,
		Radius:      radius,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *UserHandler) GetNearbyFriends(w http.ResponseWriter, r *http.Request) {
	// Parse GPS coordinates
	lat, err := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	if err != nil {
		middleware.WriteError(w, errors.ErrInvalidInput)
		return
	}
	lon, err := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
	if err != nil {
		middleware.WriteError(w, errors.ErrInvalidInput)
		return
	}
	radius, err := strconv.ParseFloat(r.URL.Query().Get("radius"), 64)
	if err != nil {
		middleware.WriteError(w, errors.ErrInvalidInput)
		return
	}
	if radius <= 0 {
		radius = 3000 // Default radius in meters
	}

	friends, err := h.userService.GetNearbyFriends(r.Context(), lat, lon, radius)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	response := NearbyFriendsResponse{
		NearbyFriends: friends,
		Count:         len(friends),
		Lat:           lat,
		Lon:           lon,
		Radius:        radius,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *UserHandler) SendFriendRequest(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ReceipientID string `json:"recipient_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		middleware.WriteError(w, errors.ErrInvalidInput)
		return
	}

	err := h.userService.SendFriendRequest(r.Context(), input.ReceipientID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Friend request sent"})
}

func (h *UserHandler) AcceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	var input struct {
		SenderID string `json:"sender_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		middleware.WriteError(w, errors.ErrInvalidInput)
		return
	}

	err := h.userService.AcceptFriendRequest(r.Context(), input.SenderID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Friend request accepted"})
}
