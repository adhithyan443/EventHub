package finance

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
)

// AccountType represents the standard classification of a financial account.
type AccountType string

const (
	AccountTypeAsset     AccountType = "ASSET"
	AccountTypeLiability AccountType = "LIABILITY"
	AccountTypeRevenue   AccountType = "REVENUE"
	AccountTypeExpense   AccountType = "EXPENSE"
	AccountTypeEquity    AccountType = "EQUITY"
)

// NormalBalance represents whether an account typically increases with a debit or credit.
type NormalBalance string

const (
	NormalBalanceDebit  NormalBalance = "DEBIT"
	NormalBalanceCredit NormalBalance = "CREDIT"
)

// Standard Chart of Accounts Control Codes for EventHub.
//
// EventHub Business Model:
// - Razorpay Clearing (1010, Asset): Escrows incoming customer ticket & fee payments.
// - Operating Bank (1020, Asset): EventHub's primary bank account for organizer settlements.
// - Organizer Payable (2010, Liability): Organizers' net ticket revenue, unsettled until event completion.
// - Customer Refund Payable (2020, Liability): Pending refunds owed back to customers on cancellation.
// - Retained Earnings (3010, Equity): Cumulative retained net earnings.
// - Platform Fee Revenue (4010, Revenue): Platform fees collected from customers at booking.
// - Payment Gateway Expense (5010, Expense): Gateway fees charged by Razorpay for payment processing.
const (
	AccountCodeRazorpayClearing      = "1010"
	AccountCodeOperatingBank         = "1020"
	AccountCodeOrganizerPayable      = "2010"
	AccountCodeCustomerRefundPayable = "2020"
	AccountCodeRetainedEarnings      = "3010"
	AccountCodePlatformFeeRevenue    = "4010"
	AccountCodePaymentGatewayExpense = "5010"
)

// Account represents an immutable general ledger financial account in the chart of accounts.
type Account struct {
	id            uuid.UUID
	code          string
	name          string
	accountType   AccountType
	normalBalance NormalBalance
	currency      string
	isActive      bool
}

// NewAccount constructs and validates a new financial account.
func NewAccount(
	id uuid.UUID,
	code string,
	name string,
	accountType AccountType,
	normalBalance NormalBalance,
	currency string,
	isActive bool,
) (*Account, error) {
	acc := &Account{
		id:            id,
		code:          strings.TrimSpace(code),
		name:          strings.TrimSpace(name),
		accountType:   AccountType(strings.ToUpper(strings.TrimSpace(string(accountType)))),
		normalBalance: NormalBalance(strings.ToUpper(strings.TrimSpace(string(normalBalance)))),
		currency:      strings.ToUpper(strings.TrimSpace(currency)),
		isActive:      isActive,
	}

	if err := acc.Validate(); err != nil {
		return nil, err
	}

	return acc, nil
}

// ID returns the unique UUID of the account.
func (a *Account) ID() uuid.UUID {
	return a.id
}

// Code returns the chart of accounts code (e.g. "1010").
func (a *Account) Code() string {
	return a.code
}

// Name returns the descriptive name of the account.
func (a *Account) Name() string {
	return a.name
}

// Type returns the account type (ASSET, LIABILITY, REVENUE, EXPENSE, EQUITY).
func (a *Account) Type() AccountType {
	return a.accountType
}

// NormalBalance returns the normal balance (DEBIT or CREDIT).
func (a *Account) NormalBalance() NormalBalance {
	return a.normalBalance
}

// Currency returns the account currency (strictly "INR" for v1).
func (a *Account) Currency() string {
	return a.currency
}

// IsActive returns whether the account is currently active.
func (a *Account) IsActive() bool {
	return a.isActive
}

// Validate verifies domain rules for a financial account.
func (a *Account) Validate() error {
	if a.id == uuid.Nil {
		return ErrInvalidAccountID
	}
	if a.code == "" {
		return ErrInvalidAccountCode
	}
	if a.name == "" {
		return ErrInvalidAccountName
	}
	if a.currency != CurrencyINR {
		return ErrUnsupportedCurrency
	}

	if !IsValidAccountType(a.accountType) {
		return ErrInvalidAccountType
	}

	if !IsValidNormalBalance(a.normalBalance) {
		return ErrInvalidNormalBalance
	}

	expectedBalance, err := ExpectedNormalBalance(a.accountType)
	if err != nil {
		return err
	}
	if a.normalBalance != expectedBalance {
		return ErrNormalBalanceMismatch
	}

	return nil
}

// IsValidAccountType checks whether an account type is supported.
func IsValidAccountType(t AccountType) bool {
	switch t {
	case AccountTypeAsset, AccountTypeLiability, AccountTypeRevenue, AccountTypeExpense, AccountTypeEquity:
		return true
	default:
		return false
	}
}

// IsValidNormalBalance checks whether a normal balance is supported.
func IsValidNormalBalance(b NormalBalance) bool {
	switch b {
	case NormalBalanceDebit, NormalBalanceCredit:
		return true
	default:
		return false
	}
}

// ExpectedNormalBalance returns the standard accounting normal balance for a given account type.
func ExpectedNormalBalance(t AccountType) (NormalBalance, error) {
	switch t {
	case AccountTypeAsset, AccountTypeExpense:
		return NormalBalanceDebit, nil
	case AccountTypeLiability, AccountTypeRevenue, AccountTypeEquity:
		return NormalBalanceCredit, nil
	default:
		return "", ErrInvalidAccountType
	}
}

// MarshalJSON provides JSON serialization for Account.
func (a *Account) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID            uuid.UUID     `json:"id"`
		Code          string        `json:"code"`
		Name          string        `json:"name"`
		Type          AccountType   `json:"type"`
		NormalBalance NormalBalance `json:"normal_balance"`
		Currency      string        `json:"currency"`
		IsActive      bool          `json:"is_active"`
	}{
		ID:            a.id,
		Code:          a.code,
		Name:          a.name,
		Type:          a.accountType,
		NormalBalance: a.normalBalance,
		Currency:      a.currency,
		IsActive:      a.isActive,
	})
}

// UnmarshalJSON provides JSON deserialization for Account.
func (a *Account) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID            uuid.UUID     `json:"id"`
		Code          string        `json:"code"`
		Name          string        `json:"name"`
		Type          AccountType   `json:"type"`
		NormalBalance NormalBalance `json:"normal_balance"`
		Currency      string        `json:"currency"`
		IsActive      bool          `json:"is_active"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	acc, err := NewAccount(aux.ID, aux.Code, aux.Name, aux.Type, aux.NormalBalance, aux.Currency, aux.IsActive)
	if err != nil {
		return err
	}

	*a = *acc
	return nil
}
