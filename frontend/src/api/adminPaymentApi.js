import apiClient from "./client.js";

/**
 * Fetch payments for admin console with pagination, search, status, ledger allocation status, event filter, and date range filters.
 *
 * @param {Object} options
 * @param {number} [options.page=1]
 * @param {number} [options.limit=10]
 * @param {string} [options.status=""]
 * @param {string} [options.allocationStatus=""]
 * @param {string} [options.from=""]
 * @param {string} [options.to=""]
 * @param {string} [options.search=""]
 * @param {string} [options.eventId=""]
 * @param {string} [options.event_id=""]
 * @returns {Promise<Object>} API response data
 */
export const getAdminPayments = async ({
  page = 1,
  limit = 10,
  status = "",
  allocationStatus = "",
  from = "",
  to = "",
  search = "",
  eventId = "",
  event_id = "",
} = {}) => {
  const params = {
    page,
    limit,
  };

  if (status && status.trim() !== "" && status.trim().toUpperCase() !== "ALL") {
    params.status = status.trim().toUpperCase();
  }

  if (
    allocationStatus &&
    allocationStatus.trim() !== "" &&
    allocationStatus.trim().toUpperCase() !== "ALL"
  ) {
    params.allocation_status = allocationStatus.trim().toUpperCase();
  }

  if (from && from.trim() !== "") {
    params.from = from.trim();
  }

  if (to && to.trim() !== "") {
    params.to = to.trim();
  }

  if (search && search.trim() !== "") {
    params.search = search.trim();
  }

  const resolvedEventId = (eventId || event_id || "").trim();
  if (resolvedEventId) {
    params.event_id = resolvedEventId;
  }

  const response = await apiClient.get("/admin/payments", {
    params,
  });

  return response.data;
};

/**
 * Fetch event-wise payment summaries with overall financial summary for admin console.
 *
 * @param {Object} options
 * @param {number} [options.page=1]
 * @param {number} [options.limit=10]
 * @param {string} [options.search=""]
 * @param {string} [options.from=""]
 * @param {string} [options.to=""]
 * @param {string} [options.eventId=""]
 * @param {string} [options.event_id=""]
 * @returns {Promise<Object>} API response data
 */
export const getAdminEventPaymentSummaries = async ({
  page = 1,
  limit = 10,
  search = "",
  from = "",
  to = "",
  eventId = "",
  event_id = "",
} = {}) => {
  const params = {
    page,
    limit,
  };

  if (search && search.trim() !== "") {
    params.search = search.trim();
  }

  if (from && from.trim() !== "") {
    params.from = from.trim();
  }

  if (to && to.trim() !== "") {
    params.to = to.trim();
  }

  const resolvedEventId = (eventId || event_id || "").trim();
  if (resolvedEventId) {
    params.event_id = resolvedEventId;
  }

  const response = await apiClient.get("/admin/payments/summary-by-event", {
    params,
  });

  return response.data;
};
