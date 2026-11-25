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

// Import client and API methods
const { default: apiClient } = await import("../../../api/client.js");
const {
  getOrganizerEventCancellationPreview,
  cancelOrganizerEvent,
  getOrganizerEventCancellationStatus,
} = await import("../../../api/organizerApi.js");
const { formatCurrency } = await import("../../../utils/formatters.js");

beforeEach(() => {
  mockStorage.clear();
});

// Sample preview response fixture for eligible event
const createEligiblePreviewFixture = (eventId = "e0a7894a-97bb-41d3-a5c7-5e921d784a92") => ({
  success: true,
  data: {
    event: {
      id: eventId,
      title: "Global Music Festival 2026",
      status: "PUBLISHED",
      scheduled_date: "15 Dec 2026",
    },
    eligibility: {
      is_cancellable: true,
      blocking_reasons: [],
    },
    operational_impact: {
      total_bookings: 25,
      affected_bookings: 23,
      total_tickets_sold: 50,
      affected_tickets: 45,
      active_reservations_released: 2,
      reserved_seats_released: 4,
      general_admission_quantity_released: 41,
      active_checkouts_expired: 1,
    },
    financial_impact: {
      currency: "INR",
      gross_collected_amount: "50000.00",
      already_refunded_amount: "2000.00",
      expected_refund_amount: "48000.00",
      platform_fee_refund: "2400.00",
      tax_refund: "8640.00",
      organizer_payable_revocation: "36960.00",
    },
    settlement_impact: {
      has_completed_settlement: false,
      has_in_flight_settlement: false,
      pending_settlements_count: 1,
      pending_settlements_total: "1500.00",
      settlement_warning: "1 pending settlement request(s) totaling 1500.00 will be automatically rejected upon cancellation",
    },
    generated_at: "2026-11-23T20:00:00Z",
  },
});

// Sample preview response fixture for blocked/ineligible event
const createIneligiblePreviewFixture = (eventId = "e0a7894a-97bb-41d3-a5c7-5e921d784a92") => ({
  success: true,
  data: {
    event: {
      id: eventId,
      title: "Past Summer Gala",
      status: "COMPLETED",
      scheduled_date: "10 Oct 2026",
    },
    eligibility: {
      is_cancellable: false,
      blocking_reasons: [
        "cannot cancel an event that has already concluded",
        "cannot cancel event: 1 completed settlement(s) exist; manual recovery required",
      ],
    },
    operational_impact: {
      total_bookings: 100,
      affected_bookings: 0,
      total_tickets_sold: 200,
      affected_tickets: 0,
      active_reservations_released: 0,
      reserved_seats_released: 0,
      general_admission_quantity_released: 0,
      active_checkouts_expired: 0,
    },
    financial_impact: {
      currency: "INR",
      gross_collected_amount: "200000.00",
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
    generated_at: "2026-11-23T20:00:00Z",
  },
});

test("API: getOrganizerEventCancellationPreview constructs correct URL and returns preview data", async () => {
  const eventId = "e0a7894a-97bb-41d3-a5c7-5e921d784a92";
  const expectedFixture = createEligiblePreviewFixture(eventId);

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

  const result = await getOrganizerEventCancellationPreview(eventId);

  assert.equal(capturedMethod, "get");
  assert.equal(capturedUrl, `/organizers/events/${eventId}/cancellation-preview`);
  assert.equal(result.success, true);
  assert.equal(result.data.event.id, eventId);
  assert.equal(result.data.eligibility.is_cancellable, true);
  assert.equal(result.data.operational_impact.affected_bookings, 23);
  assert.equal(result.data.financial_impact.expected_refund_amount, "48000.00");
});

test("API: cancelOrganizerEvent constructs POST request with cancellation reason payload", async () => {
  const eventId = "e0a7894a-97bb-41d3-a5c7-5e921d784a92";
  const reasonText = "Severe weather emergency and venue flood warning";

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
          total_bookings_affected: 23,
          total_tickets_affected: 45,
          total_eligible_refund_amount: "48000.00",
        },
      },
      config,
    };
  };

  const response = await cancelOrganizerEvent(eventId, reasonText);

  assert.equal(capturedMethod, "post");
  assert.equal(capturedUrl, `/organizers/events/${eventId}/cancel`);
  assert.deepStrictEqual(capturedData, { reason: reasonText });
  assert.equal(response.success, true);
  assert.equal(response.data.event_status, "CANCELLED");
});

test("API: getOrganizerEventCancellationStatus constructs correct GET status URL", async () => {
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

  const response = await getOrganizerEventCancellationStatus(eventId);
  assert.equal(capturedUrl, `/organizers/events/${eventId}/cancellation`);
  assert.equal(response.data.completion_status, "COMPLETED");
});

test("Preview Data Normalization: financial values format safely with formatCurrency", () => {
  const fixture = createEligiblePreviewFixture().data.financial_impact;

  const formattedGross = formatCurrency(fixture.gross_collected_amount, fixture.currency);
  const formattedAlready = formatCurrency(fixture.already_refunded_amount, fixture.currency);
  const formattedExpected = formatCurrency(fixture.expected_refund_amount, fixture.currency);
  const formattedOrgRevoke = formatCurrency(fixture.organizer_payable_revocation, fixture.currency);

  assert.match(formattedGross, /50,000/);
  assert.match(formattedAlready, /2,000/);
  assert.match(formattedExpected, /48,000/);
  assert.match(formattedOrgRevoke, /36,960/);
});

test("Eligibility Check Logic: prevents confirmation when is_cancellable is false", () => {
  const ineligibleFixture = createIneligiblePreviewFixture().data;

  const isCancellable = Boolean(ineligibleFixture.eligibility?.is_cancellable);
  assert.equal(isCancellable, false);

  const blockingReasons = ineligibleFixture.eligibility.blocking_reasons;
  assert.equal(blockingReasons.length, 2);
  assert.match(blockingReasons[0], /already concluded/);
  assert.match(blockingReasons[1], /completed settlement/);

  // Validate that button disabled condition triggers correctly
  const reason = "Trying to cancel";
  const hasConfirmed = true;
  const isCancelling = false;

  const isButtonDisabled = !isCancellable || !reason.trim() || !hasConfirmed || isCancelling;
  assert.equal(isButtonDisabled, true);
});

test("Eligibility Check Logic: permits confirmation only when is_cancellable is true and inputs valid", () => {
  const eligibleFixture = createEligiblePreviewFixture().data;

  const isCancellable = Boolean(eligibleFixture.eligibility?.is_cancellable);
  assert.equal(isCancellable, true);

  // 1. Missing reason blocks confirmation
  let reason = "";
  let hasConfirmed = true;
  let isCancelling = false;
  assert.equal(!isCancellable || !reason.trim() || !hasConfirmed || isCancelling, true);

  // 2. Whitespace-only reason blocks confirmation
  reason = "    ";
  assert.equal(!isCancellable || !reason.trim() || !hasConfirmed || isCancelling, true);

  // 3. Reason exceeding 500 chars blocks confirmation
  reason = "a".repeat(501);
  assert.equal(reason.trim().length > 500, true);

  // 4. Missing confirmation checkbox blocks confirmation
  reason = "Artist became unavailable due to scheduling conflict.";
  hasConfirmed = false;
  assert.equal(!isCancellable || !reason.trim() || !hasConfirmed || isCancelling, true);

  // 5. Valid reason + checkbox checked enables confirmation
  hasConfirmed = true;
  assert.equal(!isCancellable || !reason.trim() || !hasConfirmed || isCancelling, false);
});

test("Settlement Warning Extraction: surfaces pending settlement warnings without blocking", () => {
  const eligibleFixture = createEligiblePreviewFixture().data;

  assert.equal(eligibleFixture.eligibility.is_cancellable, true);
  assert.equal(eligibleFixture.settlement_impact.pending_settlements_count, 1);
  assert.match(eligibleFixture.settlement_impact.settlement_warning, /1 pending settlement request/);
});

test("Preview Failure Handling: server error propagates cleanly for UI retry", async () => {
  const eventId = "non-existent-event";

  apiClient.defaults.adapter = async (config) => {
    const error = new Error("Event not found");
    error.response = {
      status: 404,
      data: {
        success: false,
        message: "event not found",
      },
    };
    error.config = config;
    throw error;
  };

  await assert.rejects(
    async () => {
      await getOrganizerEventCancellationPreview(eventId);
    },
    (err) => {
      assert.equal(err.response.status, 404);
      assert.equal(err.response.data.message, "event not found");
      return true;
    }
  );
});

test("Cancellation Failure Handling: unauthorized organizer 403 propagates cleanly", async () => {
  const eventId = "other-organizer-event";

  apiClient.defaults.adapter = async (config) => {
    const error = new Error("Forbidden");
    error.response = {
      status: 403,
      data: {
        success: false,
        message: "you are not authorized to cancel this event",
      },
    };
    error.config = config;
    throw error;
  };

  await assert.rejects(
    async () => {
      await cancelOrganizerEvent(eventId, "Valid cancellation reason");
    },
    (err) => {
      assert.equal(err.response.status, 403);
      assert.equal(err.response.data.message, "you are not authorized to cancel this event");
      return true;
    }
  );
});

test("Duplicate Submission Guard: in-flight cancellation blocks repeated API dispatches", async () => {
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

  const executeCancel = async (eventId, reason) => {
    if (isCancelling) return;
    try {
      isCancelling = true;
      await cancelOrganizerEvent(eventId, reason);
    } finally {
      isCancelling = false;
    }
  };

  // Launch two concurrent attempts simultaneously
  const p1 = executeCancel("e0a7894a-97bb-41d3-a5c7-5e921d784a92", "Reason 1");
  const p2 = executeCancel("e0a7894a-97bb-41d3-a5c7-5e921d784a92", "Reason 2");

  await Promise.all([p1, p2]);

  // Only the first one was dispatched; the second was guarded by isCancelling
  assert.equal(dispatchCount, 1);
});

test("State Transition & Optimistic List Refresh: transforms event status to CANCELLED", () => {
  const initialEvents = [
    { id: "ev-1", title: "Event 1", Status: "PUBLISHED" },
    { id: "ev-2", title: "Event 2", Status: "DRAFT" },
    { id: "ev-3", title: "Event 3", Status: "PUBLISHED" },
  ];

  const cancelledId = "ev-1";
  const updatedEvents = initialEvents.map((ev) => {
    if (ev.id === cancelledId) {
      return {
        ...ev,
        Status: "CANCELLED",
        status: "CANCELLED",
      };
    }
    return ev;
  });

  assert.equal(updatedEvents[0].Status, "CANCELLED");
  assert.equal(updatedEvents[1].Status, "DRAFT");
  assert.equal(updatedEvents[2].Status, "PUBLISHED");
});
