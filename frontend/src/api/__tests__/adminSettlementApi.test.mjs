import test from "node:test";
import assert from "node:assert";
import {
  buildAdminSettlementParams,
  getAdminSettlements,
  getAdminSettlementDetails,
  approveAdminSettlement,
  rejectAdminSettlement,
  retryAdminSettlement,
} from "../adminSettlementApi.js";

test("adminSettlementApi exports all required API functions", () => {
  assert.strictEqual(typeof buildAdminSettlementParams, "function");
  assert.strictEqual(typeof getAdminSettlements, "function");
  assert.strictEqual(typeof getAdminSettlementDetails, "function");
  assert.strictEqual(typeof approveAdminSettlement, "function");
  assert.strictEqual(typeof rejectAdminSettlement, "function");
  assert.strictEqual(typeof retryAdminSettlement, "function");
});

test("buildAdminSettlementParams builds clean query params with filters", () => {
  // Empty
  assert.deepStrictEqual(buildAdminSettlementParams(), {});

  // Pagination and status
  assert.deepStrictEqual(
    buildAdminSettlementParams({ page: 1, limit: 20, status: "pending" }),
    { page: 1, limit: 20, status: "PENDING" }
  );

  // Status "ALL" ignored
  assert.deepStrictEqual(
    buildAdminSettlementParams({ page: 1, limit: 10, status: "ALL" }),
    { page: 1, limit: 10 }
  );

  // Organizer ID filtering
  const orgUUID = "123e4567-e89b-12d3-a456-426614174000";
  assert.deepStrictEqual(
    buildAdminSettlementParams({ page: 1, limit: 10, organizerId: orgUUID }),
    { page: 1, limit: 10, organizer_id: orgUUID }
  );
});

test("rejectAdminSettlement validates reason requirement", async () => {
  await assert.rejects(
    async () => {
      await rejectAdminSettlement("123", "");
    },
    {
      name: "Error",
      message: "Rejection reason is required",
    }
  );

  await assert.rejects(
    async () => {
      await rejectAdminSettlement("123", "   ");
    },
    {
      name: "Error",
      message: "Rejection reason is required",
    }
  );
});

test("admin actions validate settlementId parameter", async () => {
  await assert.rejects(
    async () => {
      await approveAdminSettlement("");
    },
    {
      name: "Error",
      message: "Invalid settlement ID",
    }
  );

  await assert.rejects(
    async () => {
      await retryAdminSettlement("");
    },
    {
      name: "Error",
      message: "Invalid settlement ID",
    }
  );
});
