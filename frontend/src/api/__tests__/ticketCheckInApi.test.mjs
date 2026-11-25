import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { checkInOrganizerTicket } from "../organizerApi.js";

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
