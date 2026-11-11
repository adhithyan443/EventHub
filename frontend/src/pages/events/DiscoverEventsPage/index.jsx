import { useEffect, useState, useCallback, useRef } from "react";
import { useSearchParams } from "react-router-dom";
import AppHeader from "../../../components/layout/AppHeader";
import HeroSearch from "./HeroSearch";
import CategoryPills from "./CategoryPills";
import FiltersSidebar from "./FiltersSidebar";
import EventCard from "./EventCard";
import Pagination from "./Pagination";
import { getPublicEvents, getCategories } from "../../../api/eventApi";

const mapEvent = (event) => {
    const rawDate = event.EventDate || event.event_date;
    const eventDate = rawDate ? new Date(rawDate) : null;

    const dateLabel = eventDate && !isNaN(eventDate.getTime())
        ? eventDate.toLocaleDateString("en-IN", {
            day: "2-digit",
            month: "short",
            year: "numeric",
        })
        : "Date unavailable";

    const venueName = event.VenueName || event.venue_name;
    const venueCity = event.VenueCity || event.venue_city;
    const eventType = event.EventType || event.event_type;

    const location =
        eventType === "ONLINE"
            ? "Online Event"
            : [venueName, venueCity]
                .filter(Boolean)
                .join(", ") || "Location unavailable";

    return {
        id: event.ID || event.id,
        title: event.Title || event.title || "Untitled Event",
        category: event.CategoryName || event.category_name || "General",
        date: dateLabel,
        location,
        price: event.StartingPrice ?? event.starting_price ?? 0,
        image:
            event.BannerImageURL ||
            event.banner_image_url ||
            event.BannerURL ||
            event.banner_url ||
            "",
        featured: false,
    };
};

export default function DiscoverEventsPage() {
    const [searchParams, setSearchParams] = useSearchParams();
    const selectedCategoryId =
        searchParams.get("category_id") ||
        searchParams.get("category") ||
        searchParams.get("categoryId") ||
        "";

    const [events, setEvents] = useState([]);
    const [categories, setCategories] = useState([]);
    const [categoriesLoading, setCategoriesLoading] = useState(true);
    const [categoriesError, setCategoriesError] = useState("");

    const selectedCategoryName = (() => {
        if (!selectedCategoryId) return "All Events";
        const found = categories.find(
            (c) => (c.ID || c.id) === selectedCategoryId
        );
        if (found) return found.Name || found.name;
        return searchParams.get("name") || "All Events";
    })();

    const [activeFilter, setActiveFilter] = useState("categories");
    const [currentPage, setCurrentPage] = useState(1);

    const [keyword, setKeyword] = useState("");
    const [city, setCity] = useState("");
    const [date, setDate] = useState("");
    const [startDate, setStartDate] = useState("");
    const [endDate, setEndDate] = useState("");
    const [minPrice, setMinPrice] = useState("");
    const [maxPrice, setMaxPrice] = useState("");
    const [sort, setSort] = useState("date_asc");

    // Debounced values for text/number inputs to prevent spamming requests on rapid typing
    const [debouncedKeyword, setDebouncedKeyword] = useState("");
    const [debouncedCity, setDebouncedCity] = useState("");
    const [debouncedMinPrice, setDebouncedMinPrice] = useState("");
    const [debouncedMaxPrice, setDebouncedMaxPrice] = useState("");
    const [debouncedStartDate, setDebouncedStartDate] = useState("");
    const [debouncedEndDate, setDebouncedEndDate] = useState("");

    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");
    const [total, setTotal] = useState(0);
    const limit = 12;

    const isInitialMount = useRef(true);

    // Fetch categories on retry
    const loadCategories = useCallback(() => {
        setCategoriesLoading(true);
        setCategoriesError("");

        getCategories()
            .then((res) => {
                const list = Array.isArray(res?.data?.categories)
                    ? res.data.categories
                    : Array.isArray(res?.data)
                    ? res.data
                    : [];
                setCategories(list);
            })
            .catch((err) => {
                console.error("Failed to load categories:", err);
                setCategoriesError("Unable to load event categories.");
            })
            .finally(() => {
                setCategoriesLoading(false);
            });
    }, []);

    useEffect(() => {
        let cancelled = false;

        getCategories()
            .then((res) => {
                if (cancelled) return;
                const list = Array.isArray(res?.data?.categories)
                    ? res.data.categories
                    : Array.isArray(res?.data)
                    ? res.data
                    : [];
                setCategories(list);
            })
            .catch((err) => {
                if (cancelled) return;
                console.error("Failed to load categories:", err);
                setCategoriesError("Unable to load event categories.");
            })
            .finally(() => {
                if (!cancelled) {
                    setCategoriesLoading(false);
                }
            });

        return () => {
            cancelled = true;
        };
    }, []);

    // Debounce filter inputs (300ms)
    useEffect(() => {
        const timer = setTimeout(() => {
            setDebouncedKeyword(keyword);
            setDebouncedCity(city);
            setDebouncedMinPrice(minPrice);
            setDebouncedMaxPrice(maxPrice);
            setDebouncedStartDate(startDate);
            setDebouncedEndDate(endDate);
        }, 300);

        return () => clearTimeout(timer);
    }, [keyword, city, minPrice, maxPrice, startDate, endDate]);

    // Reset page to 1 when any filter changes (after initial mount)
    useEffect(() => {
        if (isInitialMount.current) {
            isInitialMount.current = false;
            return;
        }
        setCurrentPage(1);
    }, [
        selectedCategoryId,
        debouncedKeyword,
        debouncedCity,
        debouncedStartDate,
        debouncedEndDate,
        debouncedMinPrice,
        debouncedMaxPrice,
        sort,
    ]);

    // Fetch events whenever page or debounced filters change
    useEffect(() => {
        let cancelled = false;

        const fetchEvents = async () => {
            try {
                setLoading(true);
                setError("");

                const response = await getPublicEvents({
                    page: currentPage,
                    limit,
                    keyword: debouncedKeyword,
                    categoryId: selectedCategoryId,
                    city: debouncedCity,
                    startDate: debouncedStartDate,
                    endDate: debouncedEndDate,
                    minPrice: debouncedMinPrice,
                    maxPrice: debouncedMaxPrice,
                    sort,
                });

                if (cancelled) return;

                const apiEvents = response?.data?.events || [];
                setEvents(apiEvents.map(mapEvent));
                setTotal(response?.data?.total || 0);
            } catch (err) {
                if (cancelled) return;
                console.error("Failed to load public events:", err);
                setError("Unable to load events. Please try again.");
            } finally {
                if (!cancelled) {
                    setLoading(false);
                }
            }
        };

        fetchEvents();

        return () => {
            cancelled = true;
        };
    }, [
        currentPage,
        selectedCategoryId,
        debouncedKeyword,
        debouncedCity,
        debouncedStartDate,
        debouncedEndDate,
        debouncedMinPrice,
        debouncedMaxPrice,
        sort,
    ]);

    // Category selection handler (used by both CategoryPills and FiltersSidebar)
    const handleSelectCategory = (categoryId, categoryName) => {
        setSearchParams((prev) => {
            const next = new URLSearchParams(prev);
            if (categoryId) {
                next.set("category_id", categoryId);
                if (categoryName && categoryName !== "All Events") {
                    next.set("name", categoryName);
                } else {
                    next.delete("name");
                }
            } else {
                next.delete("category_id");
                next.delete("category");
                next.delete("categoryId");
                next.delete("name");
            }
            return next;
        }, { replace: true });
    };

    // Hero search handler
    const handleSearch = () => {
        // Force debounced values immediately
        setDebouncedKeyword(keyword);
        setDebouncedCity(city);
        setDebouncedStartDate(date || startDate);
        setCurrentPage(1);
    };

    // When date in HeroSearch changes, also sync with startDate
    const handleHeroDateChange = (val) => {
        setDate(val);
        setStartDate(val);
        if (endDate && val && endDate < val) {
            setEndDate(val);
        }
    };

    // Reset All filters
    const handleReset = () => {
        setSearchParams({}, { replace: true });
        setActiveFilter("categories");
        setKeyword("");
        setCity("");
        setDate("");
        setStartDate("");
        setEndDate("");
        setMinPrice("");
        setMaxPrice("");
        setSort("date_asc");

        setDebouncedKeyword("");
        setDebouncedCity("");
        setDebouncedMinPrice("");
        setDebouncedMaxPrice("");
        setDebouncedStartDate("");
        setDebouncedEndDate("");

        setCurrentPage(1);
    };

    const hasActiveFilters = Boolean(
        selectedCategoryId ||
        keyword ||
        city ||
        date ||
        startDate ||
        endDate ||
        minPrice ||
        maxPrice
    );

    const totalPages = Math.max(1, Math.ceil(total / limit));

    return (
        <div className="min-h-screen bg-background">
            <AppHeader />

            <HeroSearch
                keyword={keyword}
                city={city}
                date={date}
                onKeywordChange={setKeyword}
                onCityChange={setCity}
                onDateChange={handleHeroDateChange}
                onSearch={handleSearch}
            />

            <CategoryPills
                categories={categories}
                selectedId={selectedCategoryId}
                loading={categoriesLoading}
                error={categoriesError}
                onRetry={loadCategories}
                onSelect={handleSelectCategory}
            />

            <div className="max-w-7xl mx-auto px-4 sm:px-6 py-6 sm:py-8 flex flex-col lg:flex-row gap-6 lg:gap-8 items-stretch lg:items-start">
                <FiltersSidebar
                    activeFilter={activeFilter}
                    onSelectFilter={setActiveFilter}
                    onReset={handleReset}
                    categories={categories}
                    selectedCategoryId={selectedCategoryId}
                    onSelectCategory={handleSelectCategory}
                    minPrice={minPrice}
                    maxPrice={maxPrice}
                    onMinPriceChange={setMinPrice}
                    onMaxPriceChange={setMaxPrice}
                    onClearPrice={() => {
                        setMinPrice("");
                        setMaxPrice("");
                    }}
                    startDate={startDate}
                    endDate={endDate}
                    onStartDateChange={(val) => {
                        setStartDate(val);
                        setDate(val);
                    }}
                    onEndDateChange={setEndDate}
                    onClearDate={() => {
                        setDate("");
                        setStartDate("");
                        setEndDate("");
                    }}
                    city={city}
                    onCityChange={setCity}
                    onClearCity={() => setCity("")}
                />

                <div className="flex-1 flex flex-col gap-8 min-w-0">
                    {loading && (
                        <div className="py-12 text-center text-ink/60">
                            Loading events...
                        </div>
                    )}

                    {!loading && error && (
                        <div className="py-12 text-center text-red-600">
                            <p>{error}</p>
                            <button
                                type="button"
                                onClick={() => setCurrentPage((p) => p)}
                                className="mt-3 px-4 py-2 bg-primary text-white text-sm font-semibold rounded-lg hover:bg-primary/90 transition-colors"
                            >
                                Try Again
                            </button>
                        </div>
                    )}

                    {!loading && !error && (
                        <div className="flex flex-col gap-6">
                            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                                <h2 className="font-display text-2xl sm:text-3xl font-bold text-ink">
                                    {selectedCategoryId ? `${selectedCategoryName} Events` : "All Events"}
                                    {total > 0 && (
                                        <span className="text-sm font-normal text-ink/50 ml-2">
                                            ({total})
                                        </span>
                                    )}
                                </h2>

                                <div className="flex items-center gap-2 shrink-0">
                                    <label htmlFor="event-sort" className="text-xs text-ink/60 font-medium">
                                        Sort by:
                                    </label>
                                    <select
                                        id="event-sort"
                                        value={sort}
                                        onChange={(e) => setSort(e.target.value)}
                                        className="bg-white border border-border rounded-lg px-3 py-1.5 text-xs font-semibold text-ink focus:outline-none focus:border-primary"
                                    >
                                        <option value="date_asc">Date: Soonest</option>
                                        <option value="date_desc">Date: Latest</option>
                                        <option value="price_asc">Price: Low to High</option>
                                        <option value="price_desc">Price: High to Low</option>
                                        <option value="newest">Recently Added</option>
                                    </select>
                                </div>
                            </div>

                            {events.length === 0 ? (
                                <div className="py-16 text-center text-ink/60 flex flex-col items-center gap-3">
                                    <p className="text-lg font-medium text-ink">No events found.</p>
                                    <p className="text-sm text-ink/60 max-w-sm">
                                        {hasActiveFilters
                                            ? "Try adjusting or clearing your filters to see more events."
                                            : "There are currently no upcoming events available."}
                                    </p>
                                    {hasActiveFilters && (
                                        <button
                                            type="button"
                                            onClick={handleReset}
                                            className="mt-2 text-sm font-semibold text-primary hover:underline"
                                        >
                                            Clear all filters
                                        </button>
                                    )}
                                </div>
                            ) : (
                                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-2 xl:grid-cols-3 gap-6">
                                    {events.map((event) => (
                                        <EventCard
                                            key={event.id}
                                            event={event}
                                        />
                                    ))}
                                </div>
                            )}

                            {total > limit && (
                                <Pagination
                                    currentPage={currentPage}
                                    totalPages={totalPages}
                                    onPageChange={setCurrentPage}
                                />
                            )}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}