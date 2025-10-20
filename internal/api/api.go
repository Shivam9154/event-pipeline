package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"event-pipeline/internal/config"
	"event-pipeline/internal/database"
	"event-pipeline/internal/logger"
)

// Server represents the API server
type Server struct {
	router *mux.Router
	db     *database.DB
	cfg    *config.APIConfig
	server *http.Server
}

// New creates a new API server
func New(cfg *config.APIConfig, db *database.DB) *Server {
	s := &Server{
		router: mux.NewRouter(),
		db:     db,
		cfg:    cfg,
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures the API routes
func (s *Server) setupRoutes() {
	// Health check
	s.router.HandleFunc("/health", s.healthCheck).Methods("GET")
	
	// API routes
	s.router.HandleFunc("/users/{id}", s.getUser).Methods("GET")
	s.router.HandleFunc("/orders/{id}", s.getOrder).Methods("GET")
	
	// Metrics endpoint
	s.router.Handle("/metrics", promhttp.Handler())
}

// Start starts the API server
func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:         ":" + s.cfg.Port,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Log.Infof("Starting API server on port %s", s.cfg.Port)
	return s.server.ListenAndServe()
}

// Stop gracefully stops the API server
func (s *Server) Stop(ctx context.Context) error {
	logger.Log.Info("Shutting down API server...")
	return s.server.Shutdown(ctx)
}

// healthCheck handles health check requests
func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// getUser handles GET /users/{id}
func (s *Server) getUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	user, err := s.db.GetUserWithOrders(ctx, userID)
	if err != nil {
		logger.Log.Errorf("Failed to get user: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

// getOrder handles GET /orders/{id}
func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["id"]

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	order, err := s.db.GetOrderWithPayment(ctx, orderID)
	if err != nil {
		logger.Log.Errorf("Failed to get order: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(order)
}
