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

func TestNewLedgerEntry(t *testing.T) {
	entryID := uuid.New()
	journalID := uuid.New()
	accountID := uuid.New()
	organizerID := uuid.New()
	eventID := uuid.New()
	now := time.Now()

	amt100, _ := NewPositiveMoney(decimal.NewFromInt(100), CurrencyINR)

	t.Run("valid debit entry succeeds", func(t *testing.T) {
		entry, err := NewLedgerEntry(
			entryID,
			journalID,
			accountID,
			EntryTypeDebit,
			amt100,
			&organizerID,
			&eventID,
			now,
		)
		require.NoError(t, err)
		assert.Equal(t, entryID, entry.ID())
		assert.Equal(t, journalID, entry.JournalID())
		assert.Equal(t, accountID, entry.AccountID())
		assert.Equal(t, EntryTypeDebit, entry.EntryType())
		assert.True(t, entry.Amount().Equal(amt100))
		assert.Equal(t, CurrencyINR, entry.Currency())
		assert.Equal(t, &organizerID, entry.OrganizerID())
		assert.Equal(t, &eventID, entry.EventID())
		assert.Equal(t, now, entry.CreatedAt())
	})

	t.Run("valid credit entry succeeds without subledger ids", func(t *testing.T) {
		entry, err := NewLedgerEntry(
			entryID,
			journalID,
			accountID,
			EntryTypeCredit,
			amt100,
			nil,
			nil,
			now,
		)
		require.NoError(t, err)
		assert.Equal(t, EntryTypeCredit, entry.EntryType())
		assert.Nil(t, entry.OrganizerID())
		assert.Nil(t, entry.EventID())
	})

	t.Run("invalid entry type rejected", func(t *testing.T) {
		_, err := NewLedgerEntry(
			entryID,
			journalID,
			accountID,
			"INVALID_TYPE",
			amt100,
			nil,
			nil,
			now,
		)
		assert.ErrorIs(t, err, ErrInvalidEntryType)
	})

	t.Run("non-positive amount rejected", func(t *testing.T) {
		zeroMoney := ZeroINR()
		_, err := NewLedgerEntry(
			entryID,
			journalID,
			accountID,
			EntryTypeDebit,
			zeroMoney,
			nil,
			nil,
			now,
		)
		assert.ErrorIs(t, err, ErrNonPositiveMoney)
	})

	t.Run("nil entry ID rejected", func(t *testing.T) {
		_, err := NewLedgerEntry(
			uuid.Nil,
			journalID,
			accountID,
			EntryTypeDebit,
			amt100,
			nil,
			nil,
			now,
		)
		assert.ErrorIs(t, err, ErrInvalidEntryID)
	})

	t.Run("nil account ID rejected", func(t *testing.T) {
		_, err := NewLedgerEntry(
			entryID,
			journalID,
			uuid.Nil,
			EntryTypeDebit,
			amt100,
			nil,
			nil,
			now,
		)
		assert.ErrorIs(t, err, ErrInvalidAccountID)
	})

	t.Run("nil organizer uuid pointer rejected", func(t *testing.T) {
		nilUUID := uuid.Nil
		_, err := NewLedgerEntry(
			entryID,
			journalID,
			accountID,
			EntryTypeDebit,
			amt100,
			&nilUUID,
			nil,
			now,
		)
		assert.ErrorIs(t, err, ErrInvalidOrganizerID)
	})

	t.Run("nil event uuid pointer rejected", func(t *testing.T) {
		nilUUID := uuid.Nil
		_, err := NewLedgerEntry(
			entryID,
			journalID,
			accountID,
			EntryTypeDebit,
			amt100,
			nil,
			&nilUUID,
			now,
		)
		assert.ErrorIs(t, err, ErrInvalidEventID)
	})

	t.Run("zero creation timestamp rejected with ErrInvalidEntryTimestamp", func(t *testing.T) {
		_, err := NewLedgerEntry(
			entryID,
			journalID,
			accountID,
			EntryTypeDebit,
			amt100,
			nil,
			nil,
			time.Time{},
		)
		assert.ErrorIs(t, err, ErrInvalidEntryTimestamp)
	})

	t.Run("reverse entry creates mirror opposite leg", func(t *testing.T) {
		entry, _ := NewLedgerEntry(
			entryID,
			journalID,
			accountID,
			EntryTypeDebit,
			amt100,
			&organizerID,
			&eventID,
			now,
		)

		revID := uuid.New()
		revJournalID := uuid.New()
		revTime := now.Add(time.Hour)

		revEntry, err := entry.Reverse(revID, revJournalID, revTime)
		require.NoError(t, err)
		assert.Equal(t, revID, revEntry.ID())
		assert.Equal(t, revJournalID, revEntry.JournalID())
		assert.Equal(t, accountID, revEntry.AccountID())
		assert.Equal(t, EntryTypeCredit, revEntry.EntryType()) // Reversals invert DEBIT to CREDIT
		assert.True(t, revEntry.Amount().Equal(entry.Amount()))
		assert.Equal(t, &organizerID, revEntry.OrganizerID())
		assert.Equal(t, &eventID, revEntry.EventID())
	})

	t.Run("organizer and event uuid pointers are strictly encapsulated and cloned", func(t *testing.T) {
		mutOrgID := uuid.New()
		mutEventID := uuid.New()

		entry, err := NewLedgerEntry(
			entryID,
			journalID,
			accountID,
			EntryTypeDebit,
			amt100,
			&mutOrgID,
			&mutEventID,
			now,
		)
		require.NoError(t, err)

		originalOrgID := mutOrgID
		originalEventID := mutEventID

		// Mutate caller's original variables
		mutOrgID = uuid.New()
		mutEventID = uuid.New()
		assert.Equal(t, originalOrgID, *entry.OrganizerID(), "entry must not be affected by caller's variable mutations")
		assert.Equal(t, originalEventID, *entry.EventID(), "entry must not be affected by caller's variable mutations")

		// Mutate pointer returned by getter
		retOrgPtr := entry.OrganizerID()
		*retOrgPtr = uuid.New()
		assert.Equal(t, originalOrgID, *entry.OrganizerID(), "entry internal state must not be modified via getter pointer")

		retEventPtr := entry.EventID()
		*retEventPtr = uuid.New()
		assert.Equal(t, originalEventID, *entry.EventID(), "entry internal state must not be modified via getter pointer")
	})

	t.Run("json serialization and deserialization", func(t *testing.T) {
		entry, err := NewLedgerEntry(
			entryID,
			journalID,
			accountID,
			EntryTypeDebit,
			amt100,
			&organizerID,
			&eventID,
			now.Truncate(time.Millisecond),
		)
		require.NoError(t, err)

		data, err := json.Marshal(entry)
		require.NoError(t, err)

		var unmarshaled LedgerEntry
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, entry.ID(), unmarshaled.ID())
		assert.Equal(t, entry.JournalID(), unmarshaled.JournalID())
		assert.Equal(t, entry.AccountID(), unmarshaled.AccountID())
		assert.Equal(t, entry.EntryType(), unmarshaled.EntryType())
		assert.True(t, entry.Amount().Equal(unmarshaled.Amount()))
		assert.Equal(t, entry.Currency(), unmarshaled.Currency())
		assert.Equal(t, *entry.OrganizerID(), *unmarshaled.OrganizerID())
		assert.Equal(t, *entry.EventID(), *unmarshaled.EventID())
	})
}
