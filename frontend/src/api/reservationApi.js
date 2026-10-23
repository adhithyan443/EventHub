
import apiClient from "./client";

export const createReservation = async (eventId, payload) => {
    if (!eventId) {
        throw new Error("Event ID is required to create a reservation.");
    }

    const response = await apiClient.post("/bookings/reserve", {
        eventId,
        ...payload,
    });

    return response.data;
};

export const cancelReservation = async (reservationId) => {
    if (!reservationId) {
        throw new Error("Reservation ID is required to cancel a reservation.");
    }

    const response = await apiClient.delete(`/reservations/${reservationId}`);
    return response.data;
};
