package main

import (
	"context"
	"github.com/juju/ratelimit"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

var MongoClient *mongo.Client
var PeerCollection *mongo.Collection
var WhitelistCollection *mongo.Collection
var BlacklistCollection *mongo.Collection
var UserCollection *mongo.Collection
var rateLimiter *ratelimit.Bucket

func initDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	clientOptions := options.Client().ApplyURI(Config.MongoURI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDb: %v", err)
	}
	MongoClient = client
	PeerCollection = client.Database("torrent_tracker").Collection("peers")
	WhitelistCollection = client.Database("torrent_tracker").Collection("whitelist")
	BlacklistCollection = client.Database("torrent_tracker").Collection("blacklist")
	UserCollection = client.Database("torrent_tracker").Collection("users")

	log.Println("Connected to MongoDB")
}

func initRateLimiter() {
	rateLimiter = ratelimit.NewBucketWithRate(float64(Config.RateLimit), int64(Config.RateLimit))
	log.Printf("rate limiter updated")
}
