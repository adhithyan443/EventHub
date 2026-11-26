import apiClient from "./client.js";

/**
 * Fetch system health status from the existing backend health endpoint.
 *
 * @returns {Promise<Object>} API response from the health endpoint
 */
export const getSystemHealth = async () => {
  const startTime = Date.now();
  try {
    const response = await apiClient.get("/health");
    const latency = Date.now() - startTime;
    return {
      success: true,
      data: response.data,
      latency,
      timestamp: new Date().toISOString(),
    };
  } catch (error) {
    const latency = Date.now() - startTime;
    return {
      success: false,
      error: error?.response?.data || error.message || "Backend health check failed",
      status: error?.response?.status || 500,
      latency,
      timestamp: new Date().toISOString(),
    };
  }
};

/**
 * Fetch paginated application logs from the backend.
 *
 * @param {Object} [options={}]
 * @param {number} [options.page=1]
 * @param {number} [options.limit=50]
 * @param {string} [options.level=""]
 * @param {string} [options.search=""]
 * @returns {Promise<Object>} API response data with logs and pagination
 */
export const getSystemLogs = async ({
  page = 1,
  limit = 50,
  level = "",
  search = "",
} = {}) => {
  const params = {};

  if (typeof page === "number" && page > 0) {
    params.page = page;
  }
  if (typeof limit === "number" && limit > 0) {
    params.limit = limit;
  }
  const cleanLevel = (level || "").trim().toUpperCase();
  if (cleanLevel && cleanLevel !== "ALL") {
    params.level = cleanLevel;
  }
  const cleanSearch = (search || "").trim();
  if (cleanSearch) {
    params.search = cleanSearch;
  }

  const response = await apiClient.get("/admin/system/logs", { params });
  return response.data;
};
