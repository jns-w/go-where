package models

type POI struct {
	ID          string   `json:"id" bson:"_id,omitempty"`
	Name        string   `json:"name" bson:"name"`
	Description string   `json:"description" bson:"description"`
	Type        string   `json:"type" bson:"type"`
	Location    GeoPoint `json:"location" bson:"location"`
	Tags        []string `json:"tags" bson:"tags"`
	Address     string   `json:"address" bson:"address"`
}

type GeoPoint struct {
	Type        string    `json:"type" bson:"type"`
	Coordinates []float64 `json:"coordinates" bson:"coordinates"`
}
