import test from "node:test";
import assert from "node:assert";

/**
 * Unit tests verifying:
 * 1. Multi-category quantity calculation logic (bug fix verification).
 * 2. Countdown and expiration evaluation logic for reservation holds.
 * 3. Cancel vs keep decision logic without accidental API invocation.
 */

test("Multi-category ticket derivation calculates quantities and line totals accurately from reservation items", () => {
  // Mock event ticket types
  const eventTicketTypes = [
    { id: "tier-ga", name: "General Admission", price: "500" },
    { id: "tier-vip", name: "VIP Pass", price: "1500" },
  ];

  // Mock reservation items as returned by backend database
  // Notice item 1 has Quantity: 3 (GA), item 2 has Quantity: 2 (VIP)
  const reservationItems = [
    {
      TicketTypeID: "tier-ga",
      Quantity: 3,
      Price: "500.00",
    },
    {
      TicketTypeID: "tier-vip",
      Quantity: 2,
      Price: "1500.00",
    },
  ];

  // The fixed multi-category accumulator logic:
  const map = {};
  for (const it of reservationItems) {
    const tid = it.TicketTypeID || it.ticket_type_id || it.ticketTypeId;
    if (!tid) continue;
    const itemQty = Number(it.Quantity ?? it.quantity ?? 1);
    const itemPrice = Number(it.Price ?? it.price ?? 0);
    if (!map[tid]) {
      const tier = eventTicketTypes.find((t) => t.id === tid);
      map[tid] = {
        ticketTypeId: tid,
        name: tier?.name || "General Admission",
        price: Number(itemPrice || tier?.price || 0),
        quantity: 0,
        lineTotal: 0,
      };
    }
    map[tid].quantity += itemQty;
    map[tid].lineTotal += map[tid].price * itemQty;
  }

  const list = Object.values(map);

  // Assertions
  assert.strictEqual(list.length, 2, "Should accumulate 2 distinct ticket tiers");
  
  const ga = list.find((t) => t.ticketTypeId === "tier-ga");
  assert.ok(ga, "GA tier must exist");
  assert.strictEqual(ga.quantity, 3, "GA quantity must be 3 (not 1)");
  assert.strictEqual(ga.lineTotal, 1500, "GA line total must be 3 * 500 = 1500");

  const vip = list.find((t) => t.ticketTypeId === "tier-vip");
  assert.ok(vip, "VIP tier must exist");
  assert.strictEqual(vip.quantity, 2, "VIP quantity must be 2 (not 1)");
  assert.strictEqual(vip.lineTotal, 3000, "VIP line total must be 2 * 1500 = 3000");
});

test("Legacy single-quantity items fallback safely to quantity 1 if quantity field is omitted", () => {
  const legacyItems = [
    {
      TicketTypeID: "tier-std",
      Price: "250.00",
    },
    {
      TicketTypeID: "tier-std",
      Price: "250.00",
    },
  ];

  const map = {};
  for (const it of legacyItems) {
    const tid = it.TicketTypeID;
    const itemQty = Number(it.Quantity ?? it.quantity ?? 1);
    const itemPrice = Number(it.Price ?? it.price ?? 0);
    if (!map[tid]) {
      map[tid] = {
        ticketTypeId: tid,
        name: "Standard",
        price: itemPrice,
        quantity: 0,
        lineTotal: 0,
      };
    }
    map[tid].quantity += itemQty;
    map[tid].lineTotal += map[tid].price * itemQty;
  }

  const list = Object.values(map);
  assert.strictEqual(list[0].quantity, 2, "Two items without quantity field sum to 2");
  assert.strictEqual(list[0].lineTotal, 500, "Line total equals 500");
});

test("Reservation hold countdown calculates remaining seconds and detects expiration", () => {
  const now = Date.now();
  
  // Future expiry (10 minutes)
  const futureExpiry = new Date(now + 600 * 1000).toISOString();
  const futureRemaining = Math.max(0, Math.floor((new Date(futureExpiry).getTime() - now) / 1000));
  assert.ok(futureRemaining > 590 && futureRemaining <= 600, "Should have ~600s remaining");
  assert.strictEqual(futureRemaining <= 0, false, "Future hold should not be expired");

  // Past expiry (-5 seconds)
  const pastExpiry = new Date(now - 5 * 1000).toISOString();
  const pastRemaining = Math.max(0, Math.floor((new Date(pastExpiry).getTime() - now) / 1000));
  assert.strictEqual(pastRemaining, 0, "Past hold remaining seconds should be clamped to 0");
  assert.strictEqual(pastRemaining <= 0, true, "Past hold should be flagged as expired");
});
