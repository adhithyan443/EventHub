package reservation

import (
	"context"
	"log/slog"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
)

const (
	SeatLayoutTypeSeated  = "SEATED"
	SeatLayoutTypeGeneral = "GENERAL"
)

type ReservationUsecase struct {
	transactionManager domain.TransactionManager
	logger             *slog.Logger
}

func NewReservationUsecase(
	transactionManager domain.TransactionManager,
	logger *slog.Logger,
) *ReservationUsecase {
	return &ReservationUsecase{
		transactionManager: transactionManager,
		logger:             logger,
	}
}

type CreateReservationRequest struct {
	UserID       uuid.UUID
	EventID      uuid.UUID
	SeatIDs      []uuid.UUID
	TicketTypeID *uuid.UUID
	Quantity     int
}

func (u *ReservationUsecase) CreateReservation(
	ctx context.Context,
	req CreateReservationRequest,
) (*domain.SeatReservation, error) {
	if err := validateReservationRequest(req); err != nil {
		u.logger.WarnContext(ctx, "invalid reservation request",
			slog.String("user_id", req.UserID.String()),
			slog.String("event_id", req.EventID.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	var reservation *domain.SeatReservation

	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			event, err := tx.EventRepository().FindByIDForUpdate(req.EventID)
			if err != nil {
				return err
			}

			if event.Status != domain.EventStatusPublished {
				return domain.ErrEventNotPublic
			}

			// Validate that the event type is supported.
			switch event.EventType {
			case domain.EventTypeOnline,
				domain.EventTypePhysical,
				domain.EventTypeHybrid:
				// Supported event type.
			default:
				return domain.ErrInvalidReservationRequest
			}

			setting, err := tx.EventSettingRepository().FindByEventID(req.EventID)
			if err != nil {
				return err
			}

			requestedQuantity := req.Quantity
			if len(req.SeatIDs) > 0 {
				requestedQuantity = len(req.SeatIDs)
			}

			if requestedQuantity > setting.BookingLimitPerUser {
				u.logger.WarnContext(ctx, "reservation exceeds per-booking ticket limit",
					slog.String("user_id", req.UserID.String()),
					slog.String("event_id", req.EventID.String()),
					slog.Int("requested_quantity", requestedQuantity),
					slog.Int("booking_limit", setting.BookingLimitPerUser),
				)
				return domain.ErrInvalidReservationRequest
			}

			now := time.Now()

			if setting.SalesStartDate != nil &&
				now.Before(*setting.SalesStartDate) {
				return domain.ErrInvalidReservationRequest
			}

			if setting.SalesEndDate != nil &&
				now.After(*setting.SalesEndDate) {
				return domain.ErrInvalidReservationRequest
			}

			hasSeats := len(req.SeatIDs) > 0

			if hasSeats {
				// Online events must never use reserved seating.
				if event.EventType == domain.EventTypeOnline ||
					setting.SeatLayoutType != SeatLayoutTypeSeated {
					return domain.ErrInvalidReservationRequest
				}
			} else {
				// General Admission requires GENERAL ticketing mode.
				if setting.SeatLayoutType != SeatLayoutTypeGeneral {
					return domain.ErrInvalidReservationRequest
				}
			}

			reservation = &domain.SeatReservation{
				ID:        uuid.New(),
				UserID:    req.UserID,
				EventID:   req.EventID,
				Status:    domain.ReservationStatusActive,
				ExpiresAt: now.Add(10 * time.Minute),
				CreatedAt: now,
				UpdatedAt: now,
			}

			if err := tx.SeatReservationRepository().Create(
				ctx, reservation,
			); err != nil {
				return err
			}

			if hasSeats {
				layout, err := tx.SeatLayoutRepository().FindByEventID(
					req.EventID,
				)
				if err != nil {
					return err
				}

				for _, seatID := range req.SeatIDs {
					seat, err := tx.SeatRepository().FindByID(seatID)
					if err != nil {
						return err
					}

					row, err := tx.SeatRowRepository().FindByID(
						seat.RowID,
					)
					if err != nil {
						return err
					}

					section, err := tx.SeatSectionRepository().FindByID(
						row.SectionID,
					)
					if err != nil {
						return err
					}

					if section.SeatLayoutID != layout.ID {
						return domain.ErrInvalidReservationRequest
					}

					if err := tx.SeatRepository().ReserveIfAvailable(
						ctx, seatID,
					); err != nil {
						return err
					}

					seatIDCopy := seatID
					item := &domain.SeatReservationItem{
						ID:            uuid.New(),
						ReservationID: reservation.ID,
						SeatID:        &seatIDCopy,
						Quantity:      1,
						Price:         section.Price,
						CreatedAt:     now,
					}

					if err := tx.SeatReservationRepository().CreateItem(
						ctx, item,
					); err != nil {
						return err
					}
				}
			} else {
				ticketType, err := tx.TicketTypeRepository().FindByID(
					ctx,
					*req.TicketTypeID,
				)
				if err != nil {
					return err
				}

				if ticketType.EventID != req.EventID {
					return domain.ErrTicketTypeEventMismatch
				}

				if err := tx.TicketTypeRepository().ReserveQuantityIfAvailable(
					ctx,
					ticketType.ID,
					req.Quantity,
				); err != nil {
					return err
				}

				ticketTypeID := ticketType.ID
				item := &domain.SeatReservationItem{
					ID:            uuid.New(),
					ReservationID: reservation.ID,
					TicketTypeID:  &ticketTypeID,
					Quantity:      req.Quantity,
					Price:         ticketType.Price,
					CreatedAt:     now,
				}

				if err := tx.SeatReservationRepository().CreateItem(
					ctx, item,
				); err != nil {
					return err
				}
			}

			return nil
		},
	)

	if err != nil {
		u.logger.ErrorContext(ctx, "reservation creation failed",
			slog.String("user_id", req.UserID.String()),
			slog.String("event_id", req.EventID.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	u.logger.InfoContext(ctx, "reservation created",
		slog.String("reservation_id", reservation.ID.String()),
		slog.String("user_id", req.UserID.String()),
		slog.String("event_id", req.EventID.String()),
	)

	return reservation, nil
}

func (u *ReservationUsecase) ExpireReservation(
	ctx context.Context,
	reservationID uuid.UUID,
) error {
	if reservationID == uuid.Nil {
		return domain.ErrInvalidReservationRequest
	}

	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			now := time.Now()

			expired, err := tx.SeatReservationRepository().ExpireIfActive(
				ctx,
				reservationID,
				now,
			)
			if err != nil {
				return err
			}

			// Nothing to release if the reservation was not expired.
			if !expired {
				return nil
			}

			if _, err := tx.CheckoutRepository().
				ExpireActiveByReservationID(ctx, reservationID); err != nil {
				return err
			}

			items, err := tx.SeatReservationRepository().
				FindItemsByReservationID(ctx, reservationID)
			if err != nil {
				return err
			}

			for _, item := range items {
				if item.SeatID != nil {
					if err := tx.SeatRepository().ReleaseReservedSeat(
						ctx,
						*item.SeatID,
					); err != nil {
						return err
					}

					continue
				}

				if item.TicketTypeID != nil {
					if err := tx.TicketTypeRepository().
						ReleaseReservedQuantity(
							ctx,
							*item.TicketTypeID,
							item.Quantity,
						); err != nil {
						return err
					}

					continue
				}

				return domain.ErrInvalidReservationRequest
			}

			return nil
		},
	)

	if err != nil {
		u.logger.ErrorContext(ctx, "reservation expiry failed",
			slog.String("reservation_id", reservationID.String()),
			slog.String("error", err.Error()),
		)
		return err
	}

	u.logger.InfoContext(ctx, "reservation expiry processed",
		slog.String("reservation_id", reservationID.String()),
	)

	return nil
}

func (u *ReservationUsecase) CancelReservation(
	ctx context.Context,
	reservationID uuid.UUID,
	userID uuid.UUID,
) error {

	if reservationID == uuid.Nil || userID == uuid.Nil {
		return domain.ErrInvalidReservationRequest
	}

	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			cancelled, err := tx.SeatReservationRepository().CancelIfActive(
				ctx,
				reservationID,
				userID,
			)
			if err != nil {
				return err
			}

			if !cancelled {
				return domain.ErrSeatReservationNotFound
			}

			items, err := tx.SeatReservationRepository().
				FindItemsByReservationID(ctx, reservationID)
			if err != nil {
				return err
			}

			for _, item := range items {
				if item.SeatID != nil {
					if err := tx.SeatRepository().ReleaseReservedSeat(
						ctx,
						*item.SeatID,
					); err != nil {
						return err
					}
					continue
				}

				if item.TicketTypeID != nil {
					if err := tx.TicketTypeRepository().ReleaseReservedQuantity(
						ctx,
						*item.TicketTypeID,
						item.Quantity,
					); err != nil {
						return err
					}
					continue
				}

				return domain.ErrInvalidReservationRequest
			}

			return nil
		},
	)

	if err != nil {
		u.logger.ErrorContext(ctx, "reservation cancellation failed",
			slog.String("reservation_id", reservationID.String()),
			slog.String("user_id", userID.String()),
			slog.String("error", err.Error()),
		)
		return err
	}

	u.logger.InfoContext(ctx, "reservation cancelled",
		slog.String("reservation_id", reservationID.String()),
		slog.String("user_id", userID.String()),
	)

	return nil
}

func (u *ReservationUsecase) ProcessExpiredReservations(
	ctx context.Context,
	limit int,
) error {
	if limit <= 0 {
		return domain.ErrInvalidReservationRequest
	}

	var expiredReservations []domain.SeatReservation

	err := u.transactionManager.WithinTransaction(
		func(tx domain.TransactionRepositories) error {
			var err error

			expiredReservations, err =
				tx.SeatReservationRepository().FindExpiredActive(
					ctx,
					time.Now(),
					limit,
				)

			return err
		},
	)
	if err != nil {
		u.logger.ErrorContext(
			ctx,
			"failed to fetch expired reservations",
			slog.String("error", err.Error()),
		)
		return err
	}

	var firstErr error

	for _, reservation := range expiredReservations {
		if err := u.ExpireReservation(ctx, reservation.ID); err != nil {
			u.logger.ErrorContext(
				ctx,
				"failed to process expired reservation",
				slog.String("reservation_id", reservation.ID.String()),
				slog.String("error", err.Error()),
			)

			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}

func validateReservationRequest(req CreateReservationRequest) error {

	hasSeats := len(req.SeatIDs) > 0
	hasTicketType := req.TicketTypeID != nil

	if hasSeats == hasTicketType {
		return domain.ErrInvalidReservationRequest
	}

	if hasTicketType && req.Quantity <= 0 {
		return domain.ErrInvalidReservationRequest
	}

	if hasSeats && req.Quantity != 0 {
		return domain.ErrInvalidReservationRequest
	}

	if req.UserID == uuid.Nil || req.EventID == uuid.Nil {
		return domain.ErrInvalidReservationRequest
	}

	seen := make(map[uuid.UUID]struct{}, len(req.SeatIDs))
	for _, seatID := range req.SeatIDs {
		if seatID == uuid.Nil {
			return domain.ErrInvalidReservationRequest
		}

		if _, exists := seen[seatID]; exists {
			return domain.ErrInvalidReservationRequest
		}

		seen[seatID] = struct{}{}
	}

	if hasTicketType && *req.TicketTypeID == uuid.Nil {
		return domain.ErrInvalidReservationRequest
	}

	return nil
}
