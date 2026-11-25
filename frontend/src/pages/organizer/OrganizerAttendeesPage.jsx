import { useState, useEffect } from "react";
import { formatDateTime } from "../../utils/formatters";
import notify from "../../utils/toast";
import {
  getOrganizerAttendees,
  getOrganizerAttendeeStats,
} from "../../api/organizerBookingApi";
import { getOrganizerEvents } from "../../api/organizerApi";

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

export default function OrganizerAttendeesPage() {
  // Filters State
  const [searchTerm, setSearchTerm] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
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
  const [organizerEvents, setOrganizerEvents] = useState([]);

  // UI Interactive States
  const [loading, setLoading] = useState(true);
  const [loadingEvents, setLoadingEventsList] = useState(false);
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

  // Load organizer events list for filter dropdown
  useEffect(() => {
    let cancelled = false;

    async function loadEvents() {
      setLoadingEventsList(true);
      try {
        const res = await getOrganizerEvents({ limit: 100 });
        if (cancelled) return;
        const list = Array.isArray(res?.data?.events)
          ? res.data.events
          : Array.isArray(res?.data)
          ? res.data
          : Array.isArray(res?.events)
          ? res.events
          : [];
        setOrganizerEvents(list);
      } catch (err) {
        if (!cancelled) {
          console.error("Failed to load organizer events for attendee filter dropdown:", err);
        }
      } finally {
        if (!cancelled) {
          setLoadingEventsList(false);
        }
      }
    }

    loadEvents();

    return () => {
      cancelled = true;
    };
  }, []);

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
          event_id: selectedEventId !== "ALL" ? selectedEventId : "",
          check_in_status: selectedStatus !== "ALL" ? selectedStatus : "",
          search: debouncedSearch,
        };

        const [attendeesRes, statsRes] = await Promise.all([
          getOrganizerAttendees(filterOptions),
          getOrganizerAttendeeStats(filterOptions),
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
        console.error("Error fetching organizer attendees:", err);
        const errorMsg =
          err.response?.data?.message ||
          err.response?.data?.error ||
          err.message ||
          "Failed to load attendees. Please check your connection and retry.";
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
    setSelectedEventId("ALL");
    setSelectedStatus("ALL");
    setCurrentPage(1);
    notify.info("Filters cleared");
  };

  const hasActiveFilters = Boolean(
    searchTerm.trim() || selectedEventId !== "ALL" || selectedStatus !== "ALL"
  );

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 mb-1">
            <h1 className="text-2xl font-bold text-[#141B2B] tracking-tight font-space">
              Attendee Management
            </h1>
            <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold bg-[#ccfbf1] text-[#0f766e] border border-[#99f6e4]">
              Gate Admissions
            </span>
          </div>
          <p className="text-sm text-slate-500 font-sans">
            Monitor individual admission passes, check-in status, gate progress, and attendance rates across your events.
          </p>
        </div>

        {/* Global summary badge and refresh trigger */}
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={() => setRefreshKey((k) => k + 1)}
            disabled={loading}
            className="px-3 py-1.5 rounded-xl bg-white border border-slate-200 text-xs font-semibold text-slate-700 hover:bg-slate-50 transition-colors flex items-center gap-1.5 cursor-pointer shadow-sm disabled:opacity-60"
          >
            <svg className={`w-3.5 h-3.5 ${loading ? "animate-spin text-[#00685F]" : ""}`} viewBox="0 0 20 20" fill="currentColor">
              <path fillRule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z" clipRule="evenodd" />
            </svg>
            <span>Refresh</span>
          </button>
        </div>
      </div>

      {/* Summary KPI Cards (Server-Calculated Statistics) */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
        {/* Card 1: Total Issued Tickets */}
        <div className="bg-white rounded-2xl p-5 border border-slate-200 shadow-sm transition-all hover:border-slate-300">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-semibold uppercase tracking-wider text-slate-400">Total Issued</span>
            <div className="w-8 h-8 rounded-lg bg-blue-50 text-blue-600 flex items-center justify-center">
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 110 4v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 110-4V7a2 2 0 00-2-2H5z" />
              </svg>
            </div>
          </div>
          <div className="text-2xl font-bold text-[#141B2B] font-space">{stats.total_issued_tickets ?? 0}</div>
          <p className="text-xs text-slate-500 mt-1">All tickets created</p>
        </div>

        {/* Card 2: Active Eligible Admissions */}
        <div className="bg-white rounded-2xl p-5 border border-slate-200 shadow-sm transition-all hover:border-slate-300">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-semibold uppercase tracking-wider text-slate-400">Active Admissions</span>
            <div className="w-8 h-8 rounded-lg bg-[#ccfbf1] text-[#00685F] flex items-center justify-center">
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
              </svg>
            </div>
          </div>
          <div className="text-2xl font-bold text-[#00685F] font-space">{stats.active_eligible_admissions ?? 0}</div>
          <p className="text-xs text-slate-500 mt-1">Valid + Checked-in passes</p>
        </div>

        {/* Card 3: Checked-In Attendees */}
        <div className="bg-white rounded-2xl p-5 border border-slate-200 shadow-sm transition-all hover:border-slate-300">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-semibold uppercase tracking-wider text-slate-400">Checked In</span>
            <div className="w-8 h-8 rounded-lg bg-emerald-50 text-emerald-600 flex items-center justify-center">
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
            </div>
          </div>
          <div className="text-2xl font-bold text-emerald-600 font-space">{stats.checked_in_tickets ?? 0}</div>
          <p className="text-xs text-slate-500 mt-1">Admitted at gate</p>
        </div>

        {/* Card 4: Pending Check-Ins */}
        <div className="bg-white rounded-2xl p-5 border border-slate-200 shadow-sm transition-all hover:border-slate-300">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-semibold uppercase tracking-wider text-slate-400">Pending Gate</span>
            <div className="w-8 h-8 rounded-lg bg-amber-50 text-amber-600 flex items-center justify-center">
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
          </div>
          <div className="text-2xl font-bold text-amber-600 font-space">{stats.pending_check_in_tickets ?? 0}</div>
          <p className="text-xs text-slate-500 mt-1">Awaiting arrival</p>
        </div>

        {/* Card 5: Check-In Rate (%) with Progress Bar */}
        <div className="bg-white rounded-2xl p-5 border border-slate-200 shadow-sm transition-all hover:border-slate-300 flex flex-col justify-between">
          <div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-semibold uppercase tracking-wider text-slate-400">Gate Progress</span>
              <span className="text-xs font-bold text-[#00685F]">{stats.check_in_rate || "0"}%</span>
            </div>
            <div className="text-2xl font-bold text-[#141B2B] font-space">{stats.check_in_rate || "0"}%</div>
          </div>
          <div className="mt-2">
            <div className="w-full bg-slate-100 rounded-full h-2 overflow-hidden">
              <div
                className="bg-[#00685F] h-2 rounded-full transition-all duration-500"
                style={{ width: `${Math.min(parseFloat(stats.check_in_rate) || 0, 100)}%` }}
              ></div>
            </div>
            <p className="text-[11px] text-slate-400 mt-1.5 flex justify-between">
              <span>{stats.checked_in_tickets ?? 0} admitted</span>
              <span>{stats.active_eligible_admissions ?? 0} total active</span>
            </p>
          </div>
        </div>
      </div>

      {/* Filter and Search Bar */}
      <div className="bg-white rounded-2xl p-4 sm:p-5 border border-slate-200 shadow-sm space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-12 gap-3 items-center">
          {/* Search Input */}
          <div className="md:col-span-5 relative">
            <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-slate-400">
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
            </div>
            <input
              type="text"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              placeholder="Search attendee, email, ticket code, booking #..."
              className="w-full pl-10 pr-4 py-2.5 text-sm bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-[#00685F] focus:border-transparent transition-all"
            />
            {searchTerm && (
              <button
                type="button"
                onClick={() => setSearchTerm("")}
                className="absolute inset-y-0 right-0 pr-3 flex items-center text-slate-400 hover:text-slate-600"
              >
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            )}
          </div>

          {/* Event Filter */}
          <div className="md:col-span-4">
            <select
              value={selectedEventId}
              onChange={(e) => {
                setSelectedEventId(e.target.value);
                setCurrentPage(1);
              }}
              disabled={loadingEvents}
              className="w-full px-3.5 py-2.5 text-sm bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-[#00685F] focus:border-transparent transition-all text-slate-700 font-medium disabled:opacity-60"
            >
              <option value="ALL">All Events</option>
              {organizerEvents.map((ev) => (
                <option key={ev.id} value={ev.id}>
                  {ev.title}
                </option>
              ))}
            </select>
          </div>

          {/* Check-In Status Filter */}
          <div className="md:col-span-3">
            <select
              value={selectedStatus}
              onChange={(e) => {
                setSelectedStatus(e.target.value);
                setCurrentPage(1);
              }}
              className="w-full px-3.5 py-2.5 text-sm bg-slate-50 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-[#00685F] focus:border-transparent transition-all text-slate-700 font-medium"
            >
              {CHECKIN_STATUS_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>
        </div>

        {/* Filter Summary & Reset Action */}
        {hasActiveFilters && (
          <div className="pt-2 border-t border-slate-100 flex flex-wrap items-center justify-between gap-2">
            <div className="text-xs text-slate-500 flex items-center gap-2">
              <span>Showing filtered results:</span>
              <span className="font-semibold text-slate-700">{totalRecords} total passes found</span>
            </div>
            <button
              type="button"
              onClick={handleResetFilters}
              className="inline-flex items-center gap-1.5 text-xs font-semibold text-[#00685F] hover:text-[#00524b] transition-colors cursor-pointer"
            >
              <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              Clear All Filters
            </button>
          </div>
        )}
      </div>

      {/* Attendees Table / States */}
      <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
        {/* Loading State Skeleton */}
        {loading && (
          <div className="p-8 space-y-4">
            {[1, 2, 3, 4, 5].map((idx) => (
              <div key={idx} className="h-14 bg-slate-100 animate-pulse rounded-xl"></div>
            ))}
          </div>
        )}

        {/* Error State */}
        {!loading && error && (
          <div className="p-12 text-center">
            <div className="w-12 h-12 rounded-full bg-red-100 text-red-600 flex items-center justify-center mx-auto mb-3">
              <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
            </div>
            <h3 className="text-base font-semibold text-slate-800 mb-1">Failed to Load Attendees</h3>
            <p className="text-sm text-slate-500 mb-4">{error}</p>
            <button
              type="button"
              onClick={() => setRefreshKey((k) => k + 1)}
              className="px-4 py-2 bg-[#00685F] text-white rounded-xl text-sm font-semibold hover:bg-[#00524b] transition-all cursor-pointer"
            >
              Retry
            </button>
          </div>
        )}

        {/* Empty State */}
        {!loading && !error && attendees.length === 0 && (
          <div className="p-12 text-center">
            <div className="w-12 h-12 rounded-full bg-slate-100 text-slate-400 flex items-center justify-center mx-auto mb-3">
              <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
              </svg>
            </div>
            <h3 className="text-base font-semibold text-slate-800 mb-1">No Attendees Found</h3>
            <p className="text-sm text-slate-500 mb-4 max-w-sm mx-auto">
              {hasActiveFilters
                ? "No attendee passes match your selected filter criteria. Try adjusting your search or filters."
                : "No ticket admissions have been issued for your events yet."}
            </p>
            {hasActiveFilters && (
              <button
                type="button"
                onClick={handleResetFilters}
                className="px-4 py-2 bg-[#00685F] text-white rounded-xl text-sm font-semibold hover:bg-[#00524b] transition-all cursor-pointer"
              >
                Reset Filters
              </button>
            )}
          </div>
        )}

        {/* Table View */}
        {!loading && !error && attendees.length > 0 && (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-slate-50 border-b border-slate-200 text-[11px] font-bold text-slate-500 uppercase tracking-wider font-space">
                  <th className="py-3.5 px-4">Attendee</th>
                  <th className="py-3.5 px-4">Ticket Code</th>
                  <th className="py-3.5 px-4">Event</th>
                  <th className="py-3.5 px-4">Ticket Type</th>
                  <th className="py-3.5 px-4">Seat</th>
                  <th className="py-3.5 px-4">Booking Ref</th>
                  <th className="py-3.5 px-4">Check-In Status</th>
                  <th className="py-3.5 px-4">Checked-In At</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 text-sm">
                {attendees.map((attendee) => {
                  const statusStyle =
                    attendeeStatusBadgeStyles[attendee.status] ||
                    "bg-slate-100 text-slate-600 border border-slate-200";
                  const statusLabel = attendeeStatusLabels[attendee.status] || attendee.status;
                  const attendeeName = attendee.customer?.full_name || attendee.attendee_name || "—";
                  const attendeeEmail = attendee.customer?.email || attendee.attendee_email || "—";
                  const eventTitle = attendee.event?.title || attendee.event_title || "—";
                  const ticketTierName = attendee.ticket_type_name || attendee.ticket_tier_name || "Standard Pass";
                  const bookingRef = attendee.booking_number || attendee.booking_reference || "—";

                  return (
                    <tr
                      key={attendee.ticket_id}
                      className="hover:bg-slate-50/70 transition-colors"
                    >
                      {/* Attendee Name & Email */}
                      <td className="py-3.5 px-4">
                        <div className="font-semibold text-slate-900 font-sans">
                          {attendeeName}
                        </div>
                        <div className="text-xs text-slate-500 font-sans">
                          {attendeeEmail}
                        </div>
                      </td>

                      {/* Ticket Code with Copy Button */}
                      <td className="py-3.5 px-4">
                        <div className="flex items-center gap-1.5">
                          <span className="font-mono text-xs font-semibold text-slate-700 bg-slate-100 px-2 py-1 rounded-md border border-slate-200">
                            {attendee.ticket_code}
                          </span>
                          <button
                            type="button"
                            onClick={() => handleCopyCode(attendee.ticket_code)}
                            title="Copy ticket code"
                            className="p-1 rounded hover:bg-slate-200 text-slate-400 hover:text-slate-600 transition-colors cursor-pointer"
                          >
                            {copiedCode === attendee.ticket_code ? (
                              <svg className="w-3.5 h-3.5 text-emerald-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                              </svg>
                            ) : (
                              <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                              </svg>
                            )}
                          </button>
                        </div>
                      </td>

                      {/* Event Title */}
                      <td className="py-3.5 px-4 max-w-[220px]">
                        <div className="font-medium text-slate-900 truncate" title={eventTitle}>
                          {eventTitle}
                        </div>
                      </td>

                      {/* Ticket Tier */}
                      <td className="py-3.5 px-4">
                        <span className="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-slate-100 text-slate-700 border border-slate-200">
                          {ticketTierName}
                        </span>
                      </td>

                      {/* Seat */}
                      <td className="py-3.5 px-4 font-mono text-xs">
                        {attendee.seat_label ? (
                          <span className="font-semibold text-slate-800 bg-amber-50 text-amber-800 px-2 py-0.5 rounded border border-amber-200">
                            {attendee.seat_label}
                          </span>
                        ) : (
                          <span className="text-slate-400 font-sans italic">General Admission</span>
                        )}
                      </td>

                      {/* Booking Reference */}
                      <td className="py-3.5 px-4 font-mono text-xs font-medium text-slate-700">
                        {bookingRef}
                      </td>

                      {/* Check-In Status Badge */}
                      <td className="py-3.5 px-4">
                        <span
                          className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold ${statusStyle}`}
                        >
                          <span
                            className={`w-1.5 h-1.5 rounded-full ${
                              attendee.status === "USED"
                                ? "bg-emerald-600"
                                : attendee.status === "VALID"
                                ? "bg-amber-600"
                                : attendee.status === "CANCELLED"
                                ? "bg-red-600"
                                : "bg-purple-600"
                            }`}
                          ></span>
                          {statusLabel}
                        </span>
                      </td>

                      {/* Checked-In Timestamp */}
                      <td className="py-3.5 px-4 text-xs font-sans text-slate-600">
                        {attendee.used_at ? (
                          <div className="flex flex-col">
                            <span className="font-medium text-slate-900">
                              {formatDateTime(attendee.used_at)}
                            </span>
                          </div>
                        ) : (
                          <span className="text-slate-400 italic">Not Checked In</span>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}

        {/* Pagination Bar */}
        {!loading && !error && attendees.length > 0 && (
          <div className="py-4 px-5 border-t border-slate-200 bg-white flex flex-col sm:flex-row items-center justify-between gap-4">
            {/* Range and Total */}
            <div className="text-xs text-slate-500">
              Showing <span className="font-semibold text-slate-700">{(currentPage - 1) * pageSize + 1}</span> to{" "}
              <span className="font-semibold text-slate-700">
                {Math.min(currentPage * pageSize, totalRecords)}
              </span>{" "}
              of <span className="font-semibold text-slate-700">{totalRecords}</span> attendees
            </div>

            {/* Page Size & Navigation */}
            <div className="flex items-center gap-4">
              {/* Page size dropdown */}
              <div className="flex items-center gap-2">
                <span className="text-xs text-slate-500">Rows per page:</span>
                <select
                  value={pageSize}
                  onChange={(e) => {
                    setPageSize(Number(e.target.value));
                    setCurrentPage(1);
                  }}
                  className="text-xs border border-slate-200 rounded-lg px-2.5 py-1.5 bg-slate-50 font-medium text-slate-700 focus:outline-none focus:ring-2 focus:ring-[#00685F] cursor-pointer"
                >
                  {PAGE_SIZE_OPTIONS.map((size) => (
                    <option key={size} value={size}>
                      {size}
                    </option>
                  ))}
                </select>
              </div>

              {/* Prev / Next buttons */}
              <div className="flex items-center gap-1.5">
                <button
                  type="button"
                  onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                  disabled={currentPage <= 1}
                  className="px-3 py-1.5 rounded-lg border border-slate-200 text-xs font-semibold text-slate-600 hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed transition-all cursor-pointer"
                >
                  Previous
                </button>
                <div className="text-xs font-semibold text-slate-700 px-2 font-mono">
                  {currentPage} / {totalPages}
                </div>
                <button
                  type="button"
                  onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
                  disabled={currentPage >= totalPages}
                  className="px-3 py-1.5 rounded-lg border border-slate-200 text-xs font-semibold text-slate-600 hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed transition-all cursor-pointer"
                >
                  Next
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
