import apiClient from "./client.js";

/**
 * Normalizes query parameters for organizer settlement listing.
 * Accepts either an options object or positional arguments (page, limit, status).
 *
 * @param {Object|number} [optionsOrPage]
 * @param {number} [limitArg]
 * @param {string} [statusArg]
 * @returns {Object} Clean query parameters object
 */
export const buildOrganizerSettlementParams = (optionsOrPage, limitArg, statusArg) => {
  let page;
  let limit;
  let status;

  if (typeof optionsOrPage === "object" && optionsOrPage !== null) {
    page = optionsOrPage.page;
    limit = optionsOrPage.limit;
    status = optionsOrPage.status;
  } else {
    page = optionsOrPage;
    limit = limitArg;
    status = statusArg;
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

  return params;
};

/**
 * Builds and validates payload for requesting an organizer settlement.
 *
 * @param {number|string} amount
 * @param {string} [eventId]
 * @returns {Object} Validated request payload
 */
export const buildSettlementRequestPayload = (amount, eventId) => {
  const num = typeof amount === "number" ? amount : parseFloat(amount);
  if (isNaN(num) || num <= 0) {
    throw new Error("Settlement amount must be a positive number");
  }

  const payload = {
    amount: num,
  };

  const resolvedEventId = (eventId || "").trim();
  if (resolvedEventId) {
    payload.event_id = resolvedEventId;
  }

  return payload;
};

/**
 * Fetch financial balance projection and settlement eligibility for the authenticated organizer.
 *
 * @returns {Promise<Object>} API response data with balance projection and event breakdowns
 */
export const getOrganizerSettlementBalance = async () => {
  const response = await apiClient.get("/organizers/settlements/balance");
  return response.data;
};

/**
 * Submit a new settlement disbursement request against available balance.
 *
 * @param {Object|number|string} payloadOrAmount - Payload object or numeric amount
 * @param {string} [eventIdArg] - Optional event UUID if passing positional args
 * @returns {Promise<Object>} API response data with created Settlement record
 */
export const requestOrganizerSettlement = async (payloadOrAmount, eventIdArg) => {
  let amount;
  let eventId;

  if (typeof payloadOrAmount === "object" && payloadOrAmount !== null) {
    amount = payloadOrAmount.amount;
    eventId = payloadOrAmount.eventId || payloadOrAmount.event_id;
  } else {
    amount = payloadOrAmount;
    eventId = eventIdArg;
  }

  const payload = buildSettlementRequestPayload(amount, eventId);
  const response = await apiClient.post("/organizers/settlements", payload);
  return response.data;
};

/**
 * Fetch paginated settlement requests for the authenticated organizer.
 *
 * @param {Object|number} [optionsOrPage=1]
 * @param {number} [limitArg=10]
 * @param {string} [statusArg=""]
 * @returns {Promise<Object>} API response data with settlements list and pagination
 */
export const getOrganizerSettlements = async (optionsOrPage = 1, limitArg = 10, statusArg = "") => {
  const params = buildOrganizerSettlementParams(optionsOrPage, limitArg, statusArg);
  const response = await apiClient.get("/organizers/settlements", {
    params,
  });
  return response.data;
};

/**
 * Fetch detailed information for a specific settlement record.
 *
 * @param {string} settlementId - Settlement UUID
 * @returns {Promise<Object>} API response data with single Settlement details
 */
export const getOrganizerSettlementDetails = async (settlementId) => {
  if (!settlementId || typeof settlementId !== "string") {
    throw new Error("Invalid settlement ID");
  }

  const response = await apiClient.get(`/organizers/settlements/${settlementId.trim()}`);
  return response.data;
};
