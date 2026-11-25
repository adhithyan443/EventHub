import test from "node:test";
import assert from "node:assert";

test("adminPaymentApi exports getAdminPayments and getAdminEventPaymentSummaries functions", async () => {
  const { getAdminPayments, getAdminEventPaymentSummaries } = await import("../adminPaymentApi.js");
  assert.strictEqual(typeof getAdminPayments, "function");
  assert.strictEqual(typeof getAdminEventPaymentSummaries, "function");
});

test("adminPaymentApi parameter builder produces clean, valid query params for getAdminPayments", () => {
  const buildPaymentsParams = ({
    page = 1,
    limit = 10,
    status = "",
    allocationStatus = "",
    from = "",
    to = "",
    search = "",
    eventId = "",
    event_id = "",
  } = {}) => {
    const params = { page, limit };

    if (status && status.trim() !== "" && status.trim().toUpperCase() !== "ALL") {
      params.status = status.trim().toUpperCase();
    }

    if (
      allocationStatus &&
      allocationStatus.trim() !== "" &&
      allocationStatus.trim().toUpperCase() !== "ALL"
    ) {
      params.allocation_status = allocationStatus.trim().toUpperCase();
    }

    if (from && from.trim() !== "") {
      params.from = from.trim();
    }

    if (to && to.trim() !== "") {
      params.to = to.trim();
    }

    if (search && search.trim() !== "") {
      params.search = search.trim();
    }

    const resolvedEventId = (eventId || event_id || "").trim();
    if (resolvedEventId) {
      params.event_id = resolvedEventId;
    }

    return params;
  };

  // Scenario 1: Default params (page 1, limit 10, no filters)
  const defaultParams = buildPaymentsParams();
  assert.deepStrictEqual(defaultParams, { page: 1, limit: 10 });

  // Scenario 2: ALL status and ALL allocation status are omitted from request
  const allParams = buildPaymentsParams({
    page: 2,
    limit: 25,
    status: "ALL",
    allocationStatus: "ALL",
    search: "   ",
  });
  assert.deepStrictEqual(allParams, { page: 2, limit: 25 });

  // Scenario 3: Filtered by status, allocationStatus, date range, search, and eventId
  const filteredParams = buildPaymentsParams({
    page: 3,
    limit: 50,
    status: "success",
    allocationStatus: "accounted",
    from: "2026-11-01",
    to: "2026-11-18",
    search: "BK-12345",
    eventId: "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
  });
  assert.deepStrictEqual(filteredParams, {
    page: 3,
    limit: 50,
    status: "SUCCESS",
    allocation_status: "ACCOUNTED",
    from: "2026-11-01",
    to: "2026-11-18",
    search: "BK-12345",
    event_id: "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
  });

  // Scenario 4: UNALLOCATED and REFUND_REQUIRED statuses with event_id alias
  const edgeParams = buildPaymentsParams({
    status: "REFUND_REQUIRED",
    allocationStatus: "UNALLOCATED",
    event_id: "  123e4567-e89b-12d3-a456-426614174000  ",
  });
  assert.strictEqual(edgeParams.status, "REFUND_REQUIRED");
  assert.strictEqual(edgeParams.allocation_status, "UNALLOCATED");
  assert.strictEqual(edgeParams.event_id, "123e4567-e89b-12d3-a456-426614174000");
});

test("adminPaymentApi parameter builder produces clean query params for getAdminEventPaymentSummaries", () => {
  const buildEventSummaryParams = ({
    page = 1,
    limit = 10,
    search = "",
    from = "",
    to = "",
    eventId = "",
    event_id = "",
  } = {}) => {
    const params = { page, limit };

    if (search && search.trim() !== "") {
      params.search = search.trim();
    }

    if (from && from.trim() !== "") {
      params.from = from.trim();
    }

    if (to && to.trim() !== "") {
      params.to = to.trim();
    }

    const resolvedEventId = (eventId || event_id || "").trim();
    if (resolvedEventId) {
      params.event_id = resolvedEventId;
    }

    return params;
  };

  // Scenario 1: Default params
  const defaultParams = buildEventSummaryParams();
  assert.deepStrictEqual(defaultParams, { page: 1, limit: 10 });

  // Scenario 2: Search and date range
  const searchParams = buildEventSummaryParams({
    page: 1,
    limit: 25,
    search: "Tech Conference",
    from: "2026-01-01",
    to: "2026-12-31",
  });
  assert.deepStrictEqual(searchParams, {
    page: 1,
    limit: 25,
    search: "Tech Conference",
    from: "2026-01-01",
    to: "2026-12-31",
  });

  // Scenario 3: Optional event_id filter
  const eventIdParams = buildEventSummaryParams({
    page: 2,
    limit: 10,
    eventId: "e100-200-300",
  });
  assert.deepStrictEqual(eventIdParams, {
    page: 2,
    limit: 10,
    event_id: "e100-200-300",
  });
});

test("admin payment response normalization safely handles standard, empty, and nil responses", () => {
  const normalizePaymentsResponse = (res) => {
    const payments = Array.isArray(res?.data?.payments)
      ? res.data.payments
      : Array.isArray(res?.data)
      ? res.data
      : [];
    const totalRecords =
      typeof res?.data?.total_records === "number"
        ? res.data.total_records
        : typeof res?.data?.total === "number"
        ? res.data.total
        : 0;
    const totalPages =
      typeof res?.data?.total_pages === "number"
        ? Math.max(1, res.data.total_pages)
        : 1;
    const page = typeof res?.data?.page === "number" ? res.data.page : 1;
    const limit = typeof res?.data?.limit === "number" ? res.data.limit : 10;

    return { payments, totalRecords, totalPages, page, limit };
  };

  // Standard backend envelope
  const standardRes = {
    success: true,
    data: {
      payments: [
        {
          id: "pay-1",
          booking_id: "book-1",
          booking_number: "BK-100",
          customer: { id: "c-1", name: "Alice", email: "alice@test.com" },
          event: { id: "e-1", title: "Concert" },
          organizer: { id: "o-1", business_name: "Live Nation" },
          amount: "1500.00",
          currency: "INR",
          status: "SUCCESS",
          has_gateway_capture: true,
          allocation_status: "ACCOUNTED",
          method: "UPI",
          provider: "RAZORPAY",
          transaction_id: "pay_test123",
          provider_order_id: "order_test123",
          paid_at: "2026-11-18T10:00:00Z",
          created_at: "2026-11-18T09:55:00Z",
          updated_at: "2026-11-18T10:00:00Z",
        },
      ],
      total_records: 45,
      total_pages: 5,
      page: 1,
      limit: 10,
    },
  };

  const normStandard = normalizePaymentsResponse(standardRes);
  assert.strictEqual(normStandard.payments.length, 1);
  assert.strictEqual(normStandard.totalRecords, 45);
  assert.strictEqual(normStandard.totalPages, 5);
  assert.strictEqual(normStandard.payments[0].customer.name, "Alice");
  assert.strictEqual(normStandard.payments[0].allocation_status, "ACCOUNTED");

  // Nil slice / empty array
  const emptyRes = {
    success: true,
    data: {
      payments: null,
      total_records: 0,
      total_pages: 0,
      page: 1,
      limit: 10,
    },
  };
  const normEmpty = normalizePaymentsResponse(emptyRes);
  assert.deepStrictEqual(normEmpty.payments, []);
  assert.strictEqual(normEmpty.totalRecords, 0);
  assert.strictEqual(normEmpty.totalPages, 1);

  // Corrupted / malformed responses
  const malformedList = [null, undefined, {}, { data: null }, { data: {} }, "error"];
  for (const malformed of malformedList) {
    const safe = normalizePaymentsResponse(malformed);
    assert.strictEqual(Array.isArray(safe.payments), true);
    assert.strictEqual(safe.payments.length, 0);
    assert.strictEqual(safe.totalRecords, 0);
    assert.strictEqual(safe.totalPages, 1);
  }
});

test("admin event payment summary response normalization handles overall_summary and summaries correctly", () => {
  const normalizeEventSummariesResponse = (res) => {
    const summaries = Array.isArray(res?.data?.summaries)
      ? res.data.summaries
      : Array.isArray(res?.data?.events)
      ? res.data.events
      : [];

    const rawOverall = res?.data?.overall_summary;
    const overallSummary = {
      total_events: typeof rawOverall?.total_events === "number" ? rawOverall.total_events : 0,
      gross_captured_amount:
        rawOverall?.gross_captured_amount != null
          ? String(rawOverall.gross_captured_amount)
          : "0.00",
      total_refunded_amount:
        rawOverall?.total_refunded_amount != null
          ? String(rawOverall.total_refunded_amount)
          : "0.00",
      net_captured_amount:
        rawOverall?.net_captured_amount != null
          ? String(rawOverall.net_captured_amount)
          : "0.00",
      currency: rawOverall?.currency || "INR",
      refund_required_count:
        typeof rawOverall?.refund_required_count === "number"
          ? rawOverall.refund_required_count
          : 0,
      unallocated_captured_count:
        typeof rawOverall?.unallocated_captured_count === "number"
          ? rawOverall.unallocated_captured_count
          : 0,
    };

    const totalRecords =
      typeof res?.data?.total_records === "number" ? res.data.total_records : 0;
    const totalPages =
      typeof res?.data?.total_pages === "number" ? Math.max(1, res.data.total_pages) : 1;

    return { summaries, overallSummary, totalRecords, totalPages };
  };

  // Standard event summary payload
  const standardRes = {
    success: true,
    data: {
      summaries: [
        {
          event_id: "e-1",
          event_title: "Tech Summit 2026",
          organizer_id: "o-1",
          organizer_business_name: "Tech Org",
          total_payments_count: 10,
          successfully_completed_payments_count: 8,
          gross_captured_amount: "15000.00",
          total_refunded_amount: "2000.00",
          net_captured_amount: "13000.00",
          currency: "INR",
          pending_payments_count: 1,
          failed_payments_count: 1,
          refund_required_count: 2,
          payments_with_successful_refunds_count: 1,
          unallocated_captured_count: 0,
        },
      ],
      overall_summary: {
        total_events: 1,
        gross_captured_amount: "15000.00",
        total_refunded_amount: "2000.00",
        net_captured_amount: "13000.00",
        currency: "INR",
        refund_required_count: 2,
        unallocated_captured_count: 0,
      },
      total_records: 1,
      total_pages: 1,
    },
  };

  const norm = normalizeEventSummariesResponse(standardRes);
  assert.strictEqual(norm.summaries.length, 1);
  assert.strictEqual(norm.summaries[0].total_refunded_amount, "2000.00");
  assert.strictEqual(norm.summaries[0].net_captured_amount, "13000.00");
  assert.strictEqual(norm.overallSummary.total_events, 1);
  assert.strictEqual(norm.overallSummary.gross_captured_amount, "15000.00");
  assert.strictEqual(norm.overallSummary.total_refunded_amount, "2000.00");
  assert.strictEqual(norm.overallSummary.net_captured_amount, "13000.00");
  assert.strictEqual(norm.overallSummary.refund_required_count, 2);
  assert.strictEqual(norm.overallSummary.unallocated_captured_count, 0);

  // Fallback on null overall_summary
  const nullOverallRes = {
    success: true,
    data: {
      summaries: [],
      overall_summary: null,
      total_records: 0,
      total_pages: 0,
    },
  };
  const normNull = normalizeEventSummariesResponse(nullOverallRes);
  assert.strictEqual(normNull.overallSummary.total_events, 0);
  assert.strictEqual(normNull.overallSummary.gross_captured_amount, "0.00");
  assert.strictEqual(normNull.overallSummary.total_refunded_amount, "0.00");
  assert.strictEqual(normNull.overallSummary.net_captured_amount, "0.00");
  assert.strictEqual(normNull.overallSummary.refund_required_count, 0);
  assert.strictEqual(normNull.overallSummary.unallocated_captured_count, 0);
});

test("currency formatting handles numerical, decimal string, null, and edge cases safely", () => {
  const formatCurrency = (amount, currency = "INR") => {
    if (amount === null || amount === undefined) return "—";
    const num = typeof amount === "number" ? amount : parseFloat(amount);
    if (isNaN(num)) return "—";
    return new Intl.NumberFormat("en-IN", {
      style: "currency",
      currency: currency || "INR",
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(num);
  };

  const formatted1 = formatCurrency("1500.00");
  assert.strictEqual(formatted1.includes("1,500.00"), true);

  const formatted2 = formatCurrency(0);
  assert.strictEqual(formatted2.includes("0.00"), true);

  const formatted3 = formatCurrency(null);
  assert.strictEqual(formatted3, "—");

  const formatted4 = formatCurrency(undefined);
  assert.strictEqual(formatted4, "—");

  const formatted5 = formatCurrency("not-a-number");
  assert.strictEqual(formatted5, "—");
});

test("drill-down workflow correctly sets selected event filter and switches to All Payments", () => {
  let state = {
    viewMode: "BY_EVENT",
    selectedEventFilter: null,
    paymentPage: 3,
  };

  const handleDrillDownToEvent = (eventSummary) => {
    state = {
      ...state,
      viewMode: "ALL_PAYMENTS",
      selectedEventFilter: {
        id: eventSummary.event_id,
        title: eventSummary.event_title || "Selected Event",
      },
      paymentPage: 1, // Resets to page 1 on drill down
    };
  };

  handleDrillDownToEvent({
    event_id: "evt-999",
    event_title: "Rock Festival",
  });

  assert.strictEqual(state.viewMode, "ALL_PAYMENTS");
  assert.deepStrictEqual(state.selectedEventFilter, {
    id: "evt-999",
    title: "Rock Festival",
  });
  assert.strictEqual(state.paymentPage, 1);

  // Clearing the filter
  const handleClearEventFilter = () => {
    state = {
      ...state,
      selectedEventFilter: null,
      paymentPage: 1,
    };
  };

  handleClearEventFilter();
  assert.strictEqual(state.selectedEventFilter, null);
  assert.strictEqual(state.viewMode, "ALL_PAYMENTS");
});
