package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

var Context string

type Message struct {
	Message string `json:"message"`
}

type ChatCompletion struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

type ChatResponse struct {
	Response     string `json:"response"`
	ResponseTime string `json:"response_time"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func loadEnv() {
	file, err := os.Open(".env")
	if err != nil {
		log.Printf("Could not open .env file: %v", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Error reading .env file: %v", err)
	}
}

func loadContext() {
	content, err := os.ReadFile("context.txt")
	if err != nil {
		log.Printf("Warning: Could not read context.txt: %v", err)
		Context = "No context available"
		return
	}

	Context = strings.TrimSpace(string(content))
	if Context == "" {
		Context = "No context available"
	}

	log.Println("Context loaded successfully from context.txt")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// List of allowed origins
		allowedOrigins := []string{
			"http://localhost:3000",
			"https://saturnalia.in",
		}

		// Check if the origin is in the allowed list
		isAllowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				isAllowed = true
				break
			}
		}

		// Set CORS headers only if the origin is allowed
		if isAllowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   "1.0.0",
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func chatCompletionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Method not allowed"})
		return
	}

	var msg Message
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&msg); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid request format"})
		return
	}

	if strings.TrimSpace(msg.Message) == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Message cannot be empty"})
		return
	}

	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "GROQ API key not configured"})
		return
	}

	groqPrompt := fmt.Sprintf("User Query: %s\n\nAnswer:", msg.Message)
	apiURL := "https://api.groq.com/openai/v1/chat/completions"
	requestData := map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"role": "system",
				"content": fmt.Sprintf(`You are SatBot, the friendly and knowledgeable AI assistant for the Thapar Institute of Engineering and Technology's annual techno cultural fest i.e Saturnalia.Keep responses concise but informative.

- Saturnalia is a celebration of technology, culture, and creativity
-It is golden jubilee year of Saturnalia
- Keep responses concise but informative
- Answer questions based on the provided context
- Keep responses concise but informative
- If asked about topics outside the context, politely explain that you can only discuss Saturnalia Centre related matters
- Always maintain a helpful and positive attitude
- The Saturnalia is happening from 14th to 16th November 2025. 

`, Context),
			},
			{
				"role":    "user",
				"content": groqPrompt,
			},
		},
		"model":       "moonshotai/kimi-k2-instruct-0905",
		"temperature": 0.7,
		"max_tokens":  500,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to prepare request"})
		return
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to create request"})
		return
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")

	startTime := time.Now()
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to call GROQ API"})
		return
	}
	defer resp.Body.Close()

	reader := io.LimitReader(resp.Body, 10*1024*1024)
	body, err := io.ReadAll(reader)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to read response"})
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("GROQ API error: Status %d, Body: %s", resp.StatusCode, string(body))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Limit reached for free tier"})
		return
	}

	var chatCompletion ChatCompletion
	if err := json.Unmarshal(body, &chatCompletion); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to parse response"})
		return
	}

	if len(chatCompletion.Choices) == 0 {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Limit reached for free tier"})
		return
	}

	endTime := time.Now()
	responseTime := endTime.Sub(startTime)

	go func() {
		log.Printf("Chat interaction - Question: %s, Response Time: %.4f seconds", msg.Message, responseTime.Seconds())
	}()

	response := ChatResponse{
		Response:     chatCompletion.Choices[0].Message.Content,
		ResponseTime: fmt.Sprintf("%.4f seconds", responseTime.Seconds()),
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func main() {
	loadEnv()
	loadContext()

	r := mux.NewRouter()

	r.Use(corsMiddleware)

	r.HandleFunc("/health", healthCheckHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/chat", chatCompletionHandler).Methods("POST", "OPTIONS")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Server starting on port %s", port)
	log.Printf("Health check endpoint: http://localhost:%s/health", port)
	log.Printf("Chat completion endpoint: http://localhost:%s/chat", port)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
