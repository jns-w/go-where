# Go Where

> #### Stack:
> - Golang
> - Mux
> - Redis
> - MongoDB


Go Where is a geolocation service that allows users to find nearby places of interest based on their current location; Sprinkle in user locations around the proximity, we can even provide social features. The service is built using Golang, Redis, and MongoDB, providing a fast and efficient way to retrieve location data.

## Core Features

- **Point of Interest (POI) Discovery**: Users can find nearby places of interest based on their current location.
- **User Authentication**: Users can sign up and log in to save their favorite places.
- **Real-time Location Updates**: Users can see their current location on a map and receive real-time updates.

## Architecture

When choosing technologies for real-time applications like Go Where, it is crucial to consider performant languages and efficient data storage solutions. Golang is an excellent choice for its concurrency model, which allows handling multiple requests simultaneously without blocking; Coupled with its compiled nature, it also has fast execution speeds. Redis is used for fast data retrieval and caching, ensuring low latency in serving user requests. When we need persisted data storage, MongoDB is chosen for its flexible schema and geospatial capabilities, allowing efficient querying of locations.

### Redis for Caching and Real-time Data

Redis is an ideal choice for powering the app's real-time features. It provides a fast in-memory data store that can handle high throughput and low latency, making it perfect for caching user locations and POI data. Additionally, Redis supports geospatial indexing, which allows for efficient querying of locations based on proximity.

```go
// Store user in Redis geospatial index
err = s.redisClient.GeoAdd(ctx, "users:geo", &redis.GeoLocation{
    Name:      userID,
    Longitude: lon,
    Latitude:  lat,
}).Err()
// Fetch nearby users
nearbyUsers, err := s.redisClient.GeoRadius(ctx, "users:geo", lon, lat, &redis.GeoRadiusQuery{
    Radius:  query.radius,
    Unit:    "m",
    WithDist: true,
    Sort:    "ASC",
}).Result()
```

### MongoDB for Persisted Data

Unlike user locations, POIs are not frequently updated, so a database with persisted storage like MongoDB is used for storing POI data. MongoDB's geospatial indexing capabilities is ideal for querying locations based on proximity, if we set it up correctly with a 2dsphere index. And we can do so as a backup source of data down the line. However for the current implementation, we will use the MongoDB collection to seed the Redis cache with POI data for maximum performance.

```go
// Seeding POIs from MongoDB to Redis
func (s *GeoService) seedPOIsToRedis() {
	ctx := context.Background()
	// Clear existing POI data in Redis
	err := s.RedisClient.FlushDB(ctx).Err()
	if err != nil {
		log.Printf("Failed to flush Redis DB: %v", err)
		return
	}
	log.Println("Seeding POIs into Redis...")
	// Take data from mongo and seed into Redis

	cursor, err := s.collection.Find(context.Background(), bson.M{})
	if err != nil {
		log.Printf("Failed to load POIs from MongoDB: %v", err)
		return
	}
	defer cursor.Close(context.Background())
	var pois []models.POI
	if err := cursor.All(context.Background(), &pois); err != nil {
		log.Printf("Failed to decode POIs from MongoDB: %v", err)
		return
	}
	// Iterate through each POI and store in Redis
	for _, poi := range pois {
		// Store POI data in Redis hash
		poiJSON, err := json.Marshal(poi)
		if err != nil {
			log.Printf("Failed to marshal POI %s: %v", poi.Name, err)
			continue
		}
		err = s.RedisClient.HSet(ctx, poi.ID, "data", poiJSON).Err()
		if err != nil {
			log.Printf("Failed to set POI %s in Redis: %v", poi.Name, err)
			continue
		}
		// Add to Redis Geo set
		err = s.RedisClient.GeoAdd(ctx, "pois:geo", &redis.GeoLocation{
			Name:      poi.ID,
			Longitude: poi.Location.Coordinates[0],
			Latitude:  poi.Location.Coordinates[1],
		}).Err()
		if err != nil {
			log.Printf("Failed to add POI %s to Redis Geo set: %v", poi.Name, err)
			continue
		}
	}
	log.Printf("Seeded %d POIs into Redis", len(pois))
}

```
### Authentication and User Management

User privacy is a top priority when working on a project with location data. Go Where implements secure user authentication using JWT tokens, ensuring that user data is protected. The service allows users to sign up, log in, and manage their favorite places securely.

## Conclusion

This architecture allows us plenty of flexibility and scalability. And we can easily build features on top of this foundation, such as activity recommendations, social features, and more. Anticipation of such advanced features is also why we chose Go for its performance capabilities, as it can handle high loads and concurrent requests efficiently.
