package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealthz)
	mux.HandleFunc("/echo", handleEcho)
	mux.HandleFunc("/route/", handleEcho) // catch-all for many-routes scenarios
	mux.HandleFunc("/sse", handleSSE)
	mux.HandleFunc("/mcp", handleMCP)

	log.Printf("test-backend listening on :%s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, "ok")
}

func handleEcho(w http.ResponseWriter, r *http.Request) {
	// Optional simulated delay
	if delayStr := r.URL.Query().Get("delay_ms"); delayStr != "" {
		if delayMs, err := strconv.Atoi(delayStr); err == nil && delayMs > 0 && delayMs <= 30000 {
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}
	}

	// If ?size=N, return N bytes of random data
	if sizeStr := r.URL.Query().Get("size"); sizeStr != "" {
		size, err := strconv.Atoi(sizeStr)
		if err != nil || size < 0 || size > 200*1024*1024 { // 200MB max
			http.Error(w, "invalid size parameter (0-209715200)", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", strconv.Itoa(size))
		// Use a deterministic-ish pattern for reproducibility
		buf := make([]byte, size)
		_, _ = rand.Read(buf)
		_, _ = w.Write(buf)
		return
	}

	// Otherwise echo the request body
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	if r.Body != nil {
		_, _ = io.Copy(w, r.Body)
	}
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	count := 10
	if countStr := r.URL.Query().Get("count"); countStr != "" {
		if n, err := strconv.Atoi(countStr); err == nil && n > 0 && n <= 10000 {
			count = n
		}
	}

	intervalMs := 100
	if intervalStr := r.URL.Query().Get("interval_ms"); intervalStr != "" {
		if n, err := strconv.Atoi(intervalStr); err == nil && n > 0 && n <= 60000 {
			intervalMs = n
		}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for i := 0; i < count; i++ {
		select {
		case <-r.Context().Done():
			return
		default:
		}

		_, _ = fmt.Fprintf(w, "id: %d\nevent: message\ndata: {\"seq\":%d,\"ts\":%d}\n\n",
			i, i, time.Now().UnixMilli())
		flusher.Flush()

		if i < count-1 {
			time.Sleep(time.Duration(intervalMs) * time.Millisecond)
		}
	}
}
