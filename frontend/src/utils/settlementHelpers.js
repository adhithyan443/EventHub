/**
 * Settlement & payout eligibility helper functions for EventHub.
 */

/**
 * Maps backend event financial lifecycle enum strings to user-friendly display labels.
 *
 * @param {string} lifecycle - "UPCOMING", "ONGOING", "COMPLETED", "CANCELLED"
 * @returns {string} Formatted lifecycle label
 */
export const formatEventLifecycle = (lifecycle) => {
  if (!lifecycle || typeof lifecycle !== "string") return "—";
  const normalized = lifecycle.toUpperCase().trim();
  switch (normalized) {
    case "UPCOMING":
      return "Upcoming";
    case "ONGOING":
      return "Ongoing";
    case "COMPLETED":
      return "Completed";
    case "CANCELLED":
      return "Cancelled";
    default:
      // Capitalize first letter of arbitrary lifecycle string
      return normalized.charAt(0) + normalized.slice(1).toLowerCase();
  }
};

/**
 * Returns Tailwind CSS badge classes for an event lifecycle.
 *
 * @param {string} lifecycle
 * @returns {string} Tailwind class string
 */
export const getLifecycleBadgeStyle = (lifecycle) => {
  if (!lifecycle || typeof lifecycle !== "string") {
    return "bg-gray-100 text-gray-700 border border-gray-200";
  }
  const normalized = lifecycle.toUpperCase().trim();
  switch (normalized) {
    case "UPCOMING":
      return "bg-blue-50 text-blue-700 border border-blue-200";
    case "ONGOING":
      return "bg-purple-50 text-purple-700 border border-purple-200";
    case "COMPLETED":
      return "bg-emerald-50 text-emerald-700 border border-emerald-200";
    case "CANCELLED":
      return "bg-rose-50 text-rose-700 border border-rose-200";
    default:
      return "bg-gray-100 text-gray-700 border border-gray-200";
  }
};

/**
 * Maps backend eligibility reason codes to clear, user-friendly explanations.
 *
 * @param {string} reasonCode - Backend error or restriction code
 * @param {object} [balanceData={}] - Full balance projection data
 * @returns {string} Friendly explanation
 */
export const getEligibilityReasonMessage = (reasonCode, balanceData = {}) => {
  if (!reasonCode || typeof reasonCode !== "string") {
    return "Settlement requirement pending.";
  }

  const code = reasonCode.toUpperCase().trim();
  const pendingBal = parseFloat(balanceData?.pending_balance || 0);

  switch (code) {
    case "NO_AVAILABLE_BALANCE":
      if (pendingBal > 0) {
        return "Earnings from upcoming or ongoing events are held in pending reserves until conclusion.";
      }
      return "No settled balance is currently available for disbursement.";

    case "NO_ACTIVE_BANK_ACCOUNT":
      return "A verified bank account is required to disburse settlements. Please add your bank details in Organizer Settings.";

    case "ORGANIZER_ACCOUNT_NOT_APPROVED":
      return "Your organizer account is pending administrative verification. Payouts are restricted until approved.";

    case "IN_FLIGHT_REFUNDS_PENDING":
      return "Settlement is temporarily restricted while customer refund requests are actively processing.";

    case "EVENT_NOT_COMPLETED":
      return "Earnings remain in reserve and will become eligible for settlement after events conclude.";

    case "EVENT_CANCELLED":
      return "This event was cancelled. Funds are reserved for customer refund disbursements.";

    case "IN_FLIGHT_SETTLEMENT_EXISTS":
    case "IN_FLIGHT_SETTLEMENT_HOLD":
      return "A settlement payout request is currently processing for this account.";

    default:
      // Convert screaming snake case to readable sentence
      return code
        .replace(/_/g, " ")
        .toLowerCase()
        .replace(/^\w/, (c) => c.toUpperCase()) + ".";
  }
};

/**
 * Determines whether the Request Payout button should be disabled.
 *
 * @param {boolean} balanceLoading - True if balance data is currently fetching
 * @param {object|null} balanceData - Balance projection object
 * @returns {boolean} True if button must be disabled
 */
export const isPayoutButtonDisabled = (balanceLoading, balanceData) => {
  if (balanceLoading) return true;
  if (!balanceData) return true;

  const avail = parseFloat(balanceData.available_balance || 0);
  if (avail <= 0) return true;

  if (balanceData.has_active_bank_account === false) return true;
  if (balanceData.eligibility_status === "INELIGIBLE") return true;

  return false;
};

/**
 * Returns a contextual explanation / tooltip for why the payout button is disabled or enabled.
 *
 * @param {boolean} balanceLoading - True if balance data is currently fetching
 * @param {object|null} balanceData - Balance projection object
 * @returns {string} Explanatory tooltip string
 */
export const getPayoutDisabledReason = (balanceLoading, balanceData) => {
  if (balanceLoading) {
    return "Loading balance information...";
  }
  if (!balanceData) {
    return "Balance data unavailable";
  }

  const avail = parseFloat(balanceData.available_balance || 0);
  const pending = parseFloat(balanceData.pending_balance || 0);

  if (balanceData.has_active_bank_account === false) {
    return "A verified bank account is required to request payout";
  }

  if (avail <= 0) {
    if (pending > 0) {
      return "Funds are held in pending reserves until events conclude";
    }
    return "No available balance eligible for settlement";
  }

  if (balanceData.eligibility_status === "INELIGIBLE") {
    return "Account is currently ineligible for payout";
  }

  return "Request settlement payout for eligible balance";
};
