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
