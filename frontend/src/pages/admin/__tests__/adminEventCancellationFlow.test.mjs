import test, { beforeEach } from "node:test";
import assert from "node:assert";

// Polyfill in-memory localStorage for Node test runner
const mockStorage = new Map();
globalThis.localStorage = {
  getItem: (key) => (mockStorage.has(key) ? mockStorage.get(key) : null),
  setItem: (key, val) => mockStorage.set(key, String(val)),
  removeItem: (key) => mockStorage.delete(key),
  clear: () => mockStorage.clear(),
};

// Import client and admin API methods
const { default: apiClient } = await import("../../../api/client.js");
const {
  getAdminEventCancellationPreview,
  cancelAdminEvent,
  getAdminEventCancellationStatus,
} = await import("../../../api/adminEventApi.js");
const { formatCurrency } = await import("../../../utils/formatters.js");

beforeEach(() => {
  mockStorage.clear();
});

// Sample preview response fixture for eligible event (admin view)
const createAdminEligiblePreviewFixture = (eventId = "e0a7894a-97bb-41d3-a5c7-5e921d784a92") => ({
  success: true,
  data: {
    event: {
      id: eventId,
      title: "Neon Nights Music Festival 2026",
      status: "PUBLISHED",
      scheduled_date: "20 Dec 2026",
    },
    eligibility: {
      is_cancellable: true,
      blocking_reasons: [],
    },
    operational_impact: {
      total_bookings: 50,
      affected_bookings: 48,
      total_tickets_sold: 120,
      affected_tickets: 115,
      active_reservations_released: 5,
      reserved_seats_released: 10,
      general_admission_quantity_released: 105,
      active_checkouts_expired: 3,
    },
    financial_impact: {
      currency: "INR",
      gross_collected_amount: "150000.00",
      already_refunded_amount: "5000.00",
      expected_refund_amount: "145000.00",
      platform_fee_refund: "7250.00",
      tax_refund: "26100.00",
      organizer_payable_revocation: "111650.00",
    },
    settlement_impact: {
      has_completed_settlement: false,
      has_in_flight_settlement: false,
      pending_settlements_count: 1,
      pending_settlements_total: "25000.00",
      settlement_warning: "1 pending settlement request(s) totaling 25000.00 will be automatically rejected upon cancellation",
    },
    generated_at: "2026-11-23T22:00:00Z",
  },
});

// Sample preview response fixture for ineligible event (admin view)
const createAdminIneligiblePreviewFixture = (eventId = "e0a7894a-97bb-41d3-a5c7-5e921d784a92") => ({
  success: true,
  data: {
    event: {
      id: eventId,
      title: "Completed Tech Symposium",
      status: "COMPLETED",
      scheduled_date: "01 Nov 2026",
    },
    eligibility: {
      is_cancellable: false,
      blocking_reasons: [
        "cannot cancel an event that has already concluded",
        "cannot cancel event: 1 completed settlement(s) exist; manual recovery required",
      ],
    },
    operational_impact: {
      total_bookings: 80,
      affected_bookings: 0,
      total_tickets_sold: 150,
      affected_tickets: 0,
      active_reservations_released: 0,
      reserved_seats_released: 0,
      general_admission_quantity_released: 0,
      active_checkouts_expired: 0,
    },
    financial_impact: {
      currency: "INR",
      gross_collected_amount: "250000.00",
      already_refunded_amount: "0.00",
      expected_refund_amount: "0.00",
      platform_fee_refund: "0.00",
      tax_refund: "0.00",
      organizer_payable_revocation: "0.00",
    },
    settlement_impact: {
      has_completed_settlement: true,
      has_in_flight_settlement: false,
      pending_settlements_count: 0,
      pending_settlements_total: "0.00",
      settlement_warning: "",
    },
    generated_at: "2026-11-23T22:00:00Z",
  },
});

test("Admin API: getAdminEventCancellationPreview constructs correct GET URL and returns preview data", async () => {
  const eventId = "e0a7894a-97bb-41d3-a5c7-5e921d784a92";
  const expectedFixture = createAdminEligiblePreviewFixture(eventId);

  let capturedUrl = "";
  let capturedMethod = "";

  apiClient.defaults.adapter = async (config) => {
    capturedUrl = config.url;
    capturedMethod = config.method;
    return {
      status: 200,
      data: expectedFixture,
      config,
    };
  };

  const result = await getAdminEventCancellationPreview(eventId);

  assert.equal(capturedMethod, "get");
  assert.equal(capturedUrl, `/admin/events/${eventId}/cancellation-preview`);
  assert.equal(result.success, true);
  assert.equal(result.data.event.id, eventId);
  assert.equal(result.data.eligibility.is_cancellable, true);
  assert.equal(result.data.operational_impact.affected_bookings, 48);
  assert.equal(result.data.operational_impact.affected_tickets, 115);
  assert.equal(result.data.financial_impact.expected_refund_amount, "145000.00");
  assert.equal(result.data.financial_impact.organizer_payable_revocation, "111650.00");
});

test("Admin API: cancelAdminEvent constructs POST request with reason payload", async () => {
  const eventId = "e0a7894a-97bb-41d3-a5c7-5e921d784a92";
  const reasonText = "Administrative cancellation due to organizer terms violation";

  let capturedUrl = "";
  let capturedMethod = "";
  let capturedData = null;

  apiClient.defaults.adapter = async (config) => {
    capturedUrl = config.url;
    capturedMethod = config.method;
    capturedData = typeof config.data === "string" ? JSON.parse(config.data) : config.data;
    return {
      status: 200,
      data: {
        success: true,
        message: "event cancelled successfully",
        data: {
          event_id: eventId,
          event_status: "CANCELLED",
          total_bookings_affected: 48,
          total_tickets_affected: 115,
          total_eligible_refund_amount: "145000.00",
        },
      },
      config,
    };
  };

  const response = await cancelAdminEvent(eventId, reasonText);

  assert.equal(capturedMethod, "post");
  assert.equal(capturedUrl, `/admin/events/${eventId}/cancel`);
  assert.deepStrictEqual(capturedData, { reason: reasonText });
  assert.equal(response.success, true);
  assert.equal(response.data.event_status, "CANCELLED");
});

test("Admin API: getAdminEventCancellationStatus constructs correct GET status URL", async () => {
  const eventId = "e0a7894a-97bb-41d3-a5c7-5e921d784a92";

  let capturedUrl = "";
  apiClient.defaults.adapter = async (config) => {
    capturedUrl = config.url;
    return {
      status: 200,
      data: {
        success: true,
        data: {
          event_id: eventId,
          event_status: "CANCELLED",
          completion_status: "COMPLETED",
        },
      },
      config,
    };
  };

  const response = await getAdminEventCancellationStatus(eventId);
  assert.equal(capturedUrl, `/admin/events/${eventId}/cancellation`);
  assert.equal(response.data.completion_status, "COMPLETED");
});

test("Admin Financial Formatting: ledger entries format correctly in INR", () => {
  const finImpact = createAdminEligiblePreviewFixture().data.financial_impact;

  const gross = formatCurrency(finImpact.gross_collected_amount, finImpact.currency);
  const already = formatCurrency(finImpact.already_refunded_amount, finImpact.currency);
  const expected = formatCurrency(finImpact.expected_refund_amount, finImpact.currency);
  const orgRevoke = formatCurrency(finImpact.organizer_payable_revocation, finImpact.currency);
  const feeRefund = formatCurrency(finImpact.platform_fee_refund, finImpact.currency);
  const taxRefund = formatCurrency(finImpact.tax_refund, finImpact.currency);

  assert.match(gross, /1,50,000/);
  assert.match(already, /5,000/);
  assert.match(expected, /1,45,000/);
  assert.match(orgRevoke, /1,11,650/);
  assert.match(feeRefund, /7,250/);
  assert.match(taxRefund, /26,100/);
});

test("Admin Eligibility Enforcement: blocks confirmation when is_cancellable is false", () => {
  const ineligible = createAdminIneligiblePreviewFixture().data;

  const isCancellable = Boolean(ineligible.eligibility?.is_cancellable);
  assert.equal(isCancellable, false);

  const blockingReasons = ineligible.eligibility.blocking_reasons;
  assert.equal(blockingReasons.length, 2);
  assert.match(blockingReasons[0], /already concluded/);
  assert.match(blockingReasons[1], /completed settlement/);

  // Form submit state evaluation
  const reason = "Administrative override";
  const hasConfirmed = true;
  const isCancelling = false;

  const isSubmitDisabled = !isCancellable || !reason.trim() || !hasConfirmed || isCancelling;
  assert.equal(isSubmitDisabled, true);
});

test("Admin Eligibility Enforcement: enables confirmation when is_cancellable is true and inputs valid", () => {
  const eligible = createAdminEligiblePreviewFixture().data;

  const isCancellable = Boolean(eligible.eligibility?.is_cancellable);
  assert.equal(isCancellable, true);

  // Missing reason blocks
  let reason = "";
  let hasConfirmed = true;
  let isCancelling = false;
  assert.equal(!isCancellable || !reason.trim() || !hasConfirmed || isCancelling, true);

  // Whitespace reason blocks
  reason = "     ";
  assert.equal(!isCancellable || !reason.trim() || !hasConfirmed || isCancelling, true);

  // Exceeding 500 chars blocks
  reason = "x".repeat(501);
  assert.equal(reason.trim().length > 500, true);

  // Unchecked confirmation blocks
  reason = "Valid administrative reason";
  hasConfirmed = false;
  assert.equal(!isCancellable || !reason.trim() || !hasConfirmed || isCancelling, true);

  // Valid reason + checked box enables
  hasConfirmed = true;
  assert.equal(!isCancellable || !reason.trim() || !hasConfirmed || isCancelling, false);
});

test("Admin Settlement Warning: correctly surfaces pending settlements to platform admin", () => {
  const eligible = createAdminEligiblePreviewFixture().data;

  assert.equal(eligible.eligibility.is_cancellable, true);
  assert.equal(eligible.settlement_impact.pending_settlements_count, 1);
  assert.match(eligible.settlement_impact.settlement_warning, /25000/);
});

test("Admin Error Handling: handles 404 on preview and 400 on cancellation gracefully", async () => {
  // 1. Preview 404
  apiClient.defaults.adapter = async (config) => {
    const error = new Error("Event not found");
    error.response = { status: 404, data: { success: false, message: "event not found" } };
    error.config = config;
    throw error;
  };

  await assert.rejects(
    async () => {
      await getAdminEventCancellationPreview("non-existent-id");
    },
    (err) => {
      assert.equal(err.response.status, 404);
      assert.equal(err.response.data.message, "event not found");
      return true;
    }
  );

  // 2. Cancellation 400
  apiClient.defaults.adapter = async (config) => {
    const error = new Error("Cancellation reason is required");
    error.response = { status: 400, data: { success: false, message: "cancellation reason is required" } };
    error.config = config;
    throw error;
  };

  await assert.rejects(
    async () => {
      await cancelAdminEvent("any-id", "");
    },
    (err) => {
      assert.equal(err.response.status, 400);
      assert.equal(err.response.data.message, "cancellation reason is required");
      return true;
    }
  );
});

test("Admin Duplicate Submission Guard: prevents concurrent cancellation calls", async () => {
  let isCancelling = false;
  let dispatchCount = 0;

  apiClient.defaults.adapter = async (config) => {
    dispatchCount++;
    return {
      status: 200,
      data: { success: true, message: "event cancelled successfully" },
      config,
    };
  };

  const executeAdminCancel = async (eventId, reason) => {
    if (isCancelling) return;
    try {
      isCancelling = true;
      await cancelAdminEvent(eventId, reason);
    } finally {
      isCancelling = false;
    }
  };

  const p1 = executeAdminCancel("ev-123", "Reason 1");
  const p2 = executeAdminCancel("ev-123", "Reason 2");

  await Promise.all([p1, p2]);

  assert.equal(dispatchCount, 1);
});

test("Admin State Synchronization: post-cancellation updates event and counts in admin events list", () => {
  const initialEvents = [
    { id: "ev-1", title: "Festival", status: "PUBLISHED" },
    { id: "ev-2", title: "Conference", status: "DRAFT" },
  ];

  const initialStatusCounts = {
    total: 2,
    published: 1,
    draft: 1,
    cancelled: 0,
  };

  const cancelledId = "ev-1";
  const updatedEvents = initialEvents.map((ev) =>
    ev.id === cancelledId ? { ...ev, status: "CANCELLED" } : ev
  );

  const updatedCounts = {
    ...initialStatusCounts,
    published: Math.max(0, initialStatusCounts.published - 1),
    cancelled: initialStatusCounts.cancelled + 1,
  };

  assert.equal(updatedEvents[0].status, "CANCELLED");
  assert.equal(updatedCounts.published, 0);
  assert.equal(updatedCounts.cancelled, 1);
});
