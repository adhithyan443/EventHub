import apiClient from "./client";

/**
 * Creates a persistent checkout snapshot for a valid active reservation.
 * @param {string} reservationId - UUID of the active reservation
 * @returns {Promise<object>} - Envelope containing { success: true, data: { checkout } }
 */
export const createCheckout = async (reservationId) => {
    if (!reservationId) {
        throw new Error("Reservation ID is required to create a checkout.");
    }

    const response = await apiClient.post("/bookings/checkout", {
        reservationId,
    });

    return response.data;
};
