package main

import (
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"go-server/handlers"
	"go-server/middleware"
	"go-server/services"
	"log"
	"net/http"
	"os"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	// Initialize services and handlers
	geoService := services.NewGeoService()
	poiHandler := handlers.NewPOIHandler(geoService)

	// Initialize the user handler with the user service and JWT secret
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}

	// Redis
	userService := services.NewUserService(geoService.RedisClient, jwtSecret)
	userHandler := handlers.NewUserHandler(userService, jwtSecret)

	authHandler := handlers.NewAuthHandler(userService, jwtSecret)

	r := mux.NewRouter()

	// CORS middleware
	allowedOrigins := []string{"http://localhost:3000", "http://localhost:5173"}
	r.Use(middleware.CORSMiddleware(allowedOrigins))

	// Routes

	// Auth routes
	authRouter := r.PathPrefix("/auth").Subrouter()
	authRouter.HandleFunc("/register", authHandler.RegisterUser).Methods("POST", "OPTIONS")
	authRouter.HandleFunc("/login", authHandler.LoginUser).Methods("POST", "OPTIONS")

	// User routes
	userRouter := r.PathPrefix("/user").Subrouter()
	userRouter.Use(middleware.JWTMiddleware(jwtSecret)) // Apply JWT middleware to user routes
	userRouter.HandleFunc("/ping", userHandler.PingLocation).Methods("POST", "OPTIONS")
	userRouter.HandleFunc("/nearby", userHandler.GetNearbyUsers).Methods("GET", "OPTIONS")
	userRouter.HandleFunc("/nearby-friends", userHandler.GetNearbyFriends).Methods("GET", "OPTIONS")
	userRouter.HandleFunc("/send-friend-request", userHandler.SendFriendRequest).Methods("POST", "OPTIONS")
	userRouter.HandleFunc("/accept-friend-request", userHandler.AcceptFriendRequest).Methods("POST", "OPTIONS")

	// POI routes
	r.HandleFunc("/pois", poiHandler.GetNearbyPOIs).Methods("GET", "OPTIONS")

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
