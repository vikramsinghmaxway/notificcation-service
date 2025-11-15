package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
)

type RegisterReq struct {
	DeviceID string `json:"deviceId"`
	Token    string `json:"token"`
}

type SendReq struct {
	Token string `json:"token,omitempty"` // optional: if present send to single device
	Title string `json:"title"`
	Body  string `json:"body"`
}

var (
	// in-memory store: deviceID -> token
	deviceStore = make(map[string]string)
	dsMu        = sync.RWMutex{}
)

func main() {
	// initialize firebase
	if err := InitFCM(); err != nil {
		log.Fatalf("FCM init error: %v", err)
	}

	r := chi.NewRouter()

	// register a device token
	r.Post("/register", registerHandler)
	r.Post("/send", sendHandler)
    r.Get("/url", getURL)

	log.Println("Server running on :8000")
	http.ListenAndServe(":8000", r)
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.DeviceID == "" || req.Token == "" {
		http.Error(w, "deviceId and token required", http.StatusBadRequest)
		return
	}

	dsMu.Lock()
	deviceStore[req.DeviceID] = req.Token
	dsMu.Unlock()

	log.Printf("Registered device: %s token(len=%d)\n", req.DeviceID, len(req.Token))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func sendHandler(w http.ResponseWriter, r *http.Request) {
	var req SendReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Title == "" || req.Body == "" {
		http.Error(w, "title and body required", http.StatusBadRequest)
		return
	}

	// If a token is provided in request -> send single
	if req.Token != "" {
		go func(t string) {
			if err := SendFCM(req.Title, req.Body, t); err != nil {
				log.Printf("FCM send error (single): %v\n", err)
			} else {
				log.Printf("FCM sent (single) to token len=%d\n", len(t))
			}
		}(req.Token)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "sent_to_token"})
		return
	}

	// Otherwise broadcast to all registered tokens
	dsMu.RLock()
	tokens := make([]string, 0, len(deviceStore))
	for _, tk := range deviceStore {
		tokens = append(tokens, tk)
	}
	dsMu.RUnlock()

	if len(tokens) == 0 {
		http.Error(w, "no registered devices", http.StatusBadRequest)
		return
	}

	// send concurrently but not too many goroutines (simple approach)
	for _, t := range tokens {
		go func(tok string) {
			if err := SendFCM(req.Title, req.Body, tok); err != nil {
				log.Printf("FCM send error (broadcast): %v\n", err)
			} else {
				log.Printf("FCM sent (broadcast) token len=%d\n", len(tok))
			}
		}(t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "broadcast_started",
		"target_devices": len(tokens),
	})
}


func getURL(w http.ResponseWriter, r *http.Request) {
    url := "https://google.com"
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"url": url})
}
