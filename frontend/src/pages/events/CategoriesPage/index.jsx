import { useEffect, useState, useCallback } from "react";
import { Link, useNavigate } from "react-router-dom";
import AppHeader from "../../../components/layout/AppHeader";
import Footer from "../../../components/layout/Footer";
import { getCategories } from "../../../api/eventApi";
import { SearchIcon, ChevronRightIcon, GridIcon } from "../../../components/layout/icons";

// Category-specific SVG icons for visual polish
function CategoryCardIcon({ name }) {
  const normalized = (name || "").toLowerCase();

  if (normalized.includes("music") || normalized.includes("concert")) {
    return (
      <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2">
        <path strokeLinecap="round" strokeLinejoin="round" d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" />
      </svg>
    );
  }

  if (normalized.includes("sport") || normalized.includes("fitness")) {
    return (
      <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2">
        <path strokeLinecap="round" strokeLinejoin="round" d="M14.828 14.828a4 4 0 01-5.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    );
  }

  if (normalized.includes("tech") || normalized.includes("code") || normalized.includes("software")) {
    return (
      <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2">
        <path strokeLinecap="round" strokeLinejoin="round" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
      </svg>
    );
  }

  if (normalized.includes("comedy") || normalized.includes("standup")) {
    return (
      <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2">
        <path strokeLinecap="round" strokeLinejoin="round" d="M14.828 14.828a4 4 0 01-5.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    );
  }

  if (normalized.includes("theatre") || normalized.includes("theater") || normalized.includes("art")) {
    return (
      <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2">
        <path strokeLinecap="round" strokeLinejoin="round" d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01" />
      </svg>
    );
  }

  if (normalized.includes("conference") || normalized.includes("business")) {
    return (
      <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2">
        <path strokeLinecap="round" strokeLinejoin="round" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
      </svg>
    );
  }

  if (normalized.includes("workshop") || normalized.includes("education")) {
    return (
      <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2">
        <path strokeLinecap="round" strokeLinejoin="round" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
      </svg>
    );
  }

  return <GridIcon className="w-6 h-6" />;
}

export default function CategoriesPage() {
  const navigate = useNavigate();
  const [categories, setCategories] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [searchQuery, setSearchQuery] = useState("");

  const loadCategories = useCallback(() => {
    setLoading(true);
    setError("");

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
        console.error("Failed to fetch categories:", err);
        setError("Unable to load categories. Please check your connection and try again.");
      })
      .finally(() => {
        setLoading(false);
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
        console.error("Failed to fetch categories:", err);
        setError("Unable to load categories. Please check your connection and try again.");
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, []);

  const handleCategoryClick = (category) => {
    const categoryId = category.ID || category.id;
    const categoryName = category.Name || category.name || "";
    navigate(`/events?category_id=${categoryId}&name=${encodeURIComponent(categoryName)}`);
  };

  const filteredCategories = categories.filter((cat) => {
    const name = (cat.Name || cat.name || "").toLowerCase();
    const desc = (cat.Description || cat.description || "").toLowerCase();
    const q = searchQuery.toLowerCase().trim();
    return !q || name.includes(q) || desc.includes(q);
  });

  return (
    <div className="min-h-screen bg-background flex flex-col">
      <AppHeader />

      {/* Hero Header Section */}
      <section className="bg-gradient-to-b from-primary/5 to-background py-10 sm:py-16 px-4 sm:px-6 border-b border-border">
        <div className="max-w-4xl mx-auto flex flex-col items-center text-center gap-3">
          <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-primary/10 text-primary text-xs font-semibold tracking-wide uppercase">
            <span className="w-1.5 h-1.5 rounded-full bg-primary" />
            Explore Categories
          </span>

          <h1 className="font-display text-3xl sm:text-4xl md:text-5xl font-bold text-ink">
            Discover Events by Category
          </h1>

          <p className="text-sm sm:text-base text-ink/70 max-w-2xl">
            From live music and technology summits to comedy nights and hands-on workshops, find your next unforgettable experience.
          </p>

          {/* Quick Filter Input */}
          <div className="mt-4 w-full max-w-md bg-white rounded-lg border border-border shadow-sm flex items-center gap-2 px-3.5 py-2.5">
            <span className="text-slate-400">
              <SearchIcon />
            </span>
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search categories..."
              className="bg-transparent text-sm text-ink placeholder:text-ink/40 focus:outline-none w-full"
            />
            {searchQuery && (
              <button
                type="button"
                onClick={() => setSearchQuery("")}
                className="text-xs text-ink/50 hover:text-ink font-semibold"
              >
                Clear
              </button>
            )}
          </div>
        </div>
      </section>

      {/* Main Content Area */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 py-8 sm:py-12 flex-1 w-full flex flex-col gap-8">
        {/* Loading Skeletons */}
        {loading && (
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6">
            {Array.from({ length: 8 }).map((_, i) => (
              <div
                key={`cat-skeleton-${i}`}
                className="bg-[#f9f9ff] rounded-xl border border-border p-6 flex flex-col gap-4 animate-pulse"
              >
                <div className="w-12 h-12 rounded-xl bg-border/60" />
                <div className="h-6 w-3/4 rounded bg-border/60" />
                <div className="h-4 w-full rounded bg-border/50" />
                <div className="h-4 w-1/2 rounded bg-border/50" />
                <div className="mt-auto pt-4 border-t border-border flex justify-between items-center">
                  <div className="h-4 w-24 rounded bg-border/50" />
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Error State with Retry */}
        {!loading && error && (
          <div className="py-16 text-center text-red-600 flex flex-col items-center gap-3">
            <p className="text-lg font-semibold">{error}</p>
            <button
              type="button"
              onClick={loadCategories}
              className="px-5 py-2.5 bg-primary text-white text-sm font-semibold rounded-lg hover:bg-primary/90 transition-colors shadow-sm"
            >
              Try Again
            </button>
          </div>
        )}

        {/* Empty State */}
        {!loading && !error && filteredCategories.length === 0 && (
          <div className="py-16 text-center text-ink/60 flex flex-col items-center gap-3">
            <p className="text-lg font-medium text-ink">
              {searchQuery ? "No matching categories found." : "No categories found."}
            </p>
            <p className="text-sm text-ink/60 max-w-sm">
              {searchQuery
                ? `No categories match "${searchQuery}". Try a different keyword or browse all events.`
                : "There are currently no active categories available."}
            </p>
            <Link
              to="/events"
              className="mt-2 px-5 py-2.5 bg-primary text-white text-sm font-semibold rounded-lg hover:bg-primary/90 transition-colors"
            >
              Browse All Events
            </Link>
          </div>
        )}

        {/* Active Categories Grid */}
        {!loading && !error && filteredCategories.length > 0 && (
          <div className="flex flex-col gap-6">
            <div className="flex items-center justify-between">
              <h2 className="font-display text-xl sm:text-2xl font-bold text-ink">
                All Categories ({filteredCategories.length})
              </h2>
              <Link
                to="/events"
                className="text-xs sm:text-sm font-semibold text-primary hover:underline flex items-center gap-1"
              >
                <span>Browse All Events</span>
                <ChevronRightIcon />
              </Link>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6">
              {filteredCategories.map((category) => {
                const categoryId = category.ID || category.id;
                const categoryName = category.Name || category.name || "Category";
                const categoryDesc =
                  category.Description ||
                  category.description ||
                  `Explore trending upcoming ${categoryName.toLowerCase()} experiences and tickets.`;

                return (
                  <article
                    key={categoryId}
                    onClick={() => handleCategoryClick(category)}
                    className="bg-[#f9f9ff] rounded-xl overflow-hidden border border-border p-6 flex flex-col gap-4 hover:border-primary/50 hover:shadow-md transition-all group cursor-pointer"
                  >
                    <div className="w-12 h-12 rounded-xl bg-primary/10 text-primary flex items-center justify-center font-bold text-lg group-hover:bg-primary group-hover:text-white transition-colors">
                      <CategoryCardIcon name={categoryName} />
                    </div>

                    <div className="flex flex-col gap-1.5 flex-1">
                      <h3 className="font-display text-lg sm:text-xl font-bold text-ink group-hover:text-primary transition-colors">
                        {categoryName}
                      </h3>
                      <p className="text-sm text-ink/70 line-clamp-2 leading-relaxed">
                        {categoryDesc}
                      </p>
                    </div>

                    <div className="mt-auto pt-4 border-t border-border flex items-center justify-between text-xs font-semibold text-primary">
                      <span>Explore Events</span>
                      <span className="transform group-hover:translate-x-1 transition-transform">
                        <ChevronRightIcon />
                      </span>
                    </div>
                  </article>
                );
              })}
            </div>
          </div>
        )}
      </main>

      <Footer />
    </div>
  );
}
