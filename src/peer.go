package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"net"
	"strconv"
	"time"
)

type Peer struct {
	ID             string `bson:"_id"`
	InfoHash       []byte `bson:"info_hash"`
	IP             string `bson:"ip"`
	Port           string `bson:"port"`
	Uploaded       int64  `bson:"uploaded"`
	Downloaded     int64  `bson:"downloaded"`
	Left           string `bson:"left"`
	LastAnnounce   int64  `bson:"last_announce"`
	LastUploaded   int64  `bson:"last_uploaded"`
	LastDownloaded int64  `bson:"last_downloaded"`
}

func upsertPeer(peer Peer) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	existingPeer := Peer{}
	err := PeerCollection.FindOne(ctx, bson.M{"_id": peer.ID}).Decode(&existingPeer)
	if err != nil && err != mongo.ErrNoDocuments {
		log.Printf("Failed to retrieve existing peer : %v", err)
		return
	}

	if err == nil {
		// TODO: persist this to the user object
		deltaUploaded := peer.Uploaded - existingPeer.LastUploaded
		deltaDownloaded := peer.Downloaded - existingPeer.LastDownloaded

		if deltaUploaded < 0 {
			deltaUploaded = 0 // TODO: Handle Edge cases?
		}
		if deltaDownloaded < 0 {
			deltaDownloaded = 0 // TODO: handle edge cases?
		}
		log.Printf("Deltas - Uploaded: %d, Downloaded: %d", deltaUploaded, deltaDownloaded)
		peer.Uploaded = deltaUploaded
		peer.Downloaded = deltaDownloaded
		peer.LastUploaded = peer.Uploaded
		peer.LastDownloaded = peer.Downloaded
	} else {
		peer.LastUploaded = peer.Uploaded
		peer.LastDownloaded = peer.Downloaded
	}

	log.Printf("Updated stats - Uploaded: %d, Downloaded: %d", peer.Uploaded, peer.Downloaded)

	filter := bson.M{"_id": peer.ID}
	update := bson.M{"$set": peer}
	options := options.Update().SetUpsert(true)

	_, err = PeerCollection.UpdateOne(ctx, filter, update, options)
	if err != nil {
		log.Printf("Failed to upsert peer: %v", err)
	}
	// var userDetails User
	// userId := getUserID()
	// _, err = UserCollection.FindOne(ctx, bson.M{"userID": userID}).Decode(&userDetails)
}

func deletePeer(infoHash []byte, peerID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := PeerCollection.DeleteOne(ctx, bson.M{"info_hash": infoHash, "_id": peerID})
	if err != nil {
		log.Printf("Failed to delete peer: %v", err)
	}
}

func constructTrackerResponse(infoHash []byte, ip string, port string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Printf("Querying peers for info_hash: %v, but not with IP %v or port %v", infoHash, ip, port)
	// get peers of the same torrent, but do not return the calling peer.
	cursor, err := PeerCollection.Find(ctx, bson.M{
		"$and": []interface{}{
			bson.M{"info_hash": infoHash},
			// bson.M{"ip": bson.M{"$ne": ip}},
			// bson.M{"port": bson.M{"$ne": port}}}})
		}})
	if err != nil {
		log.Printf("Failed to query peers: %v", err)
		return "d8:intervali1800e5:peerse"
	}
	defer cursor.Close(ctx)

	var peerBytes []byte
	for cursor.Next(ctx) {
		var peer Peer
		log.Printf("Got a peer, decoding:")
		if err := cursor.Decode(&peer); err != nil {
			log.Printf("Failed to decode peer: %v", err)
			continue
		}
		log.Printf("Decoded peer: %v", peer.IP)

		ipAddr := net.ParseIP(peer.IP).To4()
		if ipAddr != nil {
			peerBytes = append(peerBytes, ipAddr...)
			port, _ := strconv.Atoi(peer.Port)
			buf := make([]byte, 2)
			binary.BigEndian.PutUint16(buf, uint16(port))
			peerBytes = append(peerBytes, buf...)
		}
	}
	response := fmt.Sprintf("d8:intervali1800e5:peers%d:%se", len(peerBytes), string(peerBytes))
	log.Printf("got peerBytes: %v", peerBytes)
	log.Printf("returning response: %v", response)
	return response
}
