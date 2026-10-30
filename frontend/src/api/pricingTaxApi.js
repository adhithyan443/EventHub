import apiClient from "./client";

// Get current active configuration and scheduled future timeline
export const getPricingTaxCurrentAndTimeline = async () => {
  const response = await apiClient.get("/admin/pricing-tax");
  return response.data;
};

// Get paginated historical configuration changes
export const getPricingTaxHistory = async ({ page = 1, limit = 10 } = {}) => {
  const response = await apiClient.get("/admin/pricing-tax/history", {
    params: { page, limit },
  });
  return response.data;
};

// Publish immediate configuration change
export const publishImmediatePricingTax = async (data) => {
  const response = await apiClient.post("/admin/pricing-tax/publish", data);
  return response.data;
};

// Schedule future configuration change
export const scheduleFuturePricingTax = async (data) => {
  const response = await apiClient.post("/admin/pricing-tax/schedule", data);
  return response.data;
};

// Cancel a scheduled future configuration
export const cancelScheduledPricingTax = async (id, data) => {
  const response = await apiClient.post(`/admin/pricing-tax/scheduled/${id}/cancel`, data);
  return response.data;
};
