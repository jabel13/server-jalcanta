package main
import (
	"encoding/json"
	"log"
	"net/http"
	"time"
	"github.com/gorilla/mux"
	"github.com/jamespearly/loggly"
)

func statusHandler(w http.ResponseWriter, req *http.Request) {
	response := map[string]string{
		"system_time": time.Now().Format(time.RFC3339),
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}


func loggingMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		res := &responseWriter{ResponseWriter: w}

		next.ServeHTTP(res, r)

    	// Initialize loggly tag and client
    	var tag string = "loggly-agent"
		client := loggly.New(tag)

		logEntry := LogEntry{
			Method:   r.Method,
			SourceIP: r.RemoteAddr,
			Path:     r.URL.Path,
			Status:   res.status,
		}

		// Convert the logEntry map to a JSON string
		logData, err := json.Marshal(logEntry)
		if err != nil {
			log.Printf("Failed to marshal logEntry: %v", err)
			return
		}


		client.Send("info", string(logData))
	})
}

// Struct for the log entry
type LogEntry struct {
	Method   string `json:"method"`
	SourceIP string `json:"sourceIP"`
	Path     string `json:"path"`
	Status   int    `json:"status"`
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func main() {

	r := mux.NewRouter()
	r.HandleFunc("/status", statusHandler).Methods("GET")
	r.Use(loggingMiddleware)
	log.Fatal(http.ListenAndServe(":8080", r))
}