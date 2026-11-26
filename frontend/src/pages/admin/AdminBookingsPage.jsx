import { useState, useEffect } from "react";
import { formatCurrency, formatDateTime } from "../../utils/formatters";
import notify from "../../utils/toast";
import AdminBookingDetailDrawer from "../../components/admin/AdminBookingDetailDrawer";
import {
  getAdminBookings,
  getAdminBookingStats,
  getAdminBookingDetail,
} from "../../api/adminBookingApi";
import { getAdminEvents } from "../../api/adminEventApi";
import { getOrganizers } from "../../api/adminOrganizerApi";

// Status Badge Styling Maps
const bookingStatusBadgeStyles = {
  CONFIRMED: "bg-[#dcfce7] text-[#15803d] border border-[#bbf7d0]",
  PENDING: "bg-[#fef9c3] text-[#854d0e] border border-[#fef08a]",
  CANCELLED: "bg-[#fee2e2] text-[#b91c1c] border border-[#fecaca]",
  PARTIALLY_CANCELLED: "bg-[#ffedd5] text-[#c2410c] border border-[#fed7aa]",
};

const paymentStatusBadgeStyles = {
  SUCCESS: "bg-[#dcfce7] text-[#15803d] border border-[#bbf7d0]",
  PENDING: "bg-[#fef9c3] text-[#854d0e] border border-[#fef08a]",
  FAILED: "bg-[#fee2e2] text-[#b91c1c] border border-[#fecaca]",
  REFUND_REQUIRED: "bg-[#ffedd5] text-[#c2410c] border border-[#fed7aa]",
};

const refundStatusBadgeStyles = {
  FULL: "bg-[#fae8ff] text-[#86198f] border border-[#f5d0fe]",
  PARTIAL: "bg-[#f3e8ff] text-[#7e22ce] border border-[#e9d5ff]",
  PENDING: "bg-[#fef9c3] text-[#854d0e] border border-[#fef08a]",
  FAILED: "bg-[#fee2e2] text-[#b91c1c] border border-[#fecaca]",
};

const BOOKING_STATUS_OPTIONS = [
  { value: "ALL", label: "All Booking Statuses" },
  { value: "CONFIRMED", label: "Confirmed" },
  { value: "PENDING", label: "Pending" },
  { value: "CANCELLED", label: "Cancelled" },
  { value: "PARTIALLY_CANCELLED", label: "Partially Cancelled" },
];

const PAYMENT_STATUS_OPTIONS = [
  { value: "ALL", label: "All Payment Statuses" },
  { value: "SUCCESS", label: "Captured / Success" },
  { value: "PENDING", label: "Pending Payment" },
  { value: "FAILED", label: "Failed Payment" },
  { value: "REFUNDED", label: "Any Refund" },
  { value: "PARTIAL", label: "Partially Refunded" },
  { value: "FULL", label: "Fully Refunded" },
  { value: "PENDING_REFUND", label: "Refund Pending / Processing" },
];

const PAGE_SIZE_OPTIONS = [10, 25, 50, 100];

const INITIAL_STATS = {
  total_bookings: 0,
  confirmed_bookings: 0,
  cancelled_bookings: 0,
  partially_cancelled_bookings: 0,
  total_tickets_sold: 0,
  gross_captured_amount: "0.00",
  total_refunded_amount: "0.00",
  net_captured_amount: "0.00",
  currency: "INR",
};

export default function AdminBookingsPage() {
  // State for Filters & Search
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [selectedOrganizerId, setSelectedOrganizerId] = useState("");
  const [selectedEventId, setSelectedEventId] = useState("");
  const [bookingStatus, setBookingStatus] = useState("ALL");
  const [paymentStatus, setPaymentStatus] = useState("ALL");
  const [fromDate, setFromDate] = useState("");
  const [toDate, setToDate] = useState("");

  // Pagination State
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useState(10);
  const [totalRecords, setTotalRecords] = useState(0);
  const [totalPages, setTotalPages] = useState(1);

  // Data States
  const [bookings, setBookings] = useState([]);
  const [stats, setStats] = useState(INITIAL_STATS);
  const [organizersList, setOrganizersList] = useState([]);
  const [eventsList, setEventsList] = useState([]);

  // UI Interactive States
  const [loading, setLoading] = useState(true);
  const [loadingOrganizers, setLoadingOrganizers] = useState(false);
  const [loadingEvents, setLoadingEvents] = useState(false);
  const [error, setError] = useState(null);
  const [activeBookingDetail, setActiveBookingDetail] = useState(null);
  const [copiedCode, setCopiedCode] = useState(null);
  const [refreshKey, setRefreshKey] = useState(0);

  // Debounce search input by 350ms
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 350);
    return () => clearTimeout(timer);
  }, [search]);

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
          console.error("Failed to load organizers for filter dropdown:", err);
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
        if (selectedOrganizerId) {
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
          .map((evt) => {
            if (!evt || typeof evt !== "object") return null;
            const id = evt.id || evt.ID || evt.event_id || evt.eventId || "";
            const title =
              evt.title ||
              evt.Title ||
              evt.name ||
              evt.eventName ||
              evt.event_name ||
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
          console.error("Failed to load events for filter dropdown:", err);
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

  // Fetch bookings list and aggregate stats from backend
  useEffect(() => {
    let cancelled = false;

    async function fetchBookingsData() {
      setLoading(true);
      setError(null);

      try {
        const filterOptions = {
          page,
          limit,
          organizer_id: selectedOrganizerId,
          event_id: selectedEventId,
          booking_status: bookingStatus,
          payment_status: paymentStatus,
          search: debouncedSearch,
          from: fromDate,
          to: toDate,
        };

        const [bookingsRes, statsRes] = await Promise.all([
          getAdminBookings(filterOptions),
          getAdminBookingStats(filterOptions),
        ]);

        if (cancelled) return;

        // Process Bookings List & Pagination
        const bookingsList =
          bookingsRes?.data?.bookings ||
          bookingsRes?.bookings ||
          (Array.isArray(bookingsRes?.data) ? bookingsRes.data : []);

        const paginationInfo =
          bookingsRes?.data?.pagination ||
          bookingsRes?.data ||
          bookingsRes?.pagination ||
          {};

        setBookings(bookingsList);
        setTotalRecords(paginationInfo.total_records ?? bookingsList.length);
        setTotalPages(Math.max(1, paginationInfo.total_pages ?? Math.ceil(bookingsList.length / limit)));

        // Process Stats
        if (statsRes?.data) {
          setStats(statsRes.data);
        } else if (statsRes && typeof statsRes === "object" && statsRes.total_bookings !== undefined) {
          setStats(statsRes);
        }
      } catch (err) {
        if (cancelled) return;
        console.error("Error fetching admin bookings:", err);
        const errorMsg =
          err.response?.data?.message ||
          err.response?.data?.error ||
          err.message ||
          "Failed to load platform bookings. Please check your connection and retry.";
        setError(errorMsg);
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    fetchBookingsData();

    return () => {
      cancelled = true;
    };
  }, [
    page,
    limit,
    selectedOrganizerId,
    selectedEventId,
    bookingStatus,
    paymentStatus,
    debouncedSearch,
    fromDate,
    toDate,
    refreshKey,
  ]);

  const copyToClipboard = async (text, label) => {
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
      setCopiedCode(text);
      notify.info(`Copied ${label} to clipboard`);
      setTimeout(() => setCopiedCode(null), 2000);
    } catch {
      notify.error("Failed to copy");
    }
  };

  const handleResetFilters = () => {
    setSearch("");
    setDebouncedSearch("");
    setSelectedOrganizerId("");
    setSelectedEventId("");
    setBookingStatus("ALL");
    setPaymentStatus("ALL");
    setFromDate("");
    setToDate("");
    setPage(1);
    notify.info("Filters cleared");
  };

  const handleViewBookingDetail = async (bookingSummary) => {
    setActiveBookingDetail(bookingSummary);
    try {
      const res = await getAdminBookingDetail(bookingSummary.id);
      if (res?.data) {
        setActiveBookingDetail(res.data);
      }
    } catch (err) {
      console.error("Failed to load deep booking details:", err);
      notify.error("Could not load complete booking breakdown");
    }
  };

  const hasActiveFilters = Boolean(
    search || selectedOrganizerId || selectedEventId || bookingStatus !== "ALL" || paymentStatus !== "ALL" || fromDate || toDate
  );

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-white w-full">
      {/* Page Header */}
      <div className="px-4 sm:px-6 py-4 border-b border-[#dce2f7] flex flex-col md:flex-row md:items-center justify-between gap-4 shrink-0 bg-white">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-[#141b2b] text-xl font-bold tracking-tight">
              Platform Bookings Management
            </h1>
            <span className="bg-[#e6f4f2] text-[#00685f] text-xs font-semibold px-2.5 py-0.5 rounded-full border border-[#00685f]/20">
              Platform Overview
            </span>
          </div>
          <p className="text-[#3d4947] text-xs sm:text-sm mt-0.5">
            Monitor, inspect, and analyze all customer orders, gate admissions, payment captures, and refunds platform-wide.
          </p>
        </div>

        <button
          type="button"
          onClick={() => setRefreshKey((k) => k + 1)}
          disabled={loading}
          className="bg-[#f9f9ff] border border-[#bcc9c6] text-[#141b2b] text-xs sm:text-sm font-semibold px-3 py-1.5 rounded-lg flex items-center gap-1.5 hover:bg-[#eef0fb] transition-colors cursor-pointer self-start md:self-auto shrink-0"
        >
          <svg className={`w-3.5 h-3.5 ${loading ? "animate-spin text-[#00685f]" : ""}`} viewBox="0 0 20 20" fill="currentColor">
            <path fillRule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z" clipRule="evenodd" />
          </svg>
          <span>Refresh</span>
        </button>
      </div>

      {/* Main Content Area */}
      <div className="flex-1 overflow-y-auto p-4 sm:p-6 space-y-5">
        {/* =================================================================== */}
        {/* SUMMARY METRICS CARDS (Server-Owned Statistics)                     */}
        {/* =================================================================== */}
        <div className="grid grid-cols-2 sm:grid-cols-3 xl:grid-cols-6 gap-3.5">
          {/* Card 1: Total Bookings */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-xs flex flex-col justify-between">
            <span className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Total Orders
            </span>
            <span className="text-[#141b2b] text-xl lg:text-2xl font-bold tracking-tight">
              {stats.total_bookings ?? 0}
            </span>
            <span className="text-[11px] mt-1 text-[#6d7a77] font-medium">All platform orders</span>
          </div>

          {/* Card 2: Confirmed Bookings */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-xs flex flex-col justify-between">
            <span className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Confirmed
            </span>
            <span className="text-[#15803d] text-xl lg:text-2xl font-bold tracking-tight">
              {stats.confirmed_bookings ?? 0}
            </span>
            <span className="text-[11px] mt-1 text-[#15803d] font-medium">Active admissions</span>
          </div>

          {/* Card 3: Cancelled Bookings */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-xs flex flex-col justify-between">
            <span className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Cancelled
            </span>
            <span className="text-[#b91c1c] text-xl lg:text-2xl font-bold tracking-tight">
              {stats.cancelled_bookings ?? 0}
            </span>
            <span className="text-[11px] mt-1 text-[#b91c1c] font-medium">Cancelled orders</span>
          </div>

          {/* Card 4: Tickets Sold */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-xs flex flex-col justify-between">
            <span className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Tickets Sold
            </span>
            <span className="text-[#141b2b] text-xl lg:text-2xl font-bold tracking-tight">
              {stats.total_tickets_sold ?? 0}
            </span>
            <span className="text-[11px] mt-1 text-[#6d7a77] font-medium">Valid sold passes</span>
          </div>

          {/* Card 5: Gross Captured */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-xs flex flex-col justify-between">
            <span className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Gross Captured
            </span>
            <span className="text-[#141b2b] text-xl lg:text-2xl font-bold tracking-tight">
              {formatCurrency(stats.gross_captured_amount, stats.currency)}
            </span>
            <span className="text-[11px] mt-1 text-[#6d7a77] font-medium">Total gateway receipts</span>
          </div>

          {/* Card 6: Net Captured */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-xs flex flex-col justify-between">
            <span className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Net Captured
            </span>
            <span className="text-[#00685f] text-xl lg:text-2xl font-bold tracking-tight">
              {formatCurrency(stats.net_captured_amount, stats.currency)}
            </span>
            <span className="text-[11px] mt-1 text-[#00685f] font-medium">Captured after refunds</span>
          </div>
        </div>

        {/* =================================================================== */}
        {/* FILTER BAR                                                          */}
        {/* =================================================================== */}
        <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-3.5 space-y-3">
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-12 gap-3">
            {/* Search Input (3 columns on lg) */}
            <div className="lg:col-span-3 relative">
              <svg className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-[#6d7a77]" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z" clipRule="evenodd" />
              </svg>
              <input
                type="text"
                placeholder="Search booking #, customer, email..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full bg-white border border-[#bcc9c6] rounded-lg pl-9 pr-8 py-2 text-xs sm:text-sm text-[#141b2b] placeholder:text-[#6d7a77] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40"
              />
              {search && (
                <button
                  type="button"
                  onClick={() => setSearch("")}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 text-[#6d7a77] hover:text-[#141b2b] cursor-pointer text-xs"
                >
                  ✕
                </button>
              )}
            </div>

            {/* Organizer Dropdown (3 columns on lg) */}
            <div className="lg:col-span-3">
              <select
                value={selectedOrganizerId}
                onChange={(e) => {
                  setSelectedOrganizerId(e.target.value);
                  setSelectedEventId("");
                  setPage(1);
                }}
                disabled={loadingOrganizers}
                className="w-full bg-white border border-[#bcc9c6] rounded-lg px-3 py-2 text-xs sm:text-sm text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40 cursor-pointer disabled:opacity-60"
              >
                <option value="">
                  {loadingOrganizers ? "Loading organizers..." : "All Organizers"}
                </option>
                {organizersList.map((org) => (
                  <option key={org.id} value={org.id}>
                    {org.name}
                  </option>
                ))}
              </select>
            </div>

            {/* Event Dropdown (3 columns on lg) */}
            <div className="lg:col-span-3">
              <select
                value={selectedEventId}
                onChange={(e) => {
                  setSelectedEventId(e.target.value);
                  setPage(1);
                }}
                disabled={loadingEvents}
                className="w-full bg-white border border-[#bcc9c6] rounded-lg px-3 py-2 text-xs sm:text-sm text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40 cursor-pointer disabled:opacity-60"
              >
                <option value="">
                  {loadingEvents ? "Loading events..." : "All Events"}
                </option>
                {eventsList.map((evt) => (
                  <option key={evt.id} value={evt.id}>
                    {evt.title}
                  </option>
                ))}
              </select>
            </div>

            {/* Booking Status Dropdown (3 columns on lg) */}
            <div className="lg:col-span-3">
              <select
                value={bookingStatus}
                onChange={(e) => {
                  setBookingStatus(e.target.value);
                  setPage(1);
                }}
                className="w-full bg-white border border-[#bcc9c6] rounded-lg px-3 py-2 text-xs sm:text-sm text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40 cursor-pointer"
              >
                {BOOKING_STATUS_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Secondary Filter Row: Payment Status & Date Range */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-12 gap-3 pt-1 border-t border-[#dce2f7]/60">
            {/* Payment / Refund Status Filter (4 columns on lg) */}
            <div className="lg:col-span-4">
              <select
                value={paymentStatus}
                onChange={(e) => {
                  setPaymentStatus(e.target.value);
                  setPage(1);
                }}
                className="w-full bg-white border border-[#bcc9c6] rounded-lg px-3 py-2 text-xs sm:text-sm text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40 cursor-pointer"
              >
                {PAYMENT_STATUS_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>

            {/* From Date (3 columns on lg) */}
            <div className="lg:col-span-3 flex items-center gap-1.5">
              <span className="text-xs text-[#6d7a77] shrink-0 font-medium">From:</span>
              <input
                type="date"
                value={fromDate}
                onChange={(e) => {
                  setFromDate(e.target.value);
                  setPage(1);
                }}
                className="w-full bg-white border border-[#bcc9c6] rounded-lg px-2.5 py-1.5 text-xs text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40 cursor-pointer"
              />
            </div>

            {/* To Date (3 columns on lg) */}
            <div className="lg:col-span-3 flex items-center gap-1.5">
              <span className="text-xs text-[#6d7a77] shrink-0 font-medium">To:</span>
              <input
                type="date"
                value={toDate}
                onChange={(e) => {
                  setToDate(e.target.value);
                  setPage(1);
                }}
                className="w-full bg-white border border-[#bcc9c6] rounded-lg px-2.5 py-1.5 text-xs text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40 cursor-pointer"
              />
            </div>

            {/* Reset Filters Button (2 columns on lg) */}
            <div className="lg:col-span-2 flex items-center justify-end">
              {hasActiveFilters && (
                <button
                  type="button"
                  onClick={handleResetFilters}
                  className="w-full text-xs text-[#ba1a1a] hover:text-[#93000a] font-semibold py-1.5 px-3 rounded-lg border border-[#ba1a1a]/30 hover:bg-[#ffdad6]/20 transition-colors cursor-pointer text-center"
                >
                  Reset Filters
                </button>
              )}
            </div>
          </div>
        </div>

        {/* =================================================================== */}
        {/* BOOKINGS TABLE OR ERROR / LOADING STATE                             */}
        {/* =================================================================== */}
        {error ? (
          <div className="border border-[#ffdad6] bg-[#fff8f7] rounded-xl p-8 text-center space-y-3">
            <div className="w-10 h-10 bg-[#ba1a1a]/10 text-[#ba1a1a] rounded-full flex items-center justify-center mx-auto">
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
            </div>
            <h3 className="text-sm font-bold text-[#141b2b]">Unable to load bookings</h3>
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
                    <th className="py-3 px-4 font-semibold">Booking Reference</th>
                    <th className="py-3 px-4 font-semibold">Customer</th>
                    <th className="py-3 px-4 font-semibold">Event</th>
                    <th className="py-3 px-4 font-semibold">Organizer</th>
                    <th className="py-3 px-3 font-semibold text-center">Tickets</th>
                    <th className="py-3 px-4 font-semibold text-right">Amount</th>
                    <th className="py-3 px-4 font-semibold text-center">Status</th>
                    <th className="py-3 px-4 font-semibold text-center">Payment / Refund</th>
                    <th className="py-3 px-4 font-semibold text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#f1f3ff]">
                  {loading ? (
                    Array.from({ length: limit > 5 ? 5 : limit }).map((_, idx) => (
                      <tr key={idx} className="animate-pulse">
                        <td className="py-4 px-4"><div className="h-4 bg-gray-200 rounded w-24 mb-1"></div><div className="h-3 bg-gray-100 rounded w-16"></div></td>
                        <td className="py-4 px-4"><div className="h-4 bg-gray-200 rounded w-28 mb-1"></div><div className="h-3 bg-gray-100 rounded w-36"></div></td>
                        <td className="py-4 px-4"><div className="h-4 bg-gray-200 rounded w-32 mb-1"></div><div className="h-3 bg-gray-100 rounded w-20"></div></td>
                        <td className="py-4 px-4"><div className="h-4 bg-gray-200 rounded w-28"></div></td>
                        <td className="py-4 px-3 text-center"><div className="h-4 bg-gray-200 rounded w-8 mx-auto"></div></td>
                        <td className="py-4 px-4 text-right"><div className="h-4 bg-gray-200 rounded w-16 ml-auto"></div></td>
                        <td className="py-4 px-4 text-center"><div className="h-5 bg-gray-200 rounded-full w-20 mx-auto"></div></td>
                        <td className="py-4 px-4 text-center"><div className="h-5 bg-gray-200 rounded-full w-24 mx-auto"></div></td>
                        <td className="py-4 px-4 text-right"><div className="h-6 bg-gray-200 rounded w-16 ml-auto"></div></td>
                      </tr>
                    ))
                  ) : bookings.length === 0 ? (
                    <tr>
                      <td colSpan={9} className="py-12 text-center text-[#6d7a77]">
                        <div className="max-w-xs mx-auto space-y-2">
                          <svg className="w-8 h-8 text-[#bcc9c6] mx-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                          </svg>
                          <p className="font-semibold text-xs text-[#141b2b]">No bookings found</p>
                          <p className="text-[11px]">
                            {hasActiveFilters
                              ? "Try adjusting your search criteria or resetting filters."
                              : "No customer bookings have been placed on the platform yet."}
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
                    bookings.map((b) => {
                      const bStatus = b.booking_status?.toUpperCase() || "CONFIRMED";
                      const bBadge = bookingStatusBadgeStyles[bStatus] || "bg-[#f3f4f6] text-[#4b5563] border border-[#e5e7eb]";

                      const pStatus = b.payment_status?.toUpperCase() || "SUCCESS";
                      const pBadge = paymentStatusBadgeStyles[pStatus] || "bg-[#f3f4f6] text-[#4b5563] border border-[#e5e7eb]";

                      const rStatus = b.refund_status?.toUpperCase();
                      const rBadge = rStatus ? refundStatusBadgeStyles[rStatus] : null;

                      return (
                        <tr key={b.id} className="hover:bg-[#f9f9ff] transition-colors">
                          {/* Booking Number & Date */}
                          <td className="py-3.5 px-4 font-medium">
                            <div className="flex items-center gap-1.5">
                              <span className="font-mono font-bold text-[#00685f]">
                                {b.booking_number}
                              </span>
                              <button
                                type="button"
                                onClick={() => copyToClipboard(b.booking_number, "Booking Reference")}
                                className="text-[#6d7a77] hover:text-[#00685f] transition-colors p-0.5 cursor-pointer"
                                title="Copy reference"
                              >
                                <svg className="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
                                  <path d="M8 3a1 1 0 011-1h2a1 1 0 110 2H9a1 1 0 01-1-1z" />
                                  <path d="M6 3a2 2 0 00-2 2v11a2 2 0 002 2h8a2 2 0 002-2V5a2 2 0 00-2-2 3 3 0 01-3 3H9a3 3 0 01-3-3z" />
                                </svg>
                              </button>
                              {copiedCode === b.booking_number && (
                                <span className="text-[10px] text-[#00685f] font-semibold">Copied!</span>
                              )}
                            </div>
                            <span className="text-[11px] text-[#6d7a77] block mt-0.5">
                              {formatDateTime(b.created_at)}
                            </span>
                          </td>

                          {/* Customer */}
                          <td className="py-3.5 px-4">
                            <span className="font-semibold block text-[#141b2b]">
                              {b.customer?.full_name || "Guest Customer"}
                            </span>
                            <span className="text-[11px] font-mono text-[#6d7a77] block truncate max-w-[170px]" title={b.customer?.email}>
                              {b.customer?.email || "—"}
                            </span>
                          </td>

                          {/* Event */}
                          <td className="py-3.5 px-4 max-w-[190px]">
                            <span className="font-medium text-[#141b2b] block truncate" title={b.event?.title}>
                              {b.event?.title || "Untitled Event"}
                            </span>
                            {b.event?.venue_name && (
                              <span className="text-[11px] text-[#6d7a77] block truncate" title={b.event.venue_name}>
                                📍 {b.event.venue_name}
                              </span>
                            )}
                          </td>

                          {/* Organizer */}
                          <td className="py-3.5 px-4 max-w-[170px]">
                            <span className="font-semibold text-[#3d4947] block truncate" title={b.organizer?.business_name}>
                              {b.organizer?.business_name || "—"}
                            </span>
                            {b.organizer?.email && (
                              <span className="text-[11px] font-mono text-[#6d7a77] block truncate" title={b.organizer.email}>
                                {b.organizer.email}
                              </span>
                            )}
                          </td>

                          {/* Tickets */}
                          <td className="py-3.5 px-3 text-center font-bold text-[#141b2b]">
                            {b.ticket_count ?? 1}
                          </td>

                          {/* Amount */}
                          <td className="py-3.5 px-4 text-right font-mono font-bold text-[#141b2b]">
                            {formatCurrency(b.original_booking_total, b.currency)}
                          </td>

                          {/* Booking Status */}
                          <td className="py-3.5 px-4 text-center">
                            <span className={`inline-block text-[11px] font-bold px-2.5 py-0.5 rounded-full ${bBadge}`}>
                              {bStatus}
                            </span>
                          </td>

                          {/* Payment / Refund Status */}
                          <td className="py-3.5 px-4 text-center">
                            <div className="flex flex-col items-center gap-1">
                              <span className={`inline-block text-[10px] font-bold px-2 py-0.5 rounded-full ${pBadge}`}>
                                {pStatus}
                              </span>
                              {rBadge && (
                                <span className={`inline-block text-[10px] font-bold px-2 py-0.5 rounded-full ${rBadge}`}>
                                  REFUND: {rStatus}
                                </span>
                              )}
                            </div>
                          </td>

                          {/* Actions */}
                          <td className="py-3.5 px-4 text-right">
                            <button
                              type="button"
                              onClick={() => handleViewBookingDetail(b)}
                              className="text-xs font-semibold text-[#00685f] hover:text-[#005049] hover:underline px-2.5 py-1 rounded-md hover:bg-[#00685f]/5 transition-colors cursor-pointer"
                            >
                              View Details
                            </button>
                          </td>
                        </tr>
                      );
                    })
                  )}
                </tbody>
              </table>
            </div>

            {/* Pagination Controls */}
            {!loading && bookings.length > 0 && (
              <div className="bg-[#f9f9ff] border-t border-[#bcc9c6] px-4 py-3 flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-[#3d4947]">
                <div className="flex items-center gap-3">
                  <span>
                    Showing <strong className="text-[#141b2b]">{(page - 1) * limit + 1}</strong> to{" "}
                    <strong className="text-[#141b2b]">
                      {Math.min(page * limit, totalRecords)}
                    </strong>{" "}
                    of <strong className="text-[#141b2b]">{totalRecords}</strong> bookings
                  </span>
                  <div className="flex items-center gap-1.5 ml-2">
                    <span className="text-[11px] text-[#6d7a77]">Rows:</span>
                    <select
                      value={limit}
                      onChange={(e) => {
                        setLimit(Number(e.target.value));
                        setPage(1);
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
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                    disabled={page <= 1}
                    className="p-1.5 rounded-md border border-[#bcc9c6] bg-white text-[#141b2b] hover:bg-[#eef0fb] disabled:opacity-40 disabled:cursor-not-allowed transition-colors cursor-pointer"
                    aria-label="Previous page"
                  >
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
                    </svg>
                  </button>

                  <span className="px-3 py-1 font-semibold text-[#141b2b]">
                    Page {page} of {totalPages}
                  </span>

                  <button
                    type="button"
                    onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                    disabled={page >= totalPages}
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

      {/* Booking Detail Drawer */}
      {activeBookingDetail && (
        <AdminBookingDetailDrawer
          booking={activeBookingDetail}
          onClose={() => setActiveBookingDetail(null)}
        />
      )}
    </div>
  );
}
