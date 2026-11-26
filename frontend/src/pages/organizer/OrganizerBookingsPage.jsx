import { useState, useEffect } from "react";
import { formatCurrency, formatDateTime } from "../../utils/formatters";
import notify from "../../utils/toast";
import OrganizerBookingDetailDrawer from "../../components/organizer/OrganizerBookingDetailDrawer";
import {
  getOrganizerBookings,
  getOrganizerBookingStats,
  getOrganizerBookingDetail,
} from "../../api/organizerBookingApi";
import { getOrganizerEvents } from "../../api/organizerApi";

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

export default function OrganizerBookingsPage() {
  // State for Filters & Search
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
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
  const [organizerEvents, setOrganizerEvents] = useState([]);

  // UI Interactive States
  const [loading, setLoading] = useState(true);
  const [loadingEvents, setLoadingEventsList] = useState(false);
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

  // Load organizer events list for filter dropdown
  useEffect(() => {
    let cancelled = false;

    async function loadEvents() {
      setLoadingEventsList(true);
      try {
        const res = await getOrganizerEvents({ limit: 100 });
        if (cancelled) return;
        const rawList = Array.isArray(res?.data?.events)
          ? res.data.events
          : Array.isArray(res?.data)
          ? res.data
          : Array.isArray(res?.events)
          ? res.events
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
              ...evt,
              id: String(id),
              title: String(title).trim() || "Untitled Event",
            };
          })
          .filter(Boolean);

        setOrganizerEvents(normalizedList);
      } catch (err) {
        if (!cancelled) {
          console.error("Failed to load organizer events for filter dropdown:", err);
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
          event_id: selectedEventId,
          booking_status: bookingStatus,
          payment_status: paymentStatus,
          search: debouncedSearch,
          from: fromDate,
          to: toDate,
        };

        const [bookingsRes, statsRes] = await Promise.all([
          getOrganizerBookings(filterOptions),
          getOrganizerBookingStats(filterOptions),
        ]);

        if (cancelled) return;

        // Process Bookings List & Pagination Envelopes
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

        // Process Stats Envelope
        if (statsRes?.data) {
          setStats(statsRes.data);
        } else if (statsRes && typeof statsRes === "object" && statsRes.total_bookings !== undefined) {
          setStats(statsRes);
        }
      } catch (err) {
        if (cancelled) return;
        console.error("Error fetching organizer bookings:", err);
        const errorMsg =
          err.response?.data?.message ||
          err.response?.data?.error ||
          err.message ||
          "Failed to load bookings. Please check your connection and retry.";
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
      const res = await getOrganizerBookingDetail(bookingSummary.id);
      if (res?.data) {
        setActiveBookingDetail(res.data);
      }
    } catch (err) {
      console.error("Failed to load deep booking details:", err);
      notify.error("Could not load complete booking breakdown");
    }
  };

  const hasActiveFilters = Boolean(
    search || selectedEventId || bookingStatus !== "ALL" || paymentStatus !== "ALL" || fromDate || toDate
  );

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-white w-full">
      {/* Page Header */}
      <div className="px-4 sm:px-6 py-4 border-b border-[#dce2f7] flex flex-col md:flex-row md:items-center justify-between gap-4 shrink-0 bg-white">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-[#141b2b] text-xl font-bold tracking-tight">
              Bookings Management
            </h1>
            <span className="bg-[#e6f4f2] text-[#00685f] text-xs font-semibold px-2.5 py-0.5 rounded-full border border-[#00685f]/20">
              Orders & Admissions
            </span>
          </div>
          <p className="text-[#3d4947] text-xs sm:text-sm mt-0.5">
            View and manage attendee orders, ticket allocations, payment captures, and cancellations.
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
            <span className="text-[11px] mt-1 text-[#6d7a77] font-medium">All booking orders</span>
          </div>

          {/* Card 2: Confirmed Bookings */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-xs flex flex-col justify-between">
            <span className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Confirmed
            </span>
            <span className="text-[#15803d] text-xl lg:text-2xl font-bold tracking-tight">
              {stats.confirmed_bookings ?? 0}
            </span>
            <span className="text-[11px] mt-1 text-[#15803d] font-medium">Active reservations</span>
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
            <span className="text-[11px] mt-1 text-[#6d7a77] font-medium">Valid admissions</span>
          </div>

          {/* Card 5: Gross Captured */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-xs flex flex-col justify-between">
            <span className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Gross Captured
            </span>
            <span className="text-[#141b2b] text-xl lg:text-2xl font-bold tracking-tight">
              {formatCurrency(stats.gross_captured_amount, stats.currency)}
            </span>
            <span className="text-[11px] mt-1 text-[#6d7a77] font-medium">Total gateway captures</span>
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
            {/* Search Input (4 columns on lg) */}
            <div className="lg:col-span-4 relative">
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
                {organizerEvents.map((evt) => {
                  const eventId = evt.id || evt.ID || evt.event_id || "";
                  const eventTitle =
                    evt.title ||
                    evt.Title ||
                    evt.name ||
                    evt.eventName ||
                    "Untitled Event";
                  if (!eventId) return null;
                  return (
                    <option key={eventId} value={eventId}>
                      {eventTitle}
                    </option>
                  );
                })}
              </select>
            </div>

            {/* Booking Status Dropdown (2 columns on lg) */}
            <div className="lg:col-span-2">
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

            {/* Payment Status Dropdown (3 columns on lg) */}
            <div className="lg:col-span-3">
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
          </div>

          {/* Date Filters & Clear Button */}
          <div className="flex flex-wrap items-center justify-between gap-3 pt-2 border-t border-[#bcc9c6]/30">
            <div className="flex flex-wrap items-center gap-3">
              <div className="flex items-center gap-1.5">
                <span className="text-[11px] font-semibold text-[#6d7a77] uppercase">From:</span>
                <input
                  type="date"
                  value={fromDate}
                  onChange={(e) => {
                    setFromDate(e.target.value);
                    setPage(1);
                  }}
                  className="bg-white border border-[#bcc9c6] rounded-lg px-2.5 py-1 text-xs text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40"
                />
              </div>

              <div className="flex items-center gap-1.5">
                <span className="text-[11px] font-semibold text-[#6d7a77] uppercase">To:</span>
                <input
                  type="date"
                  value={toDate}
                  onChange={(e) => {
                    setToDate(e.target.value);
                    setPage(1);
                  }}
                  className="bg-white border border-[#bcc9c6] rounded-lg px-2.5 py-1 text-xs text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40"
                />
              </div>
            </div>

            {hasActiveFilters && (
              <button
                type="button"
                onClick={handleResetFilters}
                className="text-xs text-[#b91c1c] hover:text-[#991b1b] font-semibold flex items-center gap-1 cursor-pointer transition-colors"
              >
                <svg className="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
                </svg>
                <span>Clear All Filters</span>
              </button>
            )}
          </div>
        </div>

        {/* =================================================================== */}
        {/* BOOKINGS TABLE SECTION                                              */}
        {/* =================================================================== */}
        <div className="bg-white border border-[#bcc9c6] rounded-xl shadow-xs overflow-hidden">
          {error ? (
            <div className="p-8 text-center text-xs text-[#b91c1c]">
              <p className="font-bold text-sm">Failed to load bookings</p>
              <p className="mt-1 text-[#6d7a77]">{error}</p>
              <button
                type="button"
                onClick={() => setRefreshKey((k) => k + 1)}
                className="mt-3 px-3 py-1.5 bg-[#fef2f2] border border-[#fecaca] text-[#b91c1c] hover:bg-[#fee2e2] rounded-lg text-xs font-semibold cursor-pointer transition-colors"
              >
                Retry
              </button>
            </div>
          ) : loading ? (
            <div className="p-6 space-y-3">
              {[1, 2, 3, 4, 5].map((i) => (
                <div key={i} className="h-12 bg-gray-100 rounded-lg animate-pulse" />
              ))}
            </div>
          ) : bookings.length === 0 ? (
            <div className="py-12 px-4 text-center">
              <svg className="w-12 h-12 mx-auto text-[#bcc9c6] mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
              </svg>
              <h3 className="text-sm font-bold text-[#141b2b]">No bookings found</h3>
              <p className="text-xs text-[#6d7a77] mt-1 max-w-sm mx-auto">
                No orders match your active filter criteria. Try adjusting your search query, status filters, or date range.
              </p>
              {hasActiveFilters && (
                <button
                  type="button"
                  onClick={handleResetFilters}
                  className="mt-4 px-4 py-2 bg-[#00685f] text-white rounded-lg text-xs font-bold hover:bg-[#005149] transition-colors cursor-pointer"
                >
                  Reset Filters
                </button>
              )}
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs">
                <thead className="bg-[#f9f9ff] border-b border-[#bcc9c6] text-[#6d7a77] text-[11px] uppercase font-semibold">
                  <tr>
                    <th className="py-3 px-4">Booking Ref</th>
                    <th className="py-3 px-4">Customer</th>
                    <th className="py-3 px-4">Event</th>
                    <th className="py-3 px-3 text-center">Tickets</th>
                    <th className="py-3 px-4">Date</th>
                    <th className="py-3 px-4 text-right">Amount</th>
                    <th className="py-3 px-4 text-center">Payment / Refund</th>
                    <th className="py-3 px-3 text-center">Status</th>
                    <th className="py-3 px-4 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#bcc9c6]/30">
                  {bookings.map((b) => {
                    const bStatus = b.booking_status?.toUpperCase() || "CONFIRMED";
                    const bBadge = bookingStatusBadgeStyles[bStatus] || "bg-[#f3f4f6] text-[#4b5563]";

                    const pStatus = b.payment_status?.toUpperCase() || "SUCCESS";
                    const pBadge = paymentStatusBadgeStyles[pStatus] || "bg-[#f3f4f6] text-[#4b5563]";

                    const rStatus = b.refund_status?.toUpperCase();
                    const rBadge = rStatus ? refundStatusBadgeStyles[rStatus] : null;

                    return (
                      <tr key={b.id} className="hover:bg-[#f9f9ff]/70 transition-colors">
                        {/* Reference */}
                        <td className="py-3 px-4">
                          <div className="flex items-center gap-1.5 font-mono font-bold text-[#00685f]">
                            <span>{b.booking_number}</span>
                            <button
                              type="button"
                              onClick={() => copyToClipboard(b.booking_number, "Reference")}
                              className="text-[#6d7a77] hover:text-[#00685f] p-0.5 cursor-pointer"
                              title="Copy"
                            >
                              <svg className="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
                                <path d="M8 3a1 1 0 011-1h2a1 1 0 110 2H9a1 1 0 01-1-1z" />
                                <path d="M6 3a2 2 0 00-2 2v11a2 2 0 002 2h8a2 2 0 002-2V5a2 2 0 00-2-2 3 3 0 01-3 3H9a3 3 0 01-3-3z" />
                              </svg>
                            </button>
                            {copiedCode === b.booking_number && (
                              <span className="text-[10px] text-[#00685f] font-sans font-semibold">✓</span>
                            )}
                          </div>
                        </td>

                        {/* Customer */}
                        <td className="py-3 px-4">
                          <div className="font-bold text-[#141b2b]">{b.customer?.full_name || "—"}</div>
                          <div className="text-[11px] text-[#6d7a77]">{b.customer?.email || "—"}</div>
                        </td>

                        {/* Event */}
                        <td className="py-3 px-4 max-w-[200px]">
                          <div className="font-semibold text-[#141b2b] truncate" title={b.event?.title}>
                            {b.event?.title || "—"}
                          </div>
                          <div className="text-[11px] text-[#6d7a77] truncate">
                            {b.event?.venue_name || "Venue TBD"}
                          </div>
                        </td>

                        {/* Tickets */}
                        <td className="py-3 px-3 text-center">
                          <span className="inline-block bg-[#eef0fb] text-[#141b2b] px-2 py-0.5 rounded font-bold text-[11px]">
                            {b.ticket_count ?? 1}
                          </span>
                        </td>

                        {/* Date */}
                        <td className="py-3 px-4 text-[#3d4947] whitespace-nowrap">
                          {formatDateTime(b.created_at)}
                        </td>

                        {/* Amount */}
                        <td className="py-3 px-4 text-right font-bold text-[#141b2b] whitespace-nowrap">
                          {formatCurrency(b.original_booking_total, b.currency)}
                        </td>

                        {/* Payment & Refund Status Badges */}
                        <td className="py-3 px-4 text-center">
                          <div className="flex flex-col items-center gap-1">
                            <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${pBadge}`}>
                              {pStatus}
                            </span>
                            {rBadge && (
                              <span className={`text-[9px] font-bold px-1.5 py-0.2 rounded-full ${rBadge}`}>
                                REFUND: {rStatus}
                              </span>
                            )}
                          </div>
                        </td>

                        {/* Booking Status */}
                        <td className="py-3 px-3 text-center">
                          <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${bBadge}`}>
                            {bStatus}
                          </span>
                        </td>

                        {/* Action Drawer Button */}
                        <td className="py-3 px-4 text-right">
                          <button
                            type="button"
                            onClick={() => handleViewBookingDetail(b)}
                            className="inline-flex items-center gap-1 px-2.5 py-1.5 text-xs font-bold text-[#00685f] hover:text-white bg-[#e6f4f2] hover:bg-[#00685f] rounded-lg transition-colors cursor-pointer"
                          >
                            <svg className="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
                              <path d="M10 12a2 2 0 100-4 2 2 0 000 4z" />
                              <path fillRule="evenodd" d="M.458 10C1.732 5.943 5.522 3 10 3s8.268 2.943 9.542 7c-1.274 4.057-5.064 7-9.542 7S1.732 14.057.458 10zM14 10a4 4 0 11-8 0 4 4 0 018 0z" clipRule="evenodd" />
                            </svg>
                            <span>Details</span>
                          </button>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}

          {/* Table Footer with Pagination */}
          {!loading && !error && bookings.length > 0 && (
            <div className="px-4 py-3 border-t border-[#bcc9c6] bg-[#f9f9ff] flex flex-col sm:flex-row items-center justify-between gap-3 text-xs">
              <div className="text-[#6d7a77]">
                Showing <span className="font-bold text-[#141b2b]">{(page - 1) * limit + 1}</span> to{" "}
                <span className="font-bold text-[#141b2b]">
                  {Math.min(page * limit, totalRecords)}
                </span>{" "}
                of <span className="font-bold text-[#141b2b]">{totalRecords}</span> bookings
              </div>

              <div className="flex items-center gap-2">
                <div className="flex items-center gap-1.5 mr-2">
                  <span className="text-[#6d7a77]">Rows:</span>
                  <select
                    value={limit}
                    onChange={(e) => {
                      setLimit(Number(e.target.value));
                      setPage(1);
                    }}
                    className="bg-white border border-[#bcc9c6] rounded px-2 py-1 text-xs text-[#141b2b] cursor-pointer"
                  >
                    {PAGE_SIZE_OPTIONS.map((size) => (
                      <option key={size} value={size}>
                        {size}
                      </option>
                    ))}
                  </select>
                </div>

                <button
                  type="button"
                  disabled={page <= 1}
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  className="px-2.5 py-1 border border-[#bcc9c6] rounded bg-white font-medium text-[#141b2b] disabled:opacity-40 disabled:cursor-not-allowed hover:bg-gray-50 cursor-pointer"
                >
                  Prev
                </button>
                <span className="px-2 text-[#3d4947] font-semibold">
                  {page} / {totalPages}
                </span>
                <button
                  type="button"
                  disabled={page >= totalPages}
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  className="px-2.5 py-1 border border-[#bcc9c6] rounded bg-white font-medium text-[#141b2b] disabled:opacity-40 disabled:cursor-not-allowed hover:bg-gray-50 cursor-pointer"
                >
                  Next
                </button>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Booking Detail Slide-in Drawer */}
      <OrganizerBookingDetailDrawer
        booking={activeBookingDetail}
        onClose={() => setActiveBookingDetail(null)}
      />
    </div>
  );
}
