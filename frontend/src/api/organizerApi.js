import apiClient from "./client";

// Submit an organizer application for the authenticated customer.
export const applyAsOrganizer = async (applicationData) => {
    const response = await apiClient.post(
        "/organizers/apply",
        applicationData
    );

    return response.data;
};

// Get the authenticated customer's organizer application.
export const getOrganizerApplication = async () => {
    const response = await apiClient.get(
        "/organizers/application"
    );

    return response.data;
};

// Resubmit an existing rejected organizer application.
// Note: Currently, the backend processes both initial submission and resubmission of
// rejected applications via POST /organizers/apply (updating the existing application record in place).
// This function is isolated so that if a dedicated PUT/PATCH endpoint is introduced in the future,
// it can be configured here without touching component or store logic.
export const resubmitOrganizerApplication = async (applicationData) => {
    const response = await apiClient.post(
        "/organizers/apply",
        applicationData
    );

    return response.data;
};

// Create an event for the authenticated organizer.
export const createOrganizerEvent = async (eventData) => {
    const response = await apiClient.post(
        "/organizers/events",
        eventData
    );

    return response.data;
};

// Get active event categories.
export const getCategories = async () => {
    const response = await apiClient.get("/categories");

    return response.data;
};


export const getOrganizerProfile = async () => {
    const response = await apiClient.get("/organizers/profile");

    return response.data;
};

// Get events created by the authenticated organizer.
export const getOrganizerEvents = async ({
    page = 1,
    limit = 10,
    status = "",
    search = "",
} = {}) => {
    const response = await apiClient.get("/organizers/events", {
        params: {
            page,
            limit,
            ...(status ? { status } : {}),
            ...(search.trim() ? { search: search.trim() } : {}),
        },
    });

    return response.data;
};


export const getOrganizerEventById = async (eventId) => {
    const response = await apiClient.get(`/organizers/events/${eventId}`);

    return response.data;
};

// Update an existing draft event for the authenticated organizer.
export const updateOrganizerEvent = async (eventId, eventData) => {
    const response = await apiClient.put(
        `/organizers/events/${eventId}`,
        eventData
    );

    return response.data;
};

// Update the reserved seat layout for an existing draft event.
export const updateOrganizerSeatLayout = async (eventId, seatLayoutData) => {
    const response = await apiClient.put(
        `/organizers/events/${eventId}/seat-layout`,
        seatLayoutData
    );

    return response.data;
};

// Delete a draft event for the authenticated organizer.
export const deleteOrganizerEvent = async (eventId) => {
    const response = await apiClient.delete(
        `/organizers/events/${eventId}`
    );

    return response.data;
};


// Publish a draft event for the authenticated organizer.
export const publishOrganizerEvent = async (eventId) => {
    const response = await apiClient.patch(
        `/organizers/events/${eventId}/publish`
    );

    return response.data;
};