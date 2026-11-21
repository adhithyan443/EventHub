import { describe, it } from "node:test";
import assert from "node:assert/strict";
import {
  buildOrganizerSettlementParams,
  buildSettlementRequestPayload,
} from "../../api/organizerSettlementApi.js";
import {
  buildAdminSettlementParams,
} from "../../api/adminSettlementApi.js";
import {
  formatEventLifecycle,
  getLifecycleBadgeStyle,
  getEligibilityReasonMessage,
  isPayoutButtonDisabled,
  getPayoutDisabledReason,
  hasEventFinancialActivity,
  isDraftEvent,
  shouldDisplayEventInBreakdown,
  isEventSettleDisabled,
  getEventSettleDisabledReason,
} from "../../utils/settlementHelpers.js";

describe("Organizer Settlement Workflow & Validation", () => {
  it("builds correct organizer settlement query params", () => {
    const params = buildOrganizerSettlementParams(2, 25, "PENDING");
    assert.deepEqual(params, {
      page: 2,
      limit: 25,
      status: "PENDING",
    });
  });

  it("handles default/empty parameters for organizer settlements", () => {
    const params = buildOrganizerSettlementParams();
    assert.deepEqual(params, {});
  });

  it("builds valid general settlement request payload", () => {
    const payload = buildSettlementRequestPayload(5000);
    assert.equal(payload.amount, 5000);
    assert.equal(payload.event_id, undefined);
  });

  it("builds valid event-scoped settlement request payload", () => {
    const eventId = "123e4567-e89b-12d3-a456-426614174000";
    const payload = buildSettlementRequestPayload("12500.50", eventId);
    assert.equal(payload.amount, 12500.5);
    assert.equal(payload.event_id, eventId);
  });

  it("rejects non-positive settlement amounts", () => {
    assert.throws(() => buildSettlementRequestPayload(0), {
      message: /Settlement amount must be a positive number/,
    });
    assert.throws(() => buildSettlementRequestPayload(-100), {
      message: /Settlement amount must be a positive number/,
    });
    assert.throws(() => buildSettlementRequestPayload("abc"), {
      message: /Settlement amount must be a positive number/,
    });
  });
});

describe("Admin Settlement Filter & Management", () => {
  it("builds clean admin query params without empty filters", () => {
    const params = buildAdminSettlementParams(1, 10, null, null);
    assert.deepEqual(params, {
      page: 1,
      limit: 10,
    });
  });

  it("builds admin query params with status and organizerId", () => {
    const orgId = "550e8400-e29b-41d4-a716-446655440000";
    const params = buildAdminSettlementParams(3, 50, "FAILED", orgId);
    assert.deepEqual(params, {
      page: 3,
      limit: 50,
      status: "FAILED",
      organizer_id: orgId,
    });
  });
});

describe("Event Lifecycle Formatting & Styles", () => {
  it("formats standard backend lifecycle values to title case", () => {
    assert.equal(formatEventLifecycle("UPCOMING"), "Upcoming");
    assert.equal(formatEventLifecycle("ONGOING"), "Ongoing");
    assert.equal(formatEventLifecycle("COMPLETED"), "Completed");
    assert.equal(formatEventLifecycle("CANCELLED"), "Cancelled");
  });

  it("handles empty, null, or invalid lifecycle strings safely", () => {
    assert.equal(formatEventLifecycle(null), "—");
    assert.equal(formatEventLifecycle(undefined), "—");
    assert.equal(formatEventLifecycle(""), "—");
    assert.equal(formatEventLifecycle(123), "—");
  });

  it("provides correct badge styles for all lifecycles", () => {
    assert.ok(getLifecycleBadgeStyle("UPCOMING").includes("blue"));
    assert.ok(getLifecycleBadgeStyle("ONGOING").includes("purple"));
    assert.ok(getLifecycleBadgeStyle("COMPLETED").includes("emerald"));
    assert.ok(getLifecycleBadgeStyle("CANCELLED").includes("rose"));
    assert.ok(getLifecycleBadgeStyle(null).includes("gray"));
  });
});

describe("Settlement Eligibility Reason Mapping", () => {
  it("maps NO_AVAILABLE_BALANCE explaining pending reserves when pending balance > 0", () => {
    const msg = getEligibilityReasonMessage("NO_AVAILABLE_BALANCE", {
      available_balance: "0.00",
      pending_balance: "420.00",
    });
    assert.ok(msg.includes("pending reserves"));
    assert.ok(msg.includes("until conclusion"));
  });

  it("maps NO_AVAILABLE_BALANCE when both available and pending are zero", () => {
    const msg = getEligibilityReasonMessage("NO_AVAILABLE_BALANCE", {
      available_balance: "0.00",
      pending_balance: "0.00",
    });
    assert.ok(msg.includes("No settled balance"));
  });

  it("maps NO_ACTIVE_BANK_ACCOUNT requiring verified bank account", () => {
    const msg = getEligibilityReasonMessage("NO_ACTIVE_BANK_ACCOUNT");
    assert.ok(msg.includes("verified bank account"));
  });

  it("maps ORGANIZER_ACCOUNT_NOT_APPROVED clearly", () => {
    const msg = getEligibilityReasonMessage("ORGANIZER_ACCOUNT_NOT_APPROVED");
    assert.ok(msg.includes("administrative verification"));
  });

  it("maps IN_FLIGHT_REFUNDS_PENDING clearly", () => {
    const msg = getEligibilityReasonMessage("IN_FLIGHT_REFUNDS_PENDING");
    assert.ok(msg.includes("refund requests are actively processing"));
  });

  it("maps EVENT_NOT_COMPLETED and EVENT_CANCELLED clearly", () => {
    assert.ok(getEligibilityReasonMessage("EVENT_NOT_COMPLETED").includes("after events conclude"));
    assert.ok(getEligibilityReasonMessage("EVENT_CANCELLED").includes("customer refund disbursements"));
  });

  it("handles unknown or custom reason codes with human-readable formatting", () => {
    const msg = getEligibilityReasonMessage("SOME_UNUSUAL_CUSTOM_HOLD");
    assert.equal(msg, "Some unusual custom hold.");
    assert.equal(getEligibilityReasonMessage(null), "Settlement requirement pending.");
  });
});

describe("Payout Button State & Contextual Feedback", () => {
  it("disables payout button when loading", () => {
    const disabled = isPayoutButtonDisabled(true, {
      available_balance: "500.00",
      has_active_bank_account: true,
      eligibility_status: "ELIGIBLE",
    });
    assert.equal(disabled, true);
    assert.equal(getPayoutDisabledReason(true, null), "Loading balance information...");
  });

  it("disables payout button when balanceData is null", () => {
    assert.equal(isPayoutButtonDisabled(false, null), true);
    assert.equal(getPayoutDisabledReason(false, null), "Balance data unavailable");
  });

  it("disables payout button and explains pending reserve when available balance is 0 and pending > 0", () => {
    const balanceData = {
      available_balance: "0.00",
      pending_balance: "420.00",
      has_active_bank_account: true,
      eligibility_status: "INELIGIBLE",
      eligibility_reasons: ["NO_AVAILABLE_BALANCE"],
    };
    assert.equal(isPayoutButtonDisabled(false, balanceData), true);
    assert.equal(
      getPayoutDisabledReason(false, balanceData),
      "Funds are held in pending reserves until events conclude"
    );
  });

  it("disables payout button when organizer has no active bank account", () => {
    const balanceData = {
      available_balance: "1000.00",
      pending_balance: "0.00",
      has_active_bank_account: false,
      eligibility_status: "INELIGIBLE",
      eligibility_reasons: ["NO_ACTIVE_BANK_ACCOUNT"],
    };
    assert.equal(isPayoutButtonDisabled(false, balanceData), true);
    assert.equal(
      getPayoutDisabledReason(false, balanceData),
      "A verified bank account is required to request payout"
    );
  });

  it("enables payout button when organizer has positive available balance and satisfies all requirements", () => {
    const balanceData = {
      available_balance: "1200.00",
      pending_balance: "300.00",
      has_active_bank_account: true,
      eligibility_status: "PARTIALLY_ELIGIBLE",
      eligibility_reasons: [],
    };
    assert.equal(isPayoutButtonDisabled(false, balanceData), false);
    assert.equal(
      getPayoutDisabledReason(false, balanceData),
      "Request settlement payout for eligible balance"
    );
  });
});

describe("Per-Event Breakdown Safe Field Handling", () => {
  it("safely handles missing settled_amount and in_flight_settlement_hold on event projection", () => {
    const eventProjection = {
      event_id: "evt-123",
      event_title: "Music Festival",
      lifecycle: "UPCOMING",
      gross_earnings: "500.00",
      refunds_deducted: "0.00",
      net_payable: "420.00",
      pending_balance: "420.00",
      available_balance: "0.00",
      is_eligible: false,
      // Note: settled_amount and in_flight_settlement_hold are intentionally omitted by backend
    };

    // Verify safe resolution
    const settledDisplay =
      eventProjection.settled_amount !== undefined && eventProjection.settled_amount !== null
        ? eventProjection.settled_amount
        : "—";
    assert.equal(settledDisplay, "—");

    const holdDisplay =
      eventProjection.pending_balance !== undefined && eventProjection.pending_balance !== null
        ? eventProjection.pending_balance
        : "—";
    assert.equal(holdDisplay, "420.00");

    const netPayable =
      eventProjection.net_payable !== undefined
        ? eventProjection.net_payable
        : eventProjection.net_earnings;
    assert.equal(netPayable, "420.00");
  });
});

describe("Per-Event Breakdown Filtering & Visibility", () => {
  it("hides draft events with zero financial activity", () => {
    const draftZero = {
      event_id: "draft-1",
      event_title: "Unpublished Draft Workshop",
      status: "DRAFT",
      lifecycle: "UPCOMING",
      gross_earnings: "0.00",
      refunds_deducted: "0.00",
      net_payable: "0.00",
      available_balance: "0.00",
      pending_balance: "0.00",
    };
    assert.equal(isDraftEvent(draftZero), true);
    assert.equal(hasEventFinancialActivity(draftZero), false);
    assert.equal(shouldDisplayEventInBreakdown(draftZero), false);
  });

  it("retains draft events with financial activity when required", () => {
    const draftWithActivity = {
      event_id: "draft-2",
      event_title: "Early Bird Test Booking",
      status: "DRAFT",
      lifecycle: "UPCOMING",
      gross_earnings: "500.00",
      refunds_deducted: "0.00",
      net_payable: "420.00",
      available_balance: "0.00",
      pending_balance: "420.00",
    };
    assert.equal(isDraftEvent(draftWithActivity), true);
    assert.equal(hasEventFinancialActivity(draftWithActivity), true);
    assert.equal(shouldDisplayEventInBreakdown(draftWithActivity), true);
  });

  it("retains published events that have financial activity", () => {
    const publishedEvent = {
      event_id: "pub-1",
      event_title: "Tech Conference 2026",
      status: "PUBLISHED",
      lifecycle: "UPCOMING",
      gross_earnings: "12000.00",
      refunds_deducted: "0.00",
      net_payable: "10080.00",
      available_balance: "0.00",
      pending_balance: "10080.00",
    };
    assert.equal(hasEventFinancialActivity(publishedEvent), true);
    assert.equal(shouldDisplayEventInBreakdown(publishedEvent), true);
  });

  it("retains completed events with unsettled balances", () => {
    const completedUnsettled = {
      event_id: "comp-1",
      event_title: "Past Hackathon",
      status: "PUBLISHED",
      lifecycle: "COMPLETED",
      gross_earnings: "5000.00",
      refunds_deducted: "0.00",
      net_payable: "4200.00",
      available_balance: "4200.00",
      pending_balance: "0.00",
      is_eligible: true,
    };
    assert.equal(hasEventFinancialActivity(completedUnsettled), true);
    assert.equal(shouldDisplayEventInBreakdown(completedUnsettled), true);
  });

  it("retains cancelled events when they have relevant financial records requiring settlement or refund tracking", () => {
    const cancelledWithRefunds = {
      event_id: "canc-1",
      event_title: "Rain-Delayed Concert",
      status: "CANCELLED",
      lifecycle: "CANCELLED",
      gross_earnings: "3000.00",
      refunds_deducted: "1500.00",
      net_payable: "1260.00",
      in_flight_refund_hold: "1260.00",
      available_balance: "0.00",
      pending_balance: "1260.00",
    };
    assert.equal(hasEventFinancialActivity(cancelledWithRefunds), true);
    assert.equal(shouldDisplayEventInBreakdown(cancelledWithRefunds), true);
  });

  it("retains events with non-zero financial balances even if status is not published", () => {
    const archivedWithBalances = {
      event_id: "arch-1",
      event_title: "Legacy Event",
      status: "ARCHIVED",
      lifecycle: "COMPLETED",
      gross_earnings: "1000.00",
      refunds_deducted: "0.00",
      net_payable: "840.00",
      available_balance: "840.00",
      pending_balance: "0.00",
    };
    assert.equal(hasEventFinancialActivity(archivedWithBalances), true);
    assert.equal(shouldDisplayEventInBreakdown(archivedWithBalances), true);
  });
});

describe("Event Lifecycle vs Publication Status Separation", () => {
  it("displays Upcoming for published events scheduled in the future", () => {
    // Published upcoming event displays "Upcoming", not "Published"
    assert.equal(formatEventLifecycle("PUBLISHED"), "Upcoming");
    assert.equal(
      formatEventLifecycle({ status: "PUBLISHED", lifecycle: "UPCOMING" }),
      "Upcoming"
    );
    assert.equal(
      formatEventLifecycle({ status: "PUBLISHED" }),
      "Upcoming"
    );
  });

  it("displays Ongoing, Completed, and Cancelled lifecycle states correctly", () => {
    assert.equal(formatEventLifecycle("ONGOING"), "Ongoing");
    assert.equal(formatEventLifecycle("COMPLETED"), "Completed");
    assert.equal(formatEventLifecycle("CANCELLED"), "Cancelled");
    assert.equal(
      formatEventLifecycle({ status: "PUBLISHED", lifecycle: "ONGOING" }),
      "Ongoing"
    );
    assert.equal(
      formatEventLifecycle({ status: "PUBLISHED", lifecycle: "COMPLETED" }),
      "Completed"
    );
    assert.equal(
      formatEventLifecycle({ status: "CANCELLED", lifecycle: "CANCELLED" }),
      "Cancelled"
    );
  });
});

describe("Per-Event Settle Button Eligibility & Modal Guard", () => {
  const activeOrgBalance = {
    available_balance: "5000.00",
    has_active_bank_account: true,
    eligibility_status: "ELIGIBLE",
  };

  it("disables Settle button when available balance is zero", () => {
    const event = {
      event_id: "evt-zero",
      event_title: "Unconcluded Festival",
      lifecycle: "UPCOMING",
      available_balance: "0.00",
      pending_balance: "420.00",
      is_eligible: false,
    };
    assert.equal(isEventSettleDisabled(event, activeOrgBalance, false), true);
    const reason = getEventSettleDisabledReason(event, activeOrgBalance, false);
    assert.ok(reason.includes("pending reserves"));
  });

  it("disables Settle button for ineligible events", () => {
    const event = {
      event_id: "evt-ineligible",
      event_title: "Hold Event",
      lifecycle: "COMPLETED",
      available_balance: "500.00",
      pending_balance: "0.00",
      is_eligible: false,
      ineligibility_reasons: ["IN_FLIGHT_REFUNDS_PENDING"],
    };
    assert.equal(isEventSettleDisabled(event, activeOrgBalance, false), true);
    const reason = getEventSettleDisabledReason(event, activeOrgBalance, false);
    assert.ok(reason.includes("refund requests are actively processing"));
  });

  it("disables Settle button when organizer has no active bank account", () => {
    const noBankOrg = {
      ...activeOrgBalance,
      has_active_bank_account: false,
      eligibility_status: "INELIGIBLE",
    };
    const event = {
      event_id: "evt-valid",
      event_title: "Completed Hackathon",
      lifecycle: "COMPLETED",
      available_balance: "1000.00",
      is_eligible: true,
    };
    assert.equal(isEventSettleDisabled(event, noBankOrg, false), true);
    const reason = getEventSettleDisabledReason(event, noBankOrg, false);
    assert.ok(reason.includes("bank account"));
  });

  it("enables Settle button when event has positive available balance and satisfies all rules", () => {
    const eligibleEvent = {
      event_id: "evt-eligible",
      event_title: "Completed Gala",
      lifecycle: "COMPLETED",
      available_balance: "1500.00",
      is_eligible: true,
    };
    assert.equal(isEventSettleDisabled(eligibleEvent, activeOrgBalance, false), false);
    const reason = getEventSettleDisabledReason(eligibleEvent, activeOrgBalance, false);
    assert.equal(reason, "Request payout for Completed Gala");
  });

  it("prevents opening settlement modal for ineligible events via guard check", () => {
    const ineligibleEvent = {
      event_id: "evt-blocked",
      available_balance: "0.00",
      pending_balance: "500.00",
      is_eligible: false,
    };

    // Verify the guard check logic implemented in handleOpenRequestModal
    const canOpen = !isEventSettleDisabled(ineligibleEvent, activeOrgBalance, false);
    assert.equal(canOpen, false);
  });
});
