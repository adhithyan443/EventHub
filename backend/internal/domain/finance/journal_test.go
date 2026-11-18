package finance

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJournalTransaction_BalancedPosting(t *testing.T) {
	journalID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()
	now := time.Now()

	razorpayAccountID := uuid.New()
	organizerPayableAccountID := uuid.New()
	platformRevAccountID := uuid.New()

	t.Run("balanced journal posts successfully (1 debit, 2 credits)", func(t *testing.T) {
		// EventHub Double-Entry Booking Workflow:
		// Customer pays ₹1,000 for ticket.
		// Debit:  Razorpay Clearing ₹1,000 (Asset increases)
		// Credit: Organizer Payable ₹900 (Liability increases, tagged with organizer_id & event_id)
		// Credit: Platform Fee Revenue ₹100 (Revenue increases)
		j, err := NewJournalTransaction(
			journalID,
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			SourceTypePayment,
			uuid.New().String(),
			"idemp-pay-1001",
			CurrencyINR,
			"Ticket purchase payment captured",
			nil,
			now,
		)
		require.NoError(t, err)
		assert.Equal(t, PostingStatusDraft, j.Status())

		amt1000, _ := NewPositiveMoney(decimal.NewFromInt(1000), CurrencyINR)
		amt900, _ := NewPositiveMoney(decimal.NewFromInt(900), CurrencyINR)
		amt100, _ := NewPositiveMoney(decimal.NewFromInt(100), CurrencyINR)

		debitLeg, err := NewDebitEntry(uuid.New(), razorpayAccountID, amt1000, nil, &eventID, now)
		require.NoError(t, err)

		creditOrganizerLeg, err := NewCreditEntry(uuid.New(), organizerPayableAccountID, amt900, &organizerID, &eventID, now)
		require.NoError(t, err)

		creditPlatformLeg, err := NewCreditEntry(uuid.New(), platformRevAccountID, amt100, nil, nil, now)
		require.NoError(t, err)

		require.NoError(t, j.AddEntry(debitLeg))
		require.NoError(t, j.AddEntry(creditOrganizerLeg))
		require.NoError(t, j.AddEntry(creditPlatformLeg))

		balanced, err := j.IsBalanced()
		require.NoError(t, err)
		assert.True(t, balanced)

		totDebits, err := j.TotalDebits()
		require.NoError(t, err)
		assert.True(t, totDebits.Equal(amt1000))

		totCredits, err := j.TotalCredits()
		require.NoError(t, err)
		assert.True(t, totCredits.Equal(amt1000))

		// Post journal
		postTime := now.Add(time.Second)
		err = j.Post(postTime)
		require.NoError(t, err)
		assert.Equal(t, PostingStatusPosted, j.Status())
		require.NotNil(t, j.PostedAt())
		assert.Equal(t, postTime, *j.PostedAt())

		// Test subledger reporting helpers
		organizerEntries := j.FindEntriesByOrganizer(organizerID)
		assert.Len(t, organizerEntries, 1)
		assert.True(t, organizerEntries[0].Amount().Equal(amt900))

		eventEntries := j.FindEntriesByEvent(eventID)
		assert.Len(t, eventEntries, 2) // Debit leg and organizer credit leg
	})

	t.Run("unbalanced journal rejected upon posting", func(t *testing.T) {
		j, _ := NewJournalTransaction(
			uuid.New(),
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			SourceTypePayment,
			"src-1002",
			"idemp-pay-1002",
			CurrencyINR,
			"Unbalanced attempt",
			nil,
			now,
		)

		amt1000, _ := NewPositiveMoney(decimal.NewFromInt(1000), CurrencyINR)
		amt500, _ := NewPositiveMoney(decimal.NewFromInt(500), CurrencyINR)

		debitLeg, _ := NewDebitEntry(uuid.New(), razorpayAccountID, amt1000, nil, nil, now)
		creditLeg, _ := NewCreditEntry(uuid.New(), organizerPayableAccountID, amt500, nil, nil, now)

		require.NoError(t, j.AddEntry(debitLeg))
		require.NoError(t, j.AddEntry(creditLeg))

		balanced, err := j.IsBalanced()
		require.NoError(t, err)
		assert.False(t, balanced)

		err = j.Post(now)
		assert.ErrorIs(t, err, ErrUnbalancedJournal)
		assert.Equal(t, PostingStatusDraft, j.Status())
	})

	t.Run("multiple debits and multiple credits balance correctly", func(t *testing.T) {
		j, _ := NewJournalTransaction(
			uuid.New(),
			TransactionTypeAdjustment,
			FinancialEventSettlementAdjustment,
			SourceTypeSettlement,
			"src-settle-1",
			"idemp-settle-multi",
			CurrencyINR,
			"Complex settlement adjustment",
			nil,
			now,
		)

		m400, _ := NewPositiveMoney(decimal.NewFromInt(400), CurrencyINR)
		m600, _ := NewPositiveMoney(decimal.NewFromInt(600), CurrencyINR)
		m300, _ := NewPositiveMoney(decimal.NewFromInt(300), CurrencyINR)
		m700, _ := NewPositiveMoney(decimal.NewFromInt(700), CurrencyINR)

		d1, _ := NewDebitEntry(uuid.New(), uuid.New(), m400, nil, nil, now)
		d2, _ := NewDebitEntry(uuid.New(), uuid.New(), m600, nil, nil, now)
		c1, _ := NewCreditEntry(uuid.New(), uuid.New(), m300, nil, nil, now)
		c2, _ := NewCreditEntry(uuid.New(), uuid.New(), m700, nil, nil, now)

		require.NoError(t, j.AddEntry(d1))
		require.NoError(t, j.AddEntry(d2))
		require.NoError(t, j.AddEntry(c1))
		require.NoError(t, j.AddEntry(c2))

		// Debits: 400 + 600 = 1000. Credits: 300 + 700 = 1000.
		balanced, err := j.IsBalanced()
		require.NoError(t, err)
		assert.True(t, balanced)

		require.NoError(t, j.Post(now))
		assert.Equal(t, PostingStatusPosted, j.Status())
	})

	t.Run("missing credit leg rejected", func(t *testing.T) {
		j, _ := NewJournalTransaction(
			uuid.New(),
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			SourceTypePayment,
			"src-1003",
			"idemp-pay-1003",
			CurrencyINR,
			"Missing credit",
			nil,
			now,
		)

		m100, _ := NewPositiveMoney(decimal.NewFromInt(100), CurrencyINR)
		d1, _ := NewDebitEntry(uuid.New(), razorpayAccountID, m100, nil, nil, now)
		d2, _ := NewDebitEntry(uuid.New(), uuid.New(), m100, nil, nil, now)

		require.NoError(t, j.AddEntry(d1))
		require.NoError(t, j.AddEntry(d2))

		err := j.Post(now)
		assert.ErrorIs(t, err, ErrMissingCreditEntry)
	})

	t.Run("missing debit leg rejected", func(t *testing.T) {
		j, _ := NewJournalTransaction(
			uuid.New(),
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			SourceTypePayment,
			"src-1004",
			"idemp-pay-1004",
			CurrencyINR,
			"Missing debit",
			nil,
			now,
		)

		m100, _ := NewPositiveMoney(decimal.NewFromInt(100), CurrencyINR)
		c1, _ := NewCreditEntry(uuid.New(), organizerPayableAccountID, m100, nil, nil, now)
		c2, _ := NewCreditEntry(uuid.New(), platformRevAccountID, m100, nil, nil, now)

		require.NoError(t, j.AddEntry(c1))
		require.NoError(t, j.AddEntry(c2))

		err := j.Post(now)
		assert.ErrorIs(t, err, ErrMissingDebitEntry)
	})

	t.Run("insufficient entries rejected", func(t *testing.T) {
		j, _ := NewJournalTransaction(
			uuid.New(),
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			SourceTypePayment,
			"src-1005",
			"idemp-pay-1005",
			CurrencyINR,
			"Only one entry",
			nil,
			now,
		)

		m100, _ := NewPositiveMoney(decimal.NewFromInt(100), CurrencyINR)
		d1, _ := NewDebitEntry(uuid.New(), razorpayAccountID, m100, nil, nil, now)
		require.NoError(t, j.AddEntry(d1))

		err := j.Post(now)
		assert.ErrorIs(t, err, ErrInsufficientLedgerEntries)
	})

	t.Run("duplicate entry ID rejected when adding entry", func(t *testing.T) {
		j, _ := NewJournalTransaction(
			uuid.New(),
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			SourceTypePayment,
			"src-dup-add",
			"idemp-dup-add",
			CurrencyINR,
			"Duplicate entry check",
			nil,
			now,
		)

		sharedEntryID := uuid.New()
		m100, _ := NewPositiveMoney(decimal.NewFromInt(100), CurrencyINR)
		d1, _ := NewDebitEntry(sharedEntryID, razorpayAccountID, m100, nil, nil, now)
		d2, _ := NewDebitEntry(sharedEntryID, organizerPayableAccountID, m100, nil, nil, now)

		require.NoError(t, j.AddEntry(d1))
		err := j.AddEntry(d2)
		assert.ErrorIs(t, err, ErrDuplicateLedgerEntryID)
	})

	t.Run("TotalDebits, TotalCredits, and IsBalanced propagate arithmetic errors on currency mismatch", func(t *testing.T) {
		j, _ := NewJournalTransaction(
			uuid.New(),
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			SourceTypePayment,
			"src-arith-err",
			"idemp-arith-err",
			CurrencyINR,
			"Arithmetic error propagation test",
			nil,
			now,
		)

		// Create an entry with foreign currency (e.g. bypass NewLedgerEntry constructor)
		usdMoney := Money{amount: decimal.NewFromInt(100), currency: "USD"}
		usdDebitEntry := LedgerEntry{
			id:        uuid.New(),
			journalID: j.ID(),
			accountID: razorpayAccountID,
			entryType: EntryTypeDebit,
			amount:    usdMoney,
			currency:  "USD",
			createdAt: now,
		}
		usdCreditEntry := LedgerEntry{
			id:        uuid.New(),
			journalID: j.ID(),
			accountID: organizerPayableAccountID,
			entryType: EntryTypeCredit,
			amount:    usdMoney,
			currency:  "USD",
			createdAt: now,
		}

		j.entries = append(j.entries, usdDebitEntry, usdCreditEntry)

		_, errDebits := j.TotalDebits()
		assert.ErrorIs(t, errDebits, ErrCurrencyMismatch)

		_, errCredits := j.TotalCredits()
		assert.ErrorIs(t, errCredits, ErrCurrencyMismatch)

		_, errBalanced := j.IsBalanced()
		assert.ErrorIs(t, errBalanced, ErrCurrencyMismatch)
	})
}

func TestJournalTransaction_Immutability(t *testing.T) {
	now := time.Now()
	j, _ := NewJournalTransaction(
		uuid.New(),
		TransactionTypePayment,
		FinancialEventTicketSaleCaptured,
		SourceTypePayment,
		"src-imm-1",
		"idemp-imm-1",
		CurrencyINR,
		"Immutability test",
		nil,
		now,
	)

	m100, _ := NewPositiveMoney(decimal.NewFromInt(100), CurrencyINR)
	orgID := uuid.New()
	evID := uuid.New()
	d1, _ := NewDebitEntry(uuid.New(), uuid.New(), m100, &orgID, &evID, now)
	c1, _ := NewCreditEntry(uuid.New(), uuid.New(), m100, nil, nil, now)

	require.NoError(t, j.AddEntry(d1))
	require.NoError(t, j.AddEntry(c1))
	require.NoError(t, j.Post(now))

	t.Run("cannot add entry to posted journal", func(t *testing.T) {
		extraEntry, _ := NewDebitEntry(uuid.New(), uuid.New(), m100, nil, nil, now)
		err := j.AddEntry(extraEntry)
		assert.ErrorIs(t, err, ErrPostedJournalImmutable)
	})

	t.Run("cannot post already posted journal", func(t *testing.T) {
		err := j.Post(now)
		assert.ErrorIs(t, err, ErrJournalAlreadyPosted)
	})

	t.Run("mutating returned entries slice does not affect internal journal state", func(t *testing.T) {
		entries := j.Entries()
		require.Len(t, entries, 2)

		// Overwrite slice element
		m500, _ := NewPositiveMoney(decimal.NewFromInt(500), CurrencyINR)
		fakeEntry, _ := NewDebitEntry(uuid.New(), uuid.New(), m500, nil, nil, now)
		entries[0] = fakeEntry

		// Journal's internal entries must remain original
		freshEntries := j.Entries()
		assert.Equal(t, d1.ID(), freshEntries[0].ID())
		assert.True(t, freshEntries[0].Amount().Equal(m100))
	})

	t.Run("mutating returned entry subledger pointers does not affect internal journal state", func(t *testing.T) {
		entries := j.Entries()
		require.Len(t, entries, 2)

		orgPtr := entries[0].OrganizerID()
		require.NotNil(t, orgPtr)
		*orgPtr = uuid.New() // Attempt in-place mutation

		freshEntries := j.Entries()
		assert.Equal(t, orgID, *freshEntries[0].OrganizerID(), "internal entry organizer ID must remain unchanged")
	})
}

func TestJournalTransaction_ReversalLifecycle(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	originalID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()
	razorpayAccountID := uuid.New()
	organizerPayableAccountID := uuid.New()
	platformRevAccountID := uuid.New()

	amt1000, _ := NewPositiveMoney(decimal.NewFromInt(1000), CurrencyINR)
	amt900, _ := NewPositiveMoney(decimal.NewFromInt(900), CurrencyINR)
	amt100, _ := NewPositiveMoney(decimal.NewFromInt(100), CurrencyINR)

	j, _ := NewJournalTransaction(
		originalID,
		TransactionTypePayment,
		FinancialEventTicketSaleCaptured,
		SourceTypePayment,
		"payment-uuid-1",
		"idemp-orig-1",
		CurrencyINR,
		"Original ticket sale",
		nil,
		now,
	)

	d, _ := NewDebitEntry(uuid.New(), razorpayAccountID, amt1000, nil, &eventID, now)
	c1, _ := NewCreditEntry(uuid.New(), organizerPayableAccountID, amt900, &organizerID, &eventID, now)
	c2, _ := NewCreditEntry(uuid.New(), platformRevAccountID, amt100, nil, nil, now)

	_ = j.AddEntry(d)
	_ = j.AddEntry(c1)
	_ = j.AddEntry(c2)

	t.Run("cannot reverse unposted journal", func(t *testing.T) {
		_, err := j.CreateReversal("idemp-rev-1", "Reversal attempt", now)
		assert.ErrorIs(t, err, ErrCannotReverseUnpostedJournal)
	})

	require.NoError(t, j.Post(now))

	var reversal *JournalTransaction

	t.Run("CreateReversal creates draft reversal without prematurely marking original as reversed", func(t *testing.T) {
		revTime := now.Add(2 * time.Hour)
		var err error
		reversal, err = j.CreateReversal("idemp-rev-success", "Customer chargeback reversed", revTime)
		require.NoError(t, err)

		// Reversal journal created in DRAFT status
		assert.NotEqual(t, j.ID(), reversal.ID())
		assert.Equal(t, TransactionTypeReversal, reversal.TransactionType())
		assert.Equal(t, FinancialEventJournalReversed, reversal.FinancialEventType())
		assert.Equal(t, j.SourceType(), reversal.SourceType())
		assert.Equal(t, j.SourceID(), reversal.SourceID())
		assert.Equal(t, j.ID(), *reversal.ReversalReferenceID())
		assert.Equal(t, PostingStatusDraft, reversal.Status())
		assert.Len(t, reversal.Entries(), 3)

		// CRITICAL: The original journal must NOT be marked reversed yet!
		assert.False(t, j.IsReversed(), "original journal must not be marked reversed until reversal is posted and finalized")
		assert.Nil(t, j.ReversedByJournalID())
		assert.Nil(t, j.ReversedAt())
	})

	t.Run("MarkReversed rejects unposted reversal journal", func(t *testing.T) {
		// reversal is currently DRAFT
		assert.Equal(t, PostingStatusDraft, reversal.Status())
		err := j.MarkReversed(reversal)
		assert.ErrorIs(t, err, ErrReversalJournalNotPosted, "cannot mark reversed with an unposted reversal journal")
		assert.False(t, j.IsReversed())
	})

	t.Run("MarkReversed rejects nil reversal", func(t *testing.T) {
		err := j.MarkReversed(nil)
		assert.ErrorIs(t, err, ErrNilReversalJournal)
	})

	t.Run("MarkReversed rejects reversal with mismatched reference ID", func(t *testing.T) {
		otherJournalID := uuid.New()
		mismatchedReversal, err := NewJournalTransaction(
			uuid.New(),
			TransactionTypeReversal,
			FinancialEventJournalReversed,
			SourceTypePayment,
			"other-src",
			"idemp-other-rev",
			CurrencyINR,
			"Reversal of another journal",
			&otherJournalID,
			now,
		)
		require.NoError(t, err)
		m100, _ := NewPositiveMoney(decimal.NewFromInt(100), CurrencyINR)
		dEntry, _ := NewDebitEntry(uuid.New(), razorpayAccountID, m100, nil, nil, now)
		cEntry, _ := NewCreditEntry(uuid.New(), organizerPayableAccountID, m100, nil, nil, now)
		_ = mismatchedReversal.AddEntry(dEntry)
		_ = mismatchedReversal.AddEntry(cEntry)
		require.NoError(t, mismatchedReversal.Post(now))

		err = j.MarkReversed(mismatchedReversal)
		assert.ErrorIs(t, err, ErrReversalReferenceMismatch)
	})

	t.Run("successfully finalizing reversal state after posting reversal", func(t *testing.T) {
		revPostTime := now.Add(2 * time.Hour)
		require.NoError(t, reversal.Post(revPostTime))
		assert.Equal(t, PostingStatusPosted, reversal.Status())

		// Now finalize the reversal
		err := j.MarkReversed(reversal)
		require.NoError(t, err)

		// Verify original journal is now finalized as reversed
		assert.True(t, j.IsReversed())
		require.NotNil(t, j.ReversedByJournalID())
		assert.Equal(t, reversal.ID(), *j.ReversedByJournalID())
		require.NotNil(t, j.ReversedAt())
		assert.Equal(t, revPostTime, *j.ReversedAt())

		// Check inverted entries:
		// Original: Debit 1000 Razorpay. Reversal: Credit 1000 Razorpay.
		razorpayEntries := reversal.FindEntriesByAccount(razorpayAccountID)
		require.Len(t, razorpayEntries, 1)
		assert.Equal(t, EntryTypeCredit, razorpayEntries[0].EntryType())
		assert.True(t, razorpayEntries[0].Amount().Equal(amt1000))

		// Original: Credit 900 Organizer. Reversal: Debit 900 Organizer.
		organizerEntries := reversal.FindEntriesByOrganizer(organizerID)
		require.Len(t, organizerEntries, 1)
		assert.Equal(t, EntryTypeDebit, organizerEntries[0].EntryType())
		assert.True(t, organizerEntries[0].Amount().Equal(amt900))

		// Original: Credit 100 Platform. Reversal: Debit 100 Platform.
		platformEntries := reversal.FindEntriesByAccount(platformRevAccountID)
		require.Len(t, platformEntries, 1)
		assert.Equal(t, EntryTypeDebit, platformEntries[0].EntryType())
		assert.True(t, platformEntries[0].Amount().Equal(amt100))

		// A reversal journal itself cannot be reversed
		_, errRevReversal := reversal.CreateReversal("idemp-rev-rev", "Cannot reverse a reversal", revPostTime)
		assert.ErrorIs(t, errRevReversal, ErrCannotReverseReversalJournal)
	})

	t.Run("duplicate reversal attempts rejected after finalization", func(t *testing.T) {
		// j is already reversed
		assert.True(t, j.IsReversed())

		// CreateReversal fails
		_, errCreate := j.CreateReversal("idemp-rev-dup", "Duplicate reversal", now)
		assert.ErrorIs(t, errCreate, ErrJournalAlreadyReversed)

		// MarkReversed fails
		errMark := j.MarkReversed(reversal)
		assert.ErrorIs(t, errMark, ErrJournalAlreadyReversed)
	})

	t.Run("missing reversal idempotency key rejected", func(t *testing.T) {
		j2, _ := NewJournalTransaction(
			uuid.New(),
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			SourceTypePayment,
			"payment-uuid-2",
			"idemp-orig-2",
			CurrencyINR,
			"Original ticket sale 2",
			nil,
			now,
		)
		d2, _ := NewDebitEntry(uuid.New(), razorpayAccountID, amt1000, nil, nil, now)
		c2, _ := NewCreditEntry(uuid.New(), organizerPayableAccountID, amt1000, nil, nil, now)
		_ = j2.AddEntry(d2)
		_ = j2.AddEntry(c2)
		require.NoError(t, j2.Post(now))

		_, err := j2.CreateReversal("", "Reversal without key", now)
		assert.ErrorIs(t, err, ErrMissingIdempotencyKey)
	})

	t.Run("missing reversal description rejected", func(t *testing.T) {
		j3, _ := NewJournalTransaction(
			uuid.New(),
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			SourceTypePayment,
			"payment-uuid-3",
			"idemp-orig-3",
			CurrencyINR,
			"Original ticket sale 3",
			nil,
			now,
		)
		d3, _ := NewDebitEntry(uuid.New(), razorpayAccountID, amt1000, nil, nil, now)
		c3, _ := NewCreditEntry(uuid.New(), organizerPayableAccountID, amt1000, nil, nil, now)
		_ = j3.AddEntry(d3)
		_ = j3.AddEntry(c3)
		require.NoError(t, j3.Post(now))

		_, err := j3.CreateReversal("idemp-key-x", "", now)
		assert.ErrorIs(t, err, ErrMissingDescription)
	})
}

func TestJournalTransaction_Validation(t *testing.T) {
	now := time.Now()

	t.Run("missing source type rejected", func(t *testing.T) {
		_, err := NewJournalTransaction(
			uuid.New(),
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			"",
			"src-1",
			"key-1",
			CurrencyINR,
			"desc",
			nil,
			now,
		)
		assert.ErrorIs(t, err, ErrMissingSourceType)
	})

	t.Run("missing source id rejected", func(t *testing.T) {
		_, err := NewJournalTransaction(
			uuid.New(),
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			SourceTypePayment,
			"",
			"key-1",
			CurrencyINR,
			"desc",
			nil,
			now,
		)
		assert.ErrorIs(t, err, ErrMissingSourceID)
	})

	t.Run("missing idempotency key rejected", func(t *testing.T) {
		_, err := NewJournalTransaction(
			uuid.New(),
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			SourceTypePayment,
			"src-1",
			"",
			CurrencyINR,
			"desc",
			nil,
			now,
		)
		assert.ErrorIs(t, err, ErrMissingIdempotencyKey)
	})

	t.Run("unsupported currency rejected", func(t *testing.T) {
		_, err := NewJournalTransaction(
			uuid.New(),
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			SourceTypePayment,
			"src-1",
			"key-1",
			"EUR",
			"desc",
			nil,
			now,
		)
		assert.ErrorIs(t, err, ErrUnsupportedCurrency)
	})

	t.Run("reversal transaction type without reversal reference rejected", func(t *testing.T) {
		_, err := NewJournalTransaction(
			uuid.New(),
			TransactionTypeReversal,
			FinancialEventJournalReversed,
			SourceTypePayment,
			"src-1",
			"key-1",
			CurrencyINR,
			"desc",
			nil,
			now,
		)
		assert.ErrorIs(t, err, ErrInvalidReversalReference)
	})

	t.Run("self-referencing reversal reference rejected", func(t *testing.T) {
		selfID := uuid.New()
		_, err := NewJournalTransaction(
			selfID,
			TransactionTypeReversal,
			FinancialEventJournalReversed,
			SourceTypePayment,
			"src-1",
			"key-1",
			CurrencyINR,
			"desc",
			&selfID,
			now,
		)
		assert.ErrorIs(t, err, ErrSelfReferencingJournal)
	})

	t.Run("non-reversal transaction with reversal reference rejected", func(t *testing.T) {
		refID := uuid.New()
		_, err := NewJournalTransaction(
			uuid.New(),
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			SourceTypePayment,
			"src-1",
			"key-1",
			CurrencyINR,
			"desc",
			&refID,
			now,
		)
		assert.ErrorIs(t, err, ErrUnexpectedReversalReference)
	})
}

func TestJournalTransaction_ReconstitutionInvariants(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	postTime := now.Add(time.Minute)
	revTime := now.Add(2 * time.Hour)

	jID := uuid.New()
	revID := uuid.New()
	acc1 := uuid.New()
	acc2 := uuid.New()

	m100, _ := NewPositiveMoney(decimal.NewFromInt(100), CurrencyINR)
	d, _ := NewDebitEntry(uuid.New(), acc1, m100, nil, nil, now)
	c, _ := NewCreditEntry(uuid.New(), acc2, m100, nil, nil, now)
	entries := []LedgerEntry{d.WithJournalID(jID), c.WithJournalID(jID)}

	t.Run("valid reconstitution of posted and reversed journal succeeds", func(t *testing.T) {
		reconstituted, err := ReconstituteJournalTransaction(
			jID,
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			SourceTypePayment,
			"src-reconstitute",
			"key-reconstitute",
			CurrencyINR,
			"Reconstituted payment journal",
			entries,
			nil,
			PostingStatusPosted,
			now,
			&postTime,
			true,
			&revID,
			&revTime,
		)
		require.NoError(t, err)
		assert.Equal(t, jID, reconstituted.ID())
		assert.Equal(t, PostingStatusPosted, reconstituted.Status())
		assert.True(t, reconstituted.IsReversed())
		assert.Equal(t, revID, *reconstituted.ReversedByJournalID())
		assert.Equal(t, revTime, *reconstituted.ReversedAt())
		assert.Len(t, reconstituted.Entries(), 2)
	})

	t.Run("draft journal with postedAt rejected", func(t *testing.T) {
		_, err := ReconstituteJournalTransaction(
			jID,
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			SourceTypePayment,
			"src-draft-posted",
			"key-draft-posted",
			CurrencyINR,
			"Draft with postedAt",
			entries,
			nil,
			PostingStatusDraft,
			now,
			&postTime, // invalid for draft
			false,
			nil,
			nil,
		)
		assert.ErrorIs(t, err, ErrDraftCannotHavePostedAt)
	})

	t.Run("draft journal marked as reversed rejected", func(t *testing.T) {
		_, err := ReconstituteJournalTransaction(
			jID,
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			SourceTypePayment,
			"src-draft-rev",
			"key-draft-rev",
			CurrencyINR,
			"Draft reversed",
			entries,
			nil,
			PostingStatusDraft,
			now,
			nil,
			true, // invalid for draft
			&revID,
			&revTime,
		)
		assert.ErrorIs(t, err, ErrDraftCannotBeReversed)
	})

	t.Run("posted journal with nil postedAt rejected", func(t *testing.T) {
		_, err := ReconstituteJournalTransaction(
			jID,
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			SourceTypePayment,
			"src-posted-nil",
			"key-posted-nil",
			CurrencyINR,
			"Posted without postedAt",
			entries,
			nil,
			PostingStatusPosted,
			now,
			nil, // invalid for posted
			false,
			nil,
			nil,
		)
		assert.ErrorIs(t, err, ErrMissingPostedAt)
	})

	t.Run("posted journal with postedAt before createdAt rejected", func(t *testing.T) {
		pastTime := now.Add(-1 * time.Hour)
		_, err := ReconstituteJournalTransaction(
			jID,
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			SourceTypePayment,
			"src-posted-past",
			"key-posted-past",
			CurrencyINR,
			"Posted in past",
			entries,
			nil,
			PostingStatusPosted,
			now,
			&pastTime,
			false,
			nil,
			nil,
		)
		assert.ErrorIs(t, err, ErrInvalidJournalPostingState)
	})

	t.Run("reversed journal with missing reversing journal ID rejected", func(t *testing.T) {
		_, err := ReconstituteJournalTransaction(
			jID,
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			SourceTypePayment,
			"src-rev-noid",
			"key-rev-noid",
			CurrencyINR,
			"Missing rev id",
			entries,
			nil,
			PostingStatusPosted,
			now,
			&postTime,
			true,
			nil, // missing
			&revTime,
		)
		assert.ErrorIs(t, err, ErrInconsistentReversalMetadata)
	})

	t.Run("reversed journal with missing reversed timestamp rejected", func(t *testing.T) {
		_, err := ReconstituteJournalTransaction(
			jID,
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			SourceTypePayment,
			"src-rev-notime",
			"key-rev-notime",
			CurrencyINR,
			"Missing rev time",
			entries,
			nil,
			PostingStatusPosted,
			now,
			&postTime,
			true,
			&revID,
			nil, // missing
		)
		assert.ErrorIs(t, err, ErrInconsistentReversalMetadata)
	})

	t.Run("reversed journal with reversedAt before postedAt rejected", func(t *testing.T) {
		reversedBeforePosted := postTime.Add(-1 * time.Minute)
		_, err := ReconstituteJournalTransaction(
			jID,
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			SourceTypePayment,
			"src-rev-before",
			"key-rev-before",
			CurrencyINR,
			"Reversed before posted",
			entries,
			nil,
			PostingStatusPosted,
			now,
			&postTime,
			true,
			&revID,
			&reversedBeforePosted,
		)
		assert.ErrorIs(t, err, ErrReversalTimestampBeforePosting)
	})

	t.Run("reversal journal cannot be marked as reversed", func(t *testing.T) {
		origRefID := uuid.New()
		_, err := ReconstituteJournalTransaction(
			jID,
			TransactionTypeReversal,
			FinancialEventJournalReversed,
			SourceTypePayment,
			"src-rev-marked-rev",
			"key-rev-marked-rev",
			CurrencyINR,
			"Reversal journal marked reversed",
			entries,
			&origRefID,
			PostingStatusPosted,
			now,
			&postTime,
			true, // reversal journal marked as reversed!
			&revID,
			&revTime,
		)
		assert.ErrorIs(t, err, ErrReversalCannotBeReversed)
	})

	t.Run("journal with duplicate entry IDs rejected upon validation", func(t *testing.T) {
		dupEntryID := uuid.New()
		d1, _ := NewDebitEntry(dupEntryID, acc1, m100, nil, nil, now)
		c1, _ := NewCreditEntry(dupEntryID, acc2, m100, nil, nil, now) // same ID!
		dupEntries := []LedgerEntry{d1.WithJournalID(jID), c1.WithJournalID(jID)}

		_, err := ReconstituteJournalTransaction(
			jID,
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			SourceTypePayment,
			"src-dup-entries",
			"key-dup-entries",
			CurrencyINR,
			"Duplicate entry IDs in reconstitution",
			dupEntries,
			nil,
			PostingStatusPosted,
			now,
			&postTime,
			false,
			nil,
			nil,
		)
		assert.ErrorIs(t, err, ErrDuplicateLedgerEntryID)
	})
}

func TestJournalTransaction_JSONRoundtrip(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	postTime := now.Add(time.Minute)

	jID := uuid.New()
	acc1 := uuid.New()
	acc2 := uuid.New()

	m100, _ := NewPositiveMoney(decimal.NewFromInt(100), CurrencyINR)
	d, _ := NewDebitEntry(uuid.New(), acc1, m100, nil, nil, now)
	c, _ := NewCreditEntry(uuid.New(), acc2, m100, nil, nil, now)

	j, err := NewJournalTransaction(
		jID,
		TransactionTypePayment,
		FinancialEventTicketSaleCaptured,
		SourceTypePayment,
		"src-json",
		"key-json",
		CurrencyINR,
		"JSON roundtrip test",
		nil,
		now,
	)
	require.NoError(t, err)
	require.NoError(t, j.AddEntry(d))
	require.NoError(t, j.AddEntry(c))
	require.NoError(t, j.Post(postTime))

	data, err := json.Marshal(j)
	require.NoError(t, err)

	var unmarshaled JournalTransaction
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, j.ID(), unmarshaled.ID())
	assert.Equal(t, j.TransactionType(), unmarshaled.TransactionType())
	assert.Equal(t, j.FinancialEventType(), unmarshaled.FinancialEventType())
	assert.Equal(t, j.SourceType(), unmarshaled.SourceType())
	assert.Equal(t, j.SourceID(), unmarshaled.SourceID())
	assert.Equal(t, j.IdempotencyKey(), unmarshaled.IdempotencyKey())
	assert.Equal(t, j.Currency(), unmarshaled.Currency())
	assert.Equal(t, j.Status(), unmarshaled.Status())
	assert.Equal(t, len(j.Entries()), len(unmarshaled.Entries()))
}

func TestMarkReversedRejectsMismatchedEntries(t *testing.T) {
	now := time.Now()

	organizerID := uuid.New()
	eventID := uuid.New()

	accountA := uuid.New()
	accountB := uuid.New()

	amount100 := func() Money {
		money, err := NewPositiveMoney(
			decimal.NewFromInt(100),
			CurrencyINR,
		)
		require.NoError(t, err)
		return money
	}

	newOriginal := func() *JournalTransaction {
		j, err := NewJournalTransaction(
			uuid.New(),
			TransactionTypePayment,
			FinancialEventTicketSaleCaptured,
			SourceTypePayment,
			"payment-123",
			"original-key",
			CurrencyINR,
			"Original payment",
			nil,
			now,
		)
		require.NoError(t, err)

		debit, err := NewDebitEntry(
			uuid.New(), accountA, amount100(),
			&organizerID, &eventID, now,
		)
		require.NoError(t, err)

		credit, err := NewCreditEntry(
			uuid.New(), accountB, amount100(),
			&organizerID, &eventID, now,
		)
		require.NoError(t, err)

		require.NoError(t, j.AddEntry(debit))
		require.NoError(t, j.AddEntry(credit))
		require.NoError(t, j.Post(now))

		return j
	}

	tests := []struct {
		name   string
		mutate func(*JournalTransaction, *JournalTransaction) error
	}{
		{
			name: "wrong account",
			mutate: func(original, reversal *JournalTransaction) error {
				reversal.entries[0].accountID = uuid.New()
				return nil
			},
		},
		{
			name: "wrong amount",
			mutate: func(original, reversal *JournalTransaction) error {
				reversal.entries[0].amount = amount100()
				// Replace with a different amount.
				money, err := NewPositiveMoney(
					decimal.NewFromInt(200),
					CurrencyINR,
				)
				if err != nil {
					return err
				}
				reversal.entries[0].amount = money
				return nil
			},
		},
		{
			name: "wrong entry type",
			mutate: func(original, reversal *JournalTransaction) error {
				reversal.entries[0].entryType = original.entries[0].entryType
				return nil
			},
		},
		{
			name: "different organizer",
			mutate: func(original, reversal *JournalTransaction) error {
				differentOrganizer := uuid.New()
				reversal.entries[0].organizerID = &differentOrganizer
				return nil
			},
		},
		{
			name: "different event",
			mutate: func(original, reversal *JournalTransaction) error {
				differentEvent := uuid.New()
				reversal.entries[0].eventID = &differentEvent
				return nil
			},
		},
		{
			name: "different currency",
			mutate: func(original, reversal *JournalTransaction) error {
				reversal.entries[0].currency = "USD"
				return nil
			},
		},
		{
			name: "missing entry",
			mutate: func(original, reversal *JournalTransaction) error {
				reversal.entries = reversal.entries[:1]
				return nil
			},
		},
		{
			name: "duplicate entry",
			mutate: func(original, reversal *JournalTransaction) error {
				reversal.entries[1] = reversal.entries[0].Clone()
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := newOriginal()

			reversal, err := original.CreateReversal(
				"reversal-"+tt.name,
				"Test reversal",
				now.Add(time.Minute),
			)
			require.NoError(t, err)

			require.NoError(t, tt.mutate(original, reversal))

			// Mark as posted for this domain validation test.
			reversal.status = PostingStatusPosted
			postedAt := now.Add(2 * time.Minute)
			reversal.postedAt = &postedAt

			err = original.MarkReversed(reversal)

			assert.ErrorIs(t, err, ErrReversalEntriesMismatch)
			assert.False(t, original.isReversed)
			assert.Nil(t, original.reversedByJournalID)
			assert.Nil(t, original.reversedAt)
		})
	}
}
