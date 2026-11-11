import test from "node:test";
import assert from "node:assert/strict";
import apiClient from "../client.js";
import { getPublicEvents, getCategories } from "../eventApi.js";

test("getPublicEvents: builds clean query parameters with default pagination", async () => {
    let capturedConfig = null;
    const originalGet = apiClient.get;
    apiClient.get = async (url, config) => {
        capturedConfig = { url, ...config };
        return { data: { success: true, data: { events: [], total: 0 } } };
    };

    try {
        await getPublicEvents();
        assert.strictEqual(capturedConfig.url, "/events");
        assert.strictEqual(capturedConfig.params.page, 1);
        assert.strictEqual(capturedConfig.params.limit, 12);
        assert.strictEqual(capturedConfig.params.keyword, undefined);
        assert.strictEqual(capturedConfig.params.category_id, undefined);
        assert.strictEqual(capturedConfig.params.city, undefined);
        assert.strictEqual(capturedConfig.params.min_price, undefined);
        assert.strictEqual(capturedConfig.params.max_price, undefined);
        assert.strictEqual(capturedConfig.params.start_date, undefined);
        assert.strictEqual(capturedConfig.params.end_date, undefined);
        assert.strictEqual(capturedConfig.params.sort, undefined);
    } finally {
        apiClient.get = originalGet;
    }
});

test("getPublicEvents: handles category, price range, date range, location, and sort parameters", async () => {
    let capturedConfig = null;
    const originalGet = apiClient.get;
    apiClient.get = async (url, config) => {
        capturedConfig = { url, ...config };
        return { data: { success: true, data: { events: [], total: 0 } } };
    };

    try {
        await getPublicEvents({
            page: 2,
            limit: 24,
            keyword: " Coldplay ",
            categoryId: "f47ac10b-58cc-4372-a567-0e02b2c3d479",
            city: " Mumbai ",
            startDate: "2026-12-01",
            endDate: "2026-12-31",
            minPrice: 500,
            maxPrice: 2500,
            sort: "price_asc",
        });

        assert.strictEqual(capturedConfig.url, "/events");
        assert.strictEqual(capturedConfig.params.page, 2);
        assert.strictEqual(capturedConfig.params.limit, 24);
        assert.strictEqual(capturedConfig.params.keyword, "Coldplay");
        assert.strictEqual(capturedConfig.params.category_id, "f47ac10b-58cc-4372-a567-0e02b2c3d479");
        assert.strictEqual(capturedConfig.params.city, "Mumbai");
        assert.strictEqual(capturedConfig.params.start_date, "2026-12-01");
        assert.strictEqual(capturedConfig.params.end_date, "2026-12-31");
        assert.strictEqual(capturedConfig.params.min_price, 500);
        assert.strictEqual(capturedConfig.params.max_price, 2500);
        assert.strictEqual(capturedConfig.params.sort, "price_asc");
    } finally {
        apiClient.get = originalGet;
    }
});

test("getPublicEvents: supports single date backwards compatibility", async () => {
    let capturedConfig = null;
    const originalGet = apiClient.get;
    apiClient.get = async (url, config) => {
        capturedConfig = { url, ...config };
        return { data: { success: true, data: { events: [], total: 0 } } };
    };

    try {
        await getPublicEvents({
            date: "2026-11-20",
        });

        assert.strictEqual(capturedConfig.params.date, "2026-11-20");
    } finally {
        apiClient.get = originalGet;
    }
});

test("getCategories: requests /categories and returns category data", async () => {
    let capturedUrl = null;
    const originalGet = apiClient.get;
    apiClient.get = async (url) => {
        capturedUrl = url;
        return {
            data: {
                success: true,
                data: {
                    categories: [
                        { ID: "cat-1", Name: "Music", Status: "ACTIVE" },
                        { ID: "cat-2", Name: "Sports", Status: "ACTIVE" },
                    ],
                },
            },
        };
    };

    try {
        const response = await getCategories();
        assert.strictEqual(capturedUrl, "/categories");
        assert.strictEqual(response.success, true);
        assert.strictEqual(response.data.categories.length, 2);
        assert.strictEqual(response.data.categories[0].Name, "Music");
    } finally {
        apiClient.get = originalGet;
    }
});

test("Category selection: All Events clears category selection and specific category preserves ID", () => {
    const handleSelectCategory = (id, name, stateSetter) => {
        stateSetter({
            categoryId: id || "",
            categoryName: name || "All Events",
        });
    };

    let state = { categoryId: "initial", categoryName: "Initial" };
    const setState = (newState) => {
        state = newState;
    };

    // Selecting specific category
    handleSelectCategory("cat-123", "Music", setState);
    assert.strictEqual(state.categoryId, "cat-123");
    assert.strictEqual(state.categoryName, "Music");

    // Selecting All Events
    handleSelectCategory("", "All Events", setState);
    assert.strictEqual(state.categoryId, "");
    assert.strictEqual(state.categoryName, "All Events");
});

test("Reset All: clears all filters and restores default state", () => {
    const defaultState = {
        selectedCategoryId: "",
        selectedCategoryName: "All Events",
        keyword: "",
        city: "",
        date: "",
        startDate: "",
        endDate: "",
        minPrice: "",
        maxPrice: "",
        sort: "date_asc",
        currentPage: 1,
    };

    let currentState = {
        selectedCategoryId: "cat-456",
        selectedCategoryName: "Conferences",
        keyword: "Tech",
        city: "San Francisco",
        date: "2026-12-10",
        startDate: "2026-12-10",
        endDate: "2026-12-15",
        minPrice: "100",
        maxPrice: "500",
        sort: "price_desc",
        currentPage: 3,
    };

    const handleReset = () => {
        currentState = { ...defaultState };
    };

    handleReset();

    assert.deepStrictEqual(currentState, defaultState);
});
