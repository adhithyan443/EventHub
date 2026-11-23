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
 * @param {string} [options.eventId=""]
 * @param {string} [options.organizerId=""]
 * @param {string} [options.settlementStatus=""]
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
  eventId = "",
  organizerId = "",
  settlementStatus = "",
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

  if (settlementStatus && settlementStatus.trim() !== "" && settlementStatus.trim().toUpperCase() !== "ALL") {
    params.settlement_status = settlementStatus.trim().toUpperCase();
  }

  if (eventId && eventId.trim() !== "") {
    params.event_id = eventId.trim();
  }

  if (organizerId && organizerId.trim() !== "") {
    params.organizer_id = organizerId.trim();
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
 * Fetch consolidated platform-wide financial overview, 10 authoritative metrics,
 * and event/organizer breakdowns for the admin accounting module.
 *
 * @param {Object} [filter={}]
 * @param {string} [filter.from=""]
 * @param {string} [filter.to=""]
 * @param {string} [filter.eventId=""]
 * @param {string} [filter.organizerId=""]
 * @param {string} [filter.transactionType=""]
 * @param {string} [filter.settlementStatus=""]
 * @returns {Promise<Object>} API response data containing metrics and breakdowns
 */
export const getAdminFinancialOverview = async (filter = {}) => {
  const params = {};

  if (filter.from && filter.from.trim() !== "") {
    params.from = filter.from.trim();
  }
  if (filter.to && filter.to.trim() !== "") {
    params.to = filter.to.trim();
  }
  if (filter.eventId && filter.eventId.trim() !== "") {
    params.event_id = filter.eventId.trim();
  }
  if (filter.organizerId && filter.organizerId.trim() !== "") {
    params.organizer_id = filter.organizerId.trim();
  }
  if (filter.transactionType && filter.transactionType.trim() !== "" && filter.transactionType.trim().toUpperCase() !== "ALL") {
    params.transaction_type = filter.transactionType.trim().toUpperCase();
  }
  if (filter.settlementStatus && filter.settlementStatus.trim() !== "" && filter.settlementStatus.trim().toUpperCase() !== "ALL") {
    params.settlement_status = filter.settlementStatus.trim().toUpperCase();
  }

  const response = await apiClient.get("/admin/ledger/overview", {
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
