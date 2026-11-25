import test from "node:test";
import assert from "node:assert";

test("adminLedgerApi exports getAdminLedgerJournals, getAdminLedgerJournalById, and getAdminFinancialOverview", async () => {
  const { getAdminLedgerJournals, getAdminLedgerJournalById, getAdminFinancialOverview } =
    await import("../adminLedgerApi.js");
  assert.strictEqual(typeof getAdminLedgerJournals, "function");
  assert.strictEqual(typeof getAdminLedgerJournalById, "function");
  assert.strictEqual(typeof getAdminFinancialOverview, "function");
});

test("adminLedgerApi parameter builder correctly sanitizes query parameters", () => {
  const buildLedgerParams = ({
    page = 1,
    limit = 10,
    transactionType = "",
    status = "",
    search = "",
    referenceId = "",
    from = "",
    to = "",
    eventId = "",
    organizerId = "",
    settlementStatus = "",
  } = {}) => {
    const params = { page, limit };
    if (transactionType && transactionType.trim() !== "" && transactionType.trim().toUpperCase() !== "ALL") {
      params.transaction_type = transactionType.trim().toUpperCase();
    }
    if (status && status.trim() !== "" && status.trim().toUpperCase() !== "ALL") {
      params.status = status.trim().toUpperCase();
    }
    const querySearch = search || referenceId;
    if (querySearch && querySearch.trim() !== "") {
      params.search = querySearch.trim();
    }
    if (from && from.trim() !== "") {
      params.from = from.trim();
    }
    if (to && to.trim() !== "") {
      params.to = to.trim();
    }
    if (eventId && eventId.trim() !== "" && eventId.trim().toUpperCase() !== "ALL") {
      params.event_id = eventId.trim();
    }
    if (organizerId && organizerId.trim() !== "" && organizerId.trim().toUpperCase() !== "ALL") {
      params.organizer_id = organizerId.trim();
    }
    if (settlementStatus && settlementStatus.trim() !== "" && settlementStatus.trim().toUpperCase() !== "ALL") {
      params.settlement_status = settlementStatus.trim().toUpperCase();
    }
    return params;
  };

  assert.deepStrictEqual(buildLedgerParams(), { page: 1, limit: 10 });
  assert.deepStrictEqual(
    buildLedgerParams({ page: 2, limit: 25, transactionType: "ALL", status: "ALL" }),
    { page: 2, limit: 25 }
  );
  assert.deepStrictEqual(
    buildLedgerParams({
      transactionType: "payment",
      status: "posted",
      search: "pay_123",
      from: "2026-11-01",
      to: "2026-11-20",
      eventId: "evt-001",
      organizerId: "org-001",
      settlementStatus: "pending",
    }),
    {
      page: 1,
      limit: 10,
      transaction_type: "PAYMENT",
      status: "POSTED",
      search: "pay_123",
      from: "2026-11-01",
      to: "2026-11-20",
      event_id: "evt-001",
      organizer_id: "org-001",
      settlement_status: "PENDING",
    }
  );
  assert.deepStrictEqual(
    buildLedgerParams({ referenceId: "rfnd_abc" }),
    { page: 1, limit: 10, search: "rfnd_abc" }
  );
});

test("adminFinancialOverview parameter builder correctly maps filters", () => {
  const buildOverviewParams = ({
    from = "",
    to = "",
    eventId = "",
    organizerId = "",
    transactionType = "",
    settlementStatus = "",
  } = {}) => {
    const params = {};
    if (from && from.trim() !== "") {
      params.from = from.trim();
    }
    if (to && to.trim() !== "") {
      params.to = to.trim();
    }
    if (eventId && eventId.trim() !== "" && eventId.trim().toUpperCase() !== "ALL") {
      params.event_id = eventId.trim();
    }
    if (organizerId && organizerId.trim() !== "" && organizerId.trim().toUpperCase() !== "ALL") {
      params.organizer_id = organizerId.trim();
    }
    if (transactionType && transactionType.trim() !== "" && transactionType.trim().toUpperCase() !== "ALL") {
      params.transaction_type = transactionType.trim().toUpperCase();
    }
    if (settlementStatus && settlementStatus.trim() !== "" && settlementStatus.trim().toUpperCase() !== "ALL") {
      params.settlement_status = settlementStatus.trim().toUpperCase();
    }
    return params;
  };

  assert.deepStrictEqual(buildOverviewParams(), {});
  assert.deepStrictEqual(
    buildOverviewParams({
      from: "2026-11-01",
      to: "2026-11-21",
      eventId: "e6a2b8e4-8457-4148-8a8b-bf1b514df19d",
      organizerId: "b8c3d4e5-1234-5678-9abc-def012345678",
      transactionType: "payment",
      settlementStatus: "approved",
    }),
    {
      from: "2026-11-01",
      to: "2026-11-21",
      event_id: "e6a2b8e4-8457-4148-8a8b-bf1b514df19d",
      organizer_id: "b8c3d4e5-1234-5678-9abc-def012345678",
      transaction_type: "PAYMENT",
      settlement_status: "APPROVED",
    }
  );
  assert.deepStrictEqual(
    buildOverviewParams({
      eventId: "ALL",
      organizerId: "ALL",
      transactionType: "ALL",
      settlementStatus: "ALL",
    }),
    {}
  );
});

test("getAdminLedgerJournalById throws error when id is empty", async () => {
  const { getAdminLedgerJournalById } = await import("../adminLedgerApi.js");
  await assert.rejects(async () => {
    await getAdminLedgerJournalById("");
  }, /Journal ID is required/);
});
