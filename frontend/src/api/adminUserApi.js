import apiClient from "./client";

/**
 * Fetch registered users with pagination, search, role, and status filtering.
 *
 * @param {Object} options
 * @param {number} [options.page=1]
 * @param {number} [options.limit=10]
 * @param {string} [options.search=""]
 * @param {string} [options.role=""]
 * @param {string} [options.status=""]
 * @returns {Promise<Object>} API response data
 */
export const getUsers = async ({
  page = 1,
  limit = 10,
  search = "",
  role = "",
  status = "",
} = {}) => {
  const params = {
    page,
    limit,
  };

  if (search && search.trim() !== "") {
    params.search = search.trim();
  }

  if (role && role.trim() !== "") {
    params.role = role.trim();
  }

  if (status && status.trim() !== "") {
    params.status = status.trim();
  }

  const response = await apiClient.get("/admin/users", {
    params,
  });

  return response.data;
};

/**
 * Fetch detailed profile information for a single user by ID.
 *
 * @param {string} userId
 * @returns {Promise<Object>} API response data
 */
export const getUserById = async (userId) => {
  const response = await apiClient.get(`/admin/users/${userId}`);
  return response.data;
};

/**
 * Block / suspend a user account.
 *
 * @param {string} userId
 * @param {string} [reason=""]
 * @returns {Promise<Object>} API response data
 */
export const blockUser = async (userId, reason = "") => {
  const response = await apiClient.patch(`/admin/users/${userId}/block`, {
    reason,
  });
  return response.data;
};

/**
 * Unblock / reactivate a user account.
 *
 * @param {string} userId
 * @returns {Promise<Object>} API response data
 */
export const unblockUser = async (userId) => {
  const response = await apiClient.patch(`/admin/users/${userId}/unblock`);
  return response.data;
};
