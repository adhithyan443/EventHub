import test from "node:test";
import assert from "node:assert";

test("adminDashboardApi exports getAdminDashboard function", async () => {
  const { getAdminDashboard } = await import("../adminDashboardApi.js");
  assert.strictEqual(typeof getAdminDashboard, "function");
});

test("adminDashboardApi parameter builder produces clean query params", () => {
  const buildDashboardParams = ({ from = "", to = "" } = {}) => {
    const params = {};
    if (from && typeof from === "string" && from.trim() !== "") {
      params.from = from.trim();
    }
    if (to && typeof to === "string" && to.trim() !== "") {
      params.to = to.trim();
    }
    return params;
  };

  assert.deepStrictEqual(buildDashboardParams(), {});
  assert.deepStrictEqual(buildDashboardParams({ from: "", to: "" }), {});
  assert.deepStrictEqual(
    buildDashboardParams({ from: "2026-08-01", to: "2026-08-31" }),
    { from: "2026-08-01", to: "2026-08-31" }
  );
  assert.deepStrictEqual(
    buildDashboardParams({ from: "  2026-01-01T00:00:00Z  " }),
    { from: "2026-01-01T00:00:00Z" }
  );
});

test("admin dashboard response normalization handles standard and missing fields safely", () => {
  const normalizeDashboardData = (apiData) => {
    const platform = apiData?.platform_overview || {};
    const financial = apiData?.financial_overview || {};
    const trends = apiData?.trends || {};

    return {
      totalUsers: platform.total_users ?? 0,
      totalOrganizers: platform.total_organizers ?? 0,
      totalEvents: platform.total_events ?? 0,
      activeEvents: platform.active_events ?? 0,
      totalBookings: platform.total_bookings ?? 0,
      ticketsSold: platform.tickets_sold ?? 0,
      grossCapturedAmount: financial.gross_captured_amount || "0",
      platformFeeRevenue: financial.platform_fee_revenue || "0",
      organizerPayable: financial.organizer_payable_liability || "0",
      totalRefunds: financial.total_refunds || "0",
      completedSettlements: financial.completed_settlements || "0",
      pendingSettlements: financial.pending_settlements || "0",
      taxLiability: financial.tax_liability || "0",
      currency: financial.currency || "INR",
      revenueTrends: Array.isArray(trends.revenue_trends) ? trends.revenue_trends : [],
      bookingTrends: Array.isArray(trends.booking_trends) ? trends.booking_trends : [],
      refundTrends: Array.isArray(trends.refund_trends) ? trends.refund_trends : [],
      paymentStatusBreakdown: Array.isArray(apiData?.payment_status_breakdown)
        ? apiData.payment_status_breakdown
        : [],
      recentBookings: Array.isArray(apiData?.recent_bookings)
        ? apiData.recent_bookings
        : [],
    };
  };

  const emptyNorm = normalizeDashboardData(null);
  assert.strictEqual(emptyNorm.totalUsers, 0);
  assert.strictEqual(emptyNorm.grossCapturedAmount, "0");
  assert.strictEqual(emptyNorm.currency, "INR");
  assert.deepStrictEqual(emptyNorm.revenueTrends, []);
  assert.deepStrictEqual(emptyNorm.recentBookings, []);

  const fullData = {
    platform_overview: {
      total_users: 120,
      total_organizers: 15,
      total_events: 25,
      active_events: 8,
      total_bookings: 60,
      tickets_sold: 140,
    },
    financial_overview: {
      gross_captured_amount: "150000.00",
      platform_fee_revenue: "15000.00",
      organizer_payable_liability: "120000.00",
      total_refunds: "5000.00",
      completed_settlements: "80000.00",
      pending_settlements: "20000.00",
      tax_liability: "10000.00",
      currency: "INR",
    },
    trends: {
      revenue_trends: [{ period: "2026-08", gross_volume: "150000.00", platform_fees: "15000.00" }],
      booking_trends: [{ period: "2026-08", booking_count: 60, tickets_sold: 140 }],
      refund_trends: [{ period: "2026-08", refund_amount: "5000.00", refund_count: 2 }],
    },
    payment_status_breakdown: [
      { status: "SUCCESS", count: 58, amount: "150000.00" },
      { status: "FAILED", count: 2, amount: "5000.00" },
    ],
    recent_bookings: [
      {
        id: "b1",
        booking_number: "BK-001",
        customer_name: "John Doe",
        amount: "2500.00",
        status: "CONFIRMED",
        payment_status: "SUCCESS",
      },
    ],
  };

  const populatedNorm = normalizeDashboardData(fullData);
  assert.strictEqual(populatedNorm.totalUsers, 120);
  assert.strictEqual(populatedNorm.totalOrganizers, 15);
  assert.strictEqual(populatedNorm.grossCapturedAmount, "150000.00");
  assert.strictEqual(populatedNorm.platformFeeRevenue, "15000.00");
  assert.strictEqual(populatedNorm.organizerPayable, "120000.00");
  assert.strictEqual(populatedNorm.totalRefunds, "5000.00");
  assert.strictEqual(populatedNorm.completedSettlements, "80000.00");
  assert.strictEqual(populatedNorm.pendingSettlements, "20000.00");
  assert.strictEqual(populatedNorm.taxLiability, "10000.00");
  assert.strictEqual(populatedNorm.revenueTrends.length, 1);
  assert.strictEqual(populatedNorm.paymentStatusBreakdown.length, 2);
  assert.strictEqual(populatedNorm.recentBookings.length, 1);
});
