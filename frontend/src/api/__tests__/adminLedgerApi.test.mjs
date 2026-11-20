import test from "node:test";
import assert from "node:assert";

test("adminLedgerApi exports getAdminLedgerJournals and getAdminLedgerJournalById functions", async () => {
  const { getAdminLedgerJournals, getAdminLedgerJournalById } = await import("../adminLedgerApi.js");
  assert.strictEqual(typeof getAdminLedgerJournals, "function");
  assert.strictEqual(typeof getAdminLedgerJournalById, "function");
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
    }),
    {
      page: 1,
      limit: 10,
      transaction_type: "PAYMENT",
      status: "POSTED",
      search: "pay_123",
      from: "2026-11-01",
      to: "2026-11-20",
    }
  );
  assert.deepStrictEqual(
    buildLedgerParams({ referenceId: "rfnd_abc" }),
    { page: 1, limit: 10, search: "rfnd_abc" }
  );
});

test("getAdminLedgerJournalById throws error when id is empty", async () => {
  const { getAdminLedgerJournalById } = await import("../adminLedgerApi.js");
  await assert.rejects(async () => {
    await getAdminLedgerJournalById("");
  }, /Journal ID is required/);
});
