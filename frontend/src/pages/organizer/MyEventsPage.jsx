import { useState, useMemo, useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { ORGANIZER_ROUTES, getEventDetailsRoute } from "../../constants/eventConstants";
import StatusBadge from "../../components/organizer/events/StatusBadge";
import ProgressBar from "../../components/organizer/events/ProgressBar";
import {
  PlusIcon,
  SearchIcon,
  CategoryIcon,
  DateIcon,
  StatusIcon,
  VenueIcon,
  SortIcon,
  MoreIcon,
} from "../../components/organizer/events/OrganizerEventsIcons";
import { ORGANIZER_EVENTS, TABS_CONFIG } from "./mockOrganizerEvents";

const ITEMS_PER_PAGE = 5;

export default function MyEventsPage() {
  const navigate = useNavigate();

  const [activeTab, setActiveTab] = useState(0);
  const [search, setSearch] = useState("");
  const [selectedCategory, setSelectedCategory] = useState("ALL");
  const [selectedDateFilter, setSelectedDateFilter] = useState("ALL");
  const [selectedStatusFilter, setSelectedStatusFilter] = useState("ALL");
  const [selectedVenueFilter, setSelectedVenueFilter] = useState("ALL");
  const [selectedSort, setSelectedSort] = useState("newest");
  const [currentPage, setCurrentPage] = useState(1);

  // Dropdown open states
  const [openDropdown, setOpenDropdown] = useState(null);
  const [activeActionMenuId, setActiveActionMenuId] = useState(null);

  const dropdownRef = useRef(null);

  // Close dropdowns on outside click
  useEffect(() => {
    function handleClickOutside(event) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
        setOpenDropdown(null);
      }
      if (!event.target.closest("[data-action-menu]")) {
        setActiveActionMenuId(null);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // Compute live tab counts
  const tabCounts = useMemo(() => {
    return TABS_CONFIG.map((tab) => {
      if (tab.value === "ALL") return ORGANIZER_EVENTS.length;
      return ORGANIZER_EVENTS.filter(tab.filter).length;
    });
  }, []);

  // Categories, Venues, Dates lists
  const categories = useMemo(() => {
    const list = Array.from(new Set(ORGANIZER_EVENTS.map((e) => e.category))).filter(Boolean);
    return ["ALL", ...list];
  }, []);

  const venues = useMemo(() => {
    const list = Array.from(new Set(ORGANIZER_EVENTS.map((e) => e.venue))).filter(Boolean);
    return ["ALL", ...list];
  }, []);

  const hasActiveFilters =
    search.trim() !== "" ||
    selectedCategory !== "ALL" ||
    selectedDateFilter !== "ALL" ||
    selectedStatusFilter !== "ALL" ||
    selectedVenueFilter !== "ALL" ||
    selectedSort !== "newest";

  const handleResetFilters = () => {
    setSearch("");
    setSelectedCategory("ALL");
    setSelectedDateFilter("ALL");
    setSelectedStatusFilter("ALL");
    setSelectedVenueFilter("ALL");
    setSelectedSort("newest");
    setCurrentPage(1);
    setActiveTab(0);
  };

  // Filter & Sort pipeline
  const filteredEvents = useMemo(() => {
    return ORGANIZER_EVENTS.filter((e) => {
      // 1. Tab filter
      const tabConfig = TABS_CONFIG[activeTab];
      if (tabConfig && !tabConfig.filter(e)) {
        return false;
      }

      // 2. Search keyword
      if (search.trim()) {
        const query = search.toLowerCase();
        const matchesName = e.name.toLowerCase().includes(query);
        const matchesVenue = e.venue.toLowerCase().includes(query);
        const matchesCategory = e.category?.toLowerCase().includes(query);
        const matchesStatus = e.status.toLowerCase().includes(query);
        if (!matchesName && !matchesVenue && !matchesCategory && !matchesStatus) {
          return false;
        }
      }

      // 3. Category filter
      if (selectedCategory !== "ALL" && e.category !== selectedCategory) {
        return false;
      }

      // 4. Status filter
      if (selectedStatusFilter !== "ALL" && e.status !== selectedStatusFilter) {
        return false;
      }

      // 5. Venue filter
      if (selectedVenueFilter !== "ALL" && e.venue !== selectedVenueFilter) {
        return false;
      }

      // 6. Date filter
      if (selectedDateFilter !== "ALL") {
        if (selectedDateFilter === "Sep" && !e.date.includes("Sep")) return false;
        if (selectedDateFilter === "Oct" && !e.date.includes("Oct")) return false;
        if (selectedDateFilter === "Nov" && !e.date.includes("Nov")) return false;
      }

      return true;
    }).sort((a, b) => {
      if (selectedSort === "newest") return b.id - a.id;
      if (selectedSort === "oldest") return a.id - b.id;
      if (selectedSort === "tickets_high") return b.sold - a.sold;
      if (selectedSort === "revenue_high") {
        const parseRev = (val) => Number(String(val).replace(/[^0-9]/g, "")) || 0;
        return parseRev(b.revenue) - parseRev(a.revenue);
      }
      return 0;
    });
  }, [
    activeTab,
    search,
    selectedCategory,
    selectedStatusFilter,
    selectedVenueFilter,
    selectedDateFilter,
    selectedSort,
  ]);

  // Pagination calculation
  const totalEvents = filteredEvents.length;
  const totalPages = Math.max(1, Math.ceil(totalEvents / ITEMS_PER_PAGE));
  const effectiveCurrentPage = Math.min(currentPage, totalPages);
  const startIndex = (effectiveCurrentPage - 1) * ITEMS_PER_PAGE;
  const paginatedEvents = filteredEvents.slice(startIndex, startIndex + ITEMS_PER_PAGE);

  const handleEventClick = (eventId) => {
    navigate(getEventDetailsRoute(eventId));
  };

  return (
    <div className="p-4 sm:p-6 lg:p-8 max-w-7xl mx-auto flex flex-col gap-6 w-full">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4 border-b border-[#bcc9c6]/40">
        <div className="flex flex-col gap-1">
          <h1 className="font-bold text-[#141b2b] text-2xl sm:text-3xl tracking-[-0.64px] leading-tight">
            My Events
          </h1>
          <p className="text-[#565e74] text-sm sm:text-base">
            Create, manage, and monitor all your events.
          </p>
        </div>

        <button
          type="button"
          onClick={() => navigate(ORGANIZER_ROUTES.CREATE_STEP_1)}
          className="bg-[#00685f] hover:bg-[#005a52] text-white h-11 sm:h-12 px-5 sm:px-6 rounded-lg flex items-center justify-center gap-2 font-medium text-sm sm:text-base transition-colors shadow-xs shrink-0 cursor-pointer self-start sm:self-auto"
        >
          <PlusIcon className="size-3 text-white" />
          <span>Create Event</span>
        </button>
      </div>

      {/* Tabs Row */}
      <div className="border-b border-[#bcc9c6] flex gap-2 sm:gap-6 overflow-x-auto scrollbar-none">
        {TABS_CONFIG.map((tab, i) => {
          const isActive = activeTab === i;
          const count = tabCounts[i];

          return (
            <button
              key={tab.label}
              type="button"
              onClick={() => {
                setActiveTab(i);
                setCurrentPage(1);
              }}
              className={`flex items-center justify-center pb-3.5 pt-2 px-2 shrink-0 border-b-2 transition-all cursor-pointer ${
                isActive
                  ? "border-[#00685f] text-[#00685f] font-semibold"
                  : "border-transparent text-[#565e74] hover:text-[#141b2b] font-normal"
              }`}
              style={{ marginBottom: "-1px" }}
            >
              <span className="text-sm sm:text-base whitespace-nowrap">
                {tab.label} ({count})
              </span>
            </button>
          );
        })}
      </div>

      {/* Search & Filter Controls */}
      <div className="flex flex-col gap-3" ref={dropdownRef}>
        {/* Search Input */}
        <div className="relative w-full">
          <div className="bg-white border border-[#bcc9c6] flex h-11 sm:h-12 items-center pl-10 pr-4 rounded-lg w-full focus-within:border-[#00685f] focus-within:ring-2 focus-within:ring-[#00685f]/15 transition-all shadow-2xs">
            <input
              type="text"
              placeholder="Search events by name, venue, or category..."
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setCurrentPage(1);
              }}
              className="flex-1 text-[#141b2b] text-sm sm:text-base bg-transparent outline-none placeholder:text-[#565e74]/70"
            />
            {search && (
              <button
                type="button"
                onClick={() => setSearch("")}
                className="text-[#565e74] hover:text-[#141b2b] text-sm p-1 rounded-sm"
                aria-label="Clear search"
              >
                ✕
              </button>
            )}
          </div>
          <div className="absolute left-3.5 top-1/2 -translate-y-1/2 text-[#565e74] pointer-events-none">
            <SearchIcon className="size-4 text-[#565e74]" />
          </div>
        </div>

        {/* Filter Pills Bar */}
        <div className="flex gap-2 flex-wrap items-center">
          {/* Category Filter */}
          <div className="relative">
            <button
              type="button"
              onClick={() => setOpenDropdown(openDropdown === "category" ? null : "category")}
              className={`border flex gap-2 h-10 sm:h-11 items-center px-3.5 rounded-lg text-xs sm:text-sm font-medium transition-colors cursor-pointer ${
                selectedCategory !== "ALL"
                  ? "bg-[#00685f]/10 border-[#00685f] text-[#00685f]"
                  : "bg-white border-[#bcc9c6] text-[#141b2b] hover:bg-[#f9f9ff]"
              }`}
            >
              <CategoryIcon className="size-3 text-current" />
              <span>{selectedCategory === "ALL" ? "Category" : selectedCategory}</span>
              <span className="text-[10px] opacity-70">▼</span>
            </button>

            {openDropdown === "category" && (
              <div className="absolute left-0 mt-1.5 w-48 bg-white border border-[#bcc9c6] rounded-lg shadow-lg z-20 py-1 text-xs sm:text-sm animate-fade-in">
                {categories.map((cat) => (
                  <button
                    key={cat}
                    type="button"
                    onClick={() => {
                      setSelectedCategory(cat);
                      setOpenDropdown(null);
                      setCurrentPage(1);
                    }}
                    className={`w-full text-left px-3.5 py-2 hover:bg-[#f1f3ff] transition-colors ${
                      selectedCategory === cat ? "text-[#00685f] font-semibold bg-[#00685f]/5" : "text-[#141b2b]"
                    }`}
                  >
                    {cat === "ALL" ? "All Categories" : cat}
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Date Filter */}
          <div className="relative">
            <button
              type="button"
              onClick={() => setOpenDropdown(openDropdown === "date" ? null : "date")}
              className={`border flex gap-2 h-10 sm:h-11 items-center px-3.5 rounded-lg text-xs sm:text-sm font-medium transition-colors cursor-pointer ${
                selectedDateFilter !== "ALL"
                  ? "bg-[#00685f]/10 border-[#00685f] text-[#00685f]"
                  : "bg-white border-[#bcc9c6] text-[#141b2b] hover:bg-[#f9f9ff]"
              }`}
            >
              <DateIcon className="size-3 text-current" />
              <span>
                {selectedDateFilter === "ALL"
                  ? "Date"
                  : selectedDateFilter === "Sep"
                    ? "Sep 2026"
                    : selectedDateFilter === "Oct"
                      ? "Oct 2026"
                      : "Nov 2026"}
              </span>
              <span className="text-[10px] opacity-70">▼</span>
            </button>

            {openDropdown === "date" && (
              <div className="absolute left-0 mt-1.5 w-44 bg-white border border-[#bcc9c6] rounded-lg shadow-lg z-20 py-1 text-xs sm:text-sm animate-fade-in">
                {[
                  { label: "All Dates", value: "ALL" },
                  { label: "September 2026", value: "Sep" },
                  { label: "October 2026", value: "Oct" },
                  { label: "November 2026", value: "Nov" },
                ].map((item) => (
                  <button
                    key={item.value}
                    type="button"
                    onClick={() => {
                      setSelectedDateFilter(item.value);
                      setOpenDropdown(null);
                      setCurrentPage(1);
                    }}
                    className={`w-full text-left px-3.5 py-2 hover:bg-[#f1f3ff] transition-colors ${
                      selectedDateFilter === item.value
                        ? "text-[#00685f] font-semibold bg-[#00685f]/5"
                        : "text-[#141b2b]"
                    }`}
                  >
                    {item.label}
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Status Filter */}
          <div className="relative">
            <button
              type="button"
              onClick={() => setOpenDropdown(openDropdown === "status" ? null : "status")}
              className={`border flex gap-2 h-10 sm:h-11 items-center px-3.5 rounded-lg text-xs sm:text-sm font-medium transition-colors cursor-pointer ${
                selectedStatusFilter !== "ALL"
                  ? "bg-[#00685f]/10 border-[#00685f] text-[#00685f]"
                  : "bg-white border-[#bcc9c6] text-[#141b2b] hover:bg-[#f9f9ff]"
              }`}
            >
              <StatusIcon className="size-3 text-current" />
              <span>{selectedStatusFilter === "ALL" ? "Status" : selectedStatusFilter}</span>
              <span className="text-[10px] opacity-70">▼</span>
            </button>

            {openDropdown === "status" && (
              <div className="absolute left-0 mt-1.5 w-44 bg-white border border-[#bcc9c6] rounded-lg shadow-lg z-20 py-1 text-xs sm:text-sm animate-fade-in">
                {[
                  { label: "All Statuses", value: "ALL" },
                  { label: "Published", value: "PUBLISHED" },
                  { label: "Draft", value: "DRAFT" },
                  { label: "Ongoing", value: "ONGOING" },
                  { label: "Completed", value: "COMPLETED" },
                  { label: "Cancelled", value: "CANCELLED" },
                ].map((item) => (
                  <button
                    key={item.value}
                    type="button"
                    onClick={() => {
                      setSelectedStatusFilter(item.value);
                      setOpenDropdown(null);
                      setCurrentPage(1);
                    }}
                    className={`w-full text-left px-3.5 py-2 hover:bg-[#f1f3ff] transition-colors ${
                      selectedStatusFilter === item.value
                        ? "text-[#00685f] font-semibold bg-[#00685f]/5"
                        : "text-[#141b2b]"
                    }`}
                  >
                    {item.label}
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Venue / City Filter */}
          <div className="relative">
            <button
              type="button"
              onClick={() => setOpenDropdown(openDropdown === "venue" ? null : "venue")}
              className={`border flex gap-2 h-10 sm:h-11 items-center px-3.5 rounded-lg text-xs sm:text-sm font-medium transition-colors cursor-pointer ${
                selectedVenueFilter !== "ALL"
                  ? "bg-[#00685f]/10 border-[#00685f] text-[#00685f]"
                  : "bg-white border-[#bcc9c6] text-[#141b2b] hover:bg-[#f9f9ff]"
              }`}
            >
              <VenueIcon className="size-3 text-current" />
              <span>{selectedVenueFilter === "ALL" ? "Venue/City" : selectedVenueFilter}</span>
              <span className="text-[10px] opacity-70">▼</span>
            </button>

            {openDropdown === "venue" && (
              <div className="absolute left-0 mt-1.5 w-44 bg-white border border-[#bcc9c6] rounded-lg shadow-lg z-20 py-1 text-xs sm:text-sm animate-fade-in">
                {venues.map((v) => (
                  <button
                    key={v}
                    type="button"
                    onClick={() => {
                      setSelectedVenueFilter(v);
                      setOpenDropdown(null);
                      setCurrentPage(1);
                    }}
                    className={`w-full text-left px-3.5 py-2 hover:bg-[#f1f3ff] transition-colors ${
                      selectedVenueFilter === v
                        ? "text-[#00685f] font-semibold bg-[#00685f]/5"
                        : "text-[#141b2b]"
                    }`}
                  >
                    {v === "ALL" ? "All Venues/Cities" : v}
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Sort Filter */}
          <div className="relative ml-auto">
            <button
              type="button"
              onClick={() => setOpenDropdown(openDropdown === "sort" ? null : "sort")}
              className="bg-white border border-[#bcc9c6] flex gap-2 h-10 sm:h-11 items-center px-3.5 rounded-lg text-xs sm:text-sm text-[#141b2b] hover:bg-[#f9f9ff] transition-colors cursor-pointer"
            >
              <SortIcon className="size-3 text-[#141b2b]" />
              <span>
                Sort:{" "}
                {selectedSort === "newest"
                  ? "Newest First"
                  : selectedSort === "oldest"
                    ? "Oldest First"
                    : selectedSort === "tickets_high"
                      ? "Most Tickets Sold"
                      : "Highest Revenue"}
              </span>
              <span className="text-[10px] opacity-70">▼</span>
            </button>

            {openDropdown === "sort" && (
              <div className="absolute right-0 mt-1.5 w-48 bg-white border border-[#bcc9c6] rounded-lg shadow-lg z-20 py-1 text-xs sm:text-sm animate-fade-in">
                {[
                  { label: "Newest First", value: "newest" },
                  { label: "Oldest First", value: "oldest" },
                  { label: "Most Tickets Sold", value: "tickets_high" },
                  { label: "Highest Revenue", value: "revenue_high" },
                ].map((item) => (
                  <button
                    key={item.value}
                    type="button"
                    onClick={() => {
                      setSelectedSort(item.value);
                      setOpenDropdown(null);
                    }}
                    className={`w-full text-left px-3.5 py-2 hover:bg-[#f1f3ff] transition-colors ${
                      selectedSort === item.value
                        ? "text-[#00685f] font-semibold bg-[#00685f]/5"
                        : "text-[#141b2b]"
                    }`}
                  >
                    {item.label}
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Clear Filters Button */}
          {hasActiveFilters && (
            <button
              type="button"
              onClick={handleResetFilters}
              className="text-xs text-[#ba1a1a] hover:underline font-medium px-2 py-1 cursor-pointer"
            >
              Reset Filters
            </button>
          )}
        </div>
      </div>

      {/* Events Table Card */}
      <div className="bg-white border border-[#bcc9c6] rounded-[12px] shadow-[0px_1px_2px_0px_rgba(0,0,0,0.05)] overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full border-collapse">
            <thead>
              <tr className="bg-[#f1f3ff] border-b border-[#bcc9c6]">
                <th className="text-left px-4 py-3.5 font-bold text-[#565e74] text-xs sm:text-sm tracking-[0.8px] uppercase whitespace-nowrap min-w-[200px]">
                  EVENT
                </th>
                <th className="text-left px-4 py-3.5 font-bold text-[#565e74] text-xs sm:text-sm tracking-[0.8px] uppercase whitespace-nowrap">
                  DATE
                </th>
                <th className="text-left px-4 py-3.5 font-bold text-[#565e74] text-xs sm:text-sm tracking-[0.8px] uppercase whitespace-nowrap">
                  VENUE
                </th>
                <th className="text-left px-4 py-3.5 font-bold text-[#565e74] text-xs sm:text-sm tracking-[0.8px] uppercase whitespace-nowrap min-w-[150px]">
                  TICKETS SOLD
                </th>
                <th className="text-left px-4 py-3.5 font-bold text-[#565e74] text-xs sm:text-sm tracking-[0.8px] uppercase whitespace-nowrap">
                  REVENUE
                </th>
                <th className="text-left px-4 py-3.5 font-bold text-[#565e74] text-xs sm:text-sm tracking-[0.8px] uppercase whitespace-nowrap">
                  STATUS
                </th>
                <th className="text-right px-4 py-3.5 font-bold text-[#565e74] text-xs sm:text-sm tracking-[0.8px] uppercase whitespace-nowrap">
                  ACTIONS
                </th>
              </tr>
            </thead>

            <tbody>
              {paginatedEvents.length > 0 ? (
                paginatedEvents.map((event, idx) => (
                  <tr
                    key={event.id}
                    onClick={() => handleEventClick(event.id)}
                    className={`${
                      idx > 0 ? "border-t border-[#bcc9c6]/70" : ""
                    } hover:bg-[#f9f9ff] transition-colors cursor-pointer group`}
                  >
                    {/* Event Name & Thumbnail */}
                    <td className="px-4 py-3.5">
                      <div className="flex gap-3 items-center">
                        <div className="size-11 sm:size-12 rounded-lg bg-[#dce2f7] overflow-hidden shrink-0 border border-[#bcc9c6]/40">
                          <img
                            src={event.thumbnail || event.image}
                            alt={event.name}
                            className="size-full object-cover group-hover:scale-105 transition-transform duration-200"
                          />
                        </div>
                        <div className="flex flex-col min-w-0 max-w-[160px] sm:max-w-[240px]">
                          <span
                            className="font-semibold text-[#141b2b] text-sm sm:text-base group-hover:text-[#00685f] transition-colors truncate"
                            title={event.name}
                          >
                            {event.name}
                          </span>
                          <span className="text-xs text-[#565e74] truncate">
                            {event.category || "General"}
                          </span>
                        </div>
                      </div>
                    </td>

                    {/* Date */}
                    <td className="px-4 py-3.5">
                      <span className="text-sm sm:text-base text-[#141b2b] whitespace-nowrap">
                        {event.date}
                      </span>
                    </td>

                    {/* Venue */}
                    <td className="px-4 py-3.5">
                      <span className="text-sm sm:text-base text-[#141b2b] whitespace-nowrap">
                        {event.venue}
                      </span>
                    </td>

                    {/* Tickets Sold */}
                    <td className="px-4 py-3.5 min-w-[150px]">
                      <div className="flex flex-col gap-1.5">
                        <div className="flex items-center justify-between text-xs sm:text-sm">
                          <span className="text-[#141b2b] whitespace-nowrap font-medium">
                            {event.sold} / {event.total.toLocaleString()}
                          </span>
                          <span
                            className={`font-bold whitespace-nowrap ${
                              event.pct > 0 ? "text-[#00685f]" : "text-[#565e74]"
                            }`}
                          >
                            {event.pct}%
                          </span>
                        </div>
                        <ProgressBar pct={event.pct} />
                      </div>
                    </td>

                    {/* Revenue */}
                    <td className="px-4 py-3.5">
                      <span
                        className={`text-sm sm:text-base font-semibold whitespace-nowrap ${
                          event.pct === 0 ? "text-[#565e74]" : "text-[#141b2b]"
                        }`}
                      >
                        {event.revenue}
                      </span>
                    </td>

                    {/* Status Badge */}
                    <td className="px-4 py-3.5">
                      <StatusBadge status={event.status} />
                    </td>

                    {/* Actions Menu */}
                    <td
                      className="px-4 py-3.5 text-right relative"
                      data-action-menu
                      onClick={(e) => e.stopPropagation()}
                    >
                      <button
                        type="button"
                        onClick={() =>
                          setActiveActionMenuId(
                            activeActionMenuId === event.id ? null : event.id
                          )
                        }
                        className="p-2 rounded-lg text-[#565e74] hover:text-[#141b2b] hover:bg-[#f1f3ff] transition-colors cursor-pointer inline-flex items-center justify-center"
                        aria-label="More actions"
                      >
                        <MoreIcon className="w-1 h-4 text-current" />
                      </button>

                      {/* Dropdown Action Menu */}
                      {activeActionMenuId === event.id && (
                        <div className="absolute right-4 top-12 w-44 bg-white border border-[#bcc9c6] rounded-lg shadow-xl z-30 py-1 text-left text-xs sm:text-sm animate-fade-in">
                          <button
                            type="button"
                            onClick={() => {
                              setActiveActionMenuId(null);
                              handleEventClick(event.id);
                            }}
                            className="w-full px-3.5 py-2 hover:bg-[#f1f3ff] text-[#141b2b] font-medium flex items-center gap-2 transition-colors cursor-pointer"
                          >
                            <span>🔍</span>
                            <span>View Details</span>
                          </button>
                          <button
                            type="button"
                            onClick={() => {
                              setActiveActionMenuId(null);
                              navigate(ORGANIZER_ROUTES.CREATE_STEP_1);
                            }}
                            className="w-full px-3.5 py-2 hover:bg-[#f1f3ff] text-[#141b2b] flex items-center gap-2 transition-colors cursor-pointer"
                          >
                            <span>✏️</span>
                            <span>Edit Event</span>
                          </button>
                          <button
                            type="button"
                            onClick={() => {
                              setActiveActionMenuId(null);
                              navigator.clipboard?.writeText(window.location.origin + `/organizer/events/${event.id}`);
                            }}
                            className="w-full px-3.5 py-2 hover:bg-[#f1f3ff] text-[#141b2b] flex items-center gap-2 transition-colors cursor-pointer"
                          >
                            <span>🔗</span>
                            <span>Copy Link</span>
                          </button>
                        </div>
                      )}
                    </td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td colSpan={7} className="px-4 py-16 text-center">
                    <div className="flex flex-col items-center justify-center gap-3">
                      <div className="size-12 rounded-full bg-gray-100 flex items-center justify-center text-xl text-gray-400">
                        🔍
                      </div>
                      <p className="font-semibold text-base text-[#141b2b]">
                        No events found
                      </p>
                      <p className="text-sm text-[#565e74] max-w-sm">
                        No events match the current filter or search criteria. Try clearing some filters.
                      </p>
                      {hasActiveFilters && (
                        <button
                          type="button"
                          onClick={handleResetFilters}
                          className="mt-2 text-xs font-semibold text-[#00685f] hover:underline cursor-pointer"
                        >
                          Clear all filters
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination Bar */}
        {totalEvents > 0 && (
          <div className="bg-white border-t border-[#bcc9c6] flex flex-col sm:flex-row items-center justify-between px-4 sm:px-6 py-3.5 gap-3">
            <span className="text-[#565e74] text-xs sm:text-sm whitespace-nowrap">
              Showing{" "}
              <span className="font-semibold text-[#141b2b]">
                {totalEvents === 0 ? 0 : startIndex + 1}–{Math.min(startIndex + ITEMS_PER_PAGE, totalEvents)}
              </span>{" "}
              of{" "}
              <span className="font-semibold text-[#141b2b]">
                {totalEvents}
              </span>{" "}
              events
            </span>

            <div className="flex gap-1.5 items-center">
              <button
                type="button"
                disabled={effectiveCurrentPage <= 1}
                onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                className="border border-[#bcc9c6] flex h-8 items-center justify-center px-3 rounded text-xs text-[#565e74] hover:bg-[#f1f3ff] transition-colors disabled:opacity-40 disabled:hover:bg-transparent cursor-pointer disabled:cursor-not-allowed"
              >
                Previous
              </button>

              {Array.from({ length: totalPages }, (_, i) => i + 1).map((page) => (
                <button
                  key={page}
                  type="button"
                  onClick={() => setCurrentPage(page)}
                  className={`flex items-center justify-center rounded size-8 text-xs transition-colors cursor-pointer ${
                    effectiveCurrentPage === page
                      ? "bg-[#00685f] text-white font-bold shadow-2xs"
                      : "border border-[#bcc9c6] text-[#141b2b] hover:bg-[#f1f3ff]"
                  }`}
                >
                  {page}
                </button>
              ))}

              <button
                type="button"
                disabled={effectiveCurrentPage >= totalPages}
                onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
                className="border border-[#bcc9c6] flex h-8 items-center justify-center px-3 rounded text-xs text-[#565e74] hover:bg-[#f1f3ff] transition-colors disabled:opacity-40 disabled:hover:bg-transparent cursor-pointer disabled:cursor-not-allowed"
              >
                Next
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
