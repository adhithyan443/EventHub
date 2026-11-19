import { describe, it } from "node:test";
import assert from "node:assert/strict";
import {
  buildOrganizerSettlementParams,
  buildSettlementRequestPayload,
} from "../../api/organizerSettlementApi.js";
import {
  buildAdminSettlementParams,
} from "../../api/adminSettlementApi.js";

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
