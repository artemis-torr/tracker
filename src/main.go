package main

import (
	"encoding/binary"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

func main() {
	initConfig()
	initDB()
	initRateLimiter()
	go cleanupRoutine()
	go startUDPServer(Config.UDPPort)

	http.HandleFunc("/announce", http.HandlerFunc(announceHandler))
	http.HandleFunc("/scrape", http.HandlerFunc(scrapeHandler))
	log.Printf("Starting HTTP server on :%s...\n", Config.HTTPPort)
	if err := http.ListenAndServe(":"+Config.HTTPPort, nil); err != nil {
		log.Fatalf("HTTP server failed to start: %v", err)
	}
}

func startUDPServer(port string) {
	addr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("Failed to start UDP server: %v", err)
	}
	defer conn.Close()

	log.Printf("Starting UDP server on :%s...\n", port)

	buf := make([]byte, 2048)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Failed to read from UDP: %v", err)
			continue
		}

		go handleUDPAnnounce(conn, addr, buf[:n])
	}
}

func handleUDPAnnounce(conn *net.UDPConn, addr *net.UDPAddr, data []byte) {
	ip := addr.IP.String()

	if rateLimiter.TakeAvailable(1) < 1 {
		log.Printf("Rate limit exceeded for IP %s", ip)
		return
	}
	// Parse the UDP request (implement this function based on the UDP announce packet structure)
	// Here's a simplified example:
	// TODO: Update to match HTTP Logic
	peerID := string(data[:20])
	infoHash := []byte(data[20:40])
	port := strconv.Itoa(int(binary.BigEndian.Uint16(data[40:42])))

	peer := Peer{
		ID:           peerID,
		IP:           ip,
		InfoHash:     infoHash,
		Port:         port,
		Uploaded:     0,
		Downloaded:   0,
		Left:         "0",
		LastAnnounce: time.Now().Unix(),
	}

	upsertPeer(peer)

	response := constructTrackerResponse(infoHash, ip, port)
	if _, err := conn.WriteToUDP([]byte(response), addr); err != nil {
		log.Printf("Failed to send UDP response: %v", err)
	}
}
