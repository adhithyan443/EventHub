import apiClient from "./client.js";

/**
 * Normalizes query parameters for organizer booking endpoints.
 *
 * @param {Object} options
 * @param {number} [options.page]
 * @param {number} [options.limit]
 * @param {string} [options.eventId]
 * @param {string} [options.event_id]
 * @param {string} [options.bookingStatus]
 * @param {string} [options.booking_status]
 * @param {string} [options.paymentStatus]
 * @param {string} [options.payment_status]
 * @param {string} [options.search]
 * @param {string} [options.from]
 * @param {string} [options.to]
 * @returns {Object} Clean query parameters object
 */
export const buildOrganizerBookingParams = ({
  page,
  limit,
  eventId = "",
  event_id = "",
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
 * Normalizes query parameters for organizer attendee endpoints.
 *
 * @param {Object} options
 * @param {number} [options.page]
 * @param {number} [options.limit]
 * @param {string} [options.eventId]
 * @param {string} [options.event_id]
 * @param {string} [options.checkInStatus]
 * @param {string} [options.check_in_status]
 * @param {string} [options.status]
 * @param {string} [options.search]
 * @returns {Object} Clean query parameters object
 */
export const buildOrganizerAttendeeParams = ({
  page,
  limit,
  eventId = "",
  event_id = "",
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
 * Fetch paginated list of bookings for the authenticated organizer.
 *
 * @param {Object} [options={}]
 * @returns {Promise<Object>} API response data with bookings and pagination
 */
export const getOrganizerBookings = async (options = {}) => {
  const params = buildOrganizerBookingParams(options);
  const response = await apiClient.get("/organizers/bookings", { params });
  return response.data;
};

/**
 * Fetch aggregate booking metrics and financial totals for the organizer's scope.
 *
 * @param {Object} [options={}]
 * @returns {Promise<Object>} API response data with booking metrics
 */
export const getOrganizerBookingStats = async (options = {}) => {
  const params = buildOrganizerBookingParams(options);
  delete params.page;
  delete params.limit;
  const response = await apiClient.get("/organizers/bookings/stats", { params });
  return response.data;
};

/**
 * Fetch deep inspection breakdown for a single booking order.
 *
 * @param {string} bookingId - UUID of the booking
 * @returns {Promise<Object>} API response data with full booking detail
 */
export const getOrganizerBookingDetail = async (bookingId) => {
  if (!bookingId) {
    throw new Error("bookingId is required to fetch booking details");
  }
  const response = await apiClient.get(`/organizers/bookings/${bookingId}`);
  return response.data;
};

/**
 * Fetch paginated individual ticket admission passes for the organizer's events.
 *
 * @param {Object} [options={}]
 * @returns {Promise<Object>} API response data with attendees and pagination
 */
export const getOrganizerAttendees = async (options = {}) => {
  const params = buildOrganizerAttendeeParams(options);
  const response = await apiClient.get("/organizers/attendees", { params });
  return response.data;
};

/**
 * Fetch aggregate gate admission counts and check-in rates.
 *
 * @param {Object} [options={}]
 * @returns {Promise<Object>} API response data with gate admission statistics
 */
export const getOrganizerAttendeeStats = async (options = {}) => {
  const params = buildOrganizerAttendeeParams(options);
  delete params.page;
  delete params.limit;
  const response = await apiClient.get("/organizers/attendees/stats", { params });
  return response.data;
};
