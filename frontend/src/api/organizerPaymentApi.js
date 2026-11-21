import apiClient from "./client.js";

/**
 * Normalizes query parameters for organizer payment endpoints.
 *
 * @param {Object} options
 * @param {number} [options.page]
 * @param {number} [options.limit]
 * @param {string} [options.status]
 * @param {string} [options.allocationStatus]
 * @param {string} [options.allocation_status]
 * @param {string} [options.from]
 * @param {string} [options.to]
 * @param {string} [options.search]
 * @param {string} [options.eventId]
 * @param {string} [options.event_id]
 * @returns {Object} Clean query parameters object
 */
export const buildOrganizerPaymentParams = ({
  page,
  limit,
  status = "",
  allocationStatus = "",
  allocation_status = "",
  from = "",
  to = "",
  search = "",
  eventId = "",
  event_id = "",
} = {}) => {
  const params = {};

  if (typeof page === "number" && page > 0) {
    params.page = page;
  }

  if (typeof limit === "number" && limit > 0) {
    params.limit = limit;
  }

  const rawStatus = (status || "").trim().toUpperCase();
  if (rawStatus && rawStatus !== "ALL") {
    params.status = rawStatus;
  }

  const rawAllocation = (allocationStatus || allocation_status || "").trim().toUpperCase();
  if (rawAllocation && rawAllocation !== "ALL") {
    params.allocation_status = rawAllocation;
  }

  const rawFrom = (from || "").trim();
  if (rawFrom) {
    params.from = rawFrom;
  }

  const rawTo = (to || "").trim();
  if (rawTo) {
    params.to = rawTo;
  }

  const rawSearch = (search || "").trim();
  if (rawSearch) {
    params.search = rawSearch;
  }

  const rawEventId = (eventId || event_id || "").trim();
  if (rawEventId) {
    params.event_id = rawEventId;
  }

  return params;
};

/**
 * Fetch financial overview for the authenticated organizer.
 *
 * @param {Object} [filters={}]
 * @returns {Promise<Object>} API response data with overview metrics
 */
export const getOrganizerPaymentOverview = async (filters = {}) => {
  const params = buildOrganizerPaymentParams(filters);

  // Pagination parameters are not needed for overview
  delete params.page;
  delete params.limit;

  const response = await apiClient.get("/organizers/payments/overview", {
    params,
  });

  return response.data;
};

/**
 * Fetch event-wise payment summaries with overall summary for the authenticated organizer.
 *
 * @param {Object} [options={}]
 * @param {number} [options.page=1]
 * @param {number} [options.limit=10]
 * @returns {Promise<Object>} API response data with event summaries and overview
 */
export const getOrganizerEventPaymentSummaries = async ({
  page = 1,
  limit = 10,
  ...filters
} = {}) => {
  const params = buildOrganizerPaymentParams({
    page,
    limit,
    ...filters,
  });

  const response = await apiClient.get("/organizers/payments/summary-by-event", {
    params,
  });

  return response.data;
};

/**
 * Fetch paginated payment transactions for the authenticated organizer.
 *
 * @param {Object} [options={}]
 * @param {number} [options.page=1]
 * @param {number} [options.limit=10]
 * @returns {Promise<Object>} API response data with payments list and pagination
 */
export const getOrganizerPayments = async ({
  page = 1,
  limit = 10,
  ...filters
} = {}) => {
  const params = buildOrganizerPaymentParams({
    page,
    limit,
    ...filters,
  });

  const response = await apiClient.get("/organizers/payments", {
    params,
  });

  return response.data;
};
