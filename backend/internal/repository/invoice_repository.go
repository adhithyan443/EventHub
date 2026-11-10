package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/repository/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type invoiceRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewInvoiceRepository(db *gorm.DB, logger *slog.Logger) domain.InvoiceRepository {
	if logger == nil {
		logger = slog.Default()
	}
	return &invoiceRepository{
		db:     db,
		logger: logger,
	}
}

func (r *invoiceRepository) NextReceiptNumber(ctx context.Context, t time.Time) (string, error) {
	var seq int64
	err := r.db.WithContext(ctx).Raw("SELECT nextval('booking_receipt_number_seq')").Scan(&seq).Error
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to get next receipt sequence value", slog.Any("error", err))
		return "", fmt.Errorf("nextval sequence error: %w", err)
	}

	dateStr := t.Format("20060102")
	return fmt.Sprintf("EVH-REC-%s-%06d", dateStr, seq), nil
}

func (r *invoiceRepository) Create(ctx context.Context, invoice *domain.Invoice, items []domain.InvoiceItem) error {
	if invoice == nil {
		return errors.New("invoice is required")
	}

	db := r.db.WithContext(ctx)

	invoiceModel := invoiceToModel(invoice)
	if err := db.Create(invoiceModel).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrInvoiceAlreadyExists
		}
		r.logger.ErrorContext(ctx, "failed to create invoice record",
			slog.String("booking_id", invoice.BookingID.String()),
			slog.String("invoice_number", invoice.InvoiceNumber),
			slog.Any("error", err),
		)
		return err
	}

	if len(items) > 0 {
		itemModels := make([]models.InvoiceItemModel, 0, len(items))
		for _, item := range items {
			itemModels = append(itemModels, models.InvoiceItemModel{
				ID:            item.ID,
				InvoiceID:     invoice.ID,
				BookingItemID: item.BookingItemID,
				CategoryName:  item.CategoryName,
				SeatLabel:     item.SeatLabel,
				Quantity:      item.Quantity,
				UnitPrice:     item.UnitPrice,
				TotalPrice:    item.TotalPrice,
				CreatedAt:     item.CreatedAt,
			})
		}

		if err := db.Create(&itemModels).Error; err != nil {
			r.logger.ErrorContext(ctx, "failed to create invoice items",
				slog.String("invoice_id", invoice.ID.String()),
				slog.Any("error", err),
			)
			return err
		}
	}

	return nil
}

func (r *invoiceRepository) FindByBookingID(ctx context.Context, bookingID uuid.UUID) (*domain.Invoice, error) {
	var model models.InvoiceModel
	err := r.db.WithContext(ctx).
		Where("booking_id = ?", bookingID).
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrInvoiceNotFound
		}
		return nil, err
	}

	return invoiceModelToDomain(&model), nil
}

func (r *invoiceRepository) FindByBookingIDWithItems(ctx context.Context, bookingID uuid.UUID) (*domain.Invoice, []domain.InvoiceItem, error) {
	var model models.InvoiceModel
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("booking_id = ?", bookingID).
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, domain.ErrInvoiceNotFound
		}
		return nil, nil, err
	}

	inv := invoiceModelToDomain(&model)
	items := make([]domain.InvoiceItem, 0, len(model.Items))
	for _, it := range model.Items {
		items = append(items, invoiceItemModelToDomain(&it))
	}
	inv.Items = items

	return inv, items, nil
}

func (r *invoiceRepository) FindByInvoiceNumber(ctx context.Context, invoiceNumber string) (*domain.Invoice, []domain.InvoiceItem, error) {
	var model models.InvoiceModel
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("invoice_number = ?", invoiceNumber).
		First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, domain.ErrInvoiceNotFound
		}
		return nil, nil, err
	}

	inv := invoiceModelToDomain(&model)
	items := make([]domain.InvoiceItem, 0, len(model.Items))
	for _, it := range model.Items {
		items = append(items, invoiceItemModelToDomain(&it))
	}
	inv.Items = items

	return inv, items, nil
}

func (r *invoiceRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	result := r.db.WithContext(ctx).
		Model(&models.InvoiceModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrInvoiceNotFound
	}
	return nil
}

func invoiceToModel(d *domain.Invoice) *models.InvoiceModel {
	return &models.InvoiceModel{
		ID:             d.ID,
		InvoiceNumber:  d.InvoiceNumber,
		BookingID:      d.BookingID,
		PaymentID:      d.PaymentID,
		UserID:         d.UserID,
		EventID:        d.EventID,
		Status:         d.Status,
		Currency:       d.Currency,
		Subtotal:       d.Subtotal,
		PlatformFee:    d.PlatformFee,
		TaxAmount:      d.TaxAmount,
		DiscountAmount: d.DiscountAmount,
		TotalAmount:    d.TotalAmount,
		PaidAmount:     d.PaidAmount,
		CustomerName:   d.CustomerName,
		CustomerEmail:  d.CustomerEmail,
		CustomerPhone:  d.CustomerPhone,
		EventTitle:     d.EventTitle,
		EventDate:      d.EventDate,
		EventTimeStr:   d.EventTimeStr,
		VenueName:      d.VenueName,
		VenueAddress:   d.VenueAddress,
		IsOnline:       d.IsOnline,
		OnlineDetails:  d.OnlineDetails,
		IssuerName:     d.IssuerName,
		OrganizerName:  d.OrganizerName,
		IssuedAt:       d.IssuedAt,
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
	}
}

func invoiceModelToDomain(m *models.InvoiceModel) *domain.Invoice {
	return &domain.Invoice{
		ID:             m.ID,
		InvoiceNumber:  m.InvoiceNumber,
		BookingID:      m.BookingID,
		PaymentID:      m.PaymentID,
		UserID:         m.UserID,
		EventID:        m.EventID,
		Status:         m.Status,
		Currency:       m.Currency,
		Subtotal:       m.Subtotal,
		PlatformFee:    m.PlatformFee,
		TaxAmount:      m.TaxAmount,
		DiscountAmount: m.DiscountAmount,
		TotalAmount:    m.TotalAmount,
		PaidAmount:     m.PaidAmount,
		CustomerName:   m.CustomerName,
		CustomerEmail:  m.CustomerEmail,
		CustomerPhone:  m.CustomerPhone,
		EventTitle:     m.EventTitle,
		EventDate:      m.EventDate,
		EventTimeStr:   m.EventTimeStr,
		VenueName:      m.VenueName,
		VenueAddress:   m.VenueAddress,
		IsOnline:       m.IsOnline,
		OnlineDetails:  m.OnlineDetails,
		IssuerName:     m.IssuerName,
		OrganizerName:  m.OrganizerName,
		IssuedAt:       m.IssuedAt,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

func invoiceItemModelToDomain(m *models.InvoiceItemModel) domain.InvoiceItem {
	return domain.InvoiceItem{
		ID:            m.ID,
		InvoiceID:     m.InvoiceID,
		BookingItemID: m.BookingItemID,
		CategoryName:  m.CategoryName,
		SeatLabel:     m.SeatLabel,
		Quantity:      m.Quantity,
		UnitPrice:     m.UnitPrice,
		TotalPrice:    m.TotalPrice,
		CreatedAt:     m.CreatedAt,
	}
}
