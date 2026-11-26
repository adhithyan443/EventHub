package main

import (
	"log/slog"
	"testing"

	"github.com/adhithyan443/EventHub/backend/internal/delivery/http/handler"
	"github.com/adhithyan443/EventHub/backend/internal/token"
)

func TestSetupRouter(t *testing.T) {
	jwtSvc := token.NewJWTService("secret-key-at-least-32-bytes-long!")
	logger := slog.Default()

	engine := setupRouter(
		logger,
		&handler.AuthHandler{},
		&handler.OrganizerApplicationHandler{},
		&handler.OrganizerProfileHandler{},
		nil,
		nil,
		nil,
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
		&handler.AdminSettlementHandler{},
		nil,
		nil,
		&handler.AdminBookingHandler{},
		&handler.ProfileHandler{},
		&handler.TicketCheckInHandler{},
		jwtSvc,
		nil,
	)

	if engine == nil {
		t.Fatal("expected gin.Engine to be non-nil")
	}

	routes := engine.Routes()
	if len(routes) == 0 {
		t.Fatal("expected router to have registered routes")
	}
}
