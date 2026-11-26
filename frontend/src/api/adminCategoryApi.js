import apiClient from "./client.js";

/**
 * Fetch categories with pagination, search, status filtering, and statistics.
 *
 * @param {Object} options
 * @param {number} [options.page=1]
 * @param {number} [options.limit=10]
 * @param {string} [options.search=""]
 * @param {string} [options.status=""]
 * @returns {Promise<Object>} API response data
 */
export const getAdminCategories = async ({
  page = 1,
  limit = 10,
  search = "",
  status = "",
} = {}) => {
  const params = {
    page,
    limit,
  };

  if (search && search.trim() !== "") {
    params.search = search.trim();
  }

  if (status && status.trim() !== "" && status.toUpperCase() !== "ALL") {
    params.status = status.trim().toUpperCase();
  }

  const response = await apiClient.get("/admin/categories", {
    params,
  });

  return response.data;
};

/**
 * Create a new category.
 *
 * @param {Object} data
 * @param {string} data.name
 * @param {string} [data.description=""]
 * @returns {Promise<Object>} API response data
 */
export const createCategory = async ({ name, description = "" }) => {
  const payload = {
    name: name.trim(),
    description: (description || "").trim(),
  };

  const response = await apiClient.post("/admin/categories", payload);
  return response.data;
};

/**
 * Update an existing category by ID.
 *
 * @param {string} categoryId
 * @param {Object} data
 * @param {string} data.name
 * @param {string} [data.description=""]
 * @returns {Promise<Object>} API response data
 */
export const updateCategory = async (categoryId, { name, description = "" }) => {
  const payload = {
    name: name.trim(),
    description: (description || "").trim(),
  };

  const response = await apiClient.put(`/admin/categories/${categoryId}`, payload);
  return response.data;
};

/**
 * Toggle category status (ACTIVE / INACTIVE).
 *
 * @param {string} categoryId
 * @param {"ACTIVE"|"INACTIVE"} status
 * @returns {Promise<Object>} API response data
 */
export const toggleCategoryStatus = async (categoryId, status) => {
  const payload = {
    status: status.trim().toUpperCase(),
  };

  const response = await apiClient.patch(`/admin/categories/${categoryId}/status`, payload);
  return response.data;
};
