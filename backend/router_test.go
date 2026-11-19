package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	"github.com/adhithyan443/EventHub/backend/internal/token"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouter_OrganizerPaymentRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	jwtService := token.NewJWTService("01234567890123456789012345678901")

	// Passing nil usecase to handler for route registration test
	orgPaymentHandler := handler.NewOrganizerPaymentHandler(nil, logger)

	router := setupRouter(
		logger,
		nil,
		&handler.OrganizerApplicationHandler{},
		&handler.OrganizerProfileHandler{},
		orgPaymentHandler,
		&handler.CategoryHandler{},
		&handler.EventHandler{},
		&handler.SeatLayoutHandler{},
		&handler.ReservationHandler{},
		&handler.CheckoutHandler{},
		&handler.BookingHandler{},
		&handler.PaymentHandler{},
		&handler.TicketHandler{},
		&handler.AdminPricingTaxHandler{},
		&handler.AdminUserHandler{},
		&handler.AdminOrganizerHandler{},
		&handler.AdminEventHandler{},
		&handler.AdminPaymentHandler{},
		&handler.ProfileHandler{},
		jwtService,
		nil,
	)
	require.NotNil(t, router)

	// Verify unauthenticated requests hit the routes and get 401 Unauthorized (NOT 404 Not Found)
	testRoutes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/organizers/payments/overview"},
		{http.MethodGet, "/api/v1/organizers/payments/summary-by-event"},
		{http.MethodGet, "/api/v1/organizers/payments"},
	}

	for _, tr := range testRoutes {
		req := httptest.NewRequest(tr.method, tr.path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "Expected 401 Unauthorized on %s, not 404", tr.path)
	}
}
