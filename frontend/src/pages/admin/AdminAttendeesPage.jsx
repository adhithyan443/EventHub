import { useState, useEffect } from "react";
import { formatDateTime } from "../../utils/formatters";
import notify from "../../utils/toast";
import {
  getAdminAttendees,
  getAdminAttendeeStats,
} from "../../api/adminBookingApi";
import { getAdminEvents } from "../../api/adminEventApi";
import { getOrganizers } from "../../api/adminOrganizerApi";

// Ticket / Check-in Status Styling Maps
const attendeeStatusBadgeStyles = {
  USED: "bg-[#dcfce7] text-[#15803d] border border-[#bbf7d0]",
  VALID: "bg-[#fef9c3] text-[#854d0e] border border-[#fef08a]",
  CANCELLED: "bg-[#fee2e2] text-[#b91c1c] border border-[#fecaca]",
  REFUNDED: "bg-[#f3e8ff] text-[#7e22ce] border border-[#e9d5ff]",
};

const attendeeStatusLabels = {
  USED: "Checked In",
  VALID: "Pending Check-In",
  CANCELLED: "Cancelled",
  REFUNDED: "Refunded",
};

const CHECKIN_STATUS_OPTIONS = [
  { value: "ALL", label: "All Check-In Statuses" },
  { value: "VALID", label: "Pending Check-In (Valid)" },
  { value: "USED", label: "Checked In (Used)" },
  { value: "CANCELLED", label: "Cancelled" },
  { value: "REFUNDED", label: "Refunded" },
];

const PAGE_SIZE_OPTIONS = [10, 25, 50, 100];

const INITIAL_STATS = {
  total_issued_tickets: 0,
  active_eligible_admissions: 0,
  checked_in_tickets: 0,
  pending_check_in_tickets: 0,
  cancelled_tickets: 0,
  check_in_rate: "0",
};

export default function AdminAttendeesPage() {
  // Filters State
  const [searchTerm, setSearchTerm] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [selectedOrganizerId, setSelectedOrganizerId] = useState("ALL");
  const [selectedEventId, setSelectedEventId] = useState("ALL");
  const [selectedStatus, setSelectedStatus] = useState("ALL");

  // Pagination State
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [totalRecords, setTotalRecords] = useState(0);
  const [totalPages, setTotalPages] = useState(1);

  // Data States
  const [attendees, setAttendees] = useState([]);
  const [stats, setStats] = useState(INITIAL_STATS);
  const [organizersList, setOrganizersList] = useState([]);
  const [eventsList, setEventsList] = useState([]);

  // UI Interactive States
  const [loading, setLoading] = useState(true);
  const [loadingOrganizers, setLoadingOrganizers] = useState(false);
  const [loadingEvents, setLoadingEvents] = useState(false);
  const [error, setError] = useState(null);
  const [copiedCode, setCopiedCode] = useState(null);
  const [refreshKey, setRefreshKey] = useState(0);

  // Debounce search input by 350ms
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(searchTerm);
      setCurrentPage(1);
    }, 350);
    return () => clearTimeout(timer);
  }, [searchTerm]);

  // Load organizers list for filter dropdown
  useEffect(() => {
    let cancelled = false;

    async function loadOrganizers() {
      setLoadingOrganizers(true);
      try {
        const res = await getOrganizers({ limit: 100 });
        if (cancelled) return;
        const rawList = Array.isArray(res?.data?.organizers)
          ? res.data.organizers
          : Array.isArray(res?.organizers)
          ? res.organizers
          : Array.isArray(res?.data)
          ? res.data
          : [];

        const normalizedList = rawList
          .map((org) => {
            if (!org || typeof org !== "object") return null;
            const id = org.id || org.ID || org.organizer_id || "";
            const name =
              org.business_name ||
              org.BusinessName ||
              org.user_name ||
              org.UserName ||
              org.name ||
              "";
            if (!id) return null;
            return {
              id: String(id),
              name: String(name).trim() || "Unnamed Organizer",
            };
          })
          .filter(Boolean);

        setOrganizersList(normalizedList);
      } catch (err) {
        if (!cancelled) {
          console.error("Failed to load organizers for attendee filter dropdown:", err);
        }
      } finally {
        if (!cancelled) {
          setLoadingOrganizers(false);
        }
      }
    }

    loadOrganizers();

    return () => {
      cancelled = true;
    };
  }, []);

  // Load events list for filter dropdown (reacts to selected organizer if set)
  useEffect(() => {
    let cancelled = false;

    async function loadEvents() {
      setLoadingEvents(true);
      try {
        const queryOptions = { limit: 100 };
        if (selectedOrganizerId && selectedOrganizerId !== "ALL") {
          queryOptions.organizerId = selectedOrganizerId;
        }
        const res = await getAdminEvents(queryOptions);
        if (cancelled) return;
        const rawList = Array.isArray(res?.data?.events)
          ? res.data.events
          : Array.isArray(res?.events)
          ? res.events
          : Array.isArray(res?.data)
          ? res.data
          : [];

        const normalizedList = rawList
          .map((ev) => {
            if (!ev || typeof ev !== "object") return null;
            const id = ev.id || ev.ID || ev.event_id || ev.eventId || "";
            const title =
              ev.title ||
              ev.Title ||
              ev.name ||
              ev.eventName ||
              ev.event_name ||
              "";
            if (!id) return null;
            return {
              id: String(id),
              title: String(title).trim() || "Untitled Event",
            };
          })
          .filter(Boolean);

        setEventsList(normalizedList);
      } catch (err) {
        if (!cancelled) {
          console.error("Failed to load events for attendee filter dropdown:", err);
        }
      } finally {
        if (!cancelled) {
          setLoadingEvents(false);
        }
      }
    }

    loadEvents();

    return () => {
      cancelled = true;
    };
  }, [selectedOrganizerId]);

  // Fetch attendees list and aggregate gate stats from backend
  useEffect(() => {
    let cancelled = false;

    async function fetchAttendeesData() {
      setLoading(true);
      setError(null);

      try {
        const filterOptions = {
          page: currentPage,
          limit: pageSize,
          organizer_id: selectedOrganizerId !== "ALL" ? selectedOrganizerId : "",
          event_id: selectedEventId !== "ALL" ? selectedEventId : "",
          check_in_status: selectedStatus !== "ALL" ? selectedStatus : "",
          search: debouncedSearch,
        };

        const [attendeesRes, statsRes] = await Promise.all([
          getAdminAttendees(filterOptions),
          getAdminAttendeeStats(filterOptions),
        ]);

        if (cancelled) return;

        // Process Attendees List & Pagination Envelopes
        const attendeeList =
          attendeesRes?.data?.attendees ||
          attendeesRes?.attendees ||
          (Array.isArray(attendeesRes?.data) ? attendeesRes.data : []);

        const paginationInfo =
          attendeesRes?.data?.pagination ||
          attendeesRes?.data ||
          attendeesRes?.pagination ||
          {};

        setAttendees(attendeeList);
        setTotalRecords(paginationInfo.total_records ?? attendeeList.length);
        setTotalPages(Math.max(1, paginationInfo.total_pages ?? Math.ceil(attendeeList.length / pageSize)));

        // Process Stats Envelope
        if (statsRes?.data) {
          setStats(statsRes.data);
        } else if (statsRes && typeof statsRes === "object" && statsRes.total_issued_tickets !== undefined) {
          setStats(statsRes);
        }
      } catch (err) {
        if (cancelled) return;
        console.error("Error fetching platform attendees:", err);
        const errorMsg =
          err.response?.data?.message ||
          err.response?.data?.error ||
          err.message ||
          "Failed to load platform attendees. Please check your connection and retry.";
        setError(errorMsg);
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    fetchAttendeesData();

    return () => {
      cancelled = true;
    };
  }, [
    currentPage,
    pageSize,
    selectedOrganizerId,
    selectedEventId,
    selectedStatus,
    debouncedSearch,
    refreshKey,
  ]);

  const handleCopyCode = (code) => {
    if (navigator.clipboard) {
      navigator.clipboard.writeText(code).then(() => {
        setCopiedCode(code);
        notify.success(`Ticket code ${code} copied!`);
        setTimeout(() => setCopiedCode(null), 2000);
      });
    } else {
      notify.info(`Ticket Code: ${code}`);
    }
  };

  const handleResetFilters = () => {
    setSearchTerm("");
    setDebouncedSearch("");
    setSelectedOrganizerId("ALL");
    setSelectedEventId("ALL");
    setSelectedStatus("ALL");
    setCurrentPage(1);
    notify.info("Filters cleared");
  };

  const hasActiveFilters = Boolean(
    searchTerm.trim() || selectedOrganizerId !== "ALL" || selectedEventId !== "ALL" || selectedStatus !== "ALL"
  );

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-white w-full">
      {/* Page Header */}
      <div className="px-4 sm:px-6 py-4 border-b border-[#dce2f7] flex flex-col md:flex-row md:items-center justify-between gap-4 shrink-0 bg-white">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-[#141b2b] text-xl font-bold tracking-tight">
              Platform Attendees Management
            </h1>
            <span className="bg-[#e6f4f2] text-[#00685f] text-xs font-semibold px-2.5 py-0.5 rounded-full border border-[#00685f]/20">
              Gate Admissions & Tracking
            </span>
          </div>
          <p className="text-[#3d4947] text-xs sm:text-sm mt-0.5">
            Monitor digital ticket passes, check-in status, gate progress, and attendance rates platform-wide.
          </p>
        </div>

        {/* Global refresh trigger */}
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={() => setRefreshKey((k) => k + 1)}
            disabled={loading}
            className="bg-[#f9f9ff] border border-[#bcc9c6] text-[#141b2b] text-xs sm:text-sm font-semibold px-3 py-1.5 rounded-lg flex items-center gap-1.5 hover:bg-[#eef0fb] transition-colors cursor-pointer self-start md:self-auto shrink-0 disabled:opacity-60"
          >
            <svg className={`w-3.5 h-3.5 ${loading ? "animate-spin text-[#00685F]" : ""}`} viewBox="0 0 20 20" fill="currentColor">
              <path fillRule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z" clipRule="evenodd" />
            </svg>
            <span>Refresh</span>
          </button>
        </div>
      </div>

      {/* Main Content Area */}
      <div className="flex-1 overflow-y-auto p-4 sm:p-6 space-y-5">
        {/* =================================================================== */}
        {/* SUMMARY KPI CARDS (Server-Calculated Statistics)                    */}
        {/* =================================================================== */}
        <div className="grid grid-cols-2 sm:grid-cols-3 xl:grid-cols-6 gap-3.5">
          {/* Card 1: Total Issued Tickets */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-xs flex flex-col justify-between">
            <span className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Total Issued
            </span>
            <span className="text-[#141b2b] text-xl lg:text-2xl font-bold tracking-tight">
              {stats.total_issued_tickets ?? 0}
            </span>
            <span className="text-[11px] mt-1 text-[#6d7a77] font-medium">All tickets created</span>
          </div>

          {/* Card 2: Active Eligible Admissions */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-xs flex flex-col justify-between">
            <span className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Active Admissions
            </span>
            <span className="text-[#00685f] text-xl lg:text-2xl font-bold tracking-tight">
              {stats.active_eligible_admissions ?? 0}
            </span>
            <span className="text-[11px] mt-1 text-[#00685f] font-medium">Valid + Checked-in passes</span>
          </div>

          {/* Card 3: Checked In Attendees */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-xs flex flex-col justify-between">
            <span className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Checked In
            </span>
            <span className="text-[#15803d] text-xl lg:text-2xl font-bold tracking-tight">
              {stats.checked_in_tickets ?? 0}
            </span>
            <span className="text-[11px] mt-1 text-[#15803d] font-medium">Admitted at gate</span>
          </div>

          {/* Card 4: Pending Check-Ins */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-xs flex flex-col justify-between">
            <span className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Pending Gate
            </span>
            <span className="text-[#854d0e] text-xl lg:text-2xl font-bold tracking-tight">
              {stats.pending_check_in_tickets ?? 0}
            </span>
            <span className="text-[11px] mt-1 text-[#854d0e] font-medium">Awaiting arrival</span>
          </div>

          {/* Card 5: Cancelled Tickets */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-xs flex flex-col justify-between">
            <span className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Cancelled / Refunded
            </span>
            <span className="text-[#b91c1c] text-xl lg:text-2xl font-bold tracking-tight">
              {stats.cancelled_tickets ?? 0}
            </span>
            <span className="text-[11px] mt-1 text-[#b91c1c] font-medium">Ineligible passes</span>
          </div>

          {/* Card 6: Check-In Rate (%) with Progress Bar */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-xs flex flex-col justify-between">
            <div>
              <div className="flex items-center justify-between mb-1">
                <span className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider">
                  Gate Progress
                </span>
                <span className="text-xs font-bold text-[#00685F]">{stats.check_in_rate || "0"}%</span>
              </div>
              <div className="text-xl lg:text-2xl font-bold text-[#141B2B]">
                {stats.check_in_rate || "0"}%
              </div>
            </div>
            <div className="mt-1.5">
              <div className="w-full bg-[#dce2f7] rounded-full h-1.5 overflow-hidden">
                <div
                  className="bg-[#00685F] h-1.5 rounded-full transition-all duration-500"
                  style={{ width: `${Math.min(parseFloat(stats.check_in_rate) || 0, 100)}%` }}
                ></div>
              </div>
              <span className="text-[10px] text-[#6d7a77] mt-1 block">
                {stats.checked_in_tickets ?? 0} of {stats.active_eligible_admissions ?? 0} active
              </span>
            </div>
          </div>
        </div>

        {/* =================================================================== */}
        {/* FILTER BAR                                                          */}
        {/* =================================================================== */}
        <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-3.5 space-y-3">
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-12 gap-3 items-center">
            {/* Search Input (3 columns on lg) */}
            <div className="lg:col-span-3 relative">
              <svg className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-[#6d7a77]" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z" clipRule="evenodd" />
              </svg>
              <input
                type="text"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                placeholder="Search attendee, email, ticket #, booking..."
                className="w-full bg-white border border-[#bcc9c6] rounded-lg pl-9 pr-8 py-2 text-xs sm:text-sm text-[#141b2b] placeholder:text-[#6d7a77] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40"
              />
              {searchTerm && (
                <button
                  type="button"
                  onClick={() => setSearchTerm("")}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 text-[#6d7a77] hover:text-[#141b2b] cursor-pointer text-xs"
                >
                  ✕
                </button>
              )}
            </div>

            {/* Organizer Filter (3 columns on lg) */}
            <div className="lg:col-span-3">
              <select
                value={selectedOrganizerId}
                onChange={(e) => {
                  setSelectedOrganizerId(e.target.value);
                  setSelectedEventId("ALL");
                  setCurrentPage(1);
                }}
                disabled={loadingOrganizers}
                className="w-full bg-white border border-[#bcc9c6] rounded-lg px-3 py-2 text-xs sm:text-sm text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40 cursor-pointer disabled:opacity-60"
              >
                <option value="ALL">
                  {loadingOrganizers ? "Loading organizers..." : "All Organizers"}
                </option>
                {organizersList.map((org) => (
                  <option key={org.id} value={org.id}>
                    {org.name}
                  </option>
                ))}
              </select>
            </div>

            {/* Event Filter (3 columns on lg) */}
            <div className="lg:col-span-3">
              <select
                value={selectedEventId}
                onChange={(e) => {
                  setSelectedEventId(e.target.value);
                  setCurrentPage(1);
                }}
                disabled={loadingEvents}
                className="w-full bg-white border border-[#bcc9c6] rounded-lg px-3 py-2 text-xs sm:text-sm text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40 cursor-pointer disabled:opacity-60"
              >
                <option value="ALL">
                  {loadingEvents ? "Loading events..." : "All Events"}
                </option>
                {eventsList.map((ev) => (
                  <option key={ev.id} value={ev.id}>
                    {ev.title}
                  </option>
                ))}
              </select>
            </div>

            {/* Check-In Status Filter (2 columns on lg) */}
            <div className="lg:col-span-2">
              <select
                value={selectedStatus}
                onChange={(e) => {
                  setSelectedStatus(e.target.value);
                  setCurrentPage(1);
                }}
                className="w-full bg-white border border-[#bcc9c6] rounded-lg px-3 py-2 text-xs sm:text-sm text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40 cursor-pointer"
              >
                {CHECKIN_STATUS_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>

            {/* Reset Filters (1 column on lg) */}
            <div className="lg:col-span-1 flex items-center justify-end">
              {hasActiveFilters && (
                <button
                  type="button"
                  onClick={handleResetFilters}
                  className="w-full text-xs text-[#ba1a1a] hover:text-[#93000a] font-semibold py-2 px-2 rounded-lg border border-[#ba1a1a]/30 hover:bg-[#ffdad6]/20 transition-colors cursor-pointer text-center"
                >
                  Clear
                </button>
              )}
            </div>
          </div>
        </div>

        {/* =================================================================== */}
        {/* ATTENDEES TABLE OR ERROR / LOADING STATE                            */}
        {/* =================================================================== */}
        {error ? (
          <div className="border border-[#ffdad6] bg-[#fff8f7] rounded-xl p-8 text-center space-y-3">
            <div className="w-10 h-10 bg-[#ba1a1a]/10 text-[#ba1a1a] rounded-full flex items-center justify-center mx-auto">
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
            </div>
            <h3 className="text-sm font-bold text-[#141b2b]">Unable to load attendees</h3>
            <p className="text-xs text-[#6d7a77] max-w-md mx-auto">{error}</p>
            <button
              type="button"
              onClick={() => setRefreshKey((k) => k + 1)}
              className="bg-[#00685f] hover:bg-[#005049] text-white text-xs font-semibold px-4 py-2 rounded-lg transition-colors cursor-pointer"
            >
              Retry
            </button>
          </div>
        ) : (
          <div className="border border-[#bcc9c6] rounded-xl overflow-hidden bg-white shadow-xs">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs text-[#141b2b]">
                <thead className="bg-[#f9f9ff] border-b border-[#bcc9c6] text-[#3d4947]">
                  <tr>
                    <th className="py-3 px-4 font-semibold">Ticket Code / Pass</th>
                    <th className="py-3 px-4 font-semibold">Attendee</th>
                    <th className="py-3 px-4 font-semibold">Event</th>
                    <th className="py-3 px-4 font-semibold">Organizer</th>
                    <th className="py-3 px-4 font-semibold">Type / Seat</th>
                    <th className="py-3 px-4 font-semibold">Booking Ref</th>
                    <th className="py-3 px-4 font-semibold text-center">Ticket Status</th>
                    <th className="py-3 px-4 font-semibold text-right">Check-in Status</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#f1f3ff]">
                  {loading ? (
                    Array.from({ length: pageSize > 5 ? 5 : pageSize }).map((_, idx) => (
                      <tr key={idx} className="animate-pulse">
                        <td className="py-4 px-4"><div className="h-4 bg-gray-200 rounded w-28 mb-1"></div><div className="h-3 bg-gray-100 rounded w-16"></div></td>
                        <td className="py-4 px-4"><div className="h-4 bg-gray-200 rounded w-28 mb-1"></div><div className="h-3 bg-gray-100 rounded w-36"></div></td>
                        <td className="py-4 px-4"><div className="h-4 bg-gray-200 rounded w-32 mb-1"></div><div className="h-3 bg-gray-100 rounded w-20"></div></td>
                        <td className="py-4 px-4"><div className="h-4 bg-gray-200 rounded w-28"></div></td>
                        <td className="py-4 px-4"><div className="h-4 bg-gray-200 rounded w-24 mb-1"></div><div className="h-3 bg-gray-100 rounded w-16"></div></td>
                        <td className="py-4 px-4"><div className="h-4 bg-gray-200 rounded w-24"></div></td>
                        <td className="py-4 px-4 text-center"><div className="h-5 bg-gray-200 rounded-full w-20 mx-auto"></div></td>
                        <td className="py-4 px-4 text-right"><div className="h-5 bg-gray-200 rounded-full w-24 ml-auto"></div></td>
                      </tr>
                    ))
                  ) : attendees.length === 0 ? (
                    <tr>
                      <td colSpan={8} className="py-12 text-center text-[#6d7a77]">
                        <div className="max-w-xs mx-auto space-y-2">
                          <svg className="w-8 h-8 text-[#bcc9c6] mx-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 110 4v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 110-4V7a2 2 0 00-2-2H5z" />
                          </svg>
                          <p className="font-semibold text-xs text-[#141b2b]">No attendees found</p>
                          <p className="text-[11px]">
                            {hasActiveFilters
                              ? "Try adjusting your search criteria or resetting filters."
                              : "No attendee passes have been issued on the platform yet."}
                          </p>
                          {hasActiveFilters && (
                            <button
                              type="button"
                              onClick={handleResetFilters}
                              className="text-xs text-[#00685f] font-semibold underline hover:text-[#005049] cursor-pointer"
                            >
                              Clear all filters
                            </button>
                          )}
                        </div>
                      </td>
                    </tr>
                  ) : (
                    attendees.map((att) => {
                      const tStatus = att.status?.toUpperCase() || "VALID";
                      const tBadge = attendeeStatusBadgeStyles[tStatus] || "bg-[#f3f4f6] text-[#4b5563]";
                      const tLabel = attendeeStatusLabels[tStatus] || tStatus;

                      return (
                        <tr key={att.ticket_id} className="hover:bg-[#f9f9ff] transition-colors">
                          {/* Ticket Code & Admission # */}
                          <td className="py-3.5 px-4 font-medium">
                            <div className="flex items-center gap-1.5">
                              <span className="font-mono font-bold text-[#00685f]">
                                {att.ticket_code}
                              </span>
                              <button
                                type="button"
                                onClick={() => handleCopyCode(att.ticket_code)}
                                className="text-[#6d7a77] hover:text-[#00685f] transition-colors p-0.5 cursor-pointer"
                                title="Copy ticket code"
                              >
                                <svg className="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
                                  <path d="M8 3a1 1 0 011-1h2a1 1 0 110 2H9a1 1 0 01-1-1z" />
                                  <path d="M6 3a2 2 0 00-2 2v11a2 2 0 002 2h8a2 2 0 002-2V5a2 2 0 00-2-2 3 3 0 01-3 3H9a3 3 0 01-3-3z" />
                                </svg>
                              </button>
                              {copiedCode === att.ticket_code && (
                                <span className="text-[10px] text-[#00685f] font-semibold">Copied!</span>
                              )}
                            </div>
                            <span className="text-[11px] text-[#6d7a77] block mt-0.5">
                              Pass #{att.admission_number || 1}
                            </span>
                          </td>

                          {/* Attendee Details */}
                          <td className="py-3.5 px-4">
                            <span className="font-semibold block text-[#141b2b]">
                              {att.customer?.full_name || "Guest Attendee"}
                            </span>
                            <span className="text-[11px] font-mono text-[#6d7a77] block truncate max-w-[170px]" title={att.customer?.email}>
                              {att.customer?.email || "—"}
                            </span>
                          </td>

                          {/* Event */}
                          <td className="py-3.5 px-4 max-w-[190px]">
                            <span className="font-medium text-[#141b2b] block truncate" title={att.event?.title}>
                              {att.event?.title || "Untitled Event"}
                            </span>
                            {att.event?.venue_name && (
                              <span className="text-[11px] text-[#6d7a77] block truncate" title={att.event.venue_name}>
                                📍 {att.event.venue_name}
                              </span>
                            )}
                          </td>

                          {/* Organizer */}
                          <td className="py-3.5 px-4 max-w-[170px]">
                            <span className="font-semibold text-[#3d4947] block truncate" title={att.organizer?.business_name}>
                              {att.organizer?.business_name || "—"}
                            </span>
                            {att.organizer?.email && (
                              <span className="text-[11px] font-mono text-[#6d7a77] block truncate" title={att.organizer.email}>
                                {att.organizer.email}
                              </span>
                            )}
                          </td>

                          {/* Ticket Type & Seat */}
                          <td className="py-3.5 px-4">
                            <span className="font-medium text-[#141b2b] block">
                              {att.ticket_type_name || "Admission"}
                            </span>
                            {att.seat_label && (
                              <span className="text-[11px] text-[#00685f] block font-medium">
                                {att.seat_label}
                              </span>
                            )}
                          </td>

                          {/* Booking Ref */}
                          <td className="py-3.5 px-4 font-mono text-xs text-[#3d4947]">
                            {att.booking_number || "—"}
                          </td>

                          {/* Ticket Status */}
                          <td className="py-3.5 px-4 text-center">
                            <span className={`inline-block text-[11px] font-bold px-2.5 py-0.5 rounded-full ${tBadge}`}>
                              {tStatus}
                            </span>
                          </td>

                          {/* Check-In Status & Timestamp */}
                          <td className="py-3.5 px-4 text-right">
                            <span className={`inline-block text-[10px] font-bold px-2 py-0.5 rounded-full ${tBadge}`}>
                              {tLabel}
                            </span>
                            {att.used_at ? (
                              <span className="block text-[10px] text-[#6d7a77] mt-0.5">
                                {formatDateTime(att.used_at)}
                              </span>
                            ) : (
                              <span className="block text-[10px] text-[#6d7a77] mt-0.5">
                                Issued: {formatDateTime(att.issued_at)}
                              </span>
                            )}
                          </td>
                        </tr>
                      );
                    })
                  )}
                </tbody>
              </table>
            </div>

            {/* Pagination Controls */}
            {!loading && attendees.length > 0 && (
              <div className="bg-[#f9f9ff] border-t border-[#bcc9c6] px-4 py-3 flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-[#3d4947]">
                <div className="flex items-center gap-3">
                  <span>
                    Showing <strong className="text-[#141b2b]">{(currentPage - 1) * pageSize + 1}</strong> to{" "}
                    <strong className="text-[#141b2b]">
                      {Math.min(currentPage * pageSize, totalRecords)}
                    </strong>{" "}
                    of <strong className="text-[#141b2b]">{totalRecords}</strong> attendees
                  </span>
                  <div className="flex items-center gap-1.5 ml-2">
                    <span className="text-[11px] text-[#6d7a77]">Rows:</span>
                    <select
                      value={pageSize}
                      onChange={(e) => {
                        setPageSize(Number(e.target.value));
                        setCurrentPage(1);
                      }}
                      className="bg-white border border-[#bcc9c6] rounded px-2 py-1 text-xs text-[#141b2b] focus:outline-none cursor-pointer"
                    >
                      {PAGE_SIZE_OPTIONS.map((sz) => (
                        <option key={sz} value={sz}>
                          {sz}
                        </option>
                      ))}
                    </select>
                  </div>
                </div>

                <div className="flex items-center gap-1">
                  <button
                    type="button"
                    onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                    disabled={currentPage <= 1}
                    className="p-1.5 rounded-md border border-[#bcc9c6] bg-white text-[#141b2b] hover:bg-[#eef0fb] disabled:opacity-40 disabled:cursor-not-allowed transition-colors cursor-pointer"
                    aria-label="Previous page"
                  >
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
                    </svg>
                  </button>

                  <span className="px-3 py-1 font-semibold text-[#141b2b]">
                    Page {currentPage} of {totalPages}
                  </span>

                  <button
                    type="button"
                    onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
                    disabled={currentPage >= totalPages}
                    className="p-1.5 rounded-md border border-[#bcc9c6] bg-white text-[#141b2b] hover:bg-[#eef0fb] disabled:opacity-40 disabled:cursor-not-allowed transition-colors cursor-pointer"
                    aria-label="Next page"
                  >
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                    </svg>
                  </button>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
