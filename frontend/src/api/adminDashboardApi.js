import apiClient from "./client.js";

/**
 * Fetch consolidated dashboard metrics for the admin console.
 *
 * @param {Object} [options={}]
 * @param {string} [options.from=""] - Optional start date (RFC3339 or YYYY-MM-DD)
 * @param {string} [options.to=""] - Optional end date (RFC3339 or YYYY-MM-DD)
 * @returns {Promise<Object>} API response data containing platform_overview, financial_overview, trends, payment_status_breakdown, and recent_bookings
 */
export const getAdminDashboard = async ({ from = "", to = "" } = {}) => {
  const params = {};

  if (from && typeof from === "string" && from.trim() !== "") {
    params.from = from.trim();
  }

  if (to && typeof to === "string" && to.trim() !== "") {
    params.to = to.trim();
  }

  const response = await apiClient.get("/admin/dashboard", {
    params,
  });

  return response.data;
};
