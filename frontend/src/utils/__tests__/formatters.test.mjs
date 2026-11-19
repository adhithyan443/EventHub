import { test, describe } from "node:test";
import assert from "node:assert/strict";
import {
  formatCurrency,
  formatDateTime,
  formatRelativeTime,
} from "../formatters.js";

describe("formatCurrency", () => {
  test("formats number correctly into INR currency format", () => {
    const formatted = formatCurrency(12500.5);
    // INR formatting contains ₹ and commas
    assert.ok(formatted.includes("12,500.50"));
  });

  test("formats decimal string correctly into INR", () => {
    const formatted = formatCurrency("10800.00");
    assert.ok(formatted.includes("10,800.00"));
  });

  test("handles 0 correctly", () => {
    const formatted = formatCurrency(0);
    assert.ok(formatted.includes("0.00"));
  });

  test("handles null, undefined, and empty string by returning placeholder", () => {
    assert.equal(formatCurrency(null), "—");
    assert.equal(formatCurrency(undefined), "—");
    assert.equal(formatCurrency(""), "—");
  });

  test("handles invalid non-numeric strings by returning placeholder", () => {
    assert.equal(formatCurrency("invalid_number"), "—");
  });
});

describe("formatDateTime", () => {
  test("formats valid ISO timestamp into date and time", () => {
    const formatted = formatDateTime("2026-10-15T14:30:00Z");
    assert.notEqual(formatted, "—");
    assert.ok(formatted.includes("2026"));
  });

  test("handles null and undefined gracefully", () => {
    assert.equal(formatDateTime(null), "—");
    assert.equal(formatDateTime(undefined), "—");
    assert.equal(formatDateTime(""), "—");
  });

  test("handles invalid date strings gracefully", () => {
    assert.equal(formatDateTime("not-a-real-date"), "—");
  });
});

describe("formatRelativeTime", () => {
  test("handles just now for recent seconds", () => {
    const now = new Date();
    const formatted = formatRelativeTime(now.toISOString());
    assert.equal(formatted, "Just now");
  });

  test("handles minutes ago", () => {
    const fiveMinsAgo = new Date(Date.now() - 5 * 60 * 1000);
    const formatted = formatRelativeTime(fiveMinsAgo.toISOString());
    assert.equal(formatted, "5m ago");
  });

  test("handles hours ago", () => {
    const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60 * 1000);
    const formatted = formatRelativeTime(twoHoursAgo.toISOString());
    assert.equal(formatted, "2h ago");
  });

  test("handles invalid inputs gracefully", () => {
    assert.equal(formatRelativeTime(null), "—");
    assert.equal(formatRelativeTime(undefined), "—");
    assert.equal(formatRelativeTime("bad-date"), "—");
  });
});
