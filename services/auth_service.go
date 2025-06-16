package services

import (
	"context"
	"encoding/json"
	"go-server/models"
	"go-server/utils/errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	userService *UserService
}

// Register creates a new user
func (s *UserService) Register(ctx context.Context, username, email, password string) (string, error) {
	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", errors.Wrap(err, "HASH_ERROR", "failed to hash password", http.StatusInternalServerError)
	}

	user := models.User{
		PublicID:     uuid.New().String(),
		Username:     username,
		Email:        email,
		PasswordHash: string(passwordHash),
		FavoritePOIs: []string{},
		LastLocation: models.GeoPoint{Type: "Point", Coordinates: []float64{0, 0}},
	}

	// Insert into MongoDB
	result, err := s.collection.InsertOne(ctx, user)
	if err != nil {
		return "", errors.Wrap(err, "DB_ERROR", "failed to create user in database", http.StatusInternalServerError)
	}

	// Get the user ID
	userID := result.InsertedID.(primitive.ObjectID).Hex()
	if userID == "" {
		return "", errors.Wrap(err, "DB_ERROR", "Failed to get user ID after insertion", http.StatusInternalServerError)
	}

	// Cache in Redis
	userJSON, err := json.Marshal(user)
	if err != nil {
		return "", errors.Wrap(err, "DB_ERROR", "Failed to marshal user", http.StatusInternalServerError)
	}
	s.redisClient.Set(ctx, "user:"+userID, userJSON, 24*time.Hour)

	return user.PublicID, nil
}

// Login authenticates a user and returns a JWT
func (s *UserService) Login(ctx context.Context, username, password string) (string, error) {
	var user models.User
	err := s.collection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		return "", errors.ErrNotFound
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", errors.NewAPIError("INVALID_CREDENTIALS", "Invalid username or password", http.StatusUnauthorized)
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userID":   user.PublicID,
		"username": user.Username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", errors.Wrap(err, "JWT_ERROR", "Failed to generate token", http.StatusInternalServerError)
	}

	// Cache user in Redis
	userJSON, err := json.Marshal(user)
	if err != nil {
		return tokenString, err
	}
	s.redisClient.Set(ctx, "user:"+user.ID, userJSON, 24*time.Hour)

	return tokenString, nil
}
