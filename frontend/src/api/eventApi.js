import apiClient from "./client";

export const getPublicEvents = async ({
    page = 1,
    limit = 12,
    keyword = "",
    categoryId = "",
    city = "",
    date = "",
} = {}) => {
    const params = {
        page,
        limit,
    };

    if (keyword.trim()) {
        params.keyword = keyword.trim();
    }

    if (categoryId) {
        params.category_id = categoryId;
    }

    if (city.trim()) {
        params.city = city.trim();
    }

    if (date) {
        params.date = date;
    }

    const response = await apiClient.get("/events", {
        params,
    });

    return response.data;
};

export const getPublicEventById = async (eventId) => {
    const response = await apiClient.get(`/events/${eventId}`);
    return response.data;
};