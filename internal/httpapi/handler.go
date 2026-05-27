package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/artem/project/internal/domain"
	"github.com/artem/project/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Handler struct {
	repo   *repository.Repository
	logger *slog.Logger
}

func New(repo *repository.Repository, logger *slog.Logger) *Handler {
	return &Handler{
		repo:   repo,
		logger: logger,
	}
}

func (h *Handler) Routes() http.Handler {
	router := chi.NewRouter()

	router.Use(corsMiddleware)
	router.Use(requestLogger(h.logger))

	router.Get("/health", h.health)
	router.Get("/api/v1/subscriptions", h.listSubscriptions)
	router.Get("/api/v1/subscriptions/summary", h.summary)
	router.Get("/api/v1/subscriptions/{id}", h.getSubscription)
	router.Post("/api/v1/subscriptions", h.createSubscription)
	router.Put("/api/v1/subscriptions/{id}", h.updateSubscription)
	router.Delete("/api/v1/subscriptions/{id}", h.deleteSubscription)

	return router
}

func validateCreateSubscription(req domain.CreateSubscriptionRequest) error {
	if req.ServiceName == "" {
		return fmt.Errorf("service_name обязателен")
	}
	if req.Price <= 0 {
		return fmt.Errorf("price должен быть больше нуля")
	}
	if _, err := uuid.Parse(req.UserID); err != nil {
		return fmt.Errorf("user_id должен быть валидным UUID")
	}
	if _, err := time.Parse(domain.MonthLayout, req.StartDate); err != nil {
		return fmt.Errorf("start_date должен быть формата MM-YYYY")
	}
	if req.EndDate != "" {
		if _, err := time.Parse(domain.MonthLayout, req.EndDate); err != nil {
			return fmt.Errorf("end_date должен быть формата MM-YYYY")
		}
	}
	return nil
}

func validateUpdateSubscription(req domain.UpdateSubscriptionRequest) error {
	if req.ServiceName != nil && *req.ServiceName == "" {
		return fmt.Errorf("service_name не должно быть пустым")
	}
	if req.Price != nil && *req.Price <= 0 {
		return fmt.Errorf("price должен быть больше нуля")
	}
	if req.UserID != nil {
		if _, err := uuid.Parse(*req.UserID); err != nil {
			return fmt.Errorf("user_id должен быть валидным UUID")
		}
	}
	if req.StartDate != nil {
		if _, err := time.Parse(domain.MonthLayout, *req.StartDate); err != nil {
			return fmt.Errorf("start_date должен быть формата MM-YYYY")
		}
	}
	if req.EndDate != nil && *req.EndDate != "" {
		if _, err := time.Parse(domain.MonthLayout, *req.EndDate); err != nil {
			return fmt.Errorf("end_date должен быть формата MM-YYYY")
		}
	}
	return nil
}

func (h *Handler) updateSubscription(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		writeError(w, http.StatusBadRequest, "id должен быть похож на UUID")
		return
	}

	var req domain.UpdateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}

	if err := validateUpdateSubscription(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	subscription, err := h.repo.Update(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "subscription не найден")
			return
		}
		h.logger.Error("update subscription failed", "error", err)
		writeError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
		return
	}

	writeJSON(w, http.StatusOK, subscription)
}

func (h *Handler) deleteSubscription(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		writeError(w, http.StatusBadRequest, "id должен быть похож на UUID")
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "subscription не найден")
			return
		}
		h.logger.Error("delete subscription failed", "error", err)
		writeError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listSubscriptions(w http.ResponseWriter, r *http.Request) {
	filter := domain.ListSubscriptionsFilter{
		UserID:      r.URL.Query().Get("user_id"),
		ServiceName: r.URL.Query().Get("service_name"),
	}

	if filter.UserID != "" {
		if _, err := uuid.Parse(filter.UserID); err != nil {
			writeError(w, http.StatusBadRequest, "user_id должен быть валидным UUID")
			return
		}
	}

	items, err := h.repo.List(r.Context(), filter)
	if err != nil {
		h.logger.Error("list subscriptions failed", "error", err)
		writeError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
	})
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	periodStartRaw := r.URL.Query().Get("period_start")
	periodEndRaw := r.URL.Query().Get("period_end")

	if periodStartRaw == "" {
		writeError(w, http.StatusBadRequest, "period_start обязателен")
		return
	}
	if periodEndRaw == "" {
		writeError(w, http.StatusBadRequest, "period_end обязателен")
		return
	}

	periodStart, err := domain.ParseMonth(periodStartRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "period_start должен быть формата MM-YYYY")
		return
	}

	periodEnd, err := domain.ParseMonth(periodEndRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "period_end должен быть формата MM-YYYY")
		return
	}

	if periodEnd.Before(periodStart) {
		writeError(w, http.StatusBadRequest, "period_end должен быть не раньше period_start")
		return
	}

	filter := domain.SummaryFilter{
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		UserID:      r.URL.Query().Get("user_id"),
		ServiceName: r.URL.Query().Get("service_name"),
	}

	if filter.UserID != "" {
		if _, err := uuid.Parse(filter.UserID); err != nil {
			writeError(w, http.StatusBadRequest, "user_id должен быть валидным UUID")
			return
		}
	}

	total, err := h.repo.Summary(r.Context(), filter)
	if err != nil {
		h.logger.Error("calculate summary failed", "error", err)
		writeError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total_price":  total,
		"period_start": periodStartRaw,
		"period_end":   periodEndRaw,
	})
}

func (h *Handler) createSubscription(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}

	if err := validateCreateSubscription(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	subscription, err := h.repo.Create(r.Context(), req)
	if err != nil {
		h.logger.Error("create subscription failed", "error", err)
		writeError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
		return
	}

	writeJSON(w, http.StatusCreated, subscription)
}

func (h *Handler) getSubscription(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		writeError(w, http.StatusBadRequest, "id должен быть похож на UUID")
		return
	}

	subscription, err := h.repo.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "subscription не найден")
			return
		}
		h.logger.Error("get subscription failed", "error", err)
		writeError(w, http.StatusInternalServerError, "внутренняя ошибка сервера")
		return
	}

	writeJSON(w, http.StatusOK, subscription)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}
