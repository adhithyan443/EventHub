
import apiClient from "./client";

export const createReservation = async (eventId, payload) => {
    if (!eventId) {
        throw new Error("Event ID is required to create a reservation.");
    }

    const response = await apiClient.post(
        `/events/${eventId}/reservations`,
        payload
    );

    return response.data;
};
