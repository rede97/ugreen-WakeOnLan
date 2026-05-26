package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
)

type Device struct {
	Name string `json:"name"`
	MAC  string `json:"mac"`
}

var (
	devices []Device
	mu      sync.Mutex
	// hardcoded devices for demo; in production use a file/DB
	seedDevices = []Device{
		{Name: "My Desktop", MAC: "00:11:22:33:44:55"},
	}
)

func main() {
	devices = append(devices, seedDevices...)

	// CORS for local dev
	cors := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == "OPTIONS" {
				w.WriteHeader(200)
				return
			}
			next(w, r)
		}
	}

	http.HandleFunc("/api/devices", cors(handleDevices))
	http.HandleFunc("/api/wake", cors(handleWake))

	// Serve static files
	fs := http.FileServer(http.Dir("rootfs_common/www"))
	http.Handle("/", fs)

	addr := ":21010"
	log.Printf("WakeOnLan server starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func handleDevices(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(map[string]interface{}{"devices": devices})
	case http.MethodPost:
		var d Device
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			http.Error(w, `{"error":"invalid JSON"}`, 400)
			return
		}
		d.Name = strings.TrimSpace(d.Name)
		d.MAC = strings.TrimSpace(d.MAC)
		if d.Name == "" || d.MAC == "" {
			http.Error(w, `{"error":"name and mac required"}`, 400)
			return
		}
		devices = append(devices, d)
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(d)
	default:
		http.Error(w, `{"error":"method not allowed"}`, 405)
	}
}

func handleWake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, 405)
		return
	}

	var req struct {
		MAC string `json:"mac"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, 400)
		return
	}

	if err := sendMagicPacket(req.MAC); err != nil {
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "mac": req.MAC})
}

func sendMagicPacket(macStr string) error {
	mac, err := net.ParseMAC(macStr)
	if err != nil {
		return err
	}

	// Magic packet: 6 bytes of 0xFF followed by MAC repeated 16 times
	packet := make([]byte, 102)
	for i := 0; i < 6; i++ {
		packet[i] = 0xFF
	}
	for i := 1; i <= 16; i++ {
		copy(packet[i*6:], mac)
	}

	// Broadcast to UDP 9 (standard WOL port)
	conn, err := net.Dial("udp", "255.255.255.255:9")
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write(packet)
	return err
}
