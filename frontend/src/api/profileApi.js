import apiClient from "./client.js";

/**
 * Fetch authoritative user profile with presigned avatar URL if applicable.
 */
export const getProfile = async () => {
  const response = await apiClient.get("/profile");
  return response.data;
};

/**
 * Update full name and/or phone number.
 * @param {{ full_name: string, phone: string }} data
 */
export const updateProfile = async (data) => {
  const response = await apiClient.put("/profile", data);
  return response.data;
};

/**
 * Initiate email change request and receive OTP on new email.
 * @param {string} newEmail
 */
export const requestEmailChange = async (newEmail) => {
  const response = await apiClient.post("/profile/email/request-change", {
    new_email: newEmail,
  });
  return response.data;
};

/**
 * Resend OTP to the pending new email.
 * @param {string} newEmail
 */
export const resendEmailChangeOTP = async (newEmail) => {
  const response = await apiClient.post("/profile/email/resend-otp", {
    new_email: newEmail,
  });
  return response.data;
};

/**
 * Verify OTP to complete email change.
 * @param {{ newEmail: string, otp: string }} params
 */
export const verifyEmailChange = async ({ newEmail, otp }) => {
  const response = await apiClient.post("/profile/email/verify-change", {
    new_email: newEmail,
    otp,
  });
  return response.data;
};

/**
 * Upload new avatar image (JPEG, PNG, WebP up to 5MB).
 * @param {File} file
 */
export const uploadAvatar = async (file) => {
  const formData = new FormData();
  formData.append("avatar", file);
  const response = await apiClient.post("/profile/avatar", formData, {
    headers: {
      "Content-Type": "multipart/form-data",
    },
  });
  return response.data;
};

/**
 * Remove profile avatar and revert to default initials.
 */
export const removeAvatar = async () => {
  const response = await apiClient.delete("/profile/avatar");
  return response.data;
};
