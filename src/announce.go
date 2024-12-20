package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type ListedIP struct {
	IP string
}

var mu sync.Mutex

func announceHandler(w http.ResponseWriter, r *http.Request) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	log.Printf("URL Called: %v", r.URL)
	if err != nil {
		log.Printf("Failed to split host and port: %v", err)
		http.Error(w, "Invalid IP address", http.StatusBadRequest)
		return
	}

	acceptIP, err := isIPAllowed(ip)
	if !acceptIP || err != nil {
		log.Printf("IP %s is not allowed", ip)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if rateLimiter.TakeAvailable(1) < 1 {
		log.Printf("Rate limit exceeded for IP %s", ip)
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return
	}

	infoHash := []byte(r.URL.Query().Get("info_hash"))
	peerID := r.URL.Query().Get("peer_id")
	port := r.URL.Query().Get("port")

	uploaded, err := strconv.ParseInt(r.URL.Query().Get("uploaded"), 10, 64)
	if err != nil {
		log.Printf("Failed to parse upload stat: %v", err)
		http.Error(w, "Invalid upload value", http.StatusBadRequest)
	}
	downloaded, err := strconv.ParseInt(r.URL.Query().Get("downloaded"), 10, 64)
	if err != nil {
		log.Printf("Failed to parse download stat: %v", err)
		http.Error(w, "Invalid download value", http.StatusBadRequest)
	}
	left := r.URL.Query().Get("left")
	event := r.URL.Query().Get("event")

	peer := Peer{
		ID:           peerID,
		InfoHash:     infoHash,
		IP:           ip,
		Port:         port,
		Uploaded:     uploaded,
		Downloaded:   downloaded,
		Left:         left,
		LastAnnounce: time.Now().Unix(),
	}

	mu.Lock()
	if event == "stopped" {
		deletePeer(infoHash, peerID)
	} else {
		upsertPeer(peer)
	}
	mu.Unlock()

	response := constructTrackerResponse(infoHash, ip, port)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(response)); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}

func scrapeHandler(w http.ResponseWriter, r *http.Request) {
	infoHashes := r.URL.Query()["info_hash"]

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response := "d5:filesd"

	for _, infoHashHex := range infoHashes {
		infoHash, err := hex.DecodeString(infoHashHex)
		if err != nil {
			log.Printf("Failed to decode info_hash: %v", err)
			continue
		}

		var result struct {
			Seeders   int `bson:"seeders"`
			Leechers  int `bson:"leechers"`
			Completed int `bson:"completed"`
		}

		pipeline := []bson.M{
			{"$match": bson.M{"info_hash": infoHash}},
			{"$group": bson.M{
				"_id": nil,
				"seeders": bson.M{"$sum": bson.M{"$cond": []interface{}{
					bson.M{"$eq": []interface{}{
						"$left", 0}}, 1, 0}}},
				"leechers": bson.M{"$sum": bson.M{"$cond": []interface{}{
					bson.M{"$gt": []interface{}{"$left", 0}}, 1, 0}}},
				"completed": bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$eq": []interface{}{"$left", 0}}, 1, 0}}}, // This is a simplified example
			}},
		}

		cursor, err := PeerCollection.Aggregate(ctx, pipeline)
		if err != nil {
			log.Printf("Failed to scrape: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		defer cursor.Close(ctx)

		if cursor.Next(ctx) {
			if err := cursor.Decode(&result); err != nil {
				log.Printf("Failed to decode scrape result: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
		}
		response += fmt.Sprintf("20:%s%d:%d:%dee", infoHash, result.Seeders, result.Leechers, result.Completed)
	}
	response += "e"
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}
func isIPAllowed(ip string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// fmt.Println("Checking IP: %s", ip)

	blacklistCursor, err := BlacklistCollection.Find(ctx, bson.M{})
	defer blacklistCursor.Close(ctx)

	if err != nil {
		log.Printf("Failed to get blacklist from database: %v", err)
		return false, err
	}
	for blacklistCursor.Next(ctx) {
		var result ListedIP
		if err := blacklistCursor.Decode(&result); err != nil {
			log.Printf("Failed to decode blacklist result: %v", err)
			return false, err
		}
		if result.IP == ip {
			return false, nil
		}
	}
	whitelistCursor, err := WhitelistCollection.Find(ctx, bson.M{})
	defer whitelistCursor.Close(ctx)

	if err != nil {
		log.Printf("Failed to get blacklist from database: %v", err)
		return false, err
	}

	for whitelistCursor.Next(ctx) {
		var result ListedIP
		if err := whitelistCursor.Decode(&result); err != nil {
			log.Printf("Failed to decode whitelist result: %v", err)
			return false, err
		}
		if result.IP == ip {
			return true, nil
		}
	}
	return false, nil

}
