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