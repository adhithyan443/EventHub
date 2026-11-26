import test from "node:test";
import assert from "node:assert";
import {
  buildAdminBookingParams,
  buildAdminAttendeeParams,
} from "../adminBookingApi.js";

test("adminBookingApi exports all required API functions", async () => {
  const api = await import("../adminBookingApi.js");
  assert.strictEqual(typeof api.getAdminBookings, "function");
  assert.strictEqual(typeof api.getAdminBookingStats, "function");
  assert.strictEqual(typeof api.getAdminBookingDetail, "function");
  assert.strictEqual(typeof api.getAdminAttendees, "function");
  assert.strictEqual(typeof api.getAdminAttendeeStats, "function");
});

test("buildAdminBookingParams normalizes query options correctly", () => {
  // Empty options
  const defaultParams = buildAdminBookingParams();
  assert.deepStrictEqual(defaultParams, {});

  // Full options
  const fullParams = buildAdminBookingParams({
    page: 2,
    limit: 25,
    organizer_id: "org-uuid-1",
    event_id: "evt-uuid-1",
    booking_status: "CONFIRMED",
    payment_status: "SUCCESS",
    search: "BK-100",
    from: "2026-01-01",
    to: "2026-01-31",
  });

  assert.deepStrictEqual(fullParams, {
    page: 2,
    limit: 25,
    organizer_id: "org-uuid-1",
    event_id: "evt-uuid-1",
    booking_status: "CONFIRMED",
    payment_status: "SUCCESS",
    search: "BK-100",
    from: "2026-01-01",
    to: "2026-01-31",
  });

  // Ignores ALL options
  const allParams = buildAdminBookingParams({
    page: 1,
    limit: 10,
    organizer_id: "ALL",
    event_id: "ALL",
    booking_status: "ALL",
    payment_status: "ALL",
  });

  assert.deepStrictEqual(allParams, {
    page: 1,
    limit: 10,
  });
});

test("buildAdminAttendeeParams normalizes query options correctly", () => {
  const params = buildAdminAttendeeParams({
    page: 3,
    limit: 50,
    organizer_id: "org-1",
    event_id: "evt-1",
    check_in_status: "USED",
    search: "John",
  });

  assert.deepStrictEqual(params, {
    page: 3,
    limit: 50,
    organizer_id: "org-1",
    event_id: "evt-1",
    check_in_status: "USED",
    search: "John",
  });

  const emptyParams = buildAdminAttendeeParams({
    organizer_id: "ALL",
    event_id: "ALL",
    check_in_status: "ALL",
  });

  assert.deepStrictEqual(emptyParams, {});
});
