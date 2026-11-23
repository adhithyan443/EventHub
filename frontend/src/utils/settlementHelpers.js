/**
 * Settlement & payout eligibility helper functions for EventHub.
 */

/**
 * Maps backend event financial lifecycle enum strings to user-friendly display labels.
 * Publication status and event lifecycle remain separate concepts:
 * a published event scheduled for the future displays "Upcoming", not "Published".
 *
 * @param {string|object} lifecycle - Lifecycle string or event object
 * @returns {string} Formatted lifecycle label
 */
export const formatEventLifecycle = (lifecycle) => {
  if (!lifecycle) return "—";

  let val = lifecycle;
  if (typeof lifecycle === "object") {
    val = lifecycle.lifecycle || lifecycle.financial_lifecycle;
    if (!val && lifecycle.status) {
      val = lifecycle.status;
    }
  }

  if (!val || typeof val !== "string") return "—";
  const normalized = val.toUpperCase().trim();

  switch (normalized) {
    case "UPCOMING":
    case "PUBLISHED":
      // Publication status and event lifecycle remain separate; future published events show "Upcoming"
      return "Upcoming";
    case "ONGOING":
      return "Ongoing";
    case "COMPLETED":
      return "Completed";
    case "CANCELLED":
      return "Cancelled";
    case "DRAFT":
      return "Draft";
    default:
      return normalized.charAt(0) + normalized.slice(1).toLowerCase();
  }
};

/**
 * Returns Tailwind CSS badge classes for an event lifecycle.
 *
 * @param {string|object} lifecycle
 * @returns {string} Tailwind class string
 */
export const getLifecycleBadgeStyle = (lifecycle) => {
  if (!lifecycle) {
    return "bg-gray-100 text-gray-700 border border-gray-200";
  }

  let val = lifecycle;
  if (typeof lifecycle === "object") {
    val = lifecycle.lifecycle || lifecycle.financial_lifecycle;
    if (!val && lifecycle.status) {
      val = lifecycle.status;
    }
  }

  if (!val || typeof val !== "string") {
    return "bg-gray-100 text-gray-700 border border-gray-200";
  }

  const normalized = val.toUpperCase().trim();
  switch (normalized) {
    case "UPCOMING":
    case "PUBLISHED":
      return "bg-blue-50 text-blue-700 border border-blue-200";
    case "ONGOING":
      return "bg-purple-50 text-purple-700 border border-purple-200";
    case "COMPLETED":
      return "bg-emerald-50 text-emerald-700 border border-emerald-200";
    case "CANCELLED":
      return "bg-rose-50 text-rose-700 border border-rose-200";
    case "DRAFT":
      return "bg-amber-50 text-amber-700 border border-amber-200";
    default:
      return "bg-gray-100 text-gray-700 border border-gray-200";
  }
};

/**
 * Checks whether an event has any recorded financial transactions or balances.
 *
 * @param {object} event - Event projection object
 * @returns {boolean} True if any financial metric is non-zero
 */
export const hasEventFinancialActivity = (event) => {
  if (!event || typeof event !== "object") return false;

  const gross = parseFloat(event.gross_earnings || 0);
  const refunds = parseFloat(event.refunds_deducted || 0);
  const net = parseFloat(
    event.net_payable !== undefined ? event.net_payable : event.net_earnings || 0
  );
  const avail = parseFloat(event.available_balance || 0);
  const pending = parseFloat(event.pending_balance || 0);
  const inFlightHold = parseFloat(
    event.in_flight_refund_hold !== undefined
      ? event.in_flight_refund_hold
      : event.in_flight_settlement_hold || 0
  );
  const settled = parseFloat(event.settled_amount || 0);

  return (
    gross !== 0 ||
    refunds !== 0 ||
    net !== 0 ||
    avail !== 0 ||
    pending !== 0 ||
    inFlightHold !== 0 ||
    settled !== 0
  );
};

/**
 * Checks whether an event is in draft state.
 *
 * @param {object} event - Event projection object
 * @returns {boolean} True if event is draft
 */
export const isDraftEvent = (event) => {
  if (!event || typeof event !== "object") return false;

  const status = typeof event.status === "string" ? event.status.toUpperCase().trim() : "";
  const lifecycle = typeof event.lifecycle === "string"
    ? event.lifecycle.toUpperCase().trim()
    : typeof event.financial_lifecycle === "string"
    ? event.financial_lifecycle.toUpperCase().trim()
    : "";

  return status === "DRAFT" || lifecycle === "DRAFT" || event.is_draft === true;
};

/**
 * Evaluates visibility for the Per-Event Balance Breakdown:
 * - Hides draft events that have no financial activity.
 * - Keeps published events that have financial activity.
 * - Keeps completed events with unsettled balances.
 * - Keeps cancelled events when they have relevant financial records.
 * - Keeps events with non-zero financial balances or transaction history even if not published.
 * - Do not hide events that have legitimate financial records simply because their lifecycle is not active.
 *
 * @param {object} event - Event balance projection
 * @returns {boolean} True if event should be displayed
 */
export const shouldDisplayEventInBreakdown = (event) => {
  if (!event || typeof event !== "object") return false;

  const hasActivity = hasEventFinancialActivity(event);

  // If the event has legitimate financial activity, ALWAYS retain it
  if (hasActivity) {
    return true;
  }

  // If it has NO financial activity and is marked as draft, hide it
  if (isDraftEvent(event)) {
    return false;
  }

  // If no status is specified (or empty) and zero financial activity, hide from breakdown
  const status = typeof event.status === "string" ? event.status.toUpperCase().trim() : "";
  if (!status || status === "DRAFT") {
    return false;
  }

  // If explicitly marked as PUBLISHED with 0 current activity, retain
  return status === "PUBLISHED";
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

/**
 * Determines whether an individual event's "Settle" action button must be disabled.
 *
 * @param {object|null} event - Event projection object
 * @param {object|null} balanceData - Organizer balance projection
 * @param {boolean} balanceLoading - Balance loading state
 * @returns {boolean} True if Settle button must be disabled
 */
export const isEventSettleDisabled = (event, balanceData, balanceLoading) => {
  if (balanceLoading) return true;
  if (!event || typeof event !== "object") return true;

  const avail = parseFloat(event.available_balance || 0);
  if (avail <= 0) return true;

  if (!event.is_eligible) return true;

  if (balanceData && balanceData.has_active_bank_account === false) return true;
  if (balanceData && balanceData.eligibility_status === "INELIGIBLE") return true;

  return false;
};

/**
 * Returns a clear explanation for why an event's Settle button is disabled or enabled.
 *
 * @param {object|null} event - Event projection object
 * @param {object|null} balanceData - Organizer balance projection
 * @param {boolean} balanceLoading - Balance loading state
 * @returns {string} Explanatory tooltip message
 */
export const getEventSettleDisabledReason = (event, balanceData, balanceLoading) => {
  if (balanceLoading) {
    return "Loading balance information...";
  }
  if (!event || typeof event !== "object") {
    return "Event information unavailable";
  }

  const avail = parseFloat(event.available_balance || 0);
  const pending = parseFloat(event.pending_balance || 0);

  if (balanceData && balanceData.has_active_bank_account === false) {
    return "A verified bank account is required before requesting payouts";
  }

  if (avail <= 0) {
    if (pending > 0) {
      return "Earnings are held in pending reserves until event concludes";
    }
    return "No available balance eligible for settlement";
  }

  if (!event.is_eligible) {
    if (event.ineligibility_reasons && event.ineligibility_reasons.length > 0) {
      return getEligibilityReasonMessage(event.ineligibility_reasons[0], balanceData);
    }
    return "Event is not eligible for settlement disbursement";
  }

  if (balanceData && balanceData.eligibility_status === "INELIGIBLE") {
    return "Account is currently ineligible for payout";
  }

  return `Request payout for ${event.event_title || "event"}`;
};
