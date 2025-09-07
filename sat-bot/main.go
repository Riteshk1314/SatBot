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

// Global context variable to store content from context.txt
var Context string

// Message represents the request payload for chat completion
type Message struct {
	Message string `json:"message"`
}

// ChatCompletion represents the response structure from GROQ API
type ChatCompletion struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

// ChatResponse represents the chat completion response
type ChatResponse struct {
	Response     string `json:"response"`
	ResponseTime string `json:"response_time"`
}

// ErrorResponse represents error response structure
type ErrorResponse struct {
	Error string `json:"error"`
}

// loadEnv loads environment variables from .env file
func loadEnv() {
	file, err := os.Open(".env")
	if err != nil {
		log.Printf("Warning: Could not open .env file: %v", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Split by = sign
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		// Remove quotes if present
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}
		
		// Set environment variable only if not already set
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
	
	if err := scanner.Err(); err != nil {
		log.Printf("Error reading .env file: %v", err)
	} else {
		log.Println("Environment variables loaded from .env file")
	}
}

// loadContext reads the context from context.txt file
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

// healthCheckHandler handles the health check endpoint
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

// chatCompletionHandler handles the chat completion endpoint
func chatCompletionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Only allow POST requests
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Method not allowed"})
		return
	}
	
	// Parse request body
	var msg Message
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	
	if err := decoder.Decode(&msg); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid request format"})
		return
	}
	
	// Validate message content
	if strings.TrimSpace(msg.Message) == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Message cannot be empty"})
		return
	}
	
	// Get API key from environment
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "GROQ API key not configured"})
		return
	}
	
	// Prepare the prompt
	groqPrompt := fmt.Sprintf("User Query: %s\n\nAnswer:", msg.Message)
	
	// Prepare API request
	apiURL := "https://api.groq.com/openai/v1/chat/completions"
	
		requestData := map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"role": "system",
				"content": fmt.Sprintf(`You are SatBot, the friendly and knowledgeable AI assistant for the Thapar Institute of Engineering and Technology's annual techno cultural fest i.e Saturnalia.

Your characteristics:
- You're enthusiastic about Saturnalia and its events
- You provide accurate and helpful information
- You speak in a clear, friendly manner
- Saturnalia is a celebration of technology, culture, and creativity
-It is golden jubilee year of Saturnalia

Guidelines:
- Answer questions based on the provided context
- Keep responses concise but informative
- Use natural, conversational language
- If asked about topics outside the context, politely explain that you can only discuss Saturnalia Centre related matters
- Always maintain a helpful and positive attitude



Information, facts and keypoints for reference to answer the question asked: %s
`, Context),
			},
			{
				"role":    "user",
				"content": groqPrompt,
			},
		},
		"model":       "llama-3.1-8b-instant",
		"temperature": 0.7,
		"max_tokens":  500,
	}
	
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to prepare request"})
		return
	}
	
	// Create HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to create request"})
		return
	}
	
	// Set headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")
	
	// Make the API call and measure response time
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
	
	// Read response with size limit
	reader := io.LimitReader(resp.Body, 10*1024*1024) // 10 MB max
	body, err := io.ReadAll(reader)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to read response"})
		return
	}
	
	// Check for API errors
	if resp.StatusCode != http.StatusOK {
		log.Printf("GROQ API error: Status %d, Body: %s", resp.StatusCode, string(body))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "GROQ API returned an error"})
		return
	}
	
	// Parse the response
	var chatCompletion ChatCompletion
	if err := json.Unmarshal(body, &chatCompletion); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to parse response"})
		return
	}
	
	// Validate response structure
	if len(chatCompletion.Choices) == 0 {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "No response from GROQ API"})
		return
	}
	
	endTime := time.Now()
	responseTime := endTime.Sub(startTime)
	
	// Log the interaction (in production, you might want to use a proper logging system)
	go func() {
		log.Printf("Chat interaction - Question: %s, Response Time: %.4f seconds", 
			msg.Message, responseTime.Seconds())
	}()
	
	// Send successful response
	response := ChatResponse{
		Response:     chatCompletion.Choices[0].Message.Content,
		ResponseTime: fmt.Sprintf("%.4f seconds", responseTime.Seconds()),
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func main() {
	// Load environment variables from .env file
	loadEnv()
	
	// Load context from file
	loadContext()
	
	// Create router
	r := mux.NewRouter()
	
	// Define routes
	r.HandleFunc("/health", healthCheckHandler).Methods("GET")
	r.HandleFunc("/chat", chatCompletionHandler).Methods("POST")
	
	// Get port from environment or use default
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