import apiClient from "./client.js";

// Get organizer applications with pagination/status filtering.
export const getOrganizerApplications = async ({
    page = 1,
    limit = 10,
    status = "",
} = {}) => {
    const params = {
        page,
        limit,
    };

    // Only send status when a filter is selected.
    if (status) {
        params.status = status;
    }

    const response = await apiClient.get(
        "/admin/organizer-applications",
        {
            params,
        }
    );

    return response.data;
};

// Get one organizer application by its ID.
export const getOrganizerApplicationById = async (applicationId) => {
    const response = await apiClient.get(
        `/admin/organizer-applications/${applicationId}`
    );

    return response.data;
};

// Approve a pending organizer application.
export const approveOrganizerApplication = async (applicationId) => {
    const response = await apiClient.patch(
        `/admin/organizer-applications/${applicationId}/approve`
    );

    return response.data;
};

// Reject an organizer application.
export const rejectOrganizerApplication = async (applicationId, reason) => {
    const response = await apiClient.patch(
        `/admin/organizer-applications/${applicationId}/reject`,
        {
            reason,
        }
    );

    return response.data;
};

// Get approved organizers with pagination and search.
export const getOrganizers = async ({
    page = 1,
    limit = 10,
    search = "",
    status = "",
} = {}) => {
    const params = {
        page,
        limit,
    };

    if (search && search.trim() !== "") {
        params.search = search.trim();
    }

    if (status && status.trim() !== "") {
        params.status = status.trim();
    }

    const response = await apiClient.get(
        "/admin/organizers",
        {
            params,
        }
    );

    return response.data;
};