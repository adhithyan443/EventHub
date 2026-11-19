package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SettlementModel struct {
	ID              uuid.UUID       `gorm:"type:uuid;primaryKey"`
	OrganizerID     uuid.UUID       `gorm:"type:uuid;not null;index"`
	EventID         *uuid.UUID      `gorm:"type:uuid;index"`
	Amount          decimal.Decimal `gorm:"type:numeric(12,2);not null"`
	Currency        string          `gorm:"type:varchar(3);not null;default:'INR'"`
	Status          string          `gorm:"type:varchar(30);not null;default:'PENDING';index"`
	PayoutReference *string         `gorm:"type:varchar(255)"`
	FailureReason   *string         `gorm:"type:text"`
	RejectionReason *string         `gorm:"type:text"`
	ApprovedBy      *uuid.UUID      `gorm:"type:uuid"`
	ApprovedAt      *time.Time      `gorm:"type:timestamptz"`
	RejectedAt      *time.Time      `gorm:"type:timestamptz"`
	ProcessedAt     *time.Time      `gorm:"type:timestamptz"`
	CreatedAt       time.Time       `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt       time.Time       `gorm:"type:timestamptz;not null;default:now()"`
}

func (SettlementModel) TableName() string {
	return "settlements"
}

type settlementRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewSettlementRepository(db *gorm.DB, logger *slog.Logger) domain.SettlementRepository {
	if logger == nil {
		logger = slog.Default()
	}
	return &settlementRepository{
		db:     db,
		logger: logger,
	}
}

func (r *settlementRepository) Create(ctx context.Context, s *domain.Settlement) error {
	if s == nil {
		return errors.New("settlement cannot be nil")
	}

	model := toSettlementModel(s)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to create settlement record",
			slog.String("settlement_id", s.ID.String()),
			slog.String("organizer_id", s.OrganizerID.String()),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to create settlement: %w", err)
	}

	return nil
}

func (r *settlementRepository) Update(ctx context.Context, s *domain.Settlement) error {
	if s == nil {
		return errors.New("settlement cannot be nil")
	}

	s.UpdatedAt = time.Now().UTC()
	model := toSettlementModel(s)
	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to update settlement record",
			slog.String("settlement_id", s.ID.String()),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to update settlement: %w", err)
	}

	return nil
}

func (r *settlementRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Settlement, error) {
	if id == uuid.Nil {
		return nil, domain.ErrSettlementNotFound
	}

	var model SettlementModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrSettlementNotFound
		}
		return nil, fmt.Errorf("failed to find settlement by id: %w", err)
	}

	return toSettlementDomain(&model), nil
}

func (r *settlementRepository) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.Settlement, error) {
	if id == uuid.Nil {
		return nil, domain.ErrSettlementNotFound
	}

	var model SettlementModel
	if err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrSettlementNotFound
		}
		return nil, fmt.Errorf("failed to find settlement for update: %w", err)
	}

	return toSettlementDomain(&model), nil
}

func (r *settlementRepository) List(ctx context.Context, filter domain.SettlementFilter) (*domain.SettlementListResult, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	query := r.db.WithContext(ctx).Model(&SettlementModel{})

	if filter.OrganizerID != nil && *filter.OrganizerID != uuid.Nil {
		query = query.Where("organizer_id = ?", *filter.OrganizerID)
	}
	if filter.EventID != nil && *filter.EventID != uuid.Nil {
		query = query.Where("event_id = ?", *filter.EventID)
	}
	if filter.Status != nil && strings.TrimSpace(string(*filter.Status)) != "" {
		query = query.Where("status = ?", string(*filter.Status))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count settlements: %w", err)
	}

	var models []SettlementModel
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to query settlements: %w", err)
	}

	settlements := make([]*domain.Settlement, len(models))
	for i, m := range models {
		settlements[i] = toSettlementDomain(&m)
	}

	return &domain.SettlementListResult{
		Settlements: settlements,
		Total:       total,
		Page:        page,
		Limit:       limit,
	}, nil
}

func (r *settlementRepository) GetInFlightSettlementAmount(ctx context.Context, organizerID uuid.UUID) (decimal.Decimal, error) {
	if organizerID == uuid.Nil {
		return decimal.Zero, nil
	}

	var total decimal.Decimal
	err := r.db.WithContext(ctx).Model(&SettlementModel{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("organizer_id = ?", organizerID).
		Where("status IN ('PENDING', 'APPROVED', 'PROCESSING')").
		Scan(&total).Error
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to compute in-flight settlement amount: %w", err)
	}

	return total, nil
}

func (r *settlementRepository) GetInFlightSettlementAmountForEvent(ctx context.Context, eventID uuid.UUID) (decimal.Decimal, error) {
	if eventID == uuid.Nil {
		return decimal.Zero, nil
	}

	var total decimal.Decimal
	err := r.db.WithContext(ctx).Model(&SettlementModel{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("event_id = ?", eventID).
		Where("status IN ('PENDING', 'APPROVED', 'PROCESSING')").
		Scan(&total).Error
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to compute in-flight settlement amount for event: %w", err)
	}

	return total, nil
}

func toSettlementModel(s *domain.Settlement) *SettlementModel {
	return &SettlementModel{
		ID:              s.ID,
		OrganizerID:     s.OrganizerID,
		EventID:         s.EventID,
		Amount:          s.Amount,
		Currency:        s.Currency,
		Status:          string(s.Status),
		PayoutReference: s.PayoutReference,
		FailureReason:   s.FailureReason,
		RejectionReason: s.RejectionReason,
		ApprovedBy:      s.ApprovedBy,
		ApprovedAt:      s.ApprovedAt,
		RejectedAt:      s.RejectedAt,
		ProcessedAt:     s.ProcessedAt,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
	}
}

func toSettlementDomain(m *SettlementModel) *domain.Settlement {
	return &domain.Settlement{
		ID:              m.ID,
		OrganizerID:     m.OrganizerID,
		EventID:         m.EventID,
		Amount:          m.Amount,
		Currency:        m.Currency,
		Status:          domain.SettlementStatus(m.Status),
		PayoutReference: m.PayoutReference,
		FailureReason:   m.FailureReason,
		RejectionReason: m.RejectionReason,
		ApprovedBy:      m.ApprovedBy,
		ApprovedAt:      m.ApprovedAt,
		RejectedAt:      m.RejectedAt,
		ProcessedAt:     m.ProcessedAt,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}
