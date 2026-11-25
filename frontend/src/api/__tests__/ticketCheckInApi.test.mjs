import { describe, it } from "node:test";
import assert from "node:assert/strict";
import apiClient from "../client.js";
import {
  checkInOrganizerTicket,
  lookupManualBookingCheckIn,
  confirmManualBookingCheckIn,
} from "../organizerApi.js";

describe("checkInOrganizerTicket API", () => {
  it("throws validation error if eventId is missing", async () => {
    await assert.rejects(
      async () => {
        await checkInOrganizerTicket("", "test-payload");
      },
      {
        message: "Event ID is required for ticket check-in.",
      }
    );
  });

  it("throws validation error if qrPayload is missing or whitespace", async () => {
    await assert.rejects(
      async () => {
        await checkInOrganizerTicket("event-123", "   ");
      },
      {
        message: "QR payload is required for ticket check-in.",
      }
    );
  });
});

describe("lookupManualBookingCheckIn API", () => {
  it("throws validation error if eventId is missing", async () => {
    await assert.rejects(
      async () => {
        await lookupManualBookingCheckIn("", "EVH-12345678-abc");
      },
      {
        message: "Event ID is required for manual check-in lookup.",
      }
    );
  });

  it("throws validation error if bookingReference is missing or whitespace", async () => {
    await assert.rejects(
      async () => {
        await lookupManualBookingCheckIn("event-123", "   ");
      },
      {
        message: "Booking reference is required for manual check-in lookup.",
      }
    );
  });

  it("makes POST request to /organizers/events/:id/check-in/manual/lookup with trimmed reference", async () => {
    let capturedUrl = "";
    let capturedBody = null;
    const originalPost = apiClient.post;
    apiClient.post = async (url, body) => {
      capturedUrl = url;
      capturedBody = body;
      return { data: { success: true, data: { is_eligible: true } } };
    };

    try {
      const result = await lookupManualBookingCheckIn("event-xyz", "  EVH-949d362c-266  ");
      assert.strictEqual(capturedUrl, "/organizers/events/event-xyz/check-in/manual/lookup");
      assert.deepStrictEqual(capturedBody, { booking_reference: "EVH-949d362c-266" });
      assert.strictEqual(result.success, true);
    } finally {
      apiClient.post = originalPost;
    }
  });

  it("preserves lowercase and mixed case references without forced upper-casing", async () => {
    let capturedBody = null;
    const originalPost = apiClient.post;
    apiClient.post = async (url, body) => {
      capturedBody = body;
      return { data: { success: true } };
    };

    try {
      await lookupManualBookingCheckIn("event-xyz", "  evh-377e95eb-afe  ");
      assert.deepStrictEqual(capturedBody, { booking_reference: "evh-377e95eb-afe" });
    } finally {
      apiClient.post = originalPost;
    }
  });
});

describe("confirmManualBookingCheckIn API", () => {
  it("throws validation error if eventId is missing", async () => {
    await assert.rejects(
      async () => {
        await confirmManualBookingCheckIn("", "EVH-12345678-abc");
      },
      {
        message: "Event ID is required for manual check-in confirmation.",
      }
    );
  });

  it("throws validation error if bookingReference is missing or whitespace", async () => {
    await assert.rejects(
      async () => {
        await confirmManualBookingCheckIn("event-123", "   ");
      },
      {
        message: "Booking reference is required for manual check-in confirmation.",
      }
    );
  });

  it("makes POST request to /organizers/events/:id/check-in/manual/confirm with trimmed reference", async () => {
    let capturedUrl = "";
    let capturedBody = null;
    const originalPost = apiClient.post;
    apiClient.post = async (url, body) => {
      capturedUrl = url;
      capturedBody = body;
      return { data: { success: true, data: { checked_in_count: 2 } } };
    };

    try {
      const result = await confirmManualBookingCheckIn("event-xyz", "  EVH-949d362c-266  ");
      assert.strictEqual(capturedUrl, "/organizers/events/event-xyz/check-in/manual/confirm");
      assert.deepStrictEqual(capturedBody, { booking_reference: "EVH-949d362c-266" });
      assert.strictEqual(result.success, true);
    } finally {
      apiClient.post = originalPost;
    }
  });
});
