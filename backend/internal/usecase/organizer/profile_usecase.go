package organizer

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	appErrors "github.com/adhithyan443/EventHub/backend/internal/errors"
	"github.com/google/uuid"
)

type ProfileUsecase struct {
	organizerRepository   domain.OrganizerRepository
	profileRepository     domain.OrganizerProfileRepository
	addressRepository     domain.OrganizerAddressRepository
	bankAccountRepository domain.OrganizerBankAccountRepository
	logger                *slog.Logger
}

func NewProfileUsecase(
	organizerRepository domain.OrganizerRepository,
	profileRepository domain.OrganizerProfileRepository,
	addressRepository domain.OrganizerAddressRepository,
	bankAccountRepository domain.OrganizerBankAccountRepository,
	logger *slog.Logger,
) *ProfileUsecase {
	return &ProfileUsecase{
		organizerRepository:   organizerRepository,
		profileRepository:     profileRepository,
		addressRepository:     addressRepository,
		bankAccountRepository: bankAccountRepository,
		logger:                logger,
	}
}

func (u *ProfileUsecase) GetProfile(
	userID uuid.UUID,
) (*OrganizerProfileResponse, error) {
	organizer, err := u.organizerRepository.FindByUserID(userID)
	if err != nil {
		if errors.Is(err, domain.ErrOrganizerNotFound) {
			u.logger.Warn(
				"organizer_profile_access_denied",
				"user_id", userID,
				"reason", "organizer_not_found",
			)

			return nil, appErrors.NewForbiddenError(
				"organizer access required",
			)
		}

		u.logger.Error(
			"organizer_profile_organizer_lookup_failed",
			"user_id", userID,
			"error", err,
		)

		return nil, fmt.Errorf("failed to retrieve organizer")
	}

	if organizer.Status != "ACTIVE" {
		u.logger.Warn(
			"organizer_profile_access_denied",
			"user_id", userID,
			"organizer_id", organizer.ID,
			"reason", "organizer_not_active",
			"status", organizer.Status,
		)

		return nil, appErrors.NewForbiddenError(
			"organizer account is not active",
		)
	}

	profile, err := u.profileRepository.FindByOrganizerID(
		organizer.ID,
	)
	if err != nil {
		if errors.Is(err, domain.ErrOrganizerProfileNotFound) {
			u.logger.Warn(
				"organizer_profile_not_found",
				"user_id", userID,
				"organizer_id", organizer.ID,
			)

			return nil, appErrors.NewNotFoundError(
				"organizer profile not found",
			)
		}

		u.logger.Error(
			"organizer_profile_lookup_failed",
			"user_id", userID,
			"organizer_id", organizer.ID,
			"error", err,
		)

		return nil, fmt.Errorf("failed to retrieve organizer profile")
	}

	address, err := u.addressRepository.FindByOrganizerID(
		organizer.ID,
	)
	if err != nil && !errors.Is(err, domain.ErrOrganizerAddressNotFound) {
		u.logger.Error(
			"organizer_profile_address_lookup_failed",
			"user_id", userID,
			"organizer_id", organizer.ID,
			"error", err,
		)

		return nil, fmt.Errorf("failed to retrieve organizer address")
	}

	bankAccount, err := u.bankAccountRepository.FindByOrganizerID(
		organizer.ID,
	)
	if err != nil && !errors.Is(err, domain.ErrOrganizerBankAccountNotFound) {
		u.logger.Error(
			"organizer_profile_bank_account_lookup_failed",
			"user_id", userID,
			"organizer_id", organizer.ID,
			"error", err,
		)

		return nil, fmt.Errorf("failed to retrieve organizer bank account")
	}

	response := toOrganizerProfileResponse(
		organizer,
		profile,
		address,
		bankAccount,
	)

	u.logger.Info(
		"organizer_profile_retrieved",
		"user_id", userID,
		"organizer_id", organizer.ID,
	)

	return &response, nil
}
