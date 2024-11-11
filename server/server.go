package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"receipt-processor/config"
	"receipt-processor/models"
	"receipt-processor/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	cfg    *config.Config
	store  storage.Storage
	logger *slog.Logger
	srv    *http.Server
}

func NewServer(cfg *config.Config, store storage.Storage, logger *slog.Logger) *Server {
	return &Server{
		cfg:    cfg,
		store:  store,
		logger: logger,
	}
}

func (s *Server) Start() error {
	r := chi.NewRouter()

	// middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// routes
	r.Post("/receipts/process", s.processReceipt())
	r.Get("/receipts/{id}/points", s.getPoints())

	s.srv = &http.Server{
		Addr:         ":" + s.cfg.Port,
		Handler:      r,
		ReadTimeout:  s.cfg.ReadTimeout,
		WriteTimeout: s.cfg.WriteTimeout,
	}

	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *Server) processReceipt() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := s.logger.With(
			"handler", "processReceipt",
			"requestID", middleware.GetReqID(ctx),
		)

		var receipt models.Receipt
		if err := json.NewDecoder(r.Body).Decode(&receipt); err != nil {
			s.respondWithError(w, http.StatusBadRequest, "Invalid request payload")
			logger.Error("failed to decode request", "error", err)
			return
		}

		if err := receipt.Validate(); err != nil {
			s.respondWithError(w, http.StatusBadRequest, err.Error())
			logger.Error("invalid receipt", "error", err)
			return
		}

		points := receipt.CalculatePoints()
		id, err := s.store.SavePoints(ctx, points)
		if err != nil {
			s.respondWithError(w, http.StatusInternalServerError, "Failed to process receipt")
			logger.Error("failed to save points", "error", err)
			return
		}

		logger.Info("receipt processed successfully", "id", id, "points", points)
		s.respondWithJSON(w, http.StatusOK, models.ProcessResponse{ID: id})
	}
}

func (s *Server) getPoints() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := s.logger.With(
			"handler", "getPoints",
			"requestID", middleware.GetReqID(ctx),
		)

		id := chi.URLParam(r, "id")
		points, err := s.store.GetPoints(ctx, id)
		if err != nil {
			if err == storage.ErrNotFound {
				s.respondWithError(w, http.StatusNotFound, "Receipt not found")
			} else {
				s.respondWithError(w, http.StatusInternalServerError, "Failed to get points")
			}
			logger.Error("failed to get points", "error", err)
			return
		}

		logger.Info("points retrieved successfully", "id", id, "points", points)
		s.respondWithJSON(w, http.StatusOK, models.PointsResponse{Points: points})
	}
}

func (s *Server) respondWithError(w http.ResponseWriter, code int, message string) {
	s.respondWithJSON(w, code, map[string]string{"error": message})
}

func (s *Server) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
