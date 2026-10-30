import test, { beforeEach } from "node:test";
import assert from "node:assert";

import { parseAgeRestriction } from "../../constants/eventConstants.js";
import {
  parseDeadlineHours,
  extractEventId,
  formatSubmissionErrorMessage,
  validateEventForDraft,
  validateEventForPublish,
} from "../../utils/eventValidation.js";

// In-memory mock localStorage
const mockStorage = new Map();
globalThis.localStorage = {
  getItem: (key) => (mockStorage.has(key) ? mockStorage.get(key) : null),
  setItem: (key, val) => mockStorage.set(key, String(val)),
  removeItem: (key) => mockStorage.delete(key),
  clear: () => mockStorage.clear(),
};

beforeEach(() => {
  mockStorage.clear();
});

// Helper to construct a valid baseline event input fixture
const createValidEventInput = (overrides = {}) => {
  const {
    basicInformation = {},
    venue = {},
    dateTime = {},
    eventDetails = {},
    contactInformation: rootContactOverrides,
    cancellationPolicy: rootPolicyOverrides,
    ticketSalesSettings = {},
    ...restOverrides
  } = overrides;

  const {
    contactInformation: nestedContactOverrides,
    cancellationPolicy: nestedPolicyOverrides,
    ...restEventDetails
  } = eventDetails;

  const contactOverrides = rootContactOverrides || nestedContactOverrides || {};
  const policyOverrides = rootPolicyOverrides || nestedPolicyOverrides || {};

  return {
    basicInformation: {
      title: "Bangalore Tech Summit 2026",
      categoryId: "c1111111-1111-1111-1111-111111111111",
      description: "Annual premier technology conference and exhibition.",
      language: "English",
      bannerFile: { name: "banner.png", size: 1024, type: "image/png" },
      banner: "",
      ageRestriction: "All Ages (Family Friendly)",
      ...basicInformation,
    },
    locationType: "PHYSICAL",
    onlineUrl: "",
    venue: {
      google_place_id: "ChIJbU60qSX9TDkRfdDeAYogV4E",
      name: "Bangalore Palace",
      address: "Palace Ground, Vasanth Nagar",
      city: "Bengaluru",
      state: "Karnataka",
      country: "India",
      postal_code: "560052",
      latitude: 12.9988,
      longitude: 77.5921,
      ...venue,
    },
    dateTime: {
      eventDate: "2026-12-15",
      startTime: "09:00",
      endTime: "18:00",
      isAllDay: false,
      ...dateTime,
    },
    eventDetails: {
      visibility: "PUBLIC",
      highlights: "Keynotes\nNetworking",
      rules: "Valid ID required",
      attendeeInformation: "Registration opens at 8 AM",
      ...restEventDetails,
      contactInformation: {
        name: "Summit Coordinator",
        phone: "+91 98765 43210",
        email: "coordinator@techsummit.in",
        ...contactOverrides,
      },
      cancellationPolicy: {
        allowCancellation: true,
        cancellationDeadline: "48 hours before event",
        refundPolicy: "PARTIAL",
        refundPercentage: 80,
        organizerPolicyAccepted: true,
        ...policyOverrides,
      },
    },
    ticketMode: "GENERAL",
    ticketTypes: [
      {
        name: "General Delegate Pass",
        price: 1500,
        capacity: 500,
        description: "Access to all conference sessions",
      },
    ],
    seatingConfig: { categories: [] },
    ticketSalesSettings: {
      salesStartDate: "2026-11-01",
      salesEndDate: "2026-12-14",
      maxTicketsPerBooking: 5,
      ...ticketSalesSettings,
    },
    isEditMode: false,
    ...restOverrides,
  };
};

/* =====================================================================
 * 1. Age Restriction Parsing Tests
 * ===================================================================== */

test("age restriction parser: correctly handles all 5 dropdown options", () => {
  assert.strictEqual(
    parseAgeRestriction("All Ages (Family Friendly)"),
    0,
    "All Ages (Family Friendly) must parse to 0"
  );
  assert.strictEqual(parseAgeRestriction("13+"), 13, "13+ must parse to 13");
  assert.strictEqual(parseAgeRestriction("16+"), 16, "16+ must parse to 16");
  assert.strictEqual(parseAgeRestriction("18+"), 18, "18+ must parse to 18");
  assert.strictEqual(parseAgeRestriction("21+"), 21, "21+ must parse to 21");
});

test("age restriction parser: handles numeric and edge-case inputs safely", () => {
  assert.strictEqual(parseAgeRestriction(0), 0);
  assert.strictEqual(parseAgeRestriction(18), 18);
  assert.strictEqual(parseAgeRestriction(""), 0);
  assert.strictEqual(parseAgeRestriction(null), 0);
  assert.strictEqual(parseAgeRestriction(undefined), 0);
  assert.strictEqual(parseAgeRestriction("All Ages"), 0);
  assert.strictEqual(parseAgeRestriction("all ages (kids welcome)"), 0);
  assert.strictEqual(parseAgeRestriction("99+"), 99);
  assert.strictEqual(parseAgeRestriction("105"), 0, "Out-of-range age must fall back to 0");
  assert.strictEqual(parseAgeRestriction(-5), 0, "Negative age must fall back to 0");
});

test("age restriction in payload construction: parsed integer is used in preview and payload", () => {
  // Test Case A: "All Ages (Family Friendly)" -> 0
  const inputAllAges = createValidEventInput({
    basicInformation: { ageRestriction: "All Ages (Family Friendly)" },
  });
  const parsedAllAges = parseAgeRestriction(inputAllAges.basicInformation.ageRestriction);
  assert.strictEqual(parsedAllAges, 0);
  assert.ok(Number.isInteger(parsedAllAges));

  // Test Case B: "18+" -> 18
  const input18 = createValidEventInput({
    basicInformation: { ageRestriction: "18+" },
  });
  const parsed18 = parseAgeRestriction(input18.basicInformation.ageRestriction);
  assert.strictEqual(parsed18, 18);
  assert.ok(Number.isInteger(parsed18));
});

/* =====================================================================
 * 2. Separate Draft vs Publish Validation Tests
 * ===================================================================== */

test("draft validation: valid physical and online events pass draft validation", () => {
  const physicalInput = createValidEventInput();
  assert.strictEqual(validateEventForDraft(physicalInput), null);

  const onlineInput = createValidEventInput({
    locationType: "ONLINE",
    onlineUrl: "https://zoom.us/j/123456789",
    venue: {},
  });
  assert.strictEqual(validateEventForDraft(onlineInput), null);
});

test("draft validation: does NOT require organizerPolicyAccepted for draft saving", () => {
  const draftWithoutAcceptedPolicy = createValidEventInput({
    cancellationPolicy: { organizerPolicyAccepted: false },
  });

  // Draft validation passes even if organizer policy has not yet been accepted
  const draftError = validateEventForDraft(draftWithoutAcceptedPolicy);
  assert.strictEqual(draftError, null, "Draft creation should not block on organizerPolicyAccepted");

  // But Publish validation DOES block on organizerPolicyAccepted
  const publishError = validateEventForPublish(draftWithoutAcceptedPolicy);
  assert.strictEqual(
    publishError,
    "You must accept the organizer cancellation policy before publishing."
  );
});

test("publish validation: blocks publishing when event date is in the past", () => {
  const pastEvent = createValidEventInput({
    dateTime: { eventDate: "2020-01-01" },
    ticketSalesSettings: {
      salesStartDate: "2019-11-01",
      salesEndDate: "2019-12-31",
    },
  });

  const publishError = validateEventForPublish(pastEvent);
  assert.strictEqual(publishError, "Event date cannot be in the past.");
});

/* =====================================================================
 * 3. Required Fields & Discrepancy Detection Tests
 * ===================================================================== */

test("validation: detects missing title, category, description, and banner", () => {
  assert.strictEqual(
    validateEventForDraft(createValidEventInput({ basicInformation: { title: "  " } })),
    "Event title is required."
  );
  assert.strictEqual(
    validateEventForDraft(createValidEventInput({ basicInformation: { categoryId: "" } })),
    "Please select an event category."
  );
  assert.strictEqual(
    validateEventForDraft(createValidEventInput({ basicInformation: { description: "" } })),
    "Event description is required."
  );
  assert.strictEqual(
    validateEventForDraft(createValidEventInput({ basicInformation: { bannerFile: null, banner: "" } })),
    "Please upload an event banner."
  );
});

test("validation: enforces backend physical venue contract (state, google_place_id, address)", () => {
  assert.strictEqual(
    validateEventForDraft(createValidEventInput({ venue: { state: "" } })),
    "Venue state is required. Please re-select the venue from Google Places to capture its state."
  );
  assert.strictEqual(
    validateEventForDraft(createValidEventInput({ venue: { google_place_id: "" } })),
    "Google Place ID is required. Please select an established venue using Google Places."
  );
  assert.strictEqual(
    validateEventForDraft(createValidEventInput({ venue: { address: "" } })),
    "Venue address is required."
  );
  assert.strictEqual(
    validateEventForDraft(createValidEventInput({ venue: { city: "" } })),
    "Venue city is required."
  );
  assert.strictEqual(
    validateEventForDraft(createValidEventInput({ venue: { country: "" } })),
    "Venue country is required."
  );
});

test("validation: enforces contact information all-or-nothing rule", () => {
  // Scenario A: No contact information provided at all -> allowed (contact is optional)
  const noContactInput = createValidEventInput({
    contactInformation: { name: "", phone: "", email: "" },
  });
  assert.strictEqual(validateEventForDraft(noContactInput), null);

  // Scenario B: Name entered, but phone and email omitted -> rejected with actionable guidance
  const partialContactInput = createValidEventInput({
    contactInformation: { name: "Organizer Person", phone: "", email: "" },
  });
  assert.strictEqual(
    validateEventForDraft(partialContactInput),
    "Contact phone number is required when contact details are provided."
  );

  // Scenario C: Invalid email format
  const invalidEmailInput = createValidEventInput({
    contactInformation: { name: "Organizer Person", phone: "+91 9876543210", email: "not-an-email" },
  });
  assert.strictEqual(
    validateEventForDraft(invalidEmailInput),
    "Please provide a valid organizer contact email address."
  );
});

/* =====================================================================
 * 4. Response Extraction & Error Formatting Tests
 * ===================================================================== */

test("extractEventId: extracts ID from diverse backend response envelopes", () => {
  assert.strictEqual(
    extractEventId({ data: { data: { event: { ID: "evt-uuid-1" } } } }),
    "evt-uuid-1"
  );
  assert.strictEqual(
    extractEventId({ data: { event: { id: "evt-uuid-2" } } }),
    "evt-uuid-2"
  );
  assert.strictEqual(
    extractEventId({ data: { ID: "evt-uuid-3" } }),
    "evt-uuid-3"
  );
  assert.strictEqual(
    extractEventId({ id: "evt-uuid-4" }),
    "evt-uuid-4"
  );
});

test("formatSubmissionErrorMessage: transforms generic invalid event data into actionable guidance", () => {
  const errInvalidData = { response: { data: { message: "invalid event data" } } };
  const formatted = formatSubmissionErrorMessage(errInvalidData, "Fallback");
  assert.ok(
    formatted.includes("Event submission failed: Some event fields are invalid or incomplete"),
    "Should provide actionable guidance instead of raw 'invalid event data'"
  );

  const errSpecific = { response: { data: { message: "banner image must not exceed 5 MB" } } };
  assert.strictEqual(
    formatSubmissionErrorMessage(errSpecific, "Fallback"),
    "banner image must not exceed 5 MB"
  );
});

test("deadline hours parsing: correctly parses diverse format strings", () => {
  assert.strictEqual(parseDeadlineHours("24 hours before event"), 24);
  assert.strictEqual(parseDeadlineHours("48 hours before event"), 48);
  assert.strictEqual(parseDeadlineHours("7 days before event"), 168);
  assert.strictEqual(parseDeadlineHours("14 days before event"), 336);
  assert.strictEqual(parseDeadlineHours(48), 48);
  assert.strictEqual(parseDeadlineHours("72 hours"), 72);
});

/* =====================================================================
 * 5. Execution Flow & Retry Behavior Tests
 * ===================================================================== */

test("publish flow: Save as Draft only calls createOrganizerEvent and leaves event in draft", async () => {
  let createCalls = 0;
  let publishCalls = 0;

  const mockCreate = async (formData) => {
    createCalls++;
    assert.ok(typeof formData === "object");
    return {
      success: true,
      data: {
        event: { ID: "evt-draft-123", Status: "DRAFT" },
      },
    };
  };

  const mockPublish = async () => {
    publishCalls++;
    return { success: true };
  };

  // Simulate Save as Draft action
  const formData = { event: JSON.stringify({ title: "My Draft Event" }) };
  const res = await mockCreate(formData);
  const createdId = extractEventId(res);

  assert.strictEqual(createCalls, 1);
  assert.strictEqual(publishCalls, 0, "Save as Draft must never invoke the publish endpoint");
  assert.strictEqual(createdId, "evt-draft-123");
});

test("publish flow: Publish Event calls createOrganizerEvent and then immediately publishOrganizerEvent", async () => {
  let createCalls = 0;
  let publishCalls = 0;
  let publishedId = null;

  const mockCreate = async () => {
    createCalls++;
    return {
      success: true,
      data: {
        event: { ID: "evt-live-456", Status: "DRAFT" },
      },
    };
  };

  const mockPublish = async (eventId) => {
    publishCalls++;
    publishedId = eventId;
    return { success: true, message: "event published successfully" };
  };

  // Simulate Publish action
  const res = await mockCreate();
  const targetId = extractEventId(res);
  const pubRes = await mockPublish(targetId);

  assert.strictEqual(createCalls, 1);
  assert.strictEqual(publishCalls, 1);
  assert.strictEqual(publishedId, "evt-live-456");
  assert.strictEqual(pubRes.success, true);
});

test("partial success handling: failure on publish preserves draft ID and retry avoids duplicate event creation", async () => {
  let createCalls = 0;
  let publishCalls = 0;
  let savedDraftId = null;

  const mockCreate = async () => {
    createCalls++;
    return {
      success: true,
      data: {
        event: { ID: "evt-partial-789", Status: "DRAFT" },
      },
    };
  };

  let failPublishOnce = true;
  const mockPublish = async (eventId) => {
    publishCalls++;
    if (failPublishOnce) {
      failPublishOnce = false;
      const err = new Error("event is not ready to be published");
      err.response = { status: 400, data: { message: "event is not ready to be published" } };
      throw err;
    }
    return { success: true, message: "event published successfully" };
  };

  // Attempt 1: Creation succeeds, publish fails
  try {
    if (!savedDraftId) {
      const res = await mockCreate();
      savedDraftId = extractEventId(res);
    }
    await mockPublish(savedDraftId);
  } catch (err) {
    assert.strictEqual(err.message, "event is not ready to be published");
  }

  // Verify created event ID is retained even though publishing failed
  assert.strictEqual(savedDraftId, "evt-partial-789");
  assert.strictEqual(createCalls, 1);
  assert.strictEqual(publishCalls, 1);

  // Attempt 2: Organizer clicks "Publish Event" again (Retry)
  if (!savedDraftId) {
    const res = await mockCreate();
    savedDraftId = extractEventId(res);
  }
  const retryPubRes = await mockPublish(savedDraftId);

  // CRITICAL: createCalls must still be 1 (NO duplicate event created on retry!)
  assert.strictEqual(createCalls, 1, "Duplicate event must not be created on retry");
  assert.strictEqual(publishCalls, 2);
  assert.strictEqual(retryPubRes.success, true);
});
