import apiClient from "./client.js";

/**
 * Fetch paginated financial journals for the admin ledger view with filters.
 *
 * @param {Object} [options={}]
 * @param {number} [options.page=1]
 * @param {number} [options.limit=10]
 * @param {string} [options.transactionType=""]
 * @param {string} [options.status=""]
 * @param {string} [options.search=""]
 * @param {string} [options.referenceId=""]
 * @param {string} [options.from=""]
 * @param {string} [options.to=""]
 * @returns {Promise<Object>} API response data containing journals and pagination metadata
 */
export const getAdminLedgerJournals = async ({
  page = 1,
  limit = 10,
  transactionType = "",
  status = "",
  search = "",
  referenceId = "",
  from = "",
  to = "",
} = {}) => {
  const params = {
    page,
    limit,
  };

  if (transactionType && transactionType.trim() !== "" && transactionType.trim().toUpperCase() !== "ALL") {
    params.transaction_type = transactionType.trim().toUpperCase();
  }

  if (status && status.trim() !== "" && status.trim().toUpperCase() !== "ALL") {
    params.status = status.trim().toUpperCase();
  }

  const querySearch = search || referenceId;
  if (querySearch && querySearch.trim() !== "") {
    params.search = querySearch.trim();
  }

  if (from && from.trim() !== "") {
    params.from = from.trim();
  }

  if (to && to.trim() !== "") {
    params.to = to.trim();
  }

  const response = await apiClient.get("/admin/ledger/journals", {
    params,
  });

  return response.data;
};

/**
 * Fetch detailed journal data by its UUID, including balanced debit and credit ledger entries.
 *
 * @param {string} id - Journal UUID
 * @returns {Promise<Object>} API response data containing journal details and entries
 */
export const getAdminLedgerJournalById = async (id) => {
  if (!id) {
    throw new Error("Journal ID is required");
  }

  const response = await apiClient.get(`/admin/ledger/journals/${encodeURIComponent(id)}`);
  return response.data;
};
