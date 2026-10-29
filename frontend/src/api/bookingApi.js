import apiClient from "./client";

/**
 * Retrieves all bookings belonging to the authenticated customer.
 * @returns {Promise<{ data: Array }>}
 */
export const getMyBookings = async () => {
  const response = await apiClient.get("/bookings");
  return response.data;
};

/**
 * Retrieves a single booking with its associated items for the authenticated customer.
 * @param {string} bookingId - UUID of the booking
 * @returns {Promise<{ data: object }>}
 */
export const getBookingById = async (bookingId) => {
  if (!bookingId) {
    throw new Error("Booking ID is required.");
  }

  const response = await apiClient.get(`/bookings/${bookingId}`);
  return response.data;
};

/**
 * Retrieves the authoritative QR code image blob for a booking.
 * @param {string} bookingId - UUID of the booking
 * @returns {Promise<Blob>}
 */
export const getBookingQrBlob = async (bookingId) => {
  if (!bookingId) {
    throw new Error("Booking ID is required.");
  }

  const response = await apiClient.get(`/bookings/${bookingId}/qr`, {
    responseType: "blob",
  });
  return response.data;
};

/**
 * Downloads the booking PDF ticket pass for the authenticated customer.
 * Extracts the filename from content-disposition header and triggers browser download.
 * @param {string} bookingId - UUID of the booking
 * @returns {Promise<{ filename: string }>}
 */
export const downloadBookingPdf = async (bookingId) => {
  if (!bookingId) {
    throw new Error("Booking ID is required to download booking PDF.");
  }

  try {
    const response = await apiClient.get(`/bookings/${bookingId}/download`, {
      responseType: "blob",
    });

    let filename = `booking-${bookingId}.pdf`;
    const disposition = response.headers?.["content-disposition"];
    if (disposition) {
      const match = disposition.match(/filename=["']?([^"';]+)["']?/i);
      if (match && match[1]) {
        filename = match[1];
      }
    }

    const blob = new Blob([response.data], { type: "application/pdf" });
    const downloadUrl = window.URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = downloadUrl;
    link.setAttribute("download", filename);
    document.body.appendChild(link);
    link.click();
    link.remove();
    window.URL.revokeObjectURL(downloadUrl);

    return { filename };
  } catch (err) {
    if (err.response?.data instanceof Blob) {
      try {
        const text = await err.response.data.text();
        const parsed = JSON.parse(text);
        if (parsed?.error || parsed?.message) {
          err.message = parsed.error || parsed.message;
        }
      } catch {
        // Fallback to generic message
      }
    }
    throw err;
  }
};

/**
 * Cancels a booking for the authenticated customer and initiates refund processing if eligible.
 * @param {string} bookingId - UUID of the booking
 * @returns {Promise<{ message: string, data: object }>}
 */
export const cancelBooking = async (bookingId) => {
  if (!bookingId) {
    throw new Error("Booking ID is required.");
  }

  const response = await apiClient.post(`/bookings/${bookingId}/cancel`);
  return response.data;
};
