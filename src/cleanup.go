package main

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"log"
	"time"
)

func cleanupRoutine() {
	for {
		time.Sleep(10 * time.Minute)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		threshold := time.Now().Add(-30 * time.Minute).Unix()
		_, err := PeerCollection.DeleteMany(ctx, bson.M{"last_announce": bson.M{"$lt": threshold}})
		if err != nil {
			log.Printf("Failed to clean up stale peers: %v", err)
		}
	}

}
