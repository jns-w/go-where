package models

type User struct {
	ID                        string   `json:"id" bson:"_id,omitempty"`
	PublicID                  string   `json:"public_id" bson:"public_id"`
	Username                  string   `json:"username" bson:"username"`
	Email                     string   `json:"email" bson:"email"`
	PasswordHash              string   `json:"password_hash" bson:"password_hash"`
	FavoritePOIs              []string `json:"favorite_pois" bson:"favorite_pois"`
	LastLocation              GeoPoint `json:"last_location" bson:"last_location"`
	Friends                   []string `json:"friends" bson:"friends"`
	PendingFriendRequests     []string `json:"pending_friend_requests" bson:"pending_friend_requests"`
	PendingFriendRequestsSent []string `json:"pending_friend_requests_sent" bson:"pending_friend_requests_sent"`
}
