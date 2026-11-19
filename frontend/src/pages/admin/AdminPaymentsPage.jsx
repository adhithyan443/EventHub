import { useState, useEffect, useCallback } from "react";
import { getAdminPayments, getAdminEventPaymentSummaries } from "../../api/adminPaymentApi";
import PaymentDetailDrawer from "../../components/admin/PaymentDetailDrawer";
import notify from "../../utils/toast";

const PAYMENT_STATUS_OPTIONS = [
  { value: "ALL", label: "All Payment Statuses" },
  { value: "SUCCESS", label: "Success" },
  { value: "PENDING", label: "Pending" },
  { value: "FAILED", label: "Failed" },
  { value: "REFUND_REQUIRED", label: "Refund Required" },
  { value: "REFUNDED", label: "Refunded" },
];

const ALLOCATION_STATUS_OPTIONS = [
  { value: "ALL", label: "All Ledger Allocations" },
  { value: "ACCOUNTED", label: "Accounted" },
  { value: "UNALLOCATED", label: "Unallocated" },
  { value: "NOT_APPLICABLE", label: "Not Applicable" },
];

const PAGE_SIZE_OPTIONS = [10, 25, 50, 100];

const paymentStatusBadgeStyles = {
  SUCCESS: "bg-[#dcfce7] text-[#15803d] border border-[#bbf7d0]",
  PENDING: "bg-[#fef9c3] text-[#854d0e] border border-[#fef08a]",
  FAILED: "bg-[#fee2e2] text-[#b91c1c] border border-[#fecaca]",
  REFUND_REQUIRED: "bg-[#ffedd5] text-[#c2410c] border border-[#fed7aa]",
  REFUNDED: "bg-[#f3e8ff] text-[#7e22ce] border border-[#e9d5ff]",
};

const allocationBadgeStyles = {
  ACCOUNTED: "bg-[#ccfbf1] text-[#0f766e] border border-[#99f6e4]",
  UNALLOCATED: "bg-[#fef3c7] text-[#b45309] border border-[#fde68a]",
  NOT_APPLICABLE: "bg-[#f3f4f6] text-[#4b5563] border border-[#e5e7eb]",
};

const formatCurrency = (amount, currency = "INR") => {
  if (amount === null || amount === undefined) return "—";
  const num = typeof amount === "number" ? amount : parseFloat(amount);
  if (isNaN(num)) return "—";
  return new Intl.NumberFormat("en-IN", {
    style: "currency",
    currency: currency || "INR",
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(num);
};

const formatDateTime = (dateStr) => {
  if (!dateStr) return "—";
  const d = new Date(dateStr);
  if (isNaN(d.getTime())) return "—";
  return d.toLocaleDateString("en-IN", {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
};

export default function AdminPaymentsPage() {
  // Top-level View Mode: 'BY_EVENT' (default) or 'ALL_PAYMENTS'
  const [viewMode, setViewMode] = useState("BY_EVENT");
  const [selectedEventFilter, setSelectedEventFilter] = useState(null); // { id: string, title: string } | null
  const [copiedId, setCopiedId] = useState(null);
  const [activeDrawerPayment, setActiveDrawerPayment] = useState(null);

  // ---------------------------------------------------------------------------
  // View 1: By Event State
  // ---------------------------------------------------------------------------
  const [eventSummaries, setEventSummaries] = useState([]);
  const [overallSummary, setOverallSummary] = useState({
    total_events: 0,
    gross_captured_amount: "0.00",
    currency: "INR",
    refund_required_count: 0,
    unallocated_captured_count: 0,
  });
  const [eventTotalRecords, setEventTotalRecords] = useState(0);
  const [eventTotalPages, setEventTotalPages] = useState(1);
  const [eventPage, setEventPage] = useState(1);
  const [eventLimit, setEventLimit] = useState(10);
  const [eventSearch, setEventSearch] = useState("");
  const [debouncedEventSearch, setDebouncedEventSearch] = useState("");
  const [eventFromDate, setEventFromDate] = useState("");
  const [eventToDate, setEventToDate] = useState("");
  const [eventLoading, setEventLoading] = useState(true);
  const [eventError, setEventError] = useState("");
  const [eventRefreshKey, setEventRefreshKey] = useState(0);

  // Debounce event search input
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedEventSearch(eventSearch);
      setEventPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [eventSearch]);

  // ---------------------------------------------------------------------------
  // View 2: All Payments State
  // ---------------------------------------------------------------------------
  const [payments, setPayments] = useState([]);
  const [totalRecords, setTotalRecords] = useState(0);
  const [totalPages, setTotalPages] = useState(1);
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useState(10);
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [status, setStatus] = useState("ALL");
  const [allocationStatus, setAllocationStatus] = useState("ALL");
  const [fromDate, setFromDate] = useState("");
  const [toDate, setToDate] = useState("");
  const [paymentsLoading, setPaymentsLoading] = useState(true);
  const [paymentsError, setPaymentsError] = useState("");
  const [paymentsRefreshKey, setPaymentsRefreshKey] = useState(0);

  // Debounce all payments search input
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  // ---------------------------------------------------------------------------
  // Data Fetching: By Event
  // ---------------------------------------------------------------------------
  const fetchEventSummaries = useCallback(async (cancelledRef) => {
    try {
      setEventLoading(true);
      setEventError("");

      const res = await getAdminEventPaymentSummaries({
        page: eventPage,
        limit: eventLimit,
        search: debouncedEventSearch,
        from: eventFromDate,
        to: eventToDate,
      });

      if (cancelledRef.cancelled) return;

      if (res?.data) {
        const list = Array.isArray(res.data.summaries)
          ? res.data.summaries
          : Array.isArray(res.data.events)
          ? res.data.events
          : [];
        setEventSummaries(list);

        if (res.data.overall_summary) {
          setOverallSummary({
            total_events: res.data.overall_summary.total_events ?? 0,
            gross_captured_amount: res.data.overall_summary.gross_captured_amount ?? "0.00",
            currency: res.data.overall_summary.currency || "INR",
            refund_required_count: res.data.overall_summary.refund_required_count ?? 0,
            unallocated_captured_count: res.data.overall_summary.unallocated_captured_count ?? 0,
          });
        }

        setEventTotalRecords(
          typeof res.data.total_records === "number" ? res.data.total_records : 0
        );
        setEventTotalPages(
          typeof res.data.total_pages === "number" ? Math.max(1, res.data.total_pages) : 1
        );
      } else {
        setEventSummaries([]);
        setEventTotalRecords(0);
        setEventTotalPages(1);
      }
    } catch (err) {
      if (cancelledRef.cancelled) return;
      console.error("Failed to fetch event payment summaries:", err);
      const msg =
        err.response?.data?.message || err.message || "Failed to load event payment summaries.";
      setEventError(msg);
      setEventSummaries([]);
      setEventTotalRecords(0);
      setEventTotalPages(1);
    } finally {
      if (!cancelledRef.cancelled) {
        setEventLoading(false);
      }
    }
  }, [eventPage, eventLimit, debouncedEventSearch, eventFromDate, eventToDate]);

  useEffect(() => {
    if (viewMode !== "BY_EVENT") return;
    const cancelledRef = { cancelled: false };
    fetchEventSummaries(cancelledRef);
    return () => {
      cancelledRef.cancelled = true;
    };
  }, [fetchEventSummaries, eventRefreshKey, viewMode]);

  // ---------------------------------------------------------------------------
  // Data Fetching: All Payments
  // ---------------------------------------------------------------------------
  const fetchPayments = useCallback(async (cancelledRef) => {
    try {
      setPaymentsLoading(true);
      setPaymentsError("");

      const res = await getAdminPayments({
        page,
        limit,
        status: status === "ALL" ? "" : status,
        allocationStatus: allocationStatus === "ALL" ? "" : allocationStatus,
        from: fromDate,
        to: toDate,
        search: debouncedSearch,
        eventId: selectedEventFilter ? selectedEventFilter.id : "",
      });

      if (cancelledRef.cancelled) return;

      if (res?.data) {
        const list = Array.isArray(res.data.payments) ? res.data.payments : [];
        setPayments(list);
        setTotalRecords(
          typeof res.data.total_records === "number" ? res.data.total_records : 0
        );
        setTotalPages(
          typeof res.data.total_pages === "number" ? Math.max(1, res.data.total_pages) : 1
        );
      } else {
        setPayments([]);
        setTotalRecords(0);
        setTotalPages(1);
      }
    } catch (err) {
      if (cancelledRef.cancelled) return;
      console.error("Failed to fetch payments:", err);
      const msg = err.response?.data?.message || err.message || "Failed to load payments.";
      setPaymentsError(msg);
      setPayments([]);
      setTotalRecords(0);
      setTotalPages(1);
    } finally {
      if (!cancelledRef.cancelled) {
        setPaymentsLoading(false);
      }
    }
  }, [page, limit, status, allocationStatus, fromDate, toDate, debouncedSearch, selectedEventFilter]);

  useEffect(() => {
    if (viewMode !== "ALL_PAYMENTS") return;
    const cancelledRef = { cancelled: false };
    fetchPayments(cancelledRef);
    return () => {
      cancelledRef.cancelled = true;
    };
  }, [fetchPayments, paymentsRefreshKey, viewMode]);

  // ---------------------------------------------------------------------------
  // Action Handlers
  // ---------------------------------------------------------------------------
  const copyText = async (text, label) => {
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
      setCopiedId(text);
      notify.info(`Copied ${label} to clipboard`);
      setTimeout(() => setCopiedId(null), 2000);
    } catch {
      notify.error("Failed to copy");
    }
  };

  // Drill down from Event Summary row to All Payments view
  const handleDrillDownToEvent = (event) => {
    setSelectedEventFilter({
      id: event.event_id,
      title: event.event_title || "Selected Event",
    });
    setPage(1);
    setViewMode("ALL_PAYMENTS");
  };

  // Clear event-specific filter while staying in All Payments
  const handleClearEventFilter = () => {
    setSelectedEventFilter(null);
    setPage(1);
  };

  // Clear all filters in By Event view
  const handleClearEventFilters = () => {
    setEventSearch("");
    setDebouncedEventSearch("");
    setEventFromDate("");
    setEventToDate("");
    setEventPage(1);
  };

  // Clear all filters in All Payments view
  const handleClearPaymentsFilters = () => {
    setSearch("");
    setDebouncedSearch("");
    setStatus("ALL");
    setAllocationStatus("ALL");
    setFromDate("");
    setToDate("");
    setSelectedEventFilter(null);
    setPage(1);
  };

  const hasActiveEventFilters =
    Boolean(eventSearch.trim()) || Boolean(eventFromDate) || Boolean(eventToDate);

  const hasActivePaymentFilters =
    Boolean(search.trim()) ||
    status !== "ALL" ||
    allocationStatus !== "ALL" ||
    Boolean(fromDate) ||
    Boolean(toDate) ||
    Boolean(selectedEventFilter);

  // Pagination bounds
  const eventStartRecord = eventTotalRecords === 0 ? 0 : (eventPage - 1) * eventLimit + 1;
  const eventEndRecord = Math.min(eventPage * eventLimit, eventTotalRecords);

  const paymentStartRecord = totalRecords === 0 ? 0 : (page - 1) * limit + 1;
  const paymentEndRecord = Math.min(page * limit, totalRecords);

  const isCurrentViewLoading = viewMode === "BY_EVENT" ? eventLoading : paymentsLoading;

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-white w-full">
      {/* Top Header Bar */}
      <div className="px-4 sm:px-6 py-4 border-b border-[#dce2f7] flex flex-col md:flex-row md:items-center justify-between gap-4 shrink-0 bg-white">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-[#141b2b] text-xl font-bold tracking-tight">
              Payments & Refunds
            </h1>
            <span className="bg-[#f1f3ff] text-[#00685f] text-xs font-semibold px-2.5 py-0.5 rounded-full border border-[#dce2f7]">
              Finance Ledger
            </span>
          </div>
          <p className="text-[#3d4947] text-xs sm:text-sm mt-0.5">
            Monitor incoming customer payments, gateway capture states, and financial ledger allocations.
          </p>
        </div>

        {/* View Switcher Tabs & Refresh */}
        <div className="flex items-center gap-3 self-start md:self-auto shrink-0">
          <div className="inline-flex rounded-lg border border-[#bcc9c6] p-0.5 bg-[#f9f9ff]">
            <button
              type="button"
              onClick={() => setViewMode("BY_EVENT")}
              className={`px-3 py-1.5 text-xs font-bold rounded-md transition-all cursor-pointer ${
                viewMode === "BY_EVENT"
                  ? "bg-[#00685f] text-white shadow-xs"
                  : "text-[#3d4947] hover:text-[#141b2b] hover:bg-[#eef0fb]"
              }`}
            >
              By Event
            </button>
            <button
              type="button"
              onClick={() => setViewMode("ALL_PAYMENTS")}
              className={`px-3 py-1.5 text-xs font-bold rounded-md transition-all cursor-pointer ${
                viewMode === "ALL_PAYMENTS"
                  ? "bg-[#00685f] text-white shadow-xs"
                  : "text-[#3d4947] hover:text-[#141b2b] hover:bg-[#eef0fb]"
              }`}
            >
              All Payments
            </button>
          </div>

          <button
            type="button"
            onClick={() => {
              if (viewMode === "BY_EVENT") {
                setEventRefreshKey((k) => k + 1);
              } else {
                setPaymentsRefreshKey((k) => k + 1);
              }
            }}
            disabled={isCurrentViewLoading}
            className="bg-[#f9f9ff] border border-[#bcc9c6] text-[#141b2b] text-xs sm:text-sm font-semibold px-3 py-1.5 rounded-lg flex items-center gap-1.5 hover:bg-[#eef0fb] transition-colors cursor-pointer disabled:opacity-50"
            title="Refresh current view"
          >
            <svg
              className={`w-3.5 h-3.5 ${isCurrentViewLoading ? "animate-spin text-[#00685f]" : ""}`}
              viewBox="0 0 20 20"
              fill="currentColor"
            >
              <path
                fillRule="evenodd"
                d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z"
                clipRule="evenodd"
              />
            </svg>
            <span>Refresh</span>
          </button>
        </div>
      </div>

      {/* Main Content Area */}
      <div className="flex-1 overflow-y-auto p-4 sm:p-6 space-y-5">
        {/* =================================================================== */}
        {/* VIEW 1: BY EVENT (DEFAULT)                                          */}
        {/* =================================================================== */}
        {viewMode === "BY_EVENT" && (
          <>
            {/* Error Alert */}
            {eventError && (
              <div className="flex items-center justify-between rounded-lg border border-[#ffdad6] bg-[#fff5f4] px-4 py-3">
                <div className="flex items-center gap-2">
                  <svg className="w-5 h-5 text-[#93000a] shrink-0" viewBox="0 0 20 20" fill="currentColor">
                    <path
                      fillRule="evenodd"
                      d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                      clipRule="evenodd"
                    />
                  </svg>
                  <span className="text-[#93000a] text-sm font-medium">{eventError}</span>
                </div>
                <div className="flex items-center gap-3">
                  <button
                    type="button"
                    onClick={() => setEventRefreshKey((k) => k + 1)}
                    className="text-[#93000a] text-xs font-bold hover:underline cursor-pointer"
                  >
                    Retry
                  </button>
                  <button
                    type="button"
                    onClick={() => setEventError("")}
                    className="text-[#6d7a77] text-xs font-semibold hover:text-[#141b2b] cursor-pointer"
                  >
                    Dismiss
                  </button>
                </div>
              </div>
            )}

            {/* Overall Financial Summary Cards (Strictly Populated from overall_summary) */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
              {/* Card 1: Total Events */}
              <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
                <div className="text-[#3d4947] text-xs font-medium uppercase tracking-wide mb-1">
                  Total Events
                </div>
                <div className="text-[#141b2b] text-2xl font-bold tracking-tight">
                  {overallSummary.total_events?.toLocaleString("en-IN") ?? 0}
                </div>
                <div className="text-xs mt-1 font-medium text-[#00685f]">
                  Events with payment activity
                </div>
              </div>

              {/* Card 2: Gross Captured Amount */}
              <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
                <div className="text-[#3d4947] text-xs font-medium uppercase tracking-wide mb-1">
                  Gross Captured Amount
                </div>
                <div className="text-[#141b2b] text-2xl font-bold tracking-tight">
                  {formatCurrency(overallSummary.gross_captured_amount, overallSummary.currency)}
                </div>
                <div className="text-xs mt-1 font-medium text-[#00685f]">
                  Across all filtered events
                </div>
              </div>

              {/* Card 3: Refund Required */}
              <div
                className={`rounded-xl p-4 border shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)] ${
                  overallSummary.refund_required_count > 0
                    ? "bg-[#fff7ed] border-[#fed7aa]"
                    : "bg-[#f9f9ff] border-[#bcc9c6]"
                }`}
              >
                <div className="flex items-center justify-between mb-1">
                  <div className="text-[#3d4947] text-xs font-medium uppercase tracking-wide">
                    Refund Required
                  </div>
                  {overallSummary.refund_required_count > 0 && (
                    <span className="bg-[#ffedd5] text-[#c2410c] text-[10px] font-bold px-1.5 py-0.5 rounded-full border border-[#fed7aa]">
                      Action Needed
                    </span>
                  )}
                </div>
                <div
                  className={`text-2xl font-bold tracking-tight ${
                    overallSummary.refund_required_count > 0 ? "text-[#c2410c]" : "text-[#141b2b]"
                  }`}
                >
                  {overallSummary.refund_required_count?.toLocaleString("en-IN") ?? 0}
                </div>
                <div className="text-xs mt-1 font-medium text-[#6d7a77]">
                  Payments awaiting refund
                </div>
              </div>

              {/* Card 4: Unallocated Captured */}
              <div
                className={`rounded-xl p-4 border shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)] ${
                  overallSummary.unallocated_captured_count > 0
                    ? "bg-[#fffbeb] border-[#fde68a]"
                    : "bg-[#f9f9ff] border-[#bcc9c6]"
                }`}
              >
                <div className="flex items-center justify-between mb-1">
                  <div className="text-[#3d4947] text-xs font-medium uppercase tracking-wide">
                    Unallocated Captured
                  </div>
                  {overallSummary.unallocated_captured_count > 0 && (
                    <span className="bg-[#fef3c7] text-[#b45309] text-[10px] font-bold px-1.5 py-0.5 rounded-full border border-[#fde68a]">
                      Unreconciled
                    </span>
                  )}
                </div>
                <div
                  className={`text-2xl font-bold tracking-tight ${
                    overallSummary.unallocated_captured_count > 0
                      ? "text-[#b45309]"
                      : "text-[#141b2b]"
                  }`}
                >
                  {overallSummary.unallocated_captured_count?.toLocaleString("en-IN") ?? 0}
                </div>
                <div className="text-xs mt-1 font-medium text-[#6d7a77]">
                  Captured without active booking
                </div>
              </div>
            </div>

            {/* Filter Bar: By Event */}
            <div className="bg-white border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)] space-y-3">
              <div className="flex flex-col md:flex-row items-stretch md:items-center justify-between gap-3">
                {/* Search Input */}
                <div className="relative flex-1">
                  <svg
                    className="absolute left-3 top-1/2 -translate-y-1/2"
                    width="14"
                    height="14"
                    viewBox="0 0 13 13"
                    fill="none"
                  >
                    <path
                      d="M8.5 7.5H7.71L7.43 7.23C8.41 6.09 9 4.61 9 3C9 1.34 7.66 0 6 0C4.34 0 3 1.34 3 3C3 4.66 4.34 6 6 6C6.73 6 7.4 5.75 7.93 5.32L8.2 5.6V6.35L11.25 9.39L12.24 8.4L9.2 5.35H8.5V7.5ZM6 5.5C4.62 5.5 3.5 4.38 3.5 3C3.5 1.62 4.62 0.5 6 0.5C7.38 0.5 8.5 1.62 8.5 3C8.5 4.38 7.38 5.5 6 5.5Z"
                      fill="#6D7A77"
                    />
                  </svg>
                  <input
                    type="text"
                    placeholder="Search events by title or organizer name..."
                    value={eventSearch}
                    onChange={(e) => setEventSearch(e.target.value)}
                    className="w-full bg-white border border-[#bcc9c6] rounded-lg pl-9 pr-8 py-2 text-xs sm:text-sm text-[#141b2b] placeholder:text-[#6d7a77] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40"
                  />
                  {eventSearch && (
                    <button
                      type="button"
                      onClick={() => setEventSearch("")}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-[#6d7a77] hover:text-[#141b2b] cursor-pointer text-xs"
                    >
                      ✕
                    </button>
                  )}
                </div>

                {/* Date Filters */}
                <div className="flex items-center gap-2">
                  <div className="flex items-center gap-1.5">
                    <span className="text-[11px] font-semibold text-[#6d7a77] uppercase">From:</span>
                    <input
                      type="date"
                      value={eventFromDate}
                      onChange={(e) => {
                        setEventFromDate(e.target.value);
                        setEventPage(1);
                      }}
                      className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-2.5 py-1.5 text-xs text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40"
                    />
                  </div>
                  <div className="flex items-center gap-1.5">
                    <span className="text-[11px] font-semibold text-[#6d7a77] uppercase">To:</span>
                    <input
                      type="date"
                      value={eventToDate}
                      onChange={(e) => {
                        setEventToDate(e.target.value);
                        setEventPage(1);
                      }}
                      className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-2.5 py-1.5 text-xs text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40"
                    />
                  </div>

                  {hasActiveEventFilters && (
                    <button
                      type="button"
                      onClick={handleClearEventFilters}
                      className="text-xs text-[#00685f] hover:text-[#00524b] font-bold hover:underline cursor-pointer ml-1 whitespace-nowrap"
                    >
                      Clear filters
                    </button>
                  )}
                </div>
              </div>
            </div>

            {/* Event Payment Summary Table */}
            <div className="bg-white border border-[#bcc9c6] rounded-xl overflow-hidden shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
              <div className="px-4 py-3 border-b border-[#dce2f7] flex items-center justify-between bg-[#f1f3ff]">
                <div className="text-xs text-[#141b2b] font-bold uppercase tracking-wider">
                  Event-Wise Payment Aggregations
                </div>
                <div className="text-xs text-[#6d7a77] font-medium">
                  Showing {eventTotalRecords === 0 ? 0 : eventStartRecord}–{eventEndRecord} of{" "}
                  {eventTotalRecords} events
                </div>
              </div>

              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs min-w-[1100px]">
                  <thead className="bg-[#f9f9ff] text-[#6d7a77] uppercase tracking-wide border-b border-[#bcc9c6]">
                    <tr>
                      <th className="px-4 py-3 font-semibold">Event Name</th>
                      <th className="px-4 py-3 font-semibold">Organizer</th>
                      <th className="px-4 py-3 font-semibold text-center">Attempts</th>
                      <th className="px-4 py-3 font-semibold text-center">Success</th>
                      <th className="px-4 py-3 font-semibold text-right">Gross Captured</th>
                      <th className="px-4 py-3 font-semibold text-center">Pending</th>
                      <th className="px-4 py-3 font-semibold text-center">Failed</th>
                      <th className="px-4 py-3 font-semibold text-center">Refund Req.</th>
                      <th className="px-4 py-3 font-semibold text-center">Refunded</th>
                      <th className="px-4 py-3 font-semibold text-center">Unallocated</th>
                      <th className="px-4 py-3 font-semibold text-right">Action</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-[rgba(188,201,198,0.25)]">
                    {eventLoading ? (
                      <tr>
                        <td colSpan={11} className="px-4 py-12 text-center text-[#6d7a77] text-sm">
                          <div className="flex flex-col items-center justify-center gap-2">
                            <div className="w-6 h-6 border-2 border-[#00685f]/30 border-t-[#00685f] rounded-full animate-spin" />
                            <span className="font-medium text-xs text-[#3d4947]">
                              Loading event payment summaries...
                            </span>
                          </div>
                        </td>
                      </tr>
                    ) : eventSummaries.length === 0 ? (
                      <tr>
                        <td colSpan={11} className="px-4 py-12 text-center text-[#6d7a77] text-sm">
                          <div className="flex flex-col items-center justify-center gap-2 max-w-sm mx-auto">
                            <svg className="w-10 h-10 text-[#bcc9c6]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                strokeWidth="1.5"
                                d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                              />
                            </svg>
                            <div className="font-semibold text-[#141b2b]">No events found</div>
                            <p className="text-xs text-[#6d7a77]">
                              {hasActiveEventFilters
                                ? "No event payment records match your current filter and search criteria."
                                : "There are currently no recorded event payments in the system."}
                            </p>
                            {hasActiveEventFilters && (
                              <button
                                type="button"
                                onClick={handleClearEventFilters}
                                className="mt-2 text-xs font-bold text-[#00685f] bg-[#f0fdfa] border border-[#99f6e4] px-3 py-1.5 rounded-lg hover:bg-[#ccfbf1] transition-colors cursor-pointer"
                              >
                                Reset all filters
                              </button>
                            )}
                          </div>
                        </td>
                      </tr>
                    ) : (
                      eventSummaries.map((item, idx) => {
                        const isEven = idx % 2 === 0;
                        const hasRefundReq = Number(item.refund_required_count) > 0;
                        const hasUnallocated = Number(item.unallocated_captured_count) > 0;

                        return (
                          <tr
                            key={item.event_id || idx}
                            className={`hover:bg-[#f1f3ff] transition-colors ${
                              isEven ? "bg-white" : "bg-[#fafbff]"
                            }`}
                          >
                            {/* Event Title & ID */}
                            <td className="px-4 py-3 max-w-[220px]">
                              <div className="font-bold text-[#141b2b] truncate" title={item.event_title}>
                                {item.event_title || "Untitled Event"}
                              </div>
                              <div className="flex items-center gap-1.5 text-[10px] text-[#6d7a77] font-mono mt-0.5">
                                <span>{item.event_id ? `${item.event_id.substring(0, 8)}...` : "—"}</span>
                                {item.event_id && (
                                  <button
                                    type="button"
                                    onClick={() => copyText(item.event_id, "Event ID")}
                                    className="hover:text-[#00685f] cursor-pointer"
                                    title="Copy Event ID"
                                  >
                                    {copiedId === item.event_id ? "✓" : "📋"}
                                  </button>
                                )}
                              </div>
                            </td>

                            {/* Organizer */}
                            <td className="px-4 py-3 max-w-[170px]">
                              <div
                                className="text-[#3d4947] font-medium truncate"
                                title={item.organizer_business_name}
                              >
                                {item.organizer_business_name || "—"}
                              </div>
                            </td>

                            {/* Total Attempts */}
                            <td className="px-4 py-3 text-center font-semibold text-[#141b2b]">
                              {Number(item.total_payments_count || 0).toLocaleString("en-IN")}
                            </td>

                            {/* Successful */}
                            <td className="px-4 py-3 text-center">
                              <span className="font-semibold text-[#15803d]">
                                {Number(
                                  item.successfully_completed_payments_count || 0
                                ).toLocaleString("en-IN")}
                              </span>
                            </td>

                            {/* Gross Captured Amount */}
                            <td className="px-4 py-3 text-right whitespace-nowrap">
                              <span className="font-bold text-[#141b2b]">
                                {formatCurrency(item.gross_captured_amount, item.currency)}
                              </span>
                            </td>

                            {/* Pending */}
                            <td className="px-4 py-3 text-center text-[#854d0e]">
                              {Number(item.pending_payments_count || 0).toLocaleString("en-IN")}
                            </td>

                            {/* Failed */}
                            <td className="px-4 py-3 text-center text-[#b91c1c]">
                              {Number(item.failed_payments_count || 0).toLocaleString("en-IN")}
                            </td>

                            {/* Refund Required */}
                            <td className="px-4 py-3 text-center">
                              {hasRefundReq ? (
                                <span className="bg-[#ffedd5] text-[#c2410c] border border-[#fed7aa] px-2 py-0.5 rounded-full font-bold text-[11px]">
                                  {Number(item.refund_required_count).toLocaleString("en-IN")}
                                </span>
                              ) : (
                                <span className="text-[#6d7a77]">0</span>
                              )}
                            </td>

                            {/* Payments with Successful Refunds */}
                            <td className="px-4 py-3 text-center">
                              <span className="text-[#7e22ce] font-medium">
                                {Number(
                                  item.payments_with_successful_refunds_count || 0
                                ).toLocaleString("en-IN")}
                              </span>
                            </td>

                            {/* Unallocated Captures */}
                            <td className="px-4 py-3 text-center">
                              {hasUnallocated ? (
                                <span className="bg-[#fef3c7] text-[#b45309] border border-[#fde68a] px-2 py-0.5 rounded-full font-bold text-[11px]">
                                  {Number(item.unallocated_captured_count).toLocaleString("en-IN")}
                                </span>
                              ) : (
                                <span className="text-[#6d7a77]">0</span>
                              )}
                            </td>

                            {/* Action: View Payments */}
                            <td className="px-4 py-3 text-right whitespace-nowrap">
                              <button
                                type="button"
                                onClick={() => handleDrillDownToEvent(item)}
                                className="text-xs text-[#00685f] font-bold hover:underline cursor-pointer bg-[#f1f3ff] hover:bg-[#e1e8fd] px-3 py-1.5 rounded-lg transition-colors border border-[#dce2f7]"
                                title={`View all payments for ${item.event_title}`}
                              >
                                View Payments →
                              </button>
                            </td>
                          </tr>
                        );
                      })
                    )}
                  </tbody>
                </table>
              </div>

              {/* By Event Pagination Footer */}
              <div className="flex flex-col sm:flex-row items-center justify-between px-4 py-3 border-t border-[rgba(188,201,198,0.3)] bg-white gap-3">
                <div className="flex items-center gap-3">
                  <span className="text-[#3d4947] text-xs">
                    Showing {eventTotalRecords === 0 ? 0 : eventStartRecord} to {eventEndRecord} of{" "}
                    {eventTotalRecords} events
                  </span>

                  <div className="flex items-center gap-1.5 text-xs text-[#6d7a77]">
                    <span>Per page:</span>
                    <select
                      value={eventLimit}
                      onChange={(e) => {
                        setEventLimit(Number(e.target.value));
                        setEventPage(1);
                      }}
                      className="bg-[#f9f9ff] border border-[#bcc9c6] rounded px-2 py-1 text-xs text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40 cursor-pointer"
                    >
                      {PAGE_SIZE_OPTIONS.map((size) => (
                        <option key={size} value={size}>
                          {size}
                        </option>
                      ))}
                    </select>
                  </div>
                </div>

                {/* Event Pagination Controls */}
                <div className="flex items-center gap-1">
                  <button
                    type="button"
                    disabled={eventPage <= 1 || eventLoading}
                    onClick={() => setEventPage((p) => Math.max(1, p - 1))}
                    className="w-8 h-8 rounded bg-[#f9f9ff] border border-[#bcc9c6] text-[#3d4947] hover:bg-[#eef0fb] disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer flex items-center justify-center text-xs"
                    title="Previous page"
                  >
                    ‹
                  </button>

                  {Array.from({ length: eventTotalPages }, (_, i) => i + 1)
                    .filter((p) => p === 1 || p === eventTotalPages || Math.abs(p - eventPage) <= 1)
                    .map((p, idx, arr) => {
                      const prev = arr[idx - 1];
                      const hasGap = prev && p - prev > 1;

                      return (
                        <div key={p} className="flex items-center gap-1">
                          {hasGap && <span className="text-xs text-[#6d7a77] px-0.5">…</span>}
                          <button
                            type="button"
                            onClick={() => setEventPage(p)}
                            disabled={eventLoading}
                            className={`w-8 h-8 rounded text-xs font-semibold transition-colors cursor-pointer ${
                              eventPage === p
                                ? "bg-[#00685f] text-white"
                                : "bg-[#f9f9ff] border border-[#bcc9c6] text-[#3d4947] hover:bg-[#eef0fb]"
                            }`}
                          >
                            {p}
                          </button>
                        </div>
                      );
                    })}

                  <button
                    type="button"
                    disabled={eventPage >= eventTotalPages || eventLoading}
                    onClick={() => setEventPage((p) => Math.min(eventTotalPages, p + 1))}
                    className="w-8 h-8 rounded bg-[#f9f9ff] border border-[#bcc9c6] text-[#3d4947] hover:bg-[#eef0fb] disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer flex items-center justify-center text-xs"
                    title="Next page"
                  >
                    ›
                  </button>
                </div>
              </div>
            </div>
          </>
        )}

        {/* =================================================================== */}
        {/* VIEW 2: ALL PAYMENTS (TRANSACTION REGISTER)                         */}
        {/* =================================================================== */}
        {viewMode === "ALL_PAYMENTS" && (
          <>
            {/* Active Event Drill-Down Filter Banner */}
            {selectedEventFilter && (
              <div className="bg-[#f0fdfa] border border-[#99f6e4] rounded-xl p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 shadow-xs">
                <div className="flex items-center gap-2.5">
                  <span className="p-1.5 rounded-lg bg-[#ccfbf1] text-[#0f766e]">
                    <svg width="18" height="18" viewBox="0 0 20 20" fill="currentColor">
                      <path
                        fillRule="evenodd"
                        d="M3 4a1 1 0 011-1h12a1 1 0 011 1v2a1 1 0 01-.293.707L12 11.414V16a1 1 0 01-.293.707l-2 2A1 1 0 018 18v-6.586L3.293 6.707A1 1 0 013 6V4z"
                        clipRule="evenodd"
                      />
                    </svg>
                  </span>
                  <div>
                    <div className="text-xs font-semibold text-[#0f766e] uppercase tracking-wide">
                      Filtered by Event
                    </div>
                    <div className="text-sm font-bold text-[#141b2b]">
                      {selectedEventFilter.title}
                    </div>
                    <div className="text-[10px] text-[#6d7a77] font-mono select-all">
                      Event ID: {selectedEventFilter.id}
                    </div>
                  </div>
                </div>

                <div className="flex items-center gap-2 self-start sm:self-center">
                  <button
                    type="button"
                    onClick={handleClearEventFilter}
                    className="text-xs font-semibold px-3 py-1.5 rounded-lg bg-white border border-[#99f6e4] text-[#0f766e] hover:bg-[#ccfbf1] transition-colors cursor-pointer"
                  >
                    Clear Event Filter
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setViewMode("BY_EVENT");
                    }}
                    className="text-xs font-bold px-3 py-1.5 rounded-lg bg-[#00685f] text-white hover:bg-[#00524b] transition-colors cursor-pointer"
                  >
                    ← Return to By Event
                  </button>
                </div>
              </div>
            )}

            {/* Error Alert */}
            {paymentsError && (
              <div className="flex items-center justify-between rounded-lg border border-[#ffdad6] bg-[#fff5f4] px-4 py-3">
                <div className="flex items-center gap-2">
                  <svg className="w-5 h-5 text-[#93000a] shrink-0" viewBox="0 0 20 20" fill="currentColor">
                    <path
                      fillRule="evenodd"
                      d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                      clipRule="evenodd"
                    />
                  </svg>
                  <span className="text-[#93000a] text-sm font-medium">{paymentsError}</span>
                </div>
                <div className="flex items-center gap-3">
                  <button
                    type="button"
                    onClick={() => setPaymentsRefreshKey((k) => k + 1)}
                    className="text-[#93000a] text-xs font-bold hover:underline cursor-pointer"
                  >
                    Retry
                  </button>
                  <button
                    type="button"
                    onClick={() => setPaymentsError("")}
                    className="text-[#6d7a77] text-xs font-semibold hover:text-[#141b2b] cursor-pointer"
                  >
                    Dismiss
                  </button>
                </div>
              </div>
            )}

            {/* Payments Metric Cards */}
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
                <div className="text-[#3d4947] text-xs font-medium uppercase tracking-wide mb-1">
                  {selectedEventFilter ? "Event Payment Records" : "Total Payment Records"}
                </div>
                <div className="text-[#141b2b] text-2xl font-bold tracking-tight">
                  {totalRecords.toLocaleString("en-IN")}
                </div>
                <div className="text-xs mt-1 font-medium text-[#00685f]">
                  {selectedEventFilter
                    ? `For ${selectedEventFilter.title}`
                    : "Across all historical transactions"}
                </div>
              </div>

              <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
                <div className="text-[#3d4947] text-xs font-medium uppercase tracking-wide mb-1">
                  Active Filter State
                </div>
                <div className="text-[#141b2b] text-base font-bold tracking-tight mt-1 truncate">
                  {status === "ALL" ? "All Payment Statuses" : status}
                </div>
                <div className="text-xs mt-1 font-medium text-[#3d4947]">
                  Ledger Allocation: {allocationStatus === "ALL" ? "All Types" : allocationStatus}
                </div>
              </div>

              <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
                <div className="text-[#3d4947] text-xs font-medium uppercase tracking-wide mb-1">
                  Displaying Current Page
                </div>
                <div className="text-[#141b2b] text-2xl font-bold tracking-tight">
                  {payments.length}
                </div>
                <div className="text-xs mt-1 font-medium text-[#6d7a77]">
                  Page {page} of {totalPages} ({limit} per page)
                </div>
              </div>
            </div>

            {/* Filter Controls & Search Bar */}
            <div className="bg-white border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)] space-y-3">
              <div className="flex flex-col lg:flex-row items-stretch lg:items-center justify-between gap-3">
                {/* Search Input */}
                <div className="relative flex-1">
                  <svg
                    className="absolute left-3 top-1/2 -translate-y-1/2"
                    width="14"
                    height="14"
                    viewBox="0 0 13 13"
                    fill="none"
                  >
                    <path
                      d="M8.5 7.5H7.71L7.43 7.23C8.41 6.09 9 4.61 9 3C9 1.34 7.66 0 6 0C4.34 0 3 1.34 3 3C3 4.66 4.34 6 6 6C6.73 6 7.4 5.75 7.93 5.32L8.2 5.6V6.35L11.25 9.39L12.24 8.4L9.2 5.35H8.5V7.5ZM6 5.5C4.62 5.5 3.5 4.38 3.5 3C3.5 1.62 4.62 0.5 6 0.5C7.38 0.5 8.5 1.62 8.5 3C8.5 4.38 7.38 5.5 6 5.5Z"
                      fill="#6D7A77"
                    />
                  </svg>
                  <input
                    type="text"
                    placeholder="Search by transaction ID, order ID, booking #, customer..."
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

                {/* Clear Filters Button */}
                {hasActivePaymentFilters && (
                  <button
                    type="button"
                    onClick={handleClearPaymentsFilters}
                    className="text-xs text-[#00685f] hover:text-[#00524b] font-bold hover:underline cursor-pointer self-start lg:self-center shrink-0"
                  >
                    Clear all filters
                  </button>
                )}
              </div>

              {/* Secondary Filters Bar */}
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 pt-2 border-t border-[rgba(188,201,198,0.3)]">
                {/* Status Select */}
                <div>
                  <label className="block text-[11px] font-semibold text-[#6d7a77] uppercase tracking-wider mb-1">
                    Payment Status
                  </label>
                  <select
                    value={status}
                    onChange={(e) => {
                      setStatus(e.target.value);
                      setPage(1);
                    }}
                    className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-2.5 py-1.5 text-xs text-[#141b2b] font-medium focus:outline-none focus:ring-1 focus:ring-[#00685f]/40 cursor-pointer"
                  >
                    {PAYMENT_STATUS_OPTIONS.map((opt) => (
                      <option key={opt.value} value={opt.value}>
                        {opt.label}
                      </option>
                    ))}
                  </select>
                </div>

                {/* Ledger Allocation Status Select */}
                <div>
                  <label className="block text-[11px] font-semibold text-[#6d7a77] uppercase tracking-wider mb-1">
                    Ledger Allocation Status
                  </label>
                  <select
                    value={allocationStatus}
                    onChange={(e) => {
                      setAllocationStatus(e.target.value);
                      setPage(1);
                    }}
                    className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-2.5 py-1.5 text-xs text-[#141b2b] font-medium focus:outline-none focus:ring-1 focus:ring-[#00685f]/40 cursor-pointer"
                  >
                    {ALLOCATION_STATUS_OPTIONS.map((opt) => (
                      <option key={opt.value} value={opt.value}>
                        {opt.label}
                      </option>
                    ))}
                  </select>
                </div>

                {/* From Date */}
                <div>
                  <label className="block text-[11px] font-semibold text-[#6d7a77] uppercase tracking-wider mb-1">
                    From Date
                  </label>
                  <input
                    type="date"
                    value={fromDate}
                    onChange={(e) => {
                      setFromDate(e.target.value);
                      setPage(1);
                    }}
                    className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-2.5 py-1.5 text-xs text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40"
                  />
                </div>

                {/* To Date */}
                <div>
                  <label className="block text-[11px] font-semibold text-[#6d7a77] uppercase tracking-wider mb-1">
                    To Date
                  </label>
                  <input
                    type="date"
                    value={toDate}
                    onChange={(e) => {
                      setToDate(e.target.value);
                      setPage(1);
                    }}
                    className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-2.5 py-1.5 text-xs text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40"
                  />
                </div>
              </div>
            </div>

            {/* Payments Table Card */}
            <div className="bg-white border border-[#bcc9c6] rounded-xl overflow-hidden shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
              {/* Table Header Bar */}
              <div className="px-4 py-3 border-b border-[#dce2f7] flex items-center justify-between bg-[#f1f3ff]">
                <div className="text-xs text-[#141b2b] font-bold uppercase tracking-wider">
                  {selectedEventFilter
                    ? `Payments for: ${selectedEventFilter.title}`
                    : "Payments Register"}
                </div>
                <div className="text-xs text-[#6d7a77] font-medium">
                  Showing {totalRecords === 0 ? 0 : paymentStartRecord}–{paymentEndRecord} of{" "}
                  {totalRecords} records
                </div>
              </div>

              {/* Table */}
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs min-w-[950px]">
                  <thead className="bg-[#f9f9ff] text-[#6d7a77] uppercase tracking-wide border-b border-[#bcc9c6]">
                    <tr>
                      <th className="px-4 py-3 font-semibold">Payment / Tx ID</th>
                      <th className="px-4 py-3 font-semibold">Customer</th>
                      <th className="px-4 py-3 font-semibold">Event</th>
                      <th className="px-4 py-3 font-semibold">Organizer</th>
                      <th className="px-4 py-3 font-semibold">Amount</th>
                      <th className="px-4 py-3 font-semibold">Payment Status</th>
                      <th className="px-4 py-3 font-semibold">Ledger Allocation</th>
                      <th className="px-4 py-3 font-semibold">Method</th>
                      <th className="px-4 py-3 font-semibold">Date</th>
                      <th className="px-4 py-3 font-semibold text-right">Action</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-[rgba(188,201,198,0.25)]">
                    {paymentsLoading ? (
                      <tr>
                        <td colSpan={10} className="px-4 py-12 text-center text-[#6d7a77] text-sm">
                          <div className="flex flex-col items-center justify-center gap-2">
                            <div className="w-6 h-6 border-2 border-[#00685f]/30 border-t-[#00685f] rounded-full animate-spin" />
                            <span className="font-medium text-xs text-[#3d4947]">
                              Loading payments...
                            </span>
                          </div>
                        </td>
                      </tr>
                    ) : payments.length === 0 ? (
                      <tr>
                        <td colSpan={10} className="px-4 py-12 text-center text-[#6d7a77] text-sm">
                          <div className="flex flex-col items-center justify-center gap-2 max-w-sm mx-auto">
                            <svg className="w-10 h-10 text-[#bcc9c6]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                strokeWidth="1.5"
                                d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                              />
                            </svg>
                            <div className="font-semibold text-[#141b2b]">No payments found</div>
                            <p className="text-xs text-[#6d7a77]">
                              {hasActivePaymentFilters
                                ? "No payment records match your current filter and search criteria."
                                : "There are currently no recorded payments in the system."}
                            </p>
                            {hasActivePaymentFilters && (
                              <button
                                type="button"
                                onClick={handleClearPaymentsFilters}
                                className="mt-2 text-xs font-bold text-[#00685f] bg-[#f0fdfa] border border-[#99f6e4] px-3 py-1.5 rounded-lg hover:bg-[#ccfbf1] transition-colors cursor-pointer"
                              >
                                Reset all filters
                              </button>
                            )}
                          </div>
                        </td>
                      </tr>
                    ) : (
                      payments.map((p, idx) => {
                        const statusKey = p.status?.toUpperCase() || "PENDING";
                        const statusClass =
                          paymentStatusBadgeStyles[statusKey] ||
                          "bg-[#f3f4f6] text-[#4b5563] border border-[#e5e7eb]";

                        const allocationKey = p.allocation_status?.toUpperCase() || "NOT_APPLICABLE";
                        const allocationClass =
                          allocationBadgeStyles[allocationKey] ||
                          "bg-[#f3f4f6] text-[#4b5563] border border-[#e5e7eb]";

                        const dateDisplay = formatDateTime(p.paid_at || p.created_at);
                        const isEven = idx % 2 === 0;

                        return (
                          <tr
                            key={p.id}
                            className={`hover:bg-[#f1f3ff] transition-colors cursor-pointer ${
                              isEven ? "bg-white" : "bg-[#fafbff]"
                            }`}
                            onClick={() => setActiveDrawerPayment(p)}
                          >
                            {/* Payment ID & Transaction ID */}
                            <td className="px-4 py-3" onClick={(e) => e.stopPropagation()}>
                              <div className="flex flex-col">
                                <div className="flex items-center gap-1.5">
                                  <span className="font-mono text-[#141b2b] text-[11px] font-semibold">
                                    {p.id ? `${p.id.substring(0, 8)}...` : "—"}
                                  </span>
                                  {p.id && (
                                    <button
                                      type="button"
                                      onClick={() => copyText(p.id, "Payment ID")}
                                      className="text-[#6d7a77] hover:text-[#00685f] transition-colors cursor-pointer p-0.5"
                                      title="Copy full Payment ID"
                                    >
                                      {copiedId === p.id ? (
                                        <span className="text-[10px] text-[#00685f] font-bold">✓</span>
                                      ) : (
                                        <svg width="11" height="11" viewBox="0 0 20 20" fill="currentColor">
                                          <path d="M8 3a1 1 0 011-1h2a1 1 0 110 2H9a1 1 0 01-1-1z" />
                                          <path d="M6 3a2 2 0 00-2 2v11a2 2 0 002 2h8a2 2 0 002-2V5a2 2 0 00-2-2 3 3 0 01-3 3H9a3 3 0 01-3-3z" />
                                        </svg>
                                      )}
                                    </button>
                                  )}
                                </div>
                                <div className="text-[10px] text-[#6d7a77] font-mono mt-0.5">
                                  Tx: {p.transaction_id ? p.transaction_id : "—"}
                                </div>
                                {p.booking_number && (
                                  <div className="text-[10px] text-[#00685f] font-semibold mt-0.5">
                                    #{p.booking_number}
                                  </div>
                                )}
                              </div>
                            </td>

                            {/* Customer */}
                            <td className="px-4 py-3">
                              <div className="font-semibold text-[#141b2b]">
                                {p.customer?.name || "Anonymous Customer"}
                              </div>
                              <div className="text-[#6d7a77] text-[11px]">
                                {p.customer?.email || "—"}
                              </div>
                            </td>

                            {/* Event */}
                            <td className="px-4 py-3 max-w-[180px]">
                              <div className="font-medium text-[#141b2b] truncate" title={p.event?.title || ""}>
                                {p.event?.title || "—"}
                              </div>
                            </td>

                            {/* Organizer */}
                            <td className="px-4 py-3 max-w-[150px]">
                              <div className="text-[#3d4947] truncate" title={p.organizer?.business_name || ""}>
                                {p.organizer?.business_name || "—"}
                              </div>
                            </td>

                            {/* Amount */}
                            <td className="px-4 py-3 whitespace-nowrap">
                              <span className="font-bold text-[#141b2b] text-xs">
                                {formatCurrency(p.amount, p.currency)}
                              </span>
                            </td>

                            {/* Payment Status */}
                            <td className="px-4 py-3 whitespace-nowrap">
                              <span
                                className={`px-2 py-0.5 rounded-full text-[10px] font-bold uppercase ${statusClass}`}
                              >
                                {statusKey}
                              </span>
                            </td>

                            {/* Ledger Allocation Status */}
                            <td className="px-4 py-3 whitespace-nowrap">
                              <span
                                className={`px-2 py-0.5 rounded-full text-[10px] font-bold uppercase ${allocationClass}`}
                              >
                                {allocationKey}
                              </span>
                            </td>

                            {/* Method / Provider */}
                            <td className="px-4 py-3 whitespace-nowrap text-[#3d4947]">
                              <div className="font-medium">{p.method || "—"}</div>
                              <div className="text-[10px] text-[#6d7a77]">{p.provider || "RAZORPAY"}</div>
                            </td>

                            {/* Date */}
                            <td className="px-4 py-3 whitespace-nowrap text-[#6d7a77]">
                              {dateDisplay}
                            </td>

                            {/* Action */}
                            <td className="px-4 py-3 text-right whitespace-nowrap" onClick={(e) => e.stopPropagation()}>
                              <button
                                type="button"
                                onClick={() => setActiveDrawerPayment(p)}
                                className="text-xs text-[#00685f] font-semibold hover:underline cursor-pointer bg-[#f1f3ff] hover:bg-[#e1e8fd] px-2.5 py-1 rounded transition-colors"
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

              {/* Payments Pagination Footer */}
              <div className="flex flex-col sm:flex-row items-center justify-between px-4 py-3 border-t border-[rgba(188,201,198,0.3)] bg-white gap-3">
                <div className="flex items-center gap-3">
                  <span className="text-[#3d4947] text-xs">
                    Showing {totalRecords === 0 ? 0 : paymentStartRecord} to {paymentEndRecord} of{" "}
                    {totalRecords} payments
                  </span>

                  <div className="flex items-center gap-1.5 text-xs text-[#6d7a77]">
                    <span>Per page:</span>
                    <select
                      value={limit}
                      onChange={(e) => {
                        setLimit(Number(e.target.value));
                        setPage(1);
                      }}
                      className="bg-[#f9f9ff] border border-[#bcc9c6] rounded px-2 py-1 text-xs text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40 cursor-pointer"
                    >
                      {PAGE_SIZE_OPTIONS.map((size) => (
                        <option key={size} value={size}>
                          {size}
                        </option>
                      ))}
                    </select>
                  </div>
                </div>

                {/* Pagination Controls */}
                <div className="flex items-center gap-1">
                  <button
                    type="button"
                    disabled={page <= 1 || paymentsLoading}
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                    className="w-8 h-8 rounded bg-[#f9f9ff] border border-[#bcc9c6] text-[#3d4947] hover:bg-[#eef0fb] disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer flex items-center justify-center text-xs"
                    title="Previous page"
                  >
                    ‹
                  </button>

                  {Array.from({ length: totalPages }, (_, i) => i + 1)
                    .filter((p) => p === 1 || p === totalPages || Math.abs(p - page) <= 1)
                    .map((p, idx, arr) => {
                      const prev = arr[idx - 1];
                      const hasGap = prev && p - prev > 1;

                      return (
                        <div key={p} className="flex items-center gap-1">
                          {hasGap && <span className="text-xs text-[#6d7a77] px-0.5">…</span>}
                          <button
                            type="button"
                            onClick={() => setPage(p)}
                            disabled={paymentsLoading}
                            className={`w-8 h-8 rounded text-xs font-semibold transition-colors cursor-pointer ${
                              page === p
                                ? "bg-[#00685f] text-white"
                                : "bg-[#f9f9ff] border border-[#bcc9c6] text-[#3d4947] hover:bg-[#eef0fb]"
                            }`}
                          >
                            {p}
                          </button>
                        </div>
                      );
                    })}

                  <button
                    type="button"
                    disabled={page >= totalPages || paymentsLoading}
                    onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                    className="w-8 h-8 rounded bg-[#f9f9ff] border border-[#bcc9c6] text-[#3d4947] hover:bg-[#eef0fb] disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer flex items-center justify-center text-xs"
                    title="Next page"
                  >
                    ›
                  </button>
                </div>
              </div>
            </div>
          </>
        )}
      </div>

      {/* Payment Detail Drawer */}
      {activeDrawerPayment && (
        <PaymentDetailDrawer
          payment={activeDrawerPayment}
          onClose={() => setActiveDrawerPayment(null)}
        />
      )}
    </div>
  );
}
