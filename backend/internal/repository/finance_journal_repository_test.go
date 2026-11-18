package repository_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"github.com/adhithyan443/EventHub/backend/internal/domain/finance"
	"github.com/adhithyan443/EventHub/backend/internal/repository"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFinanceJournalRepository_Scenarios(t *testing.T) {
	db := initTestDatabase(t)
	repo := repository.NewFinanceJournalRepository(db, slog.Default())
	accA, accB := setupAccounts(t, db)
	ctx := context.Background()

	amt100, err := finance.NewMoney(decimal.NewFromInt(100), finance.CurrencyINR)
	require.NoError(t, err)
	amt200, err := finance.NewMoney(decimal.NewFromInt(200), finance.CurrencyINR)
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Second)

	t.Run("Successful journal recording returns POSTED journal", func(t *testing.T) {
		jID := uuid.New()
		key := "succ_rec_" + uuid.New().String()
		d1, _ := finance.NewDebitEntry(uuid.New(), accA, amt100, nil, nil, now)
		c1, _ := finance.NewCreditEntry(uuid.New(), accB, amt100, nil, nil, now)

		j, err := finance.NewJournalTransaction(
			jID,
			finance.TransactionTypePayment,
			finance.FinancialEventTicketSaleCaptured,
			finance.SourceTypePayment,
			"src_1",
			key,
			finance.CurrencyINR,
			"Ticket purchase journal",
			nil,
			now,
		)
		require.NoError(t, err)
		require.NoError(t, j.AddEntry(d1))
		require.NoError(t, j.AddEntry(c1))
		assert.Equal(t, finance.PostingStatusDraft, j.Status())

		posted, err := repo.RecordJournal(ctx, j)
		require.NoError(t, err)
		require.NotNil(t, posted)
		assert.Equal(t, jID, posted.ID())
		assert.Equal(t, finance.PostingStatusPosted, posted.Status(), "Returned journal must be POSTED")
		assert.NotNil(t, posted.PostedAt(), "Returned journal must have non-nil PostedAt")
		assert.Equal(t, 2, len(posted.Entries()))
	})

	t.Run("Unbalanced journal rejection", func(t *testing.T) {
		d1, _ := finance.NewDebitEntry(uuid.New(), accA, amt100, nil, nil, now)
		c1, _ := finance.NewCreditEntry(uuid.New(), accB, amt200, nil, nil, now) // 100 != 200

		j, err := finance.NewJournalTransaction(
			uuid.New(),
			finance.TransactionTypePayment,
			finance.FinancialEventTicketSaleCaptured,
			finance.SourceTypePayment,
			"src_unbal",
			"key_unbal_"+uuid.New().String(),
			finance.CurrencyINR,
			"Unbalanced journal",
			nil,
			now,
		)
		require.NoError(t, err)
		require.NoError(t, j.AddEntry(d1))
		require.NoError(t, j.AddEntry(c1))

		_, err = repo.RecordJournal(ctx, j)
		require.Error(t, err)
		assert.True(t, errors.Is(err, finance.ErrUnbalancedJournal))
	})

	t.Run("Invalid journal rejection", func(t *testing.T) {
		// Nil journal
		_, err := repo.RecordJournal(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot record nil journal")

		// FindByID with nil UUID
		found, err := repo.FindByID(ctx, uuid.Nil)
		require.Error(t, err)
		assert.Nil(t, found)
		assert.True(t, errors.Is(err, finance.ErrInvalidJournalID))

		// FindByIdempotencyKey with empty string
		found, err = repo.FindByIdempotencyKey(ctx, "   ")
		require.Error(t, err)
		assert.Nil(t, found)
		assert.True(t, errors.Is(err, finance.ErrMissingIdempotencyKey))
	})

	t.Run("Idempotent retry with matching fingerprint returns existing journal", func(t *testing.T) {
		key := "idem_match_" + uuid.New().String()
		d1, _ := finance.NewDebitEntry(uuid.New(), accA, amt100, nil, nil, now)
		c1, _ := finance.NewCreditEntry(uuid.New(), accB, amt100, nil, nil, now)
		j := buildValidJournal(
			t,
			uuid.New(),
			finance.TransactionTypePayment,
			finance.FinancialEventTicketSaleCaptured,
			finance.SourceTypePayment,
			"src_match",
			key,
			finance.CurrencyINR,
			"Match journal",
			nil,
			now,
			d1,
			c1,
		)

		first, err := repo.RecordJournal(ctx, j)
		require.NoError(t, err)
		require.NotNil(t, first)

		// Second call with same payload and same key
		second, err := repo.RecordJournal(ctx, j)
		require.NoError(t, err)
		require.NotNil(t, second)
		assert.Equal(t, first.ID(), second.ID(), "Must return identical journal on idempotent replay")
		assert.Equal(t, first.Status(), second.Status())
	})

	t.Run("Idempotency conflict with different fingerprint", func(t *testing.T) {
		key := "idem_conflict_" + uuid.New().String()

		d1, _ := finance.NewDebitEntry(uuid.New(), accA, amt100, nil, nil, now)
		c1, _ := finance.NewCreditEntry(uuid.New(), accB, amt100, nil, nil, now)
		j1 := buildValidJournal(
			t,
			uuid.New(),
			finance.TransactionTypePayment,
			finance.FinancialEventTicketSaleCaptured,
			finance.SourceTypePayment,
			"src_conf",
			key,
			finance.CurrencyINR,
			"Conflict journal 1",
			nil,
			now,
			d1,
			c1,
		)

		res1, err := repo.RecordJournal(ctx, j1)
		require.NoError(t, err)
		require.NotNil(t, res1)

		// Different payload: amount is 200 instead of 100
		d2, _ := finance.NewDebitEntry(uuid.New(), accA, amt200, nil, nil, now)
		c2, _ := finance.NewCreditEntry(uuid.New(), accB, amt200, nil, nil, now)
		j2 := buildValidJournal(
			t,
			uuid.New(),
			finance.TransactionTypePayment,
			finance.FinancialEventTicketSaleCaptured,
			finance.SourceTypePayment,
			"src_conf",
			key,
			finance.CurrencyINR,
			"Conflict journal 2",
			nil,
			now,
			d2,
			c2,
		)

		res2, err := repo.RecordJournal(ctx, j2)
		require.Error(t, err)
		assert.Nil(t, res2)
		assert.True(t, errors.Is(err, domain.ErrIdempotencyConflict), "Must return ErrIdempotencyConflict")
	})

	t.Run("Journal retrieval by ID and idempotency key", func(t *testing.T) {
		key := "lookup_key_" + uuid.New().String()
		d1, _ := finance.NewDebitEntry(uuid.New(), accA, amt100, nil, nil, now)
		c1, _ := finance.NewCreditEntry(uuid.New(), accB, amt100, nil, nil, now)
		j := buildValidJournal(
			t,
			uuid.New(),
			finance.TransactionTypePayment,
			finance.FinancialEventTicketSaleCaptured,
			finance.SourceTypePayment,
			"src_lookup",
			key,
			finance.CurrencyINR,
			"Lookup journal",
			nil,
			now,
			d1,
			c1,
		)

		recorded, err := repo.RecordJournal(ctx, j)
		require.NoError(t, err)

		// Lookup by ID
		byID, err := repo.FindByID(ctx, recorded.ID())
		require.NoError(t, err)
		assert.Equal(t, recorded.ID(), byID.ID())
		assert.Equal(t, key, byID.IdempotencyKey())

		// Lookup by Idempotency Key
		byKey, err := repo.FindByIdempotencyKey(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, recorded.ID(), byKey.ID())

		// Non-existent lookups
		missingID, err := repo.FindByID(ctx, uuid.New())
		require.Error(t, err)
		assert.Nil(t, missingID)
		assert.True(t, errors.Is(err, domain.ErrFinanceJournalNotFound))

		missingKey, err := repo.FindByIdempotencyKey(ctx, "NON_EXISTENT_KEY")
		require.Error(t, err)
		assert.Nil(t, missingKey)
		assert.True(t, errors.Is(err, domain.ErrFinanceJournalNotFound))
	})

	t.Run("Successful reversal and duplicate reversal prevention", func(t *testing.T) {
		// 1. Post original journal
		origKey := "orig_for_rev_" + uuid.New().String()
		d1, _ := finance.NewDebitEntry(uuid.New(), accA, amt100, nil, nil, now)
		c1, _ := finance.NewCreditEntry(uuid.New(), accB, amt100, nil, nil, now)
		orig := buildValidJournal(
			t,
			uuid.New(),
			finance.TransactionTypePayment,
			finance.FinancialEventTicketSaleCaptured,
			finance.SourceTypePayment,
			"src_rev",
			origKey,
			finance.CurrencyINR,
			"Original journal to reverse",
			nil,
			now,
			d1,
			c1,
		)
		postedOrig, err := repo.RecordJournal(ctx, orig)
		require.NoError(t, err)
		assert.False(t, postedOrig.IsReversed())

		// 2. Create reversal journal via domain method
		revKey := "rev_exec_" + uuid.New().String()
		revJournal, err := postedOrig.CreateReversal(revKey, "Reversal for refund", now.Add(time.Minute))
		require.NoError(t, err)

		postedRev, err := repo.RecordJournal(ctx, revJournal)
		require.NoError(t, err)
		require.NotNil(t, postedRev)
		assert.Equal(t, finance.PostingStatusPosted, postedRev.Status())
		assert.Equal(t, finance.TransactionTypeReversal, postedRev.TransactionType())

		// 3. Verify original journal in DB is now marked reversed
		reloadedOrig, err := repo.FindByID(ctx, postedOrig.ID())
		require.NoError(t, err)
		assert.True(t, reloadedOrig.IsReversed(), "Original journal must be marked reversed")
		require.NotNil(t, reloadedOrig.ReversedByJournalID())
		assert.Equal(t, postedRev.ID(), *reloadedOrig.ReversedByJournalID())

		// 4. Duplicate reversal attempt must be rejected
		dupRevKey := "rev_dup_" + uuid.New().String()
		dupRevJournal, err := postedOrig.CreateReversal(dupRevKey, "Duplicate reversal", now.Add(2*time.Minute))
		if err == nil {
			_, err = repo.RecordJournal(ctx, dupRevJournal)
		}
		require.Error(t, err)
		assert.True(t, errors.Is(err, finance.ErrJournalAlreadyReversed))
	})

	t.Run("Reversal mismatch rejection", func(t *testing.T) {
		origKey := "orig_mismatch_" + uuid.New().String()
		d1, _ := finance.NewDebitEntry(uuid.New(), accA, amt100, nil, nil, now)
		c1, _ := finance.NewCreditEntry(uuid.New(), accB, amt100, nil, nil, now)
		orig := buildValidJournal(
			t,
			uuid.New(),
			finance.TransactionTypePayment,
			finance.FinancialEventTicketSaleCaptured,
			finance.SourceTypePayment,
			"src_mismatch",
			origKey,
			finance.CurrencyINR,
			"Original for mismatch",
			nil,
			now,
			d1,
			c1,
		)
		postedOrig, err := repo.RecordJournal(ctx, orig)
		require.NoError(t, err)

		// Create a manual reversal journal with mismatched entries (different amount)
		origID := postedOrig.ID()
		revD1, _ := finance.NewDebitEntry(uuid.New(), accB, amt200, nil, nil, now)  // 200 instead of 100
		revC1, _ := finance.NewCreditEntry(uuid.New(), accA, amt200, nil, nil, now) // 200 instead of 100
		badRev, err := finance.NewJournalTransaction(
			uuid.New(),
			finance.TransactionTypeReversal,
			finance.FinancialEventJournalReversed,
			finance.SourceTypeRefund,
			"src_mismatch_rev",
			"bad_rev_key_"+uuid.New().String(),
			finance.CurrencyINR,
			"Bad reversal",
			&origID,
			now.Add(time.Minute),
		)
		require.NoError(t, err)
		require.NoError(t, badRev.AddEntry(revD1))
		require.NoError(t, badRev.AddEntry(revC1))

		_, err = repo.RecordJournal(ctx, badRev)
		require.Error(t, err)
		assert.True(t, errors.Is(err, finance.ErrReversalEntriesMismatch))
	})

	t.Run("Atomic rollback when entry persistence fails", func(t *testing.T) {
		badAccID := uuid.New() // Non-existent account ID will trigger FK constraint error
		d1, _ := finance.NewDebitEntry(uuid.New(), accA, amt100, nil, nil, now)
		badC1, _ := finance.NewCreditEntry(uuid.New(), badAccID, amt100, nil, nil, now)

		failKey := "rollback_test_" + uuid.New().String()
		j := buildValidJournal(
			t,
			uuid.New(),
			finance.TransactionTypePayment,
			finance.FinancialEventTicketSaleCaptured,
			finance.SourceTypePayment,
			"src_fail",
			failKey,
			finance.CurrencyINR,
			"Will fail on entry FK",
			nil,
			now,
			d1,
			badC1,
		)

		res, err := repo.RecordJournal(ctx, j)
		require.Error(t, err)
		assert.Nil(t, res)

		// Verify zero rows persisted
		var jCount int64
		db.Table("finance_journals").Where("id = ?", j.ID()).Count(&jCount)
		assert.Equal(t, int64(0), jCount, "Journal header must be rolled back")

		var kCount int64
		db.Table("finance_idempotency_keys").Where("idempotency_key = ?", failKey).Count(&kCount)
		assert.Equal(t, int64(0), kCount, "Idempotency key must be rolled back")
	})

	t.Run("Concurrent requests with identical idempotency key", func(t *testing.T) {
		key := "conc_idem_suite_" + uuid.New().String()
		d1, _ := finance.NewDebitEntry(uuid.New(), accA, amt100, nil, nil, now)
		c1, _ := finance.NewCreditEntry(uuid.New(), accB, amt100, nil, nil, now)
		j := buildValidJournal(
			t,
			uuid.New(),
			finance.TransactionTypePayment,
			finance.FinancialEventTicketSaleCaptured,
			finance.SourceTypePayment,
			"src_conc",
			key,
			finance.CurrencyINR,
			"Concurrent journal",
			nil,
			now,
			d1,
			c1,
		)

		concurrency := 10
		var wg sync.WaitGroup
		errs := make([]error, concurrency)
		results := make([]*finance.JournalTransaction, concurrency)

		wg.Add(concurrency)
		for i := 0; i < concurrency; i++ {
			go func(idx int) {
				defer wg.Done()
				results[idx], errs[idx] = repo.RecordJournal(ctx, j)
			}(i)
		}
		wg.Wait()

		for i := 0; i < concurrency; i++ {
			require.NoError(t, errs[i], "Goroutine %d must succeed", i)
			require.NotNil(t, results[i])
			assert.Equal(t, j.ID(), results[i].ID())
			assert.Equal(t, finance.PostingStatusPosted, results[i].Status())
		}
	})
}
