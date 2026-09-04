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