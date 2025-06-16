package services

import (
	"context"
	"encoding/json"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go-server/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"os"
	"strconv"
)

type GeoService struct {
	collection  *mongo.Collection
	pois        []models.POI  // In-memory cache of POIs
	RedisClient *redis.Client // Redis client for geo queries
}

func NewGeoService() *GeoService {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default configuration")
	}
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		log.Fatal("MONGODB_URI environment variable is not set")
	}
	// Connect to MongoDB
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("MongoDB connection failed: %v", err)
	}
	// Check MongoDB connection
	if err := client.Ping(context.Background(), nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}
	log.Println("Connected to MongoDB")
	collection := client.Database("poi_db").Collection("pois")

	// Instantiate GeoService with MongoDB collection
	service := &GeoService{collection: collection} // Initialize GeoService with collection

	// Initialize Redis client
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		log.Fatal("REDIS_ADDR environment variable is not set")
	}
	redisDBStr := os.Getenv("REDIS_DB")
	if redisDBStr == "" {
		log.Fatal("REDIS_DB environment variable is not set")
	}
	redisDB, err := strconv.Atoi(redisDBStr)
	if err != nil {
		log.Fatalf("Invalid REDIS_DB value: %v", err)
	}
	service.RedisClient = redis.NewClient(&redis.Options{
		Addr: redisAddr, // Redis server address
		DB:   redisDB,   // Use default DB
	})
	if err := service.RedisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Seed sample data if collection is empty
	count, err := collection.CountDocuments(context.Background(), bson.M{})
	if err != nil {
		log.Fatalf("Failed to count documents: %v", err)
	}

	if count <= 0 {
		log.Println("No POIs found in MongoDB, seeding sample data...")
		// Seed sample POIs into MongoDB
		service.seedPOIsToMongo(collection)
		// Load POIs into memory
		service.seedPOIsToRedis()
	} else {
		// If POIs exist, load them into redis
		// Seed Redis with POIs
		service.seedPOIsToRedis()
	}

	return service
}

// FindNearbyPOIs with Redis
func (s *GeoService) FindNearbyPOIs(ctx context.Context, lat, lon, radius float64, poiType string) ([]models.POI, error) {
	geoResults, err := s.RedisClient.GeoRadius(ctx, "pois:geo", lon, lat, &redis.GeoRadiusQuery{
		Radius:    radius,
		Unit:      "km",
		WithCoord: true,
		WithDist:  true,
		Sort:      "ASC",
		Count:     50,
	}).Result()
	if err != nil {
		log.Printf("Redis GeoRadius error: %v", err)
		return nil, err
	}

	var results []models.POI
	for _, geoResult := range geoResults {
		poiJSON, err := s.RedisClient.HGet(ctx, geoResult.Name, "data").Result()
		if err != nil {
			log.Printf("Redis Get error for POI %s: %v", geoResult.Name, err)
			continue
		}
		var poi models.POI
		if err := json.Unmarshal([]byte(poiJSON), &poi); err != nil {
			log.Printf("Failed to unmarshal POI %s: %v", geoResult.Name, err)
			continue
		}
		// Skip if type filter doesn't match
		if poiType != "" && poi.Type != poiType {
			continue
		}
		distance := geoResult.Dist * 1000 // Convert km to meters
		if distance <= radius {
			poiRes := models.POI{
				ID:          poi.ID,
				Name:        poi.Name,
				Description: poi.Description,
				Type:        poi.Type,
				Location:    poi.Location,
				Tags:        poi.Tags,
				Address:     poi.Address,
			}
			results = append(results, poiRes)
		}
	}

	log.Printf("Found %d POIs within %f meters", len(results), radius)
	// Sort by distance (closest first)
	return results, nil
}

// Seed Redis with POIs
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

func (s *GeoService) seedPOIsToMongo(collection *mongo.Collection) {
	log.Printf("Seeding sample POIs into MongoDB... %v", collection.Name())
	// read json file with POIs
	file, err := os.Open("./data/sg-pois.json")
	if err != nil {
		log.Fatalf("Failed to open POI file: %v", err)
		return
	}
	defer file.Close()

	var pois []models.POI
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&pois); err != nil {
		log.Fatalf("Failed to decode POI JSON: %v", err)
		return
	}

	log.Printf("Seeding %d POIs into MongoDB...", len(pois))

	// convert to interface{} for MongoDB
	var interfacePois []any
	for _, poi := range pois {
		interfacePois = append(interfacePois, poi)
	}

	result, err := collection.InsertMany(context.Background(), interfacePois)
	if err != nil {
		log.Fatalf("Failed to seed POIs: %v", err)
	}
	log.Printf("Inserted %d POIs into MongoDB", len(result.InsertedIDs))
}
