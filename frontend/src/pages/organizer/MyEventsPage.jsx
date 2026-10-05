import { useState, useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";

import {
  ORGANIZER_ROUTES,
  getEventDetailsRoute,
  getEditEventRoute,
} from "../../constants/eventConstants";

import StatusBadge from "../../components/organizer/events/StatusBadge";
import ProgressBar from "../../components/organizer/events/ProgressBar";

import {
  PlusIcon,
  SearchIcon,
  StatusIcon,
  MoreIcon,
} from "../../components/organizer/events/OrganizerEventsIcons";

import { getOrganizerEvents } from "../../api/organizerApi";

const ITEMS_PER_PAGE = 10;

const STATUS_TABS = [
  { label: "All", value: "ALL" },
  { label: "Published", value: "PUBLISHED" },
  { label: "Draft", value: "DRAFT" },
  { label: "Completed", value: "COMPLETED" },
  { label: "Cancelled", value: "CANCELLED" },
];

const STATUS_FILTERS = [
  { label: "All Statuses", value: "ALL" },
  { label: "Published", value: "PUBLISHED" },
  { label: "Draft", value: "DRAFT" },
  { label: "Completed", value: "COMPLETED" },
  { label: "Cancelled", value: "CANCELLED" },
];

export default function MyEventsPage() {
  const navigate = useNavigate();

  const [activeTab, setActiveTab] = useState("ALL");
  const [search, setSearch] = useState("");
  const [selectedStatusFilter, setSelectedStatusFilter] = useState("ALL");

  const [events, setEvents] = useState([]);
  const [currentPage, setCurrentPage] = useState(1);
  const [totalEvents, setTotalEvents] = useState(0);

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const [reloadKey, setReloadKey] = useState(0);

  const [openDropdown, setOpenDropdown] = useState(null);
  const [activeActionMenuId, setActiveActionMenuId] = useState(null);

  const dropdownRef = useRef(null);

  /*
   * Close dropdowns when clicking outside.
   */
  useEffect(() => {
    function handleClickOutside(event) {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target)
      ) {
        setOpenDropdown(null);
      }

      if (!event.target.closest("[data-action-menu]")) {
        setActiveActionMenuId(null);
      }
    }

    document.addEventListener("mousedown", handleClickOutside);

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, []);

  /*
   * Fetch organizer events from API.
   *
   * Search is debounced by 300ms.
   */
  useEffect(() => {
    let cancelled = false;

    const timer = setTimeout(async () => {
      try {
        setLoading(true);
        setError("");

        const response = await getOrganizerEvents({
          page: currentPage,
          limit: ITEMS_PER_PAGE,
          status:
            selectedStatusFilter === "ALL"
              ? ""
              : selectedStatusFilter,
          search: search.trim(),
        });

        if (cancelled) {
          return;
        }

        const responseEvents = response?.data?.events ?? [];
        const responseTotal = Number(response?.data?.total ?? 0);

        setEvents(responseEvents);
        setTotalEvents(responseTotal);
      } catch (err) {
        if (cancelled) {
          return;
        }

        console.error(
          "Failed to fetch organizer events:",
          err
        );

        setEvents([]);
        setTotalEvents(0);

        setError(
          err?.response?.data?.message ||
          "Failed to load events. Please try again."
        );
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }, 300);

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [currentPage, selectedStatusFilter, search, reloadKey]);

  /*
   * Server-side pagination.
   */
  const totalPages = Math.max(
    1,
    Math.ceil(totalEvents / ITEMS_PER_PAGE)
  );

  const currentPageStart =
    totalEvents === 0
      ? 0
      : (currentPage - 1) * ITEMS_PER_PAGE + 1;

  const currentPageEnd = Math.min(
    currentPage * ITEMS_PER_PAGE,
    totalEvents
  );

  const hasActiveFilters =
    search.trim() !== "" ||
    selectedStatusFilter !== "ALL";

  const handleResetFilters = () => {
    setSearch("");
    setSelectedStatusFilter("ALL");
    setActiveTab("ALL");
    setCurrentPage(1);
    setOpenDropdown(null);
  };

  const handleStatusChange = (status) => {
    setSelectedStatusFilter(status);
    setActiveTab(status);
    setCurrentPage(1);
    setOpenDropdown(null);
  };

  const handleEventClick = (eventId) => {
    if (!eventId) {
      return;
    }

    navigate(getEventDetailsRoute(eventId));
  };

  const handleCopyEventLink = async (eventId) => {
    if (!eventId) {
      return;
    }

    try {
      const eventPath = getEventDetailsRoute(eventId);

      await navigator.clipboard?.writeText(
        `${window.location.origin}${eventPath}`
      );
    } catch (err) {
      console.error("Failed to copy event link:", err);
    }

    setActiveActionMenuId(null);
  };

  const formatEventDate = (eventDate) => {
    if (!eventDate) {
      return "—";
    }

    const date = new Date(eventDate);

    if (Number.isNaN(date.getTime())) {
      return "—";
    }

    return date.toLocaleDateString("en-IN", {
      day: "2-digit",
      month: "short",
      year: "numeric",
    });
  };

  const getTicketPercentage = (event) => {
    const capacity = Number(event?.TicketCapacity ?? 0);
    const sold = Number(event?.TicketsSold ?? 0);

    if (capacity <= 0) {
      return 0;
    }

    return Math.min(
      100,
      Math.round((sold / capacity) * 100)
    );
  };

  const getVenueName = (event) => {
    if (
      event?.VenueName &&
      event.VenueName.trim() !== ""
    ) {
      return event.VenueCity
        ? `${event.VenueName}, ${event.VenueCity}`
        : event.VenueName;
    }

    if (event?.EventType === "ONLINE") {
      return "Online Event";
    }

    return "—";
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
          onClick={() =>
            navigate(ORGANIZER_ROUTES.CREATE_STEP_1)
          }
          className="bg-[#00685f] hover:bg-[#005a52] text-white h-11 sm:h-12 px-5 sm:px-6 rounded-lg flex items-center justify-center gap-2 font-medium text-sm sm:text-base transition-colors shadow-xs shrink-0 cursor-pointer self-start sm:self-auto"
        >
          <PlusIcon className="size-3 text-white" />

          <span>Create Event</span>
        </button>
      </div>

      {/* Tabs Row */}
      <div className="border-b border-[#bcc9c6] flex gap-2 sm:gap-6 overflow-x-auto scrollbar-none">
        {STATUS_TABS.map((tab) => {
          const isActive = activeTab === tab.value;

          return (
            <button
              key={tab.value}
              type="button"
              onClick={() =>
                handleStatusChange(tab.value)
              }
              className={`flex items-center justify-center pb-3.5 pt-2 px-2 shrink-0 border-b-2 transition-all cursor-pointer ${isActive
                ? "border-[#00685f] text-[#00685f] font-semibold"
                : "border-transparent text-[#565e74] hover:text-[#141b2b] font-normal"
                }`}
              style={{ marginBottom: "-1px" }}
            >
              <span className="text-sm sm:text-base whitespace-nowrap">
                {tab.label}
              </span>
            </button>
          );
        })}
      </div>

      {/* Search & Filter Controls */}
      <div
        className="flex flex-col gap-3"
        ref={dropdownRef}
      >
        {/* Search Input */}
        <div className="relative w-full">
          <div className="bg-white border border-[#bcc9c6] flex h-11 sm:h-12 items-center pl-10 pr-4 rounded-lg w-full focus-within:border-[#00685f] focus-within:ring-2 focus-within:ring-[#00685f]/15 transition-all shadow-2xs">
            <input
              type="text"
              placeholder="Search events by title..."
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
                onClick={() => {
                  setSearch("");
                  setCurrentPage(1);
                }}
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

        {/* 
        <div className="flex gap-2 flex-wrap items-center">
          
          <div className="relative">
            <button
              type="button"
              onClick={() => setOpenDropdown(openDropdown === "category" ? null : "category")}
              className={`border flex gap-2 h-10 sm:h-11 items-center px-3.5 rounded-lg text-xs sm:text-sm font-medium transition-colors cursor-pointer ${selectedCategory !== "ALL"
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
                    className={`w-full text-left px-3.5 py-2 hover:bg-[#f1f3ff] transition-colors ${selectedCategory === cat ? "text-[#00685f] font-semibold bg-[#00685f]/5" : "text-[#141b2b]"
                      }`}
                  >
                    {cat === "ALL" ? "All Categories" : cat}
                  </button>
                ))}
              </div>
            )}
          </div>
          
          <div className="relative">
            <button
              type="button"
              onClick={() => setOpenDropdown(openDropdown === "date" ? null : "date")}
              className={`border flex gap-2 h-10 sm:h-11 items-center px-3.5 rounded-lg text-xs sm:text-sm font-medium transition-colors cursor-pointer ${selectedDateFilter !== "ALL"
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
                    className={`w-full text-left px-3.5 py-2 hover:bg-[#f1f3ff] transition-colors ${selectedDateFilter === item.value
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
         
          <div className="relative">
            <button
              type="button"
              onClick={() => setOpenDropdown(openDropdown === "status" ? null : "status")}
              className={`border flex gap-2 h-10 sm:h-11 items-center px-3.5 rounded-lg text-xs sm:text-sm font-medium transition-colors cursor-pointer ${selectedStatusFilter !== "ALL"
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
                    className={`w-full text-left px-3.5 py-2 hover:bg-[#f1f3ff] transition-colors ${selectedStatusFilter === item.value
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
          
          <div className="relative">
            <button
              type="button"
              onClick={() => setOpenDropdown(openDropdown === "venue" ? null : "venue")}
              className={`border flex gap-2 h-10 sm:h-11 items-center px-3.5 rounded-lg text-xs sm:text-sm font-medium transition-colors cursor-pointer ${selectedVenueFilter !== "ALL"
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
                    className={`w-full text-left px-3.5 py-2 hover:bg-[#f1f3ff] transition-colors ${selectedVenueFilter === v
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
                    className={`w-full text-left px-3.5 py-2 hover:bg-[#f1f3ff] transition-colors ${selectedSort === item.value
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
*/}
        {/* Filter Pills */}
        <div className="flex gap-2 flex-wrap items-center">
          {/* Status Filter */}
          <div className="relative">
            <button
              type="button"
              onClick={() =>
                setOpenDropdown(
                  openDropdown === "status"
                    ? null
                    : "status"
                )
              }
              className={`border flex gap-2 h-10 sm:h-11 items-center px-3.5 rounded-lg text-xs sm:text-sm font-medium transition-colors cursor-pointer ${selectedStatusFilter !== "ALL"
                ? "bg-[#00685f]/10 border-[#00685f] text-[#00685f]"
                : "bg-white border-[#bcc9c6] text-[#141b2b] hover:bg-[#f9f9ff]"
                }`}
            >
              <StatusIcon className="size-3 text-current" />

              <span>
                {selectedStatusFilter === "ALL"
                  ? "Status"
                  : selectedStatusFilter}
              </span>

              <span className="text-[10px] opacity-70">
                ▼
              </span>
            </button>

            {openDropdown === "status" && (
              <div className="absolute left-0 mt-1.5 w-44 bg-white border border-[#bcc9c6] rounded-lg shadow-lg z-20 py-1 text-xs sm:text-sm">
                {STATUS_FILTERS.map((item) => (
                  <button
                    key={item.value}
                    type="button"
                    onClick={() =>
                      handleStatusChange(
                        item.value
                      )
                    }
                    className={`w-full text-left px-3.5 py-2 hover:bg-[#f1f3ff] transition-colors ${selectedStatusFilter ===
                      item.value
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

          {/* Reset Filters */}
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

      {/* Error State */}
      {error && (
        <div className="border border-[#ba1a1a]/30 bg-[#fff5f5] rounded-lg px-4 py-3 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <p className="text-sm text-[#ba1a1a]">
            {error}
          </p>

          <button
            type="button"
            onClick={() => {
              setError("");
              setReloadKey((key) => key + 1);
            }}
            className="text-sm font-semibold text-[#00685f] hover:underline cursor-pointer shrink-0"
          >
            Try again
          </button>
        </div>
      )}

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
                  STATUS
                </th>

                <th className="text-right px-4 py-3.5 font-bold text-[#565e74] text-xs sm:text-sm tracking-[0.8px] uppercase whitespace-nowrap">
                  ACTIONS
                </th>
              </tr>
            </thead>

            <tbody>
              {loading ? (
                <tr>
                  <td
                    colSpan={6}
                    className="px-4 py-16 text-center"
                  >
                    <div className="flex flex-col items-center justify-center gap-3">
                      <div className="size-8 border-2 border-[#00685f]/20 border-t-[#00685f] rounded-full animate-spin" />

                      <p className="text-sm text-[#565e74]">
                        Loading events...
                      </p>
                    </div>
                  </td>
                </tr>
              ) : events.length > 0 ? (
                events.map((event, idx) => {
                  const ticketCapacity = Number(
                    event?.TicketCapacity ?? 0
                  );

                  const ticketsSold = Number(
                    event?.TicketsSold ?? 0
                  );

                  const ticketPercentage =
                    getTicketPercentage(event);

                  const eventId = event?.ID;

                  return (
                    <tr
                      key={eventId}
                      onClick={() =>
                        handleEventClick(
                          eventId
                        )
                      }
                      className={`${idx > 0
                        ? "border-t border-[#bcc9c6]/70"
                        : ""
                        } hover:bg-[#f9f9ff] transition-colors cursor-pointer group`}
                    >
                      {/* Event */}
                      <td className="px-4 py-3.5">
                        <div className="flex gap-3 items-center">
                          <div className="size-11 sm:size-12 rounded-lg bg-[#dce2f7] overflow-hidden shrink-0 border border-[#bcc9c6]/40 flex items-center justify-center">
                            {event?.BannerImageURL ? (
                              <img
                                src={event.BannerImageURL}
                                alt={event?.Title || "Event banner"}
                                className="size-full object-cover"
                              />
                            ) : (
                              <span className="text-[#565e74] text-xs">
                                —
                              </span>
                            )}
                          </div>

                          <div className="flex flex-col min-w-0 max-w-[160px] sm:max-w-[240px]">
                            <span
                              className="font-semibold text-[#141b2b] text-sm sm:text-base group-hover:text-[#00685f] transition-colors truncate"
                              title={
                                event?.Title
                              }
                            >
                              {event?.Title ||
                                "Untitled Event"}
                            </span>

                            <span className="text-xs text-[#565e74] truncate">
                              {event?.CategoryName ||
                                "General"}
                            </span>
                          </div>
                        </div>
                      </td>

                      {/* Date */}
                      <td className="px-4 py-3.5">
                        <span className="text-sm sm:text-base text-[#141b2b] whitespace-nowrap">
                          {formatEventDate(
                            event?.EventDate
                          )}
                        </span>
                      </td>

                      {/* Venue */}
                      <td className="px-4 py-3.5">
                        <span className="text-sm sm:text-base text-[#141b2b] whitespace-nowrap">
                          {getVenueName(event)}
                        </span>
                      </td>

                      {/* Tickets Sold */}
                      <td className="px-4 py-3.5 min-w-[150px]">
                        <div className="flex flex-col gap-1.5">
                          <div className="flex items-center justify-between text-xs sm:text-sm">
                            <span className="text-[#141b2b] whitespace-nowrap font-medium">
                              {ticketsSold.toLocaleString()}{" "}
                              /{" "}
                              {ticketCapacity.toLocaleString()}
                            </span>

                            <span
                              className={`font-bold whitespace-nowrap ${ticketPercentage >
                                0
                                ? "text-[#00685f]"
                                : "text-[#565e74]"
                                }`}
                            >
                              {
                                ticketPercentage
                              }
                              %
                            </span>
                          </div>

                          <ProgressBar
                            pct={
                              ticketPercentage
                            }
                          />
                        </div>
                      </td>

                      {/* Status */}
                      <td className="px-4 py-3.5">
                        <StatusBadge
                          status={
                            event?.Status
                          }
                        />
                      </td>

                      {/* Actions */}
                      <td
                        className="px-4 py-3.5 text-right relative"
                        data-action-menu
                        onClick={(e) =>
                          e.stopPropagation()
                        }
                      >
                        <button
                          type="button"
                          onClick={() =>
                            setActiveActionMenuId(
                              activeActionMenuId ===
                                eventId
                                ? null
                                : eventId
                            )
                          }
                          className="p-2 rounded-lg text-[#565e74] hover:text-[#141b2b] hover:bg-[#f1f3ff] transition-colors cursor-pointer inline-flex items-center justify-center"
                          aria-label="More actions"
                        >
                          <MoreIcon className="w-1 h-4 text-current" />
                        </button>

                        {activeActionMenuId ===
                          eventId && (
                            <div className="absolute right-4 top-12 w-44 bg-white border border-[#bcc9c6] rounded-lg shadow-xl z-30 py-1 text-left text-xs sm:text-sm">
                              <button
                                type="button"
                                onClick={() => {
                                  setActiveActionMenuId(
                                    null
                                  );

                                  handleEventClick(
                                    eventId
                                  );
                                }}
                                className="w-full px-3.5 py-2 hover:bg-[#f1f3ff] text-[#141b2b] font-medium flex items-center gap-2 transition-colors cursor-pointer"
                              >
                                <span>
                                  🔍
                                </span>

                                <span>
                                  View
                                  Details
                                </span>
                              </button>

                              {String(event?.Status || "").toUpperCase() === "DRAFT" ? (
                                <button
                                  type="button"
                                  onClick={() => {
                                    setActiveActionMenuId(
                                      null
                                    );

                                    navigate(
                                      getEditEventRoute(
                                        eventId
                                      )
                                    );
                                  }}
                                  className="w-full px-3.5 py-2 hover:bg-[#f1f3ff] text-[#141b2b] flex items-center gap-2 transition-colors cursor-pointer"
                                >
                                  <span>
                                    ✏️
                                  </span>

                                  <span>
                                    Edit Event
                                  </span>
                                </button>
                              ) : (
                                <button
                                  type="button"
                                  disabled
                                  title="Only draft events can be edited."
                                  className="w-full px-3.5 py-2 text-[#565e74]/50 flex items-center gap-2 cursor-not-allowed"
                                >
                                  <span className="opacity-50">
                                    ✏️
                                  </span>

                                  <span>
                                    Edit Event
                                  </span>
                                </button>
                              )}

                              <button
                                type="button"
                                onClick={() =>
                                  handleCopyEventLink(
                                    eventId
                                  )
                                }
                                className="w-full px-3.5 py-2 hover:bg-[#f1f3ff] text-[#141b2b] flex items-center gap-2 transition-colors cursor-pointer"
                              >
                                <span>
                                  🔗
                                </span>

                                <span>
                                  Copy Link
                                </span>
                              </button>
                            </div>
                          )}
                      </td>
                    </tr>
                  );
                })
              ) : (
                <tr>
                  <td
                    colSpan={6}
                    className="px-4 py-16 text-center"
                  >
                    <div className="flex flex-col items-center justify-center gap-3">
                      <div className="size-12 rounded-full bg-gray-100 flex items-center justify-center text-xl text-gray-400">
                        🔍
                      </div>

                      <p className="font-semibold text-base text-[#141b2b]">
                        No events found
                      </p>

                      <p className="text-sm text-[#565e74] max-w-sm">
                        {hasActiveFilters
                          ? "No events match the current filter or search criteria."
                          : "You have not created any events yet."}
                      </p>

                      {hasActiveFilters && (
                        <button
                          type="button"
                          onClick={
                            handleResetFilters
                          }
                          className="mt-2 text-xs font-semibold text-[#00685f] hover:underline cursor-pointer"
                        >
                          Clear all
                          filters
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {!loading && totalEvents > 0 && (
          <div className="bg-white border-t border-[#bcc9c6] flex flex-col sm:flex-row items-center justify-between px-4 sm:px-6 py-3.5 gap-3">
            <span className="text-[#565e74] text-xs sm:text-sm whitespace-nowrap">
              Showing{" "}
              <span className="font-semibold text-[#141b2b]">
                {currentPageStart}–
                {currentPageEnd}
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
                disabled={currentPage <= 1}
                onClick={() =>
                  setCurrentPage((page) =>
                    Math.max(1, page - 1)
                  )
                }
                className="border border-[#bcc9c6] flex h-8 items-center justify-center px-3 rounded text-xs text-[#565e74] hover:bg-[#f1f3ff] transition-colors disabled:opacity-40 disabled:hover:bg-transparent cursor-pointer disabled:cursor-not-allowed"
              >
                Previous
              </button>

              {Array.from(
                { length: totalPages },
                (_, index) => index + 1
              ).map((page) => (
                <button
                  key={page}
                  type="button"
                  onClick={() =>
                    setCurrentPage(page)
                  }
                  className={`flex items-center justify-center rounded size-8 text-xs transition-colors cursor-pointer ${currentPage === page
                    ? "bg-[#00685f] text-white font-bold shadow-2xs"
                    : "border border-[#bcc9c6] text-[#141b2b] hover:bg-[#f1f3ff]"
                    }`}
                >
                  {page}
                </button>
              ))}

              <button
                type="button"
                disabled={
                  currentPage >= totalPages
                }
                onClick={() =>
                  setCurrentPage((page) =>
                    Math.min(
                      totalPages,
                      page + 1
                    )
                  )
                }
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













