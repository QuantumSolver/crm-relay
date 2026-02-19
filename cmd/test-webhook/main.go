package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

func main() {
	log.Println("Starting test webhook server on port 3000...")

	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Log headers
		log.Printf("Headers:")
		for key, values := range r.Header {
			for _, value := range values {
				log.Printf("  %s: %s", key, value)
			}
		}

		// Read body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(w, "Error reading body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Log body
		log.Printf("Body: %s", string(body))

		// Parse JSON
		var data map[string]interface{}
		if err := json.Unmarshal(body, &data); err == nil {
			log.Printf("Parsed data: %+v", data)
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Webhook received successfully",
		})
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "healthy",
		})
	})

	log.Println("Test webhook server is running on http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
