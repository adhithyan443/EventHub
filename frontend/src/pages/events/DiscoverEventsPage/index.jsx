import { useState } from "react";
import AppHeader from "../../../components/layout/AppHeader";
import HeroSearch from "./HeroSearch";
import CategoryPills from "./CategoryPills";
import FiltersSidebar from "./FiltersSidebar";
import FeaturedEvents from "./FeaturedEvents";
import EventCard from "./EventCard";
import Pagination from "./Pagination";
import { mockEvents } from "./mockEvents";

export default function DiscoverEventsPage() {
    const [selectedCategory, setSelectedCategory] = useState("All Events");
    const [activeFilter, setActiveFilter] = useState("categories");
    const [currentPage, setCurrentPage] = useState(1);

    const featuredEvents = mockEvents.filter((e) => e.featured);

    function handleReset() {
        setSelectedCategory("All Events");
        setActiveFilter("categories");
    }

    return (
        <div className="min-h-screen bg-background">
            <AppHeader />
            <HeroSearch />
            <CategoryPills selected={selectedCategory} onSelect={setSelectedCategory} />

            <div className="max-w-7xl mx-auto px-6 py-8 flex gap-8 items-start">
                <FiltersSidebar
                    activeFilter={activeFilter}
                    onSelectFilter={setActiveFilter}
                    onReset={handleReset}
                />

                <div className="flex-1 flex flex-col gap-8">
                    <FeaturedEvents events={featuredEvents} />

                    <div className="border-t border-border pt-8 flex flex-col gap-6">
                        <h2 className="font-display text-3xl font-bold text-ink">All Events</h2>
                        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                            {mockEvents.map((event) => (
                                <EventCard key={event.id} event={event} />
                            ))}
                        </div>
                        <Pagination currentPage={currentPage} totalPages={12} onPageChange={setCurrentPage} />
                    </div>
                </div>
            </div>
        </div>
    );
}