package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/skshohagmiah/flin/internal/queue"
)

// HTTPServer wraps the Flin server to expose HTTP API endpoints
type HTTPServer struct {
	server *Server
	queue  *queue.Queue
	router *http.ServeMux
	addr   string
}

// Response structures
type ErrorResponse struct {
	Error string `json:"error"`
}

type StatusResponse struct {
	NodeID string `json:"nodeId"`
	Status string `json:"status"`
	Time   string `json:"time"`
}

type KVItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Size  int64  `json:"size"`
}

type KVListResponse struct {
	Items []KVItem `json:"items"`
	Total int      `json:"total"`
}

type QueueItem struct {
	Name  string `json:"name"`
	Depth int    `json:"depth"`
}

type QueueListResponse struct {
	Items []QueueItem `json:"items"`
	Total int         `json:"total"`
}

type SetRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	TTL   int    `json:"ttl,omitempty"`
}

type DeleteRequest struct {
	Key string `json:"key"`
}

type UpdateRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	TTL   int    `json:"ttl,omitempty"`
}

type GetResponse struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type PushRequest struct {
	Queue   string `json:"queue"`
	Message string `json:"message"`
}

type PopRequest struct {
	Queue string `json:"queue"`
}

type PopResponse struct {
	Message string `json:"message"`
}

// NewHTTPServer creates a new HTTP API server for Flin
func NewHTTPServer(server *Server, q *queue.Queue, addr string) *HTTPServer {
	hs := &HTTPServer{
		server: server,
		queue:  q,
		router: http.NewServeMux(),
		addr:   addr,
	}

	// Register routes
	hs.registerRoutes()

	return hs
}

// registerRoutes sets up all HTTP API routes
func (hs *HTTPServer) registerRoutes() {
	// Health check
	hs.router.HandleFunc("/health", hs.handleHealth)
	hs.router.HandleFunc("/status", hs.handleStatus)

	// KV Store routes
	hs.router.HandleFunc("/kv/keys", hs.handleKVKeys)
	hs.router.HandleFunc("/kv/get", hs.handleKVGet)
	hs.router.HandleFunc("/kv/set", hs.handleKVSet)
	hs.router.HandleFunc("/kv/delete", hs.handleKVDelete)
	hs.router.HandleFunc("/kv/update", hs.handleKVUpdate)

	// Queue routes
	hs.router.HandleFunc("/queues", hs.handleQueuesGet)
	hs.router.HandleFunc("/queues/push", hs.handleQueuePush)
	hs.router.HandleFunc("/queues/pop", hs.handleQueuePop)

	// CORS middleware wrapper
	hs.router.Handle("/", corsMiddleware(hs.router))
}

// Start starts the HTTP server
func (hs *HTTPServer) Start() error {
	log.Printf("🌐 HTTP API Server listening on %s", hs.addr)
	return http.ListenAndServe(hs.addr, hs.router)
}

// Handler functions

func (hs *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (hs *HTTPServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(StatusResponse{
		NodeID: hs.server.nodeID,
		Status: "running",
		Time:   time.Now().UTC().String(),
	})
}

// KV Store handlers

func (hs *HTTPServer) handleKVKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get all keys and their values
	kvPairs, err := hs.server.GetKVStore().ScanKeysWithValues("")
	if err != nil {
		writeError(w, "Failed to scan keys: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to items format
	items := make([]KVItem, 0, len(kvPairs))
	for key, value := range kvPairs {
		items = append(items, KVItem{
			Key:   key,
			Value: string(value),
			Size:  int64(len(value)),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(KVListResponse{
		Items: items,
		Total: len(items),
	})
}

func (hs *HTTPServer) handleKVGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		writeError(w, "key parameter is required", http.StatusBadRequest)
		return
	}

	value, err := hs.server.store.Get(key)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, "Key not found", http.StatusNotFound)
		} else {
			writeError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if value == nil {
		writeError(w, "Key not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GetResponse{
		Key:   key,
		Value: string(value),
	})
}

func (hs *HTTPServer) handleKVSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Key == "" || req.Value == "" {
		writeError(w, "key and value are required", http.StatusBadRequest)
		return
	}

	ttl := 0
	if req.TTL > 0 {
		ttl = req.TTL
	}

	err := hs.server.store.Set(req.Key, []byte(req.Value), time.Duration(ttl)*time.Second)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (hs *HTTPServer) handleKVDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Key == "" {
		writeError(w, "key is required", http.StatusBadRequest)
		return
	}

	err := hs.server.store.Delete(req.Key)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (hs *HTTPServer) handleKVUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Key == "" || req.Value == "" {
		writeError(w, "key and value are required", http.StatusBadRequest)
		return
	}

	// Verify key exists
	_, err := hs.server.store.Get(req.Key)
	if err != nil {
		writeError(w, "Key does not exist", http.StatusNotFound)
		return
	}

	ttl := 0
	if req.TTL > 0 {
		ttl = req.TTL
	}

	err = hs.server.store.Set(req.Key, []byte(req.Value), time.Duration(ttl)*time.Second)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// Queue handlers

func (hs *HTTPServer) handleQueuesGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// For now, return empty list as we need to implement queue iteration
	// TODO: Implement queue iteration
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(QueueListResponse{
		Items: []QueueItem{},
		Total: 0,
	})
}

func (hs *HTTPServer) handleQueuePush(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Queue == "" || req.Message == "" {
		writeError(w, "queue and message are required", http.StatusBadRequest)
		return
	}

	err := hs.queue.Push(req.Queue, []byte(req.Message))
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (hs *HTTPServer) handleQueuePop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PopRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Queue == "" {
		writeError(w, "queue is required", http.StatusBadRequest)
		return
	}

	message, err := hs.queue.Pop(req.Queue)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if message == nil {
		writeError(w, "Queue is empty", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PopResponse{
		Message: string(message),
	})
}

// Utility functions

func writeError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

// corsMiddleware adds CORS headers to allow requests from Next.js frontend
func corsMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow all origins (change for production)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	}
}
