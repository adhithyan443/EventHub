import apiClient from "./client.js";

export const getPublicEvents = async ({
    page = 1,
    limit = 12,
    keyword = "",
    categoryId = "",
    city = "",
    date = "",
    startDate = "",
    endDate = "",
    minPrice = "",
    maxPrice = "",
    sort = "",
} = {}) => {
    const params = {
        page,
        limit,
    };

    if (keyword && typeof keyword === "string" && keyword.trim()) {
        params.keyword = keyword.trim();
    }

    if (categoryId) {
        params.category_id = categoryId;
    }

    if (city && typeof city === "string" && city.trim()) {
        params.city = city.trim();
    }

    if (date) {
        params.date = date;
    }

    if (startDate) {
        params.start_date = startDate;
    }

    if (endDate) {
        params.end_date = endDate;
    }

    if (minPrice !== "" && minPrice !== null && minPrice !== undefined) {
        params.min_price = minPrice;
    }

    if (maxPrice !== "" && maxPrice !== null && maxPrice !== undefined) {
        params.max_price = maxPrice;
    }

    if (sort) {
        params.sort = sort;
    }

    const response = await apiClient.get("/events", {
        params,
    });

    return response.data;
};

export const getCategories = async () => {
    const response = await apiClient.get("/categories");
    return response.data;
};

export const getPublicEventById = async (eventId) => {
    const response = await apiClient.get(`/events/${eventId}`);
    return response.data;
};

export const getPublicEventSeatLayout = async (eventId) => {
    const response = await apiClient.get(`/events/${eventId}/seat-layout`);
    return response.data;
};


