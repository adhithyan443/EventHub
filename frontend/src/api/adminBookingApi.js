import apiClient from "./client.js";

/**
 * Normalizes query parameters for admin booking endpoints.
 *
 * @param {Object} options
 * @param {number} [options.page]
 * @param {number} [options.limit]
 * @param {string} [options.eventId]
 * @param {string} [options.event_id]
 * @param {string} [options.organizerId]
 * @param {string} [options.organizer_id]
 * @param {string} [options.bookingStatus]
 * @param {string} [options.booking_status]
 * @param {string} [options.paymentStatus]
 * @param {string} [options.payment_status]
 * @param {string} [options.search]
 * @param {string} [options.from]
 * @param {string} [options.to]
 * @returns {Object} Clean query parameters object
 */
export const buildAdminBookingParams = ({
  page,
  limit,
  eventId = "",
  event_id = "",
  organizerId = "",
  organizer_id = "",
  bookingStatus = "",
  booking_status = "",
  paymentStatus = "",
  payment_status = "",
  search = "",
  from = "",
  to = "",
} = {}) => {
  const params = {};

  if (typeof page === "number" && page > 0) {
    params.page = page;
  }

  if (typeof limit === "number" && limit > 0) {
    params.limit = limit;
  }

  const rawEventId = (eventId || event_id || "").trim();
  if (rawEventId && rawEventId !== "ALL") {
    params.event_id = rawEventId;
  }

  const rawOrgId = (organizerId || organizer_id || "").trim();
  if (rawOrgId && rawOrgId !== "ALL") {
    params.organizer_id = rawOrgId;
  }

  const rawBookingStatus = (bookingStatus || booking_status || "").trim().toUpperCase();
  if (rawBookingStatus && rawBookingStatus !== "ALL") {
    params.booking_status = rawBookingStatus;
  }

  const rawPaymentStatus = (paymentStatus || payment_status || "").trim().toUpperCase();
  if (rawPaymentStatus && rawPaymentStatus !== "ALL") {
    params.payment_status = rawPaymentStatus;
  }

  const rawSearch = (search || "").trim();
  if (rawSearch) {
    params.search = rawSearch;
  }

  const rawFrom = (from || "").trim();
  if (rawFrom) {
    params.from = rawFrom;
  }

  const rawTo = (to || "").trim();
  if (rawTo) {
    params.to = rawTo;
  }

  return params;
};

/**
 * Normalizes query parameters for admin attendee endpoints.
 *
 * @param {Object} options
 * @param {number} [options.page]
 * @param {number} [options.limit]
 * @param {string} [options.eventId]
 * @param {string} [options.event_id]
 * @param {string} [options.organizerId]
 * @param {string} [options.organizer_id]
 * @param {string} [options.checkInStatus]
 * @param {string} [options.check_in_status]
 * @param {string} [options.status]
 * @param {string} [options.search]
 * @returns {Object} Clean query parameters object
 */
export const buildAdminAttendeeParams = ({
  page,
  limit,
  eventId = "",
  event_id = "",
  organizerId = "",
  organizer_id = "",
  checkInStatus = "",
  check_in_status = "",
  status = "",
  search = "",
} = {}) => {
  const params = {};

  if (typeof page === "number" && page > 0) {
    params.page = page;
  }

  if (typeof limit === "number" && limit > 0) {
    params.limit = limit;
  }

  const rawEventId = (eventId || event_id || "").trim();
  if (rawEventId && rawEventId !== "ALL") {
    params.event_id = rawEventId;
  }

  const rawOrgId = (organizerId || organizer_id || "").trim();
  if (rawOrgId && rawOrgId !== "ALL") {
    params.organizer_id = rawOrgId;
  }

  const rawCheckInStatus = (checkInStatus || check_in_status || status || "").trim().toUpperCase();
  if (rawCheckInStatus && rawCheckInStatus !== "ALL") {
    params.check_in_status = rawCheckInStatus;
  }

  const rawSearch = (search || "").trim();
  if (rawSearch) {
    params.search = rawSearch;
  }

  return params;
};

/**
 * Fetch paginated list of bookings across the platform for Admin.
 *
 * @param {Object} [options={}]
 * @returns {Promise<Object>} API response data with bookings and pagination
 */
export const getAdminBookings = async (options = {}) => {
  const params = buildAdminBookingParams(options);
  const response = await apiClient.get("/admin/bookings", { params });
  return response.data;
};

/**
 * Fetch aggregate platform-wide booking metrics and financial totals for Admin.
 *
 * @param {Object} [options={}]
 * @returns {Promise<Object>} API response data with booking metrics
 */
export const getAdminBookingStats = async (options = {}) => {
  const params = buildAdminBookingParams(options);
  delete params.page;
  delete params.limit;
  const response = await apiClient.get("/admin/bookings/stats", { params });
  return response.data;
};

/**
 * Fetch deep inspection breakdown for a single booking order across any organizer for Admin.
 *
 * @param {string} bookingId - UUID of the booking
 * @returns {Promise<Object>} API response data with full booking detail
 */
export const getAdminBookingDetail = async (bookingId) => {
  if (!bookingId) {
    throw new Error("bookingId is required to fetch booking details");
  }
  const response = await apiClient.get(`/admin/bookings/${bookingId}`);
  return response.data;
};

/**
 * Fetch paginated platform-wide individual ticket admission passes for Admin.
 *
 * @param {Object} [options={}]
 * @returns {Promise<Object>} API response data with attendees and pagination
 */
export const getAdminAttendees = async (options = {}) => {
  const params = buildAdminAttendeeParams(options);
  const response = await apiClient.get("/admin/attendees", { params });
  return response.data;
};

/**
 * Fetch aggregate platform-wide gate admission counts and check-in rates for Admin.
 *
 * @param {Object} [options={}]
 * @returns {Promise<Object>} API response data with gate admission statistics
 */
export const getAdminAttendeeStats = async (options = {}) => {
  const params = buildAdminAttendeeParams(options);
  delete params.page;
  delete params.limit;
  const response = await apiClient.get("/admin/attendees/stats", { params });
  return response.data;
};
