import test from "node:test";
import assert from "node:assert";
import {
  buildOrganizerSettlementParams,
  getOrganizerSettlementBalance,
  requestOrganizerSettlement,
  getOrganizerSettlements,
  getOrganizerSettlementDetails,
} from "../organizerSettlementApi.js";

test("organizerSettlementApi exports all required API functions", () => {
  assert.strictEqual(typeof buildOrganizerSettlementParams, "function");
  assert.strictEqual(typeof getOrganizerSettlementBalance, "function");
  assert.strictEqual(typeof requestOrganizerSettlement, "function");
  assert.strictEqual(typeof getOrganizerSettlements, "function");
  assert.strictEqual(typeof getOrganizerSettlementDetails, "function");
});

test("buildOrganizerSettlementParams builds clean query params", () => {
  // Empty
  assert.deepStrictEqual(buildOrganizerSettlementParams(), {});

  // Standard pagination
  assert.deepStrictEqual(buildOrganizerSettlementParams({ page: 1, limit: 10 }), { page: 1, limit: 10 });

  // ALL status omitted
  assert.deepStrictEqual(buildOrganizerSettlementParams({ page: 2, limit: 25, status: "ALL" }), { page: 2, limit: 25 });

  // Normal status uppercased
  assert.deepStrictEqual(buildOrganizerSettlementParams({ page: 1, limit: 10, status: "pending" }), {
    page: 1,
    limit: 10,
    status: "PENDING",
  });
});

test("requestOrganizerSettlement validates input amount", async () => {
  await assert.rejects(
    async () => {
      await requestOrganizerSettlement({ amount: 0 });
    },
    {
      name: "Error",
      message: "Settlement amount must be a positive number",
    }
  );

  await assert.rejects(
    async () => {
      await requestOrganizerSettlement({ amount: -100 });
    },
    {
      name: "Error",
      message: "Settlement amount must be a positive number",
    }
  );

  await assert.rejects(
    async () => {
      await requestOrganizerSettlement({ amount: "invalid" });
    },
    {
      name: "Error",
      message: "Settlement amount must be a positive number",
    }
  );
});

test("getOrganizerSettlementDetails validates settlementId", async () => {
  await assert.rejects(
    async () => {
      await getOrganizerSettlementDetails("");
    },
    {
      name: "Error",
      message: "Invalid settlement ID",
    }
  );
});
