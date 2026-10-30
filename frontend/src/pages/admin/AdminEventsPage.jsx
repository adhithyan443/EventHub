import { useState, useEffect } from "react";
import { getAdminEvents, getCategories } from "../../api/adminEventApi";
import AdminEventDetailDrawer from "../../components/admin/AdminEventDetailDrawer";
import EditEventModal from "../../components/admin/EditEventModal";
import PublishEventModal from "../../components/admin/PublishEventModal";
import DeleteEventModal from "../../components/admin/DeleteEventModal";
import notify from "../../utils/toast";

const PER_PAGE = 10;

const statusBadgeStyles = {
  PUBLISHED: "bg-[#dcfce7] text-[#15803d] border border-[#bbf7d0]",
  DRAFT: "bg-[#fef9c3] text-[#854d0e] border border-[#fef08a]",
  CANCELLED: "bg-[#fee2e2] text-[#b91c1c] border border-[#fecaca]",
};

const eventTypeBadgeStyles = {
  PHYSICAL: "bg-[#e0f2fe] text-[#0369a1] border border-[#bae6fd]",
  ONLINE: "bg-[#f3e8ff] text-[#6b21a8] border border-[#e9d5ff]",
  HYBRID: "bg-[#fef3c7] text-[#92400e] border border-[#fde68a]",
};

export default function AdminEventsPage() {
  const [events, setEvents] = useState([]);
  const [total, setTotal] = useState(0);
  const [statusCounts, setStatusCounts] = useState({
    total: 0,
    published: 0,
    draft: 0,
    cancelled: 0,
  });

  const [categories, setCategories] = useState([]);
  const [activeStatus, setActiveStatus] = useState("all");
  const [activeCategory, setActiveCategory] = useState("all");
  const [activeEventType, setActiveEventType] = useState("all");
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  // Modals & Drawer State
  const [activeDrawerEvent, setActiveDrawerEvent] = useState(null);
  const [activeEditEvent, setActiveEditEvent] = useState(null);
  const [activePublishEvent, setActivePublishEvent] = useState(null);
  const [activeDeleteEvent, setActiveDeleteEvent] = useState(null);

  const [refreshKey, setRefreshKey] = useState(0);

  // Load categories once safely
  useEffect(() => {
    let cancelled = false;

    getCategories()
      .then((res) => {
        if (cancelled) return;
        const list = Array.isArray(res?.data?.categories)
          ? res.data.categories
          : Array.isArray(res?.data)
          ? res.data
          : Array.isArray(res)
          ? res
          : [];
        setCategories(list);
      })
      .catch((err) => {
        if (cancelled) return;
        console.error("Failed to load categories:", err);
        setCategories([]);
      });

    return () => {
      cancelled = true;
    };
  }, []);

  // Debounce search by 300ms
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  // Fetch events when filters, page, debounced search, or refreshKey change
  useEffect(() => {
    let cancelled = false;

    const fetchEvents = async () => {
      try {
        setLoading(true);
        setError("");

        const res = await getAdminEvents({
          page,
          limit: PER_PAGE,
          search: debouncedSearch,
          status: activeStatus === "all" ? "" : activeStatus,
          categoryId: activeCategory === "all" ? "" : activeCategory,
          eventType: activeEventType === "all" ? "" : activeEventType,
        });

        if (cancelled) return;

        if (res?.data) {
          const eventsList = Array.isArray(res.data.events)
            ? res.data.events
            : Array.isArray(res.data)
            ? res.data
            : [];
          setEvents(eventsList);
          setTotal(typeof res.data.total === "number" ? res.data.total : 0);
          if (res.data.status_counts) {
            setStatusCounts(res.data.status_counts);
          }
        } else {
          setEvents([]);
          setTotal(0);
        }
      } catch (err) {
        if (cancelled) return;
        console.error("Failed to fetch events:", err);
        const msg =
          err.response?.data?.message || err.message || "Failed to load events.";
        setError(msg);
        setEvents([]);
        setTotal(0);
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
  }, [page, activeStatus, activeCategory, activeEventType, debouncedSearch, refreshKey]);

  const handleStatusChange = (status) => {
    setActiveStatus(status);
    setPage(1);
  };

  const handleClearFilters = () => {
    setSearch("");
    setDebouncedSearch("");
    setActiveStatus("all");
    setActiveCategory("all");
    setActiveEventType("all");
    setPage(1);
  };

  const handleEventUpdated = () => {
    setRefreshKey((k) => k + 1);
    if (activeDrawerEvent) {
      setActiveDrawerEvent(null);
    }
  };

  const handleEventPublished = () => {
    setRefreshKey((k) => k + 1);
    if (activeDrawerEvent) {
      setActiveDrawerEvent(null);
    }
  };

  const handleEventDeleted = () => {
    setRefreshKey((k) => k + 1);
    if (activeDrawerEvent) {
      setActiveDrawerEvent(null);
    }
  };

  const totalPages = Math.max(1, Math.ceil(total / PER_PAGE));

  const statusTabs = [
    { key: "all", label: "All Events", count: statusCounts?.total ?? 0 },
    { key: "published", label: "Published", count: statusCounts?.published ?? 0 },
    { key: "draft", label: "Draft", count: statusCounts?.draft ?? 0 },
    { key: "cancelled", label: "Cancelled", count: statusCounts?.cancelled ?? 0 },
  ];

  const formatDate = (dateStr) => {
    if (!dateStr) return "—";
    const d = new Date(dateStr);
    return isNaN(d.getTime())
      ? "—"
      : d.toLocaleDateString("en-IN", {
          day: "numeric",
          month: "short",
          year: "numeric",
        });
  };

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-white w-full">
      {/* Page Header */}
      <div className="px-4 sm:px-6 py-4 border-b border-[#dce2f7] flex flex-col sm:flex-row sm:items-center justify-between gap-3 shrink-0">
        <div>
          <h1 className="text-[#141b2b] text-xl font-bold tracking-tight">
            Events Management
          </h1>
          <p className="text-[#3d4947] text-xs sm:text-sm">
            Monitor, inspect, update, and manage all events across the platform.
          </p>
        </div>
        <button
          type="button"
          onClick={() => notify.info("Exporting events list...")}
          className="bg-[#f9f9ff] border border-[#bcc9c6] text-[#141b2b] text-xs font-semibold px-3.5 py-2 rounded-lg flex items-center gap-2 hover:bg-[#eef0fb] transition-colors cursor-pointer self-start sm:self-auto shrink-0"
        >
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
            <path
              d="M6 0L10.5 4.5H7.5V9H4.5V4.5H1.5L6 0ZM0 10.5H12V12H0V10.5Z"
              fill="currentColor"
            />
          </svg>
          Export Events
        </button>
      </div>

      {/* Main Content Area */}
      <div className="flex-1 overflow-y-auto p-4 sm:p-6 space-y-5">
        {/* Error Alert */}
        {error && (
          <div className="flex items-center justify-between rounded-lg border border-[#ffdad6] bg-[#fff5f4] px-4 py-3">
            <span className="text-[#ba1a1a] text-xs font-medium">{error}</span>
            <button
              type="button"
              onClick={() => setError("")}
              className="text-[#ba1a1a] text-xs font-semibold cursor-pointer hover:underline"
            >
              Dismiss
            </button>
          </div>
        )}

        {/* Platform-Wide Aggregate Summary Metric Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
            <div className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider mb-1">
              Total Events
            </div>
            <div className="text-[#141b2b] text-2xl font-bold tracking-tight">
              {statusCounts?.total ?? 0}
            </div>
            <div className="text-xs mt-1 text-[#00685f] font-medium">
              Platform-wide total created
            </div>
          </div>

          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
            <div className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider mb-1">
              Published Events
            </div>
            <div className="text-[#141b2b] text-2xl font-bold tracking-tight">
              {statusCounts?.published ?? 0}
            </div>
            <div className="text-xs mt-1 text-[#15803d] font-medium">
              Live & open for ticket bookings
            </div>
          </div>

          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
            <div className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider mb-1">
              Draft Events
            </div>
            <div className="text-[#141b2b] text-2xl font-bold tracking-tight">
              {statusCounts?.draft ?? 0}
            </div>
            <div className="text-xs mt-1 text-[#854d0e] font-medium">
              Incomplete or pending publish
            </div>
          </div>

          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
            <div className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider mb-1">
              Cancelled Events
            </div>
            <div className="text-[#141b2b] text-2xl font-bold tracking-tight">
              {statusCounts?.cancelled ?? 0}
            </div>
            <div className="text-xs mt-1 text-[#ba1a1a] font-medium">
              Cancelled events (view-only)
            </div>
          </div>
        </div>

        {/* Filter Bar: Tabs, Search, and Dropdowns */}
        <div className="bg-white border border-[#dce2f7] rounded-xl p-4 space-y-4 shadow-sm">
          {/* Status Tabs */}
          <div className="flex items-center gap-1 border-b border-[#f1f3ff] pb-3 overflow-x-auto text-xs">
            {statusTabs.map((tab) => {
              const isActive = activeStatus === tab.key;
              return (
                <button
                  key={tab.key}
                  type="button"
                  onClick={() => handleStatusChange(tab.key)}
                  className={`px-3 py-1.5 rounded-lg font-semibold flex items-center gap-2 transition-colors cursor-pointer shrink-0 ${
                    isActive
                      ? "bg-[#141b2b] text-white shadow-sm"
                      : "text-[#3d4947] hover:bg-[#eef0fb] hover:text-[#141b2b]"
                  }`}
                >
                  <span>{tab.label}</span>
                  <span
                    className={`text-[10px] px-1.5 py-0.2 rounded-full font-bold ${
                      isActive
                        ? "bg-white/20 text-white"
                        : "bg-[#eef0fb] text-[#3d4947]"
                    }`}
                  >
                    {tab.count}
                  </span>
                </button>
              );
            })}
          </div>

          {/* Search and Dropdown Controls */}
          <div className="flex flex-col md:flex-row items-stretch md:items-center gap-3">
            {/* Search Input */}
            <div className="relative flex-1">
              <svg
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                className="absolute left-3 top-1/2 -translate-y-1/2 text-[#3d4947]"
              >
                <circle cx="11" cy="11" r="8" />
                <line x1="21" y1="21" x2="16.65" y2="16.65" />
              </svg>
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search events by title, organizer, or venue..."
                className="w-full text-xs pl-9 pr-8 py-2 border border-[#bcc9c6] rounded-lg focus:outline-none focus:border-[#141b2b] bg-[#f9f9ff]"
              />
              {search && (
                <button
                  type="button"
                  onClick={() => setSearch("")}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 text-[#3d4947] hover:text-[#141b2b] text-xs cursor-pointer"
                >
                  ✕
                </button>
              )}
            </div>

            {/* Event Type Filter */}
            <div className="w-full md:w-44">
              <select
                value={activeEventType}
                onChange={(e) => {
                  setActiveEventType(e.target.value);
                  setPage(1);
                }}
                className="w-full text-xs px-3 py-2 border border-[#bcc9c6] rounded-lg focus:outline-none focus:border-[#141b2b] bg-white cursor-pointer"
              >
                <option value="all">All Event Types</option>
                <option value="PHYSICAL">Physical Venue</option>
                <option value="ONLINE">Online Stream</option>
                <option value="HYBRID">Hybrid</option>
              </select>
            </div>

            {/* Category Filter */}
            <div className="w-full md:w-44">
              <select
                value={activeCategory}
                onChange={(e) => {
                  setActiveCategory(e.target.value);
                  setPage(1);
                }}
                className="w-full text-xs px-3 py-2 border border-[#bcc9c6] rounded-lg focus:outline-none focus:border-[#141b2b] bg-white cursor-pointer"
              >
                <option value="all">All Categories</option>
                {(Array.isArray(categories) ? categories : []).map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </select>
            </div>

            {/* Clear Filters (if active) */}
            {(search || activeStatus !== "all" || activeCategory !== "all" || activeEventType !== "all") && (
              <button
                type="button"
                onClick={handleClearFilters}
                className="text-xs text-[#ba1a1a] hover:underline font-semibold px-2 py-1 cursor-pointer shrink-0 self-center"
              >
                Reset Filters
              </button>
            )}
          </div>
        </div>

        {/* Events Data Table */}
        <div className="bg-white border border-[#dce2f7] rounded-xl overflow-hidden shadow-sm">
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-[#f9f9ff] border-b border-[#dce2f7] text-[#3d4947] text-[11px] font-bold uppercase tracking-wider">
                  <th className="py-3 px-4">Event</th>
                  <th className="py-3 px-4">Organizer</th>
                  <th className="py-3 px-4">Schedule & Venue</th>
                  <th className="py-3 px-4">Capacity & Sales</th>
                  <th className="py-3 px-4">Status</th>
                  <th className="py-3 px-4 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#f1f3ff] text-xs">
                {loading ? (
                  Array.from({ length: 5 }).map((_, idx) => (
                    <tr key={idx} className="animate-pulse">
                      <td className="py-4 px-4">
                        <div className="flex items-center gap-3">
                          <div className="w-12 h-10 bg-slate-200 rounded-lg shrink-0" />
                          <div className="space-y-1.5 flex-1">
                            <div className="w-3/4 h-3 bg-slate-200 rounded" />
                            <div className="w-1/3 h-2.5 bg-slate-100 rounded" />
                          </div>
                        </div>
                      </td>
                      <td className="py-4 px-4">
                        <div className="w-28 h-3 bg-slate-200 rounded mb-1" />
                        <div className="w-20 h-2.5 bg-slate-100 rounded" />
                      </td>
                      <td className="py-4 px-4">
                        <div className="w-24 h-3 bg-slate-200 rounded mb-1" />
                        <div className="w-16 h-2.5 bg-slate-100 rounded" />
                      </td>
                      <td className="py-4 px-4">
                        <div className="w-20 h-3 bg-slate-200 rounded mb-1" />
                        <div className="w-14 h-2.5 bg-slate-100 rounded" />
                      </td>
                      <td className="py-4 px-4">
                        <div className="w-16 h-5 bg-slate-200 rounded-full" />
                      </td>
                      <td className="py-4 px-4 text-right">
                        <div className="w-24 h-6 bg-slate-200 rounded ml-auto" />
                      </td>
                    </tr>
                  ))
                ) : events.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="py-12 text-center text-[#3d4947]">
                      <div className="flex flex-col items-center justify-center gap-2">
                        <div className="w-12 h-12 rounded-full bg-[#f1f3ff] flex items-center justify-center text-[#141b2b]">
                          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                            <rect x="3" y="4" width="18" height="18" rx="2" ry="2" />
                            <line x1="16" y1="2" x2="16" y2="6" />
                            <line x1="8" y1="2" x2="8" y2="6" />
                            <line x1="3" y1="10" x2="21" y2="10" />
                          </svg>
                        </div>
                        <p className="font-semibold text-sm text-[#141b2b]">No events found</p>
                        <p className="text-xs text-[#3d4947]">
                          Try adjusting your search query, status tab, or category filter.
                        </p>
                        <button
                          type="button"
                          onClick={handleClearFilters}
                          className="mt-2 text-xs font-semibold text-[#00685f] hover:underline cursor-pointer"
                        >
                          Clear all filters
                        </button>
                      </div>
                    </td>
                  </tr>
                ) : (
                  (Array.isArray(events) ? events : []).map((ev) => {
                    if (!ev) return null;

                    const statusClass =
                      statusBadgeStyles[ev.status?.toUpperCase()] ||
                      "bg-[#f1f3ff] text-[#141b2b]";
                    const typeClass =
                      eventTypeBadgeStyles[ev.event_type?.toUpperCase()] ||
                      "bg-[#f1f3ff] text-[#141b2b]";

                    const banner = ev.banner_image_url || ev.banner_url;

                    return (
                      <tr
                        key={ev.id}
                        className="hover:bg-[#f9f9ff] transition-colors"
                      >
                        {/* Event Title, Category, Type */}
                        <td className="py-3.5 px-4">
                          <div className="flex items-center gap-3 min-w-[200px]">
                            {banner ? (
                              <img
                                src={banner}
                                alt={ev.title || "Event banner"}
                                className="w-12 h-9 rounded-lg object-cover bg-slate-100 shrink-0 border border-[#dce2f7]"
                                onError={(e) => {
                                  e.target.style.display = "none";
                                }}
                              />
                            ) : (
                              <div className="w-12 h-9 rounded-lg bg-[#eef0fb] text-[#141b2b] flex items-center justify-center font-bold text-xs shrink-0">
                                📅
                              </div>
                            )}
                            <div className="min-w-0">
                              <span
                                onClick={() => setActiveDrawerEvent(ev)}
                                className="font-bold text-[#141b2b] hover:text-[#00685f] hover:underline cursor-pointer block truncate"
                              >
                                {ev.title || "Untitled Event"}
                              </span>
                              <div className="flex items-center gap-1.5 mt-0.5">
                                <span className="text-[10px] text-[#3d4947]">
                                  {ev.category_name || "General"}
                                </span>
                                <span className="text-[#bcc9c6]">•</span>
                                <span
                                  className={`text-[9px] font-bold px-1.5 py-0.2 rounded ${typeClass}`}
                                >
                                  {ev.event_type || "PHYSICAL"}
                                </span>
                              </div>
                            </div>
                          </div>
                        </td>

                        {/* Organizer */}
                        <td className="py-3.5 px-4">
                          <div className="min-w-[140px]">
                            <span className="font-semibold text-[#141b2b] block truncate">
                              {ev.organizer_business_name || "Organizer"}
                            </span>
                            <span className="text-[11px] text-[#3d4947] truncate block">
                              {ev.organizer_email || "—"}
                            </span>
                          </div>
                        </td>

                        {/* Schedule & Venue */}
                        <td className="py-3.5 px-4">
                          <div className="min-w-[130px]">
                            <span className="font-medium text-[#141b2b] block">
                              {formatDate(ev.event_date)}
                            </span>
                            <span className="text-[11px] text-[#3d4947] block truncate">
                              {ev.event_type === "ONLINE"
                                ? "Online Event"
                                : ev.venue_city || ev.venue_name || "Physical Venue"}
                            </span>
                          </div>
                        </td>

                        {/* Capacity & Sales */}
                        <td className="py-3.5 px-4">
                          <div className="min-w-[110px]">
                            <div className="flex items-center gap-1">
                              <span className="font-semibold text-[#00685f]">
                                {ev.tickets_sold ?? 0}
                              </span>
                              <span className="text-[#3d4947]">
                                / {ev.ticket_capacity || "—"}
                              </span>
                            </div>
                            <span className="text-[10px] text-[#3d4947]">
                              {ev.tickets_available ?? 0} available
                            </span>
                          </div>
                        </td>

                        {/* Status */}
                        <td className="py-3.5 px-4">
                          <span
                            className={`text-[10px] font-bold px-2 py-0.5 rounded-full inline-block ${statusClass}`}
                          >
                            {ev.status}
                          </span>
                        </td>

                        {/* Action Buttons (Strictly NO Cancel Action Button) */}
                        <td className="py-3.5 px-4 text-right">
                          <div className="flex items-center justify-end gap-1.5 shrink-0">
                            {/* View Details */}
                            <button
                              type="button"
                              onClick={() => setActiveDrawerEvent(ev)}
                              className="text-xs font-semibold px-2.5 py-1 text-[#141b2b] hover:bg-[#eef0fb] rounded-md transition-colors cursor-pointer"
                              title="View Event Details"
                            >
                              Details
                            </button>

                            {/* Publish (Draft only) */}
                            {ev.status === "DRAFT" && (
                              <button
                                type="button"
                                onClick={() => setActivePublishEvent(ev)}
                                className="text-xs font-semibold px-2.5 py-1 bg-[#00685f] hover:bg-[#005049] text-white rounded-md transition-colors cursor-pointer shadow-sm"
                                title="Publish Draft Event"
                              >
                                Publish
                              </button>
                            )}

                            {/* Edit (Draft or Published, not Cancelled) */}
                            {ev.status !== "CANCELLED" && (
                              <button
                                type="button"
                                onClick={() => setActiveEditEvent(ev)}
                                className="text-xs font-semibold px-2.5 py-1 border border-[#bcc9c6] text-[#141b2b] hover:bg-[#eef0fb] rounded-md transition-colors cursor-pointer"
                                title="Edit Event"
                              >
                                Edit
                              </button>
                            )}

                            {/* Delete */}
                            <button
                              type="button"
                              onClick={() => setActiveDeleteEvent(ev)}
                              className="text-xs font-semibold px-2 py-1 text-[#ba1a1a] hover:bg-[#fff5f4] rounded-md transition-colors cursor-pointer"
                              title="Delete Event"
                            >
                              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                                <polyline points="3 6 5 6 21 6" />
                                <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                              </svg>
                            </button>
                          </div>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>

          {/* Pagination Controls */}
          <div className="p-4 border-t border-[#dce2f7] bg-[#f9f9ff] flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-[#3d4947]">
            <div>
              Showing{" "}
              <span className="font-semibold text-[#141b2b]">
                {events.length > 0 ? (page - 1) * PER_PAGE + 1 : 0}
              </span>{" "}
              to{" "}
              <span className="font-semibold text-[#141b2b]">
                {Math.min(page * PER_PAGE, total)}
              </span>{" "}
              of <span className="font-semibold text-[#141b2b]">{total}</span>{" "}
              events
            </div>

            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1 || loading}
                className="px-3 py-1.5 border border-[#bcc9c6] rounded-lg text-[#141b2b] hover:bg-[#eef0fb] transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed font-semibold"
              >
                Previous
              </button>
              <span className="px-2 font-medium">
                Page {page} of {totalPages}
              </span>
              <button
                type="button"
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages || loading}
                className="px-3 py-1.5 border border-[#bcc9c6] rounded-lg text-[#141b2b] hover:bg-[#eef0fb] transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed font-semibold"
              >
                Next
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Slide-over Event Detail Drawer */}
      {activeDrawerEvent && (
        <AdminEventDetailDrawer
          event={activeDrawerEvent}
          onClose={() => setActiveDrawerEvent(null)}
          onEditClick={(ev) => {
            setActiveEditEvent(ev);
          }}
          onPublishClick={(ev) => {
            setActivePublishEvent(ev);
          }}
          onDeleteClick={(ev) => {
            setActiveDeleteEvent(ev);
          }}
        />
      )}

      {/* Edit Event Modal */}
      {activeEditEvent && (
        <EditEventModal
          event={activeEditEvent}
          isOpen={Boolean(activeEditEvent)}
          onClose={() => setActiveEditEvent(null)}
          onSuccess={handleEventUpdated}
        />
      )}

      {/* Publish Event Modal */}
      {activePublishEvent && (
        <PublishEventModal
          event={activePublishEvent}
          isOpen={Boolean(activePublishEvent)}
          onClose={() => setActivePublishEvent(null)}
          onSuccess={handleEventPublished}
        />
      )}

      {/* Delete Event Modal */}
      {activeDeleteEvent && (
        <DeleteEventModal
          event={activeDeleteEvent}
          isOpen={Boolean(activeDeleteEvent)}
          onClose={() => setActiveDeleteEvent(null)}
          onSuccess={handleEventDeleted}
        />
      )}
    </div>
  );
}
