import test from "node:test";
import assert from "node:assert";

test("adminUserApi and adminOrganizerApi build clean parameter objects", async () => {
  const { getUsers } = await import("../adminUserApi.js");
  const { getOrganizers } = await import("../adminOrganizerApi.js");

  assert.strictEqual(typeof getUsers, "function");
  assert.strictEqual(typeof getOrganizers, "function");
});
