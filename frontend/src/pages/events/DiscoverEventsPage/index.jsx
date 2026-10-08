import { useEffect, useState } from "react";
import AppHeader from "../../../components/layout/AppHeader";
import HeroSearch from "./HeroSearch";
import CategoryPills from "./CategoryPills";
import FiltersSidebar from "./FiltersSidebar";
import FeaturedEvents from "./FeaturedEvents";
import EventCard from "./EventCard";
import Pagination from "./Pagination";
import { getPublicEvents } from "../../../api/eventApi";

const mapEvent = (event) => {
    const eventDate = event.EventDate
        ? new Date(event.EventDate)
        : null;

    const dateLabel = eventDate
        ? eventDate.toLocaleDateString("en-IN", {
            day: "2-digit",
            month: "short",
            year: "numeric",
        })
        : "Date unavailable";

    const location =
        event.EventType === "ONLINE"
            ? "Online Event"
            : [event.VenueName, event.VenueCity]
                .filter(Boolean)
                .join(", ") || "Location unavailable";

    return {
        id: event.ID,
        title: event.Title,
        category: event.CategoryName,
        date: dateLabel,
        location,
        price: event.StartingPrice ?? 0,
        image:
            event.BannerImageURL ||
            event.BannerURL ||
            "",
        featured: false,
    };
};

export default function DiscoverEventsPage() {
    const [events, setEvents] = useState([]);
    const [selectedCategory, setSelectedCategory] =
        useState("All Events");

    const [activeFilter, setActiveFilter] =
        useState("categories");

    const [currentPage, setCurrentPage] = useState(1);

    const [keyword, setKeyword] = useState("");
    const [city, setCity] = useState("");
    const [date, setDate] = useState("");

    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");

    const [total, setTotal] = useState(0);
    const limit = 12;

    const loadEvents = async () => {
        try {
            setLoading(true);
            setError("");

            const response = await getPublicEvents({
                page: currentPage,
                limit,
                keyword,
                city,
                date,
            });

            const apiEvents = response?.data?.events || [];

            setEvents(apiEvents.map(mapEvent));
            setTotal(response?.data?.total || 0);
        } catch (err) {
            console.error("Failed to load public events:", err);
            setError("Unable to load events. Please try again.");
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        let cancelled = false;

        const fetchEvents = async () => {
            try {
                setLoading(true);
                setError("");

                const response = await getPublicEvents({
                    page: currentPage,
                    limit,
                    keyword,
                    city,
                    date,
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
    }, [currentPage]);

    const handleSearch = () => {
        setCurrentPage(1);
        loadEvents();
    };

    const handleReset = () => {
        setSelectedCategory("All Events");
        setActiveFilter("categories");
        setKeyword("");
        setCity("");
        setDate("");
        setCurrentPage(1);
    };

    const totalPages = Math.max(
        1,
        Math.ceil(total / limit)
    );

    const featuredEvents = events.slice(0, 3);

    return (
        <div className="min-h-screen bg-background">
            <AppHeader />

            <HeroSearch
                keyword={keyword}
                city={city}
                date={date}
                onKeywordChange={setKeyword}
                onCityChange={setCity}
                onDateChange={setDate}
                onSearch={handleSearch}
            />

            <CategoryPills
                selected={selectedCategory}
                onSelect={(category) => {
                    setSelectedCategory(category);
                    setCurrentPage(1);
                }}
            />

            <div className="max-w-7xl mx-auto px-4 sm:px-6 py-6 sm:py-8 flex flex-col lg:flex-row gap-6 lg:gap-8 items-stretch lg:items-start">
                <FiltersSidebar
                    activeFilter={activeFilter}
                    onSelectFilter={setActiveFilter}
                    onReset={handleReset}
                />

                <div className="flex-1 flex flex-col gap-8 min-w-0">
                    {loading && (
                        <div className="py-12 text-center text-ink/60">
                            Loading events...
                        </div>
                    )}

                    {!loading && error && (
                        <div className="py-12 text-center text-red-600">
                            {error}
                        </div>
                    )}

                    {!loading && !error && (
                        <>
                            {featuredEvents.length > 0 && (
                                <FeaturedEvents
                                    events={featuredEvents}
                                />
                            )}

                            <div className="border-t border-border pt-8 flex flex-col gap-6">
                                <h2 className="font-display text-2xl sm:text-3xl font-bold text-ink">
                                    All Events
                                </h2>

                                {events.length === 0 ? (
                                    <div className="py-12 text-center text-ink/60">
                                        No events found.
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

                                <Pagination
                                    currentPage={currentPage}
                                    totalPages={totalPages}
                                    onPageChange={setCurrentPage}
                                />
                            </div>
                        </>
                    )}
                </div>
            </div>
        </div>
    );
}