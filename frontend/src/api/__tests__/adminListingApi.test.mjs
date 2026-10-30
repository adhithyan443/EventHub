import test from "node:test";
import assert from "node:assert";

test("adminUserApi, adminOrganizerApi, and adminEventApi build clean parameter objects", async () => {
  const { getUsers } = await import("../adminUserApi.js");
  const { getOrganizers } = await import("../adminOrganizerApi.js");
  const {
    getAdminEvents,
    getAdminEventById,
    updateAdminEvent,
    publishAdminEvent,
    deleteAdminEvent,
    getCategories,
  } = await import("../adminEventApi.js");

  assert.strictEqual(typeof getUsers, "function");
  assert.strictEqual(typeof getOrganizers, "function");
  assert.strictEqual(typeof getAdminEvents, "function");
  assert.strictEqual(typeof getAdminEventById, "function");
  assert.strictEqual(typeof updateAdminEvent, "function");
  assert.strictEqual(typeof publishAdminEvent, "function");
  assert.strictEqual(typeof deleteAdminEvent, "function");
  assert.strictEqual(typeof getCategories, "function");
});

test("category response normalization protects against non-array map crashes", () => {
  // Scenario 1: Standard backend envelope: { success: true, data: { categories: [...] } }
  const standardResponse = {
    success: true,
    data: {
      categories: [
        { id: "cat-1", name: "Music" },
        { id: "cat-2", name: "Technology" },
      ],
    },
  };

  const normalizeCategories = (res) =>
    Array.isArray(res?.data?.categories)
      ? res.data.categories
      : Array.isArray(res?.data)
      ? res.data
      : Array.isArray(res)
      ? res
      : [];

  const normalized1 = normalizeCategories(standardResponse);
  assert.strictEqual(Array.isArray(normalized1), true);
  assert.strictEqual(normalized1.length, 2);
  assert.doesNotThrow(() => normalized1.map((c) => c.name));

  // Scenario 2: Flat data array envelope: { success: true, data: [...] }
  const flatResponse = {
    success: true,
    data: [{ id: "cat-3", name: "Sports" }],
  };
  const normalized2 = normalizeCategories(flatResponse);
  assert.strictEqual(Array.isArray(normalized2), true);
  assert.strictEqual(normalized2.length, 1);
  assert.doesNotThrow(() => normalized2.map((c) => c.name));

  // Scenario 3: Corrupt or unexpected response (null, object without array, undefined)
  const corruptedResponses = [
    null,
    undefined,
    {},
    { data: null },
    { data: {} },
    { data: { categories: null } },
    "error",
  ];

  for (const corrupt of corruptedResponses) {
    const safe = normalizeCategories(corrupt);
    assert.strictEqual(Array.isArray(safe), true);
    assert.strictEqual(safe.length, 0);
    assert.doesNotThrow(() => safe.map((c) => c.name));
  }
});

test("admin event list response normalization handles null slices safely", () => {
  const normalizeEventList = (res) => {
    const events = Array.isArray(res?.data?.events)
      ? res.data.events
      : Array.isArray(res?.data)
      ? res.data
      : [];
    const total = typeof res?.data?.total === "number" ? res.data.total : 0;
    const statusCounts = res?.data?.status_counts || {
      total: 0,
      published: 0,
      draft: 0,
      cancelled: 0,
    };
    return { events, total, statusCounts };
  };

  // Nil slice serialized to null in Go
  const goNilSliceResponse = {
    success: true,
    data: {
      events: null,
      total: 0,
      status_counts: null,
    },
  };

  const result = normalizeEventList(goNilSliceResponse);
  assert.strictEqual(Array.isArray(result.events), true);
  assert.strictEqual(result.events.length, 0);
  assert.strictEqual(result.total, 0);
  assert.strictEqual(result.statusCounts.total, 0);
  assert.doesNotThrow(() => result.events.map((ev) => ev.title));
});
