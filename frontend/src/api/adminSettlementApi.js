import apiClient from "./client.js";

/**
 * Normalizes query parameters for admin settlement listing.
 * Accepts either an options object or positional arguments (page, limit, status, organizerId).
 *
 * @param {Object|number} [optionsOrPage]
 * @param {number} [limitArg]
 * @param {string} [statusArg]
 * @param {string} [organizerIdArg]
 * @returns {Object} Clean query parameters object
 */
export const buildAdminSettlementParams = (optionsOrPage, limitArg, statusArg, organizerIdArg) => {
  let page;
  let limit;
  let status;
  let organizerId;

  if (typeof optionsOrPage === "object" && optionsOrPage !== null) {
    page = optionsOrPage.page;
    limit = optionsOrPage.limit;
    status = optionsOrPage.status;
    organizerId = optionsOrPage.organizerId || optionsOrPage.organizer_id;
  } else {
    page = optionsOrPage;
    limit = limitArg;
    status = statusArg;
    organizerId = organizerIdArg;
  }

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

  const rawOrgId = (organizerId || "").trim();
  if (rawOrgId) {
    params.organizer_id = rawOrgId;
  }

  return params;
};

/**
 * Fetch paginated settlements across all organizers with optional filters for admin console.
 * Accepts either an options object or positional arguments (page, limit, status, organizerId).
 *
 * @param {Object|number} [optionsOrPage=1]
 * @param {number} [limitArg=10]
 * @param {string} [statusArg=""]
 * @param {string} [organizerIdArg=""]
 * @returns {Promise<Object>} API response data with settlements list and pagination
 */
export const getAdminSettlements = async (
  optionsOrPage = 1,
  limitArg = 10,
  statusArg = "",
  organizerIdArg = ""
) => {
  const params = buildAdminSettlementParams(optionsOrPage, limitArg, statusArg, organizerIdArg);
  const response = await apiClient.get("/admin/settlements", {
    params,
  });
  return response.data;
};

/**
 * Fetch detailed settlement information for admin review.
 *
 * @param {string} settlementId - Settlement UUID
 * @returns {Promise<Object>} API response data with full Settlement record
 */
export const getAdminSettlementDetails = async (settlementId) => {
  if (!settlementId || typeof settlementId !== "string") {
    throw new Error("Invalid settlement ID");
  }

  const response = await apiClient.get(`/admin/settlements/${settlementId.trim()}`);
  return response.data;
};

/**
 * Approve a PENDING settlement and trigger payout disbursement.
 *
 * @param {string} settlementId - Settlement UUID
 * @returns {Promise<Object>} API response data with updated Settlement record
 */
export const approveAdminSettlement = async (settlementId) => {
  if (!settlementId || typeof settlementId !== "string") {
    throw new Error("Invalid settlement ID");
  }

  const response = await apiClient.post(`/admin/settlements/${settlementId.trim()}/approve`);
  return response.data;
};

/**
 * Builds and validates the request payload for rejecting a settlement.
 * Sends both `reason` and `rejection_reason` for complete contract compatibility.
 *
 * @param {string} reason - Rejection explanation
 * @returns {{ reason: string, rejection_reason: string }}
 */
export const buildRejectSettlementPayload = (reason) => {
  const trimmedReason = (reason || "").trim();
  if (!trimmedReason) {
    throw new Error("Rejection reason is required");
  }

  return {
    reason: trimmedReason,
    rejection_reason: trimmedReason,
  };
};

/**
 * Reject a PENDING or FAILED settlement with an administrative explanation.
 *
 * @param {string} settlementId - Settlement UUID
 * @param {string} reason - Rejection explanation
 * @returns {Promise<Object>} API response data with rejected Settlement record
 */
export const rejectAdminSettlement = async (settlementId, reason) => {
  if (!settlementId || typeof settlementId !== "string") {
    throw new Error("Invalid settlement ID");
  }

  const payload = buildRejectSettlementPayload(reason);

  const response = await apiClient.post(`/admin/settlements/${settlementId.trim()}/reject`, payload);
  return response.data;
};

/**
 * Retry payout execution for a FAILED settlement record.
 *
 * @param {string} settlementId - Settlement UUID
 * @returns {Promise<Object>} API response data with retried Settlement record
 */
export const retryAdminSettlement = async (settlementId) => {
  if (!settlementId || typeof settlementId !== "string") {
    throw new Error("Invalid settlement ID");
  }

  const response = await apiClient.post(`/admin/settlements/${settlementId.trim()}/retry`);
  return response.data;
};
