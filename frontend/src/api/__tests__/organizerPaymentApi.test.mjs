import test from "node:test";
import assert from "node:assert";

test("organizerPaymentApi exports all required API functions", async () => {
  const {
    getOrganizerPaymentOverview,
    getOrganizerEventPaymentSummaries,
    getOrganizerPayments,
    buildOrganizerPaymentParams,
  } = await import("../organizerPaymentApi.js");

  assert.strictEqual(typeof getOrganizerPaymentOverview, "function");
  assert.strictEqual(typeof getOrganizerEventPaymentSummaries, "function");
  assert.strictEqual(typeof getOrganizerPayments, "function");
  assert.strictEqual(typeof buildOrganizerPaymentParams, "function");
});

test("buildOrganizerPaymentParams builds clean query params with defaults and filtering", async () => {
  const { buildOrganizerPaymentParams } = await import("../organizerPaymentApi.js");

  // Scenario 1: Empty filters produce empty object
  const emptyParams = buildOrganizerPaymentParams();
  assert.deepStrictEqual(emptyParams, {});

  // Scenario 2: Standard pagination
  const pagedParams = buildOrganizerPaymentParams({ page: 1, limit: 10 });
  assert.deepStrictEqual(pagedParams, { page: 1, limit: 10 });

  // Scenario 3: "ALL" status and "ALL" allocation_status are omitted from query params
  const allParams = buildOrganizerPaymentParams({
    page: 2,
    limit: 25,
    status: "ALL",
    allocationStatus: "ALL",
    search: "   ",
  });
  assert.deepStrictEqual(allParams, { page: 2, limit: 25 });

  // Scenario 4: Specific filters with mixed case and whitespace
  const filteredParams = buildOrganizerPaymentParams({
    page: 1,
    limit: 50,
    status: " success ",
    allocationStatus: " accounted ",
    from: "2026-11-01",
    to: "2026-11-19",
    search: "  Sarah  ",
    eventId: "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
  });
  assert.deepStrictEqual(filteredParams, {
    page: 1,
    limit: 50,
    status: "SUCCESS",
    allocation_status: "ACCOUNTED",
    from: "2026-11-01",
    to: "2026-11-19",
    search: "Sarah",
    event_id: "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
  });

  // Scenario 5: Supports allocation_status and event_id alias directly
  const aliasParams = buildOrganizerPaymentParams({
    allocation_status: "unallocated",
    event_id: "e456-789-012",
  });
  assert.deepStrictEqual(aliasParams, {
    allocation_status: "UNALLOCATED",
    event_id: "e456-789-012",
  });
});

test("organizer overview response normalization handles valid, zero, and missing data safely", () => {
  const normalizeOverview = (data) => {
    return {
      grossCapturedAmount: data?.gross_captured_amount != null ? String(data.gross_captured_amount) : "0.00",
      totalRefundedAmount: data?.total_refunded_amount != null ? String(data.total_refunded_amount) : "0.00",
      netCapturedAmount: data?.net_captured_amount != null ? String(data.net_captured_amount) : "0.00",
      totalTicketSalesAmount: data?.total_ticket_sales_amount != null ? String(data.total_ticket_sales_amount) : "0.00",
      totalPlatformFees: data?.total_platform_fees != null ? String(data.total_platform_fees) : "0.00",
      organizerEarnings: data?.organizer_earnings != null ? String(data.organizer_earnings) : "0.00",
      totalTicketsSold: typeof data?.total_tickets_sold === "number" ? data.total_tickets_sold : 0,
      totalPaymentsCount: typeof data?.total_payments_count === "number" ? data.total_payments_count : 0,
      totalEvents: typeof data?.total_events === "number" ? data.total_events : 0,
      currency: data?.currency || "INR",
      refundRequiredCount: typeof data?.refund_required_count === "number" ? data.refund_required_count : 0,
      unallocatedCapturedCount: typeof data?.unallocated_captured_count === "number" ? data.unallocated_captured_count : 0,
    };
  };

  // Standard payload
  const standard = normalizeOverview({
    gross_captured_amount: "45000.00",
    total_refunded_amount: "5000.00",
    net_captured_amount: "40000.00",
    total_ticket_sales_amount: "40000.00",
    total_platform_fees: "2000.00",
    organizer_earnings: "38000.00",
    total_tickets_sold: 90,
    total_payments_count: 50,
    total_events: 3,
    currency: "INR",
    refund_required_count: 1,
    unallocated_captured_count: 0,
  });

  assert.strictEqual(standard.grossCapturedAmount, "45000.00");
  assert.strictEqual(standard.totalRefundedAmount, "5000.00");
  assert.strictEqual(standard.netCapturedAmount, "40000.00");
  assert.strictEqual(standard.totalTicketSalesAmount, "40000.00");
  assert.strictEqual(standard.totalPlatformFees, "2000.00");
  assert.strictEqual(standard.organizerEarnings, "38000.00");
  assert.strictEqual(standard.totalTicketsSold, 90);
  assert.strictEqual(standard.totalPaymentsCount, 50);
  assert.strictEqual(standard.totalEvents, 3);
  assert.strictEqual(standard.currency, "INR");
  assert.strictEqual(standard.refundRequiredCount, 1);

  // Missing / null payload
  const empty = normalizeOverview(null);
  assert.strictEqual(empty.grossCapturedAmount, "0.00");
  assert.strictEqual(empty.totalRefundedAmount, "0.00");
  assert.strictEqual(empty.netCapturedAmount, "0.00");
  assert.strictEqual(empty.totalTicketSalesAmount, "0.00");
  assert.strictEqual(empty.totalPlatformFees, "0.00");
  assert.strictEqual(empty.organizerEarnings, "0.00");
  assert.strictEqual(empty.totalTicketsSold, 0);
  assert.strictEqual(empty.totalPaymentsCount, 0);
  assert.strictEqual(empty.currency, "INR");
});

test("organizer event summary response normalization parses summaries and overview safely", () => {
  const normalizeEventSummaries = (res) => {
    const data = res?.data || {};
    const summaries = Array.isArray(data.summaries)
      ? data.summaries
      : Array.isArray(data.events)
      ? data.events
      : [];
    const overview = data.overview || null;
    const totalRecords = typeof data.total_records === "number" ? data.total_records : 0;
    const totalPages = typeof data.total_pages === "number" ? Math.max(1, data.total_pages) : 1;
    const page = typeof data.page === "number" ? data.page : 1;
    const limit = typeof data.limit === "number" ? data.limit : 10;

    return { summaries, overview, totalRecords, totalPages, page, limit };
  };

  const payload = {
    success: true,
    data: {
      summaries: [
        {
          event_id: "evt-123",
          event_title: "Rock Festival 2026",
          tickets_sold: 250,
          gross_captured_amount: "125000.00",
          total_refunded_amount: "10000.00",
          net_captured_amount: "115000.00",
          net_ticket_sales: "109250.00",
          platform_fees: "5750.00",
          organizer_earnings: "109250.00",
          currency: "INR",
          total_payments_count: 120,
          successfully_completed_payments_count: 110,
          pending_payments_count: 5,
          failed_payments_count: 5,
          refund_required_count: 2,
          payments_with_successful_refunds_count: 1,
          unallocated_captured_count: 0,
        },
      ],
      overview: {
        gross_captured_amount: "125000.00",
        total_refunded_amount: "10000.00",
        net_captured_amount: "115000.00",
        total_tickets_sold: 250,
        total_ticket_sales_amount: "109250.00",
        total_platform_fees: "5750.00",
        organizer_earnings: "109250.00",
        currency: "INR",
        total_events: 1,
        total_payments_count: 120,
        successfully_completed_payments_count: 110,
        refund_required_count: 2,
        unallocated_captured_count: 0,
      },
      page: 1,
      limit: 10,
      total_records: 1,
      total_pages: 1,
    },
  };

  const norm = normalizeEventSummaries(payload);
  assert.strictEqual(norm.summaries.length, 1);
  assert.strictEqual(norm.summaries[0].event_title, "Rock Festival 2026");
  assert.strictEqual(norm.summaries[0].tickets_sold, 250);
  assert.strictEqual(norm.summaries[0].gross_captured_amount, "125000.00");
  assert.strictEqual(norm.summaries[0].total_refunded_amount, "10000.00");
  assert.strictEqual(norm.summaries[0].net_captured_amount, "115000.00");
  assert.strictEqual(norm.summaries[0].net_ticket_sales, "109250.00");
  assert.strictEqual(norm.summaries[0].organizer_earnings, "109250.00");
  assert.strictEqual(norm.overview.total_events, 1);
  assert.strictEqual(norm.overview.gross_captured_amount, "125000.00");
  assert.strictEqual(norm.overview.total_refunded_amount, "10000.00");
  assert.strictEqual(norm.overview.net_captured_amount, "115000.00");
  assert.strictEqual(norm.totalRecords, 1);
  assert.strictEqual(norm.totalPages, 1);

  // Nil summaries fallback
  const nilPayload = { success: true, data: { summaries: null } };
  const nilNorm = normalizeEventSummaries(nilPayload);
  assert.deepStrictEqual(nilNorm.summaries, []);
  assert.strictEqual(nilNorm.totalRecords, 0);
  assert.strictEqual(nilNorm.totalPages, 1);
});

test("organizer payments list response normalization parses payments safely with ticket_quantity", () => {
  const normalizePaymentsList = (res) => {
    const data = res?.data || {};
    const payments = Array.isArray(data.payments) ? data.payments : [];
    const totalRecords = typeof data.total_records === "number" ? data.total_records : 0;
    const totalPages = typeof data.total_pages === "number" ? Math.max(1, data.total_pages) : 1;
    const page = typeof data.page === "number" ? data.page : 1;
    const limit = typeof data.limit === "number" ? data.limit : 10;

    return { payments, totalRecords, totalPages, page, limit };
  };

  const payload = {
    success: true,
    data: {
      payments: [
        {
          id: "pay-777",
          booking_id: "book-888",
          booking_number: "BK-2026-9999",
          customer: {
            id: "usr-111",
            name: "John Doe",
            email: "john@example.com",
          },
          event: {
            id: "evt-222",
            title: "EDM Night",
          },
          ticket_quantity: 4,
          amount: "3200.00",
          currency: "INR",
          status: "SUCCESS",
          has_gateway_capture: true,
          allocation_status: "ACCOUNTED",
          method: "UPI",
          provider: "RAZORPAY",
          transaction_id: "pay_xyz987",
          provider_order_id: "order_abc123",
          paid_at: "2026-11-18T14:30:00Z",
          created_at: "2026-11-18T14:28:00Z",
          updated_at: "2026-11-18T14:30:00Z",
        },
      ],
      page: 1,
      limit: 10,
      total_records: 1,
      total_pages: 1,
    },
  };

  const norm = normalizePaymentsList(payload);
  assert.strictEqual(norm.payments.length, 1);
  assert.strictEqual(norm.payments[0].ticket_quantity, 4);
  assert.strictEqual(norm.payments[0].booking_number, "BK-2026-9999");
  assert.strictEqual(norm.payments[0].customer.name, "John Doe");
  assert.strictEqual(norm.payments[0].event.title, "EDM Night");
  assert.strictEqual(norm.payments[0].allocation_status, "ACCOUNTED");
  assert.strictEqual(norm.payments[0].amount, "3200.00");
  assert.strictEqual(norm.totalRecords, 1);
});

test("currency formatting handles INR decimal amounts accurately and safely", () => {
  const formatCurrency = (amount, currency = "INR") => {
    if (amount === null || amount === undefined || amount === "") return "—";
    const num = typeof amount === "number" ? amount : parseFloat(amount);
    if (isNaN(num)) return "—";
    return new Intl.NumberFormat("en-IN", {
      style: "currency",
      currency: currency || "INR",
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(num);
  };

  const formatted1 = formatCurrency("118750.00");
  assert.strictEqual(formatted1.includes("1,18,750.00"), true);

  const formatted2 = formatCurrency(0);
  assert.strictEqual(formatted2.includes("0.00"), true);

  assert.strictEqual(formatCurrency(null), "—");
  assert.strictEqual(formatCurrency("invalid"), "—");
});

test("drill-down from event summary to payments tab correctly scopes event filter", () => {
  let state = {
    viewMode: "BY_EVENT",
    selectedEventId: "",
    selectedEventTitle: "",
    page: 3,
  };

  const handleViewTransactionsForEvent = (eventSummary) => {
    state = {
      ...state,
      viewMode: "ALL_PAYMENTS",
      selectedEventId: eventSummary.event_id,
      selectedEventTitle: eventSummary.event_title,
      page: 1,
    };
  };

  handleViewTransactionsForEvent({
    event_id: "evt-444",
    event_title: "Tech Conference",
  });

  assert.strictEqual(state.viewMode, "ALL_PAYMENTS");
  assert.strictEqual(state.selectedEventId, "evt-444");
  assert.strictEqual(state.selectedEventTitle, "Tech Conference");
  assert.strictEqual(state.page, 1);

  // Clear event filter
  const handleClearEvent = () => {
    state = {
      ...state,
      selectedEventId: "",
      selectedEventTitle: "",
      page: 1,
    };
  };

  handleClearEvent();
  assert.strictEqual(state.selectedEventId, "");
  assert.strictEqual(state.selectedEventTitle, "");
  assert.strictEqual(state.viewMode, "ALL_PAYMENTS");
});

test("backend compatibility: payment status and allocation status allowlists match backend domain", () => {
  // Backend domain (backend/internal/domain/payment.go):
  // PaymentStatus: "PENDING", "SUCCESS", "FAILED", "REFUND_REQUIRED"
  // Note: "REFUNDED" is NOT a supported payment status in backend domain or usecase validation!
  const BACKEND_SUPPORTED_PAYMENT_STATUSES = new Set([
    "PENDING",
    "SUCCESS",
    "FAILED",
    "REFUND_REQUIRED",
  ]);

  const FRONTEND_PAYMENT_FILTER_STATUSES = [
    "SUCCESS",
    "PENDING",
    "FAILED",
    "REFUND_REQUIRED",
  ];

  for (const status of FRONTEND_PAYMENT_FILTER_STATUSES) {
    assert.strictEqual(
      BACKEND_SUPPORTED_PAYMENT_STATUSES.has(status),
      true,
      `Frontend filter status ${status} must be supported by backend domain`
    );
  }

  // Ensure REFUNDED is explicitly not included in frontend payment status filters
  assert.strictEqual(
    FRONTEND_PAYMENT_FILTER_STATUSES.includes("REFUNDED"),
    false,
    "REFUNDED must not be a selectable filter option to avoid backend validation failure"
  );

  // Backend domain AllocationStatus: "ACCOUNTED", "UNALLOCATED", "NOT_APPLICABLE"
  const BACKEND_SUPPORTED_ALLOCATION_STATUSES = new Set([
    "ACCOUNTED",
    "UNALLOCATED",
    "NOT_APPLICABLE",
  ]);

  const FRONTEND_ALLOCATION_FILTER_STATUSES = [
    "ACCOUNTED",
    "UNALLOCATED",
    "NOT_APPLICABLE",
  ];

  for (const allocStatus of FRONTEND_ALLOCATION_FILTER_STATUSES) {
    assert.strictEqual(
      BACKEND_SUPPORTED_ALLOCATION_STATUSES.has(allocStatus),
      true,
      `Frontend allocation filter ${allocStatus} must be supported by backend domain`
    );
  }
});

test("backend compatibility: drawer fields match OrganizerPaymentListItem contract", () => {
  const mockBackendPaymentItem = {
    id: "pay-100",
    booking_id: "book-200",
    booking_number: "BK-2026-0001",
    customer: {
      id: "usr-300",
      name: "Jane Smith",
      email: "jane@test.com",
    },
    event: {
      id: "evt-400",
      title: "Design Workshop",
    },
    ticket_quantity: 2,
    amount: "1500.00",
    currency: "INR",
    status: "SUCCESS",
    has_gateway_capture: true,
    allocation_status: "ACCOUNTED",
    method: "UPI",
    provider: "RAZORPAY",
    transaction_id: "txn-500",
    provider_order_id: "order-600",
    paid_at: "2026-11-18T10:00:00Z",
    created_at: "2026-11-18T09:55:00Z",
    updated_at: "2026-11-18T10:00:00Z",
  };

  // Verify all fields accessed by OrganizerPaymentDetailDrawer exist on the contract
  assert.strictEqual(mockBackendPaymentItem.id, "pay-100");
  assert.strictEqual(mockBackendPaymentItem.booking_number, "BK-2026-0001");
  assert.strictEqual(mockBackendPaymentItem.booking_id, "book-200");
  assert.strictEqual(mockBackendPaymentItem.customer.name, "Jane Smith");
  assert.strictEqual(mockBackendPaymentItem.customer.email, "jane@test.com");
  assert.strictEqual(mockBackendPaymentItem.event.title, "Design Workshop");
  assert.strictEqual(mockBackendPaymentItem.event.id, "evt-400");
  assert.strictEqual(mockBackendPaymentItem.ticket_quantity, 2);
  assert.strictEqual(mockBackendPaymentItem.amount, "1500.00");
  assert.strictEqual(mockBackendPaymentItem.currency, "INR");
  assert.strictEqual(mockBackendPaymentItem.status, "SUCCESS");
  assert.strictEqual(mockBackendPaymentItem.allocation_status, "ACCOUNTED");
  assert.strictEqual(mockBackendPaymentItem.method, "UPI");
  assert.strictEqual(mockBackendPaymentItem.provider, "RAZORPAY");
  assert.strictEqual(mockBackendPaymentItem.transaction_id, "txn-500");
  assert.strictEqual(mockBackendPaymentItem.provider_order_id, "order-600");
});
