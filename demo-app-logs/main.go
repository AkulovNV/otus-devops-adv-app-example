package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type LogEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Level       string    `json:"level"`
	Service     string    `json:"service"`
	Method      string    `json:"method"`
	Endpoint    string    `json:"endpoint"`
	StatusCode  int       `json:"status_code"`
	Duration    float64   `json:"duration_ms"`
	UserID      string    `json:"user_id,omitempty"`
	Message     string    `json:"message"`
	TraceID     string    `json:"trace_id"`
	Error       string    `json:"error,omitempty"`
	RequestSize int       `json:"request_size,omitempty"`
	Environment string    `json:"environment"`
}

type ResponseData struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data,omitempty"`
}

var (
	serviceName = getEnv("SERVICE_NAME", "demo-api")
	environment = getEnv("ENVIRONMENT", "development")
	version     = getEnv("VERSION", "1.0.0")
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func generateTraceID() string {
	return fmt.Sprintf("trace-%d-%d", time.Now().Unix(), rand.Intn(10000))
}

func logRequest(method, endpoint string, statusCode int, duration float64, userID, message, traceID, errorMsg string) {
	logEntry := LogEntry{
		Timestamp:   time.Now(),
		Level:       getLogLevel(statusCode),
		Service:     serviceName,
		Method:      method,
		Endpoint:    endpoint,
		StatusCode:  statusCode,
		Duration:    duration,
		UserID:      userID,
		Message:     message,
		TraceID:     traceID,
		Error:       errorMsg,
		Environment: environment,
	}

	logJSON, _ := json.Marshal(logEntry)
	fmt.Println(string(logJSON))
}

func getLogLevel(statusCode int) string {
	switch {
	case statusCode >= 500:
		return "error"
	case statusCode >= 400:
		return "warn"
	case statusCode >= 200:
		return "info"
	default:
		return "debug"
	}
}

func simulateWork(minMs, maxMs int) time.Duration {
	duration := time.Duration(rand.Intn(maxMs-minMs)+minMs) * time.Millisecond
	time.Sleep(duration)
	return duration
}

// Health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	traceID := generateTraceID()

	duration := simulateWork(10, 50)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := ResponseData{
		Message:   "Service is healthy",
		Timestamp: time.Now(),
		Data: map[string]string{
			"version":     version,
			"service":     serviceName,
			"environment": environment,
		},
	}

	json.NewEncoder(w).Encode(response)

	logRequest(r.Method, r.URL.Path, http.StatusOK,
		float64(time.Since(start).Nanoseconds())/1e6,
		"", "Health check successful", traceID, "")
}

// Users endpoint with various response scenarios
func usersHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	traceID := generateTraceID()
	userID := r.URL.Query().Get("user_id")

	// Simulate different response scenarios
	scenario := rand.Intn(100)
	var statusCode int
	var message string
	var errorMsg string
	var responseData ResponseData

	switch {
	case scenario < 70: // 70% success
		duration := simulateWork(50, 200)
		statusCode = http.StatusOK
		message = "Users retrieved successfully"
		responseData = ResponseData{
			Message:   "Users retrieved",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"users": []map[string]interface{}{
					{"id": 1, "name": "John Doe", "active": true},
					{"id": 2, "name": "Jane Smith", "active": true},
				},
				"total": 2,
			},
		}
	case scenario < 85: // 15% client errors
		duration := simulateWork(20, 100)
		statusCode = http.StatusBadRequest
		message = "Invalid user request"
		errorMsg = "Missing required parameter"
		responseData = ResponseData{
			Message:   "Bad Request",
			Timestamp: time.Now(),
		}
	case scenario < 95: // 10% not found
		duration := simulateWork(30, 80)
		statusCode = http.StatusNotFound
		message = "User not found"
		errorMsg = fmt.Sprintf("User %s not found in database", userID)
		responseData = ResponseData{
			Message:   "User not found",
			Timestamp: time.Now(),
		}
	default: // 5% server errors
		duration := simulateWork(200, 1000)
		statusCode = http.StatusInternalServerError
		message = "Database connection failed"
		errorMsg = "Connection timeout to database after 5 seconds"
		responseData = ResponseData{
			Message:   "Internal Server Error",
			Timestamp: time.Now(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(responseData)

	logRequest(r.Method, r.URL.Path, statusCode,
		float64(time.Since(start).Nanoseconds())/1e6,
		userID, message, traceID, errorMsg)
}

// Orders endpoint with heavy processing simulation
func ordersHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	traceID := generateTraceID()

	// Simulate heavy processing with different latencies for histogram_quantile demo
	scenario := rand.Intn(100)
	var duration time.Duration
	var statusCode int
	var message string
	var errorMsg string

	switch {
	case scenario < 40: // 40% fast responses (50-200ms)
		duration = simulateWork(50, 200)
		statusCode = http.StatusOK
		message = "Orders retrieved quickly from cache"
	case scenario < 70: // 30% medium responses (200-500ms)
		duration = simulateWork(200, 500)
		statusCode = http.StatusOK
		message = "Orders retrieved from database"
	case scenario < 85: // 15% slow responses (500-1000ms)
		duration = simulateWork(500, 1000)
		statusCode = http.StatusOK
		message = "Orders retrieved with complex query"
	case scenario < 95: // 10% very slow responses (1-2s)
		duration = simulateWork(1000, 2000)
		statusCode = http.StatusOK
		message = "Orders retrieved with slow report generation"
	default: // 5% timeout errors
		duration = simulateWork(3000, 5000)
		statusCode = http.StatusGatewayTimeout
		message = "Order service timeout"
		errorMsg = "Downstream service timeout after 5 seconds"
	}

	responseData := ResponseData{
		Message:   "Orders processed",
		Timestamp: time.Now(),
	}

	if statusCode == http.StatusOK {
		responseData.Data = map[string]interface{}{
			"orders": []map[string]interface{}{
				{"id": 1001, "amount": 125.50, "status": "completed"},
				{"id": 1002, "amount": 75.25, "status": "pending"},
			},
			"processing_time_ms": duration.Milliseconds(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(responseData)

	logRequest(r.Method, r.URL.Path, statusCode,
		float64(time.Since(start).Nanoseconds())/1e6,
		"", message, traceID, errorMsg)
}

// Load testing endpoint
func loadHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	traceID := generateTraceID()

	// Parse load parameters
	requests := 10
	if reqParam := r.URL.Query().Get("requests"); reqParam != "" {
		if parsed, err := strconv.Atoi(reqParam); err == nil {
			requests = parsed
		}
	}

	delay := 100
	if delayParam := r.URL.Query().Get("delay"); delayParam != "" {
		if parsed, err := strconv.Atoi(delayParam); err == nil {
			delay = parsed
		}
	}

	// Generate multiple log entries to simulate load
	for i := 0; i < requests; i++ {
		go func(reqNum int) {
			reqTraceID := fmt.Sprintf("%s-req-%d", traceID, reqNum)
			duration := simulateWork(delay/2, delay*2)

			// Vary the status codes for realistic load testing
			var statusCode int
			var message string
			var errorMsg string

			scenario := rand.Intn(100)
			switch {
			case scenario < 80:
				statusCode = http.StatusOK
				message = fmt.Sprintf("Load test request %d completed", reqNum)
			case scenario < 90:
				statusCode = http.StatusTooManyRequests
				message = fmt.Sprintf("Rate limit exceeded for request %d", reqNum)
				errorMsg = "Too many requests"
			default:
				statusCode = http.StatusInternalServerError
				message = fmt.Sprintf("Load test request %d failed", reqNum)
				errorMsg = "Service overloaded"
			}

			logRequest("GET", "/api/load", statusCode,
				duration.Seconds()*1000,
				fmt.Sprintf("load-user-%d", reqNum), message, reqTraceID, errorMsg)
		}(i)

		time.Sleep(time.Duration(10) * time.Millisecond) // Small delay between requests
	}

	responseData := ResponseData{
		Message:   fmt.Sprintf("Initiated %d load test requests", requests),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"requests": requests,
			"delay_ms": delay,
			"trace_id": traceID,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(responseData)

	logRequest(r.Method, r.URL.Path, http.StatusAccepted,
		float64(time.Since(start).Nanoseconds())/1e6,
		"", fmt.Sprintf("Load test initiated with %d requests", requests), traceID, "")
}

func main() {
	rand.Seed(time.Now().UnixNano())

	r := mux.NewRouter()

	// API routes
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/health", healthHandler).Methods("GET")
	api.HandleFunc("/users", usersHandler).Methods("GET")
	api.HandleFunc("/orders", ordersHandler).Methods("GET", "POST")
	api.HandleFunc("/load", loadHandler).Methods("GET")

	// Root endpoint
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		response := ResponseData{
			Message:   "Demo API for Loki logging",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"service":     serviceName,
				"version":     version,
				"environment": environment,
				"endpoints": []string{
					"/api/health",
					"/api/users",
					"/api/orders",
					"/api/load?requests=N&delay=N",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

		logRequest(r.Method, r.URL.Path, http.StatusOK, 10, "", "Root endpoint accessed", generateTraceID(), "")
	})

	port := getEnv("PORT", "8080")

	// Log service startup
	startupLog := LogEntry{
		Timestamp:   time.Now(),
		Level:       "info",
		Service:     serviceName,
		Message:     fmt.Sprintf("Starting %s on port %s", serviceName, port),
		Environment: environment,
		TraceID:     generateTraceID(),
	}
	startupJSON, _ := json.Marshal(startupLog)
	fmt.Println(string(startupJSON))

	fmt.Printf("Server starting on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
