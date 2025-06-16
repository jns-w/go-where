package services

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go-server/models"
	"go-server/utils/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

type UserService struct {
	collection  *mongo.Collection
	redisClient *redis.Client
	jwtSecret   string
}

type NearbyUsers struct {
	Username string  `json:"username"`
	UserID   string  `json:"user_id"`            // Public ID of the user
	Distance float64 `json:"distance,omitempty"` // Optional, can be used to return distance from the queried point
	Lat      float64 `json:"lat,omitempty"`      // Optional, can be used to return user's last known latitude
	Lon      float64 `json:"lon,omitempty"`      // Optional, can be used to return user's last known longitude
}

func NewUserService(redisClient *redis.Client, jwtSecret string) *UserService {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Printf("MongoDB connection failed, user persistence disabled: %v", err)
	}
	collection := client.Database("poi_db").Collection("users")

	// Ensure unique index on username and email
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}, {Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err = collection.Indexes().CreateOne(context.Background(), indexModel)
	if err != nil {
		log.Printf("Failed to create unique index on users: %v", err)
	}

	return &UserService{
		collection:  collection,
		redisClient: redisClient,
		jwtSecret:   jwtSecret,
	}
}

// GetUser retrieves a user from Redis or MongoDB
func (s *UserService) GetUser(ctx context.Context, userID string) (models.User, error) {
	var user models.User

	// Check Redis first
	userJSON, err := s.redisClient.Get(ctx, "user:"+userID).Result()
	if err == nil {
		if err := json.Unmarshal([]byte(userJSON), &user); err != nil {
			log.Printf("Failed to unmarshal user %s: %v", userID, err)
		} else {
			return user, nil
		}
	}

	err = s.collection.FindOne(ctx, bson.M{"public_id": bson.M{"$eq": userID}}).Decode(&user)
	if err != nil {
		return models.User{}, err
	}

	// Cache in Redis
	userJSONBytes, err := json.Marshal(user)
	if err != nil {
		return user, err
	}
	s.redisClient.Set(ctx, "user:"+userID, userJSONBytes, 24*time.Hour)

	return user, nil
}

// UpdateUser updates user information in MongoDB and Redis
func (s *UserService) UserLocationPing(ctx context.Context, lat, lon float64) error {
	// Get the userID from the context
	userID, ok := ctx.Value("userID").(string)
	if !ok || userID == "" {
		return errors.ErrUnauthorized
	}

	// Validate coordinates
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return fmt.Errorf("invalid coordinates: lat=%f, lon=%f", lat, lon)
	}
	// Validate user existence
	_, err := s.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %v", err)
	}

	// Log the location update
	log.Printf("Updating location for user %s: lat=%f, lon=%f", userID, lat, lon)

	// Update MongoDB
	// userObjID, err := primitive.ObjectIDFromHex(userID)
	// if err != nil {
	// 	return fmt.Errorf("invalid userID: %v", err)
	// }
	update := bson.M{
		"$set": bson.M{
			"lastLocation": bson.M{
				"type":        "Point",
				"coordinates": []float64{lon, lat},
			},
		},
	}
	_, err = s.collection.UpdateOne(ctx, bson.M{"public_id": userID}, update)
	if err != nil {
		log.Printf("Failed to update MongoDB user location: %v", err)
		return err
	}

	// Update Redis with TTL (e.g., 5 minutes)
	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	user.LastLocation = models.GeoPoint{Type: "Point", Coordinates: []float64{lon, lat}}
	userJSON, err := json.Marshal(user)
	if err != nil {
		return err
	}
	ttl := 5 * time.Minute
	err = s.redisClient.Set(ctx, "user:"+user.PublicID, userJSON, ttl).Err()
	if err != nil {
		log.Printf("Failed to update Redis user location: %v", err)
		return err
	}

	// Store in Redis geospatial index
	err = s.redisClient.GeoAdd(ctx, "users:geo", &redis.GeoLocation{
		Name:      user.PublicID,
		Longitude: lon,
		Latitude:  lat,
	}).Err()
	if err != nil {
		log.Printf("Failed to update Redis geospatial index: %v", err)
		return err
	}
	// Set TTL on geospatial entry
	s.redisClient.Expire(ctx, "users:geo", ttl)

	log.Printf("Updated location for user %s: lat=%f, lon=%f", user.PublicID, lat, lon)
	return nil
}

// GetNearbyUsers retrieves users within a specified radius from a given location
func (s *UserService) GetNearbyUsers(ctx context.Context, lat, lon float64, radius float64) ([]NearbyUsers, error) {
	// Get the userID from the context
	userID, ok := ctx.Value("userID").(string)
	if !ok || userID == "" {
		return nil, errors.ErrUnauthorized
	}
	// Validate userID
	if userID == "" {
		return nil, errors.ErrInvalidInput
	}

	// Validate coordinates
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return nil, errors.ErrInvalidInput
	}

	// Get nearby users from Redis geospatial index
	geoResults, err := s.redisClient.GeoRadius(ctx, "users:geo", lon, lat, &redis.GeoRadiusQuery{
		Radius:    radius,
		Unit:      "km",
		WithCoord: true,
		WithDist:  true,
	}).Result()
	if err != nil {
		log.Printf("Failed to get nearby users from Redis: %v", err)
		return nil, err
	}

	var users []NearbyUsers
	for _, geoResult := range geoResults {
		if geoResult.Name == userID {
			// Skip the user themselves
			continue
		}
		publicID := geoResult.Name
		userData, err := s.GetUser(ctx, publicID)
		user := NearbyUsers{
			Username: userData.Username,
			UserID:   userData.PublicID,
			Lat:      geoResult.Latitude,
			Lon:      geoResult.Longitude,
			Distance: geoResult.Dist,
		}
		if err != nil {
			log.Printf("Failed to get user %s: %v", publicID, err)
			continue
		}
		users = append(users, user)
	}

	return users, nil
}

func (s *UserService) GetNearbyFriends(ctx context.Context, lat, lon float64, radius float64) ([]NearbyUsers, error) {
	// Get the userID from the context
	userID, ok := ctx.Value("userID").(string)
	if !ok || userID == "" {
		return nil, errors.ErrUnauthorized
	}
	// Validate coordinates
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return nil, errors.ErrInvalidInput
	}

	// Get the user from the database, populate friends
	var user models.User
	err := s.collection.FindOne(ctx, bson.M{"public_id": userID}).Decode(&user)

	if err != nil {
		return nil, fmt.Errorf("user not found: %v", err)
	}

	// Get nearby users from Redis geospatial index
	geoResults, err := s.redisClient.GeoRadius(ctx, "users:geo", lon, lat, &redis.GeoRadiusQuery{
		Radius:    radius,
		Unit:      "km",
		WithCoord: true,
		WithDist:  true,
	}).Result()
	if err != nil {
		log.Printf("Failed to get nearby users from Redis: %v", err)
		return nil, fmt.Errorf("failed to get nearby users: %v", err)
	}

	var nearbyFriends []NearbyUsers
	for _, geoResult := range geoResults {
		if geoResult.Name == userID {
			// Skip the user themselves
			continue
		}
		// Check if the user is a friend
		for _, friendID := range user.Friends {
			if geoResult.Name == friendID {
				// Get user data
				friendData, err := s.GetUser(ctx, geoResult.Name)
				if err != nil {
					log.Printf("Failed to get user %s: %v", geoResult.Name, err)
					continue
				}
				nearbyFriend := NearbyUsers{
					Username: friendData.Username,
					UserID:   friendData.PublicID,
					Lat:      geoResult.Latitude,
					Lon:      geoResult.Longitude,
					Distance: geoResult.Dist,
				}
				nearbyFriends = append(nearbyFriends, nearbyFriend)
			}
		}
	}
	return nearbyFriends, nil
}

func (s *UserService) SendFriendRequest(ctx context.Context, recipientID string) error {
	// Get the userID from the context
	userID, ok := ctx.Value("userID").(string)
	if !ok || userID == "" {
		return errors.ErrUnauthorized
	}

	if userID == recipientID {
		return errors.ErrInvalidInput
	}

	// Verify users exist
	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("Sender not found")
	}
	recipient, err := s.GetUser(ctx, recipientID)
	if err != nil {
		return fmt.Errorf("Recipient not found")
	}

	for _, friendID := range recipient.Friends {
		if friendID == user.ID {
			return fmt.Errorf("Already friends")
		}
	}
	for _, pendingID := range recipient.PendingFriendRequests {
		if pendingID == user.ID {
			return fmt.Errorf("Already pending friend request")
		}
	}

	userObjID, err := primitive.ObjectIDFromHex(user.ID)
	if err != nil {
		return fmt.Errorf("Invalid user ID: %v", err)
	}

	recipientObjID, err := primitive.ObjectIDFromHex(recipient.ID)
	if err != nil {
		return fmt.Errorf("Invalid recipient ID: %v", err)
	}

	update := bson.M{
		"$addToSet": bson.M{
			"pending_friend_requests": userID, // Add sender's ID to recipient's pending requests
		},
	}
	_, err = s.collection.UpdateOne(ctx, bson.M{"_id": recipientObjID}, update)
	if err != nil {
		log.Printf("Failed to send friend request: %v", err)
		return fmt.Errorf("Failed to send friend request")
	}

	// Update sender's pending requests
	updateSender := bson.M{
		"$addToSet": bson.M{
			"pending_friend_requests_sent": recipientID,
		},
	}
	_, err = s.collection.UpdateOne(ctx, bson.M{"_id": userObjID}, updateSender)
	if err != nil {
		log.Printf("Failed to update sender's pending requests: %v", err)
		return fmt.Errorf("Failed to update sender's pending requests")
	}

	log.Printf("Friend request sent from %s to %s", user.Username, recipient.Username)
	return nil
}

func (s *UserService) AcceptFriendRequest(ctx context.Context, senderID string) error {
	// Get the userID from the context
	userID, ok := ctx.Value("userID").(string)
	if !ok || userID == "" {
		return errors.ErrUnauthorized
	}
	if userID == senderID {
		return fmt.Errorf("Cannot accept friend request from self")
	}

	// Verify users exist
	sender, err := s.GetUser(ctx, senderID)
	if err != nil {
		return fmt.Errorf("Sender not found")
	}
	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("User not found")
	}

	userObjID, err := primitive.ObjectIDFromHex(user.ID)
	if err != nil {
		return fmt.Errorf("Invalid user ID: %v", err)
	}
	senderObjID, err := primitive.ObjectIDFromHex(sender.ID)
	if err != nil {
		return fmt.Errorf("Invalid sender ID: %v", err)
	}

	log.Printf("Checking if there is a pending friend request from %s to %s", sender.Username, user.Username)
	// Check if there is a pending friend
	exists, err := s.collection.FindOne(ctx, bson.M{
		"_id":                     userObjID,
		"pending_friend_requests": bson.M{"$in": []string{senderID}},
	}).DecodeBytes()
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("No pending friend request from %s to %s", sender.Username, user.Username)
		}
		log.Printf("Failed to check pending friend request: %v", err)
		return fmt.Errorf("Failed to check pending friend request")
	}
	if exists == nil {
		return fmt.Errorf("No pending friend request from %s to %s", sender.Username, user.Username)
	}

	// Add to friends list
	updateUser := bson.M{
		"$addToSet": bson.M{
			"friends": senderID, // Add sender's ID to user's friends
		},
		"$pull": bson.M{
			"pending_friend_requests": senderID, // Remove from user's pending requests
		},
	}

	_, err = s.collection.UpdateOne(ctx, bson.M{"_id": userObjID}, updateUser)

	if err != nil {
		log.Printf("Failed to accept friend request: %v", err)
		return fmt.Errorf("Failed to accept friend request")
	}
	// Update sender's friends list
	updateSender := bson.M{
		"$addToSet": bson.M{
			"friends": userID,
		},
		"$pull": bson.M{
			"pending_friend_requests_sent": userID, // Remove from sender's pending requests
		},
	}
	_, err = s.collection.UpdateOne(ctx, bson.M{"_id": senderObjID}, updateSender)
	if err != nil {
		log.Printf("Failed to update sender's friends list: %v", err)
		return fmt.Errorf("Failed to update sender's friends list")
	}
	log.Printf("Friend request accepted from %s to %s", sender.Username, user.Username)
	return nil
}
