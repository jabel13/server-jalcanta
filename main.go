package main
import (
	"encoding/json"
	"log"
	"net/http"
	"time"
	"github.com/gorilla/mux"
	"github.com/jamespearly/loggly"
)

// Struct for the log entry
type LogEntry struct {
	Method   string `json:"method"`
	SourceIP string `json:"sourceIP"`
	Path     string `json:"path"`
	Status   int    `json:"status"`
}

// custom response writer to capture the HTTP status code
type responseWriter struct {
	http.ResponseWriter
	status int
}


// responds with the system's current time
func statusHandler(w http.ResponseWriter, req *http.Request) {
	response := map[string]string{
		"system_time": time.Now().Format(time.RFC3339),
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// logs incoming requests using Loggly
func loggingMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		res := &responseWriter{ResponseWriter: w}

		next.ServeHTTP(res, r)

    	// Initialize loggly tag and client
    	var tag string = "loggly-agent"
		client := loggly.New(tag)

		// Create a log entry with details about the request
		logEntry := LogEntry{
			Method:   r.Method,
			SourceIP: r.RemoteAddr,
			Path:     r.URL.Path,
			Status:   res.status,
		}

		// Convert the log entry to a JSON string
		logData, err := json.Marshal(logEntry)
		if err != nil {
			log.Printf("Failed to marshal logEntry: %v", err)
			return
		}

		// Send the log to loggly
		client.Send("info", string(logData))
	})
}

// // WriteHeader captures the status code before writing it
func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func main() {
	// Set up the router
	r := mux.NewRouter()
	
	// Define the route for "/status"
	r.HandleFunc("/status", statusHandler).Methods("GET")

	// Add the middleware
	r.Use(loggingMiddleware)

	// Start server on port 8080
	log.Fatal(http.ListenAndServe(":8080", r))
}