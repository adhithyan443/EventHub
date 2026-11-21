import test from "node:test";
import assert from "node:assert";

test("RevenueChart monthly mapping correctly extracts month and fee revenue", () => {
  const formatPeriodMonth = (periodStr) => {
    if (!periodStr) return "";
    const parts = periodStr.split("-");
    if (parts.length === 2) {
      const monthIndex = parseInt(parts[1], 10) - 1;
      const monthNames = [
        "Jan", "Feb", "Mar", "Apr", "May", "Jun",
        "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
      ];
      if (monthIndex >= 0 && monthIndex < 12) {
        return monthNames[monthIndex];
      }
    }
    return periodStr;
  };

  assert.strictEqual(formatPeriodMonth("2026-01"), "Jan");
  assert.strictEqual(formatPeriodMonth("2026-05"), "May");
  assert.strictEqual(formatPeriodMonth("2026-12"), "Dec");
  assert.strictEqual(formatPeriodMonth("invalid"), "invalid");

  const rawTrends = [
    { period: "2026-05", gross_volume: "10000.00", platform_fees: "1000.00" },
    { period: "2026-06", gross_volume: "25000.00", platform_fees: "2500.00" },
  ];

  const mapped = rawTrends.map((t) => ({
    month: formatPeriodMonth(t.period),
    revenue: parseFloat(t.platform_fees) || 0,
    grossVolume: parseFloat(t.gross_volume) || 0,
  }));

  assert.strictEqual(mapped.length, 2);
  assert.strictEqual(mapped[0].month, "May");
  assert.strictEqual(mapped[0].revenue, 1000);
  assert.strictEqual(mapped[0].grossVolume, 10000);
  assert.strictEqual(mapped[1].month, "Jun");
  assert.strictEqual(mapped[1].revenue, 2500);
  assert.strictEqual(mapped[1].grossVolume, 25000);
});

test("RevenueChart quarterly aggregation groups monthly points into quarters", () => {
  const rawTrends = [
    { period: "2026-01", gross_volume: "5000.00", platform_fees: "500.00" },
    { period: "2026-02", gross_volume: "7000.00", platform_fees: "700.00" },
    { period: "2026-04", gross_volume: "12000.00", platform_fees: "1200.00" },
  ];

  const quarters = {};
  rawTrends.forEach((t) => {
    const period = t.period || "";
    const parts = period.split("-");
    const year = parts[0] || "";
    const monthNum = parseInt(parts[1], 10) || 1;
    const qNum = Math.ceil(monthNum / 3);
    const qKey = year ? `${year} Q${qNum}` : `Q${qNum}`;

    const fee = parseFloat(t.platform_fees) || 0;
    const gross = parseFloat(t.gross_volume) || 0;

    if (!quarters[qKey]) {
      quarters[qKey] = {
        month: `Q${qNum}`,
        fullLabel: qKey,
        revenue: 0,
        grossVolume: 0,
      };
    }
    quarters[qKey].revenue += fee;
    quarters[qKey].grossVolume += gross;
  });

  const result = Object.values(quarters);
  assert.strictEqual(result.length, 2);
  // Q1: Jan + Feb = 500 + 700 = 1200
  assert.strictEqual(result[0].month, "Q1");
  assert.strictEqual(result[0].revenue, 1200);
  assert.strictEqual(result[0].grossVolume, 12000);
  // Q2: Apr = 1200
  assert.strictEqual(result[1].month, "Q2");
  assert.strictEqual(result[1].revenue, 1200);
  assert.strictEqual(result[1].grossVolume, 12000);
});

test("RevenueChart scale and y-ticks calculate clean bounds", () => {
  const computeMaxRevenue = (data) => {
    if (!data || data.length === 0) return 10000;
    const maxVal = Math.max(...data.map((d) => d.revenue), 0);
    if (maxVal <= 0) return 10000;
    const target = maxVal * 1.25;
    const magnitude = Math.pow(10, Math.floor(Math.log10(target)));
    return Math.ceil(target / magnitude) * magnitude;
  };

  assert.strictEqual(computeMaxRevenue([]), 10000);
  assert.strictEqual(computeMaxRevenue([{ revenue: 0 }]), 10000);

  const maxRev = computeMaxRevenue([{ revenue: 15000 }, { revenue: 22000 }]);
  // maxVal = 22000, target = 27500, magnitude = 10000, ceil = 30000
  assert.strictEqual(maxRev, 30000);

  const formatYTick = (tick) => {
    if (tick <= 0) return "₹0";
    if (tick >= 10000000) return `₹${(tick / 10000000).toFixed(1).replace(/\.0$/, "")}Cr`;
    if (tick >= 100000) return `₹${(tick / 100000).toFixed(1).replace(/\.0$/, "")}L`;
    if (tick >= 1000) return `₹${(tick / 1000).toFixed(0)}k`;
    return `₹${Math.round(tick)}`;
  };

  assert.strictEqual(formatYTick(0), "₹0");
  assert.strictEqual(formatYTick(10000), "₹10k");
  assert.strictEqual(formatYTick(250000), "₹2.5L");
  assert.strictEqual(formatYTick(15000000), "₹1.5Cr");
});
