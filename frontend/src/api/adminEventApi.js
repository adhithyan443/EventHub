import apiClient from "./client.js";

/**
 * Fetch platform events for admin management with filters and pagination.
 *
 * @param {Object} options
 * @param {number} [options.page=1]
 * @param {number} [options.limit=10]
 * @param {string} [options.search=""]
 * @param {string} [options.status=""]
 * @param {string} [options.categoryId=""]
 * @param {string} [options.eventType=""]
 * @param {string} [options.organizerId=""]
 * @returns {Promise<Object>} API response data
 */
export const getAdminEvents = async ({
  page = 1,
  limit = 10,
  search = "",
  status = "",
  categoryId = "",
  eventType = "",
  organizerId = "",
} = {}) => {
  const params = {
    page,
    limit,
  };

  if (search && search.trim() !== "") {
    params.search = search.trim();
  }

  if (status && status.trim() !== "" && status !== "all") {
    params.status = status.trim().toUpperCase();
  }

  if (categoryId && categoryId.trim() !== "" && categoryId !== "all") {
    params.category_id = categoryId.trim();
  }

  if (eventType && eventType.trim() !== "" && eventType !== "all") {
    params.event_type = eventType.trim().toUpperCase();
  }

  if (organizerId && organizerId.trim() !== "" && organizerId !== "all") {
    params.organizer_id = organizerId.trim();
  }

  const response = await apiClient.get("/admin/events", {
    params,
  });

  return response.data;
};

/**
 * Fetch comprehensive event details for admin drawer.
 *
 * @param {string} eventId
 * @returns {Promise<Object>} API response data
 */
export const getAdminEventById = async (eventId) => {
  const response = await apiClient.get(`/admin/events/${eventId}`);
  return response.data;
};

/**
 * Update event metadata or settings.
 *
 * @param {string} eventId
 * @param {Object} eventData
 * @returns {Promise<Object>} API response data
 */
export const updateAdminEvent = async (eventId, eventData) => {
  const response = await apiClient.put(`/admin/events/${eventId}`, eventData);
  return response.data;
};

/**
 * Publish an eligible draft event as admin.
 *
 * @param {string} eventId
 * @returns {Promise<Object>} API response data
 */
export const publishAdminEvent = async (eventId) => {
  const response = await apiClient.patch(`/admin/events/${eventId}/publish`);
  return response.data;
};

/**
 * Permanently delete an event (allowed only with 0 confirmed bookings & 0 active reservations).
 *
 * @param {string} eventId
 * @returns {Promise<Object>} API response data
 */
export const deleteAdminEvent = async (eventId) => {
  const response = await apiClient.delete(`/admin/events/${eventId}`);
  return response.data;
};

/**
 * Get active categories for filter and edit dropdowns.
 *
 * @returns {Promise<Object>}
 */
export const getCategories = async () => {
  const response = await apiClient.get("/categories");
  return response.data;
};
