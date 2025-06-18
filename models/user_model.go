package models

type User struct {
	ID                        string   `json:"id,omitempty" bson:"_id,omitempty"`
	PublicID                  string   `json:"public_id" bson:"public_id"`
	Username                  string   `json:"username" bson:"username"`
	Email                     string   `json:"email,omitempty" bson:"email,omitempty"`
	PasswordHash              string   `json:"password_hash,omitempty" bson:"password_hash,omitempty"`
	FavoritePOIs              []string `json:"favorite_pois,omitempty" bson:"favorite_pois"`
	LastLocation              GeoPoint `json:"last_location,omitempty" bson:"last_location,omitempty"`
	Friends                   []string `json:"friends,omitempty" bson:"friends,omitempty"`
	PendingFriendRequests     []string `json:"pending_friend_requests,omitempty" bson:"pending_friend_requests,omitempty"`
	PendingFriendRequestsSent []string `json:"pending_friend_requests_sent,omitempty" bson:"pending_friend_requests_sent,omitempty"`
}
