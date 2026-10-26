import apiClient from "./client";

// In-flight deduplication map: prevents duplicate concurrent cancel requests for the same reservation ID
const inFlightCancellations = new Map();

/**
 * Creates a ticket or seat reservation.
 * @param {string} eventId - UUID of the event
 * @param {object} payload - Either { seat_ids: [...] } or { ticket_type_id, quantity }
 */
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

/**
 * Cancels an active reservation on the backend.
 * Handles duplicate calls via in-flight promise caching.
 * Idempotently handles HTTP 404 (already expired, cancelled, or completed).
 * @param {string} reservationId - UUID of the reservation to cancel
 */
export const cancelReservation = async (reservationId) => {
    if (!reservationId) {
        throw new Error("Reservation ID is required to cancel a reservation.");
    }

    if (inFlightCancellations.has(reservationId)) {
        return inFlightCancellations.get(reservationId);
    }

    const cancelPromise = (async () => {
        try {
            console.info("[ReservationAPI] Cancelling reservation", {
                reservation_id: reservationId,
            });
            const response = await apiClient.delete(`/reservations/${reservationId}`);
            return response.data;
        } catch (err) {
            // HTTP 404 means the reservation is already cancelled, expired, or completed.
            // This is an expected idempotent outcome.
            if (err?.response?.status === 404) {
                console.info("[ReservationAPI] Reservation already inactive or not found (404 tolerated)", {
                    reservation_id: reservationId,
                });
                return {
                    success: true,
                    message: "Reservation already inactive",
                    alreadyInactive: true,
                };
            }
            console.error("[ReservationAPI] Reservation cancellation failed", {
                reservation_id: reservationId,
                status: err?.response?.status,
                message: err?.message,
            });
            throw err;
        } finally {
            inFlightCancellations.delete(reservationId);
        }
    })();

    inFlightCancellations.set(reservationId, cancelPromise);
    return cancelPromise;
};

/**
 * Reads and validates cached reservation data from sessionStorage for a specific event.
 * @param {string} eventId - Event UUID
 * @returns {object|null} Parsed reservation metadata or null
 */
export const getStoredReservation = (eventId) => {
    if (!eventId) return null;
    try {
        const raw = sessionStorage.getItem(`eventhub_reservation_${eventId}`);
        if (!raw) return null;
        const parsed = JSON.parse(raw);
        const res = parsed.reservation;
        if (!res) return null;

        const id = res.ID || res.id;
        const expiresAtRaw = res.ExpiresAt || res.expires_at;
        const expiresAt = expiresAtRaw ? new Date(expiresAtRaw).getTime() : 0;

        return {
            ...parsed,
            reservationId: id,
            expiresAt,
            isExpired: expiresAt > 0 && expiresAt <= Date.now(),
        };
    } catch {
        return null;
    }
};

/**
 * Removes cached reservation data from sessionStorage for a specific event.
 * @param {string} eventId - Event UUID
 */
export const clearStoredReservation = (eventId) => {
    if (!eventId) return;
    try {
        sessionStorage.removeItem(`eventhub_reservation_${eventId}`);
    } catch {
        // Ignore storage removal errors
    }
};

/**
 * Cancels any stored reservation for an event and clears storage on success or 404.
 * Preserves storage if cancellation fails due to network/server errors so state is not lost.
 * @param {string} eventId - Event UUID
 */
export const cancelStoredReservation = async (eventId) => {
    const stored = getStoredReservation(eventId);
    if (!stored?.reservationId) {
        clearStoredReservation(eventId);
        return null;
    }

    // Do not clear storage if cancelReservation throws a genuine network/server error.
    const result = await cancelReservation(stored.reservationId);
    clearStoredReservation(eventId);
    return result;
};
