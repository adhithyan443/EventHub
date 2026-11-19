import { useState, useEffect, useMemo } from "react";
import { useSearchParams } from "react-router-dom";
import {
  getOrganizerPaymentOverview,
  getOrganizerEventPaymentSummaries,
  getOrganizerPayments,
} from "../../api/organizerPaymentApi";
import { getOrganizerEvents } from "../../api/organizerApi";
import OrganizerPaymentDetailDrawer from "../../components/organizer/OrganizerPaymentDetailDrawer";
import notify from "../../utils/toast";

const PAYMENT_STATUS_OPTIONS = [
  { value: "ALL", label: "All Payment Statuses" },
  { value: "SUCCESS", label: "Success" },
  { value: "PENDING", label: "Pending" },
  { value: "FAILED", label: "Failed" },
  { value: "REFUND_REQUIRED", label: "Refund Required" },
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
  if (amount === null || amount === undefined || amount === "") return "—";
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

export default function OrganizerPaymentsPage() {
  const [searchParams] = useSearchParams();
  const urlEventId = searchParams.get("event_id") || "";
  const urlView = searchParams.get("view") === "ALL_PAYMENTS" ? "ALL_PAYMENTS" : "BY_EVENT";

  // View Switcher: 'BY_EVENT' (default) or 'ALL_PAYMENTS'
  const [viewMode, setViewMode] = useState(urlView);
  const [activeDrawerPayment, setActiveDrawerPayment] = useState(null);
  const [copiedBooking, setCopiedBooking] = useState(null);

  // Available organizer events for dropdown
  const [organizerEvents, setOrganizerEvents] = useState([]);
  const [loadingEventsList, setLoadingEventsList] = useState(false);

  // Global filters
  const [selectedEventId, setSelectedEventId] = useState(urlEventId);
  const [status, setStatus] = useState("ALL");
  const [allocationStatus, setAllocationStatus] = useState("ALL");
  const [fromDate, setFromDate] = useState("");
  const [toDate, setToDate] = useState("");
  const [search, setSearch] = useState("");

  // Overview metrics state (strictly loaded from backend)
  const [overview, setOverview] = useState({
    gross_captured_amount: "0.00",
    total_ticket_sales_amount: "0.00",
    total_platform_fees: "0.00",
    organizer_earnings: "0.00",
    total_tickets_sold: 0,
    total_payments_count: 0,
    total_events: 0,
    currency: "INR",
    refund_required_count: 0,
    unallocated_captured_count: 0,
  });
  const [overviewLoading, setOverviewLoading] = useState(true);

  // View 1: Event-wise Revenue state
  const [eventSummaries, setEventSummaries] = useState([]);
  const [eventPage, setEventPage] = useState(1);
  const [eventLimit, setEventLimit] = useState(10);
  const [eventTotalRecords, setEventTotalRecords] = useState(0);
  const [eventTotalPages, setEventTotalPages] = useState(1);
  const [eventLoading, setEventLoading] = useState(true);
  const [eventError, setEventError] = useState("");

  // View 2: All Payments state
  const [payments, setPayments] = useState([]);
  const [paymentPage, setPaymentPage] = useState(1);
  const [paymentLimit, setPaymentLimit] = useState(10);
  const [paymentTotalRecords, setPaymentTotalRecords] = useState(0);
  const [paymentTotalPages, setPaymentTotalPages] = useState(1);
  const [paymentsLoading, setPaymentsLoading] = useState(true);
  const [paymentsError, setPaymentsError] = useState("");

  // Refresh trigger counter
  const [refreshTrigger, setRefreshTrigger] = useState(0);

  // Derive selectedEventTitle cleanly with useMemo
  const selectedEventTitle = useMemo(() => {
    if (!selectedEventId) return "";
    const found = organizerEvents.find((e) => String(e.id) === String(selectedEventId));
    return found?.title || "";
  }, [selectedEventId, organizerEvents]);

  // Load organizer events list for filter dropdown once on mount
  useEffect(() => {
    let cancelled = false;
    const timer = setTimeout(async () => {
      setLoadingEventsList(true);
      try {
        const res = await getOrganizerEvents({ limit: 100 });
        if (cancelled) return;
        const list = Array.isArray(res?.data?.events)
          ? res.data.events
          : Array.isArray(res?.data)
          ? res.data
          : [];
        setOrganizerEvents(list);
      } catch (err) {
        if (!cancelled) {
          console.error("Failed to load organizer events for filter:", err);
        }
      } finally {
        if (!cancelled) {
          setLoadingEventsList(false);
        }
      }
    }, 0);

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, []);

  // Fetch Overview Data from backend
  useEffect(() => {
    let cancelled = false;

    const timer = setTimeout(async () => {
      setOverviewLoading(true);
      try {
        const res = await getOrganizerPaymentOverview({
          status,
          allocationStatus,
          from: fromDate,
          to: toDate,
          search: search.trim(),
          eventId: selectedEventId,
        });

        if (cancelled) return;

        if (res?.data) {
          setOverview({
            gross_captured_amount:
              res.data.gross_captured_amount != null
                ? String(res.data.gross_captured_amount)
                : "0.00",
            total_ticket_sales_amount:
              res.data.total_ticket_sales_amount != null
                ? String(res.data.total_ticket_sales_amount)
                : "0.00",
            total_platform_fees:
              res.data.total_platform_fees != null
                ? String(res.data.total_platform_fees)
                : "0.00",
            organizer_earnings:
              res.data.organizer_earnings != null
                ? String(res.data.organizer_earnings)
                : "0.00",
            total_tickets_sold:
              typeof res.data.total_tickets_sold === "number"
                ? res.data.total_tickets_sold
                : 0,
            total_payments_count:
              typeof res.data.total_payments_count === "number"
                ? res.data.total_payments_count
                : 0,
            total_events:
              typeof res.data.total_events === "number" ? res.data.total_events : 0,
            currency: res.data.currency || "INR",
            refund_required_count:
              typeof res.data.refund_required_count === "number"
                ? res.data.refund_required_count
                : 0,
            unallocated_captured_count:
              typeof res.data.unallocated_captured_count === "number"
                ? res.data.unallocated_captured_count
                : 0,
          });
        }
      } catch (err) {
        if (!cancelled) {
          console.error("Failed to load organizer payment overview:", err);
        }
      } finally {
        if (!cancelled) {
          setOverviewLoading(false);
        }
      }
    }, 300);

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [status, allocationStatus, fromDate, toDate, search, selectedEventId, refreshTrigger]);

  // Fetch Event-wise Summaries when viewMode is 'BY_EVENT'
  useEffect(() => {
    if (viewMode !== "BY_EVENT") return;

    let cancelled = false;

    const timer = setTimeout(async () => {
      setEventLoading(true);
      setEventError("");
      try {
        const res = await getOrganizerEventPaymentSummaries({
          page: eventPage,
          limit: eventLimit,
          status,
          allocationStatus,
          from: fromDate,
          to: toDate,
          search: search.trim(),
          eventId: selectedEventId,
        });

        if (cancelled) return;

        if (res?.data) {
          const list = Array.isArray(res.data.summaries)
            ? res.data.summaries
            : Array.isArray(res.data.events)
            ? res.data.events
            : [];
          setEventSummaries(list);

          if (res.data.overview) {
            setOverview((prev) => ({
              ...prev,
              gross_captured_amount:
                res.data.overview.gross_captured_amount != null
                  ? String(res.data.overview.gross_captured_amount)
                  : prev.gross_captured_amount,
              total_ticket_sales_amount:
                res.data.overview.total_ticket_sales_amount != null
                  ? String(res.data.overview.total_ticket_sales_amount)
                  : prev.total_ticket_sales_amount,
              total_platform_fees:
                res.data.overview.total_platform_fees != null
                  ? String(res.data.overview.total_platform_fees)
                  : prev.total_platform_fees,
              organizer_earnings:
                res.data.overview.organizer_earnings != null
                  ? String(res.data.overview.organizer_earnings)
                  : prev.organizer_earnings,
              total_tickets_sold:
                typeof res.data.overview.total_tickets_sold === "number"
                  ? res.data.overview.total_tickets_sold
                  : prev.total_tickets_sold,
              total_payments_count:
                typeof res.data.overview.total_payments_count === "number"
                  ? res.data.overview.total_payments_count
                  : prev.total_payments_count,
              total_events:
                typeof res.data.overview.total_events === "number"
                  ? res.data.overview.total_events
                  : prev.total_events,
              currency: res.data.overview.currency || prev.currency,
              refund_required_count:
                typeof res.data.overview.refund_required_count === "number"
                  ? res.data.overview.refund_required_count
                  : prev.refund_required_count,
              unallocated_captured_count:
                typeof res.data.overview.unallocated_captured_count === "number"
                  ? res.data.overview.unallocated_captured_count
                  : prev.unallocated_captured_count,
            }));
          }

          setEventTotalRecords(
            typeof res.data.total_records === "number" ? res.data.total_records : 0
          );
          setEventTotalPages(
            typeof res.data.total_pages === "number"
              ? Math.max(1, res.data.total_pages)
              : 1
          );
        } else {
          setEventSummaries([]);
          setEventTotalRecords(0);
          setEventTotalPages(1);
        }
      } catch (err) {
        if (!cancelled) {
          console.error("Failed to load event payment summaries:", err);
          const msg =
            err.response?.data?.message ||
            err.message ||
            "Failed to load event payment summaries.";
          setEventError(msg);
          setEventSummaries([]);
          setEventTotalRecords(0);
          setEventTotalPages(1);
        }
      } finally {
        if (!cancelled) {
          setEventLoading(false);
        }
      }
    }, 300);

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [
    viewMode,
    eventPage,
    eventLimit,
    status,
    allocationStatus,
    fromDate,
    toDate,
    search,
    selectedEventId,
    refreshTrigger,
  ]);

  // Fetch Payments List when viewMode is 'ALL_PAYMENTS'
  useEffect(() => {
    if (viewMode !== "ALL_PAYMENTS") return;

    let cancelled = false;

    const timer = setTimeout(async () => {
      setPaymentsLoading(true);
      setPaymentsError("");
      try {
        const res = await getOrganizerPayments({
          page: paymentPage,
          limit: paymentLimit,
          status,
          allocationStatus,
          from: fromDate,
          to: toDate,
          search: search.trim(),
          eventId: selectedEventId,
        });

        if (cancelled) return;

        if (res?.data) {
          const list = Array.isArray(res.data.payments) ? res.data.payments : [];
          setPayments(list);
          setPaymentTotalRecords(
            typeof res.data.total_records === "number" ? res.data.total_records : 0
          );
          setPaymentTotalPages(
            typeof res.data.total_pages === "number"
              ? Math.max(1, res.data.total_pages)
              : 1
          );
        } else {
          setPayments([]);
          setPaymentTotalRecords(0);
          setPaymentTotalPages(1);
        }
      } catch (err) {
        if (!cancelled) {
          console.error("Failed to load organizer payments:", err);
          const msg =
            err.response?.data?.message || err.message || "Failed to load payments.";
          setPaymentsError(msg);
          setPayments([]);
          setPaymentTotalRecords(0);
          setPaymentTotalPages(1);
        }
      } finally {
        if (!cancelled) {
          setPaymentsLoading(false);
        }
      }
    }, 300);

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [
    viewMode,
    paymentPage,
    paymentLimit,
    status,
    allocationStatus,
    fromDate,
    toDate,
    search,
    selectedEventId,
    refreshTrigger,
  ]);

  // Drill down from Event Summary row to Transactions view
  const handleDrillDownToEvent = (eventSummary) => {
    setSelectedEventId(eventSummary.event_id);
    setPaymentPage(1);
    setViewMode("ALL_PAYMENTS");
  };

  // Clear single event filter
  const handleClearEventFilter = () => {
    setSelectedEventId("");
    setEventPage(1);
    setPaymentPage(1);
  };

  // Reset all filters to default
  const handleResetFilters = () => {
    setSearch("");
    setStatus("ALL");
    setAllocationStatus("ALL");
    setFromDate("");
    setToDate("");
    setSelectedEventId("");
    setEventPage(1);
    setPaymentPage(1);
  };

  const hasActiveFilters = useMemo(() => {
    return (
      Boolean(search.trim()) ||
      status !== "ALL" ||
      allocationStatus !== "ALL" ||
      Boolean(fromDate) ||
      Boolean(toDate) ||
      Boolean(selectedEventId)
    );
  }, [search, status, allocationStatus, fromDate, toDate, selectedEventId]);

  // Quick copy helper
  const copyText = async (text, label) => {
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
      setCopiedBooking(text);
      notify.info(`Copied ${label} to clipboard`);
      setTimeout(() => setCopiedBooking(null), 2000);
    } catch {
      notify.error("Failed to copy");
    }
  };

  // Pagination bounds calculation
  const eventStartRecord = eventTotalRecords === 0 ? 0 : (eventPage - 1) * eventLimit + 1;
  const eventEndRecord = Math.min(eventPage * eventLimit, eventTotalRecords);

  const paymentStartRecord =
    paymentTotalRecords === 0 ? 0 : (paymentPage - 1) * paymentLimit + 1;
  const paymentEndRecord = Math.min(paymentPage * paymentLimit, paymentTotalRecords);

  const isCurrentViewLoading = viewMode === "BY_EVENT" ? eventLoading : paymentsLoading;

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-white w-full">
      {/* Top Header Bar */}
      <div className="px-4 sm:px-6 py-4 border-b border-[#dce2f7] flex flex-col md:flex-row md:items-center justify-between gap-4 shrink-0 bg-white">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-[#141b2b] text-xl font-bold tracking-tight">
              Payments & Financial Summaries
            </h1>
            <span className="bg-[#e6f4f2] text-[#00685f] text-xs font-semibold px-2.5 py-0.5 rounded-full border border-[#00685f]/20">
              Organizer Ledger
            </span>
          </div>
          <p className="text-[#3d4947] text-xs sm:text-sm mt-0.5">
            Monitor incoming customer payments, ticket sales, platform fees, and net organizer earnings.
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
              All Transactions
            </button>
          </div>

          <button
            type="button"
            onClick={() => setRefreshTrigger((k) => k + 1)}
            disabled={isCurrentViewLoading || overviewLoading}
            className="bg-[#f9f9ff] border border-[#bcc9c6] text-[#141b2b] text-xs sm:text-sm font-semibold px-3 py-1.5 rounded-lg flex items-center gap-1.5 hover:bg-[#eef0fb] transition-colors cursor-pointer disabled:opacity-50"
            title="Refresh current view"
          >
            <svg
              className={`w-3.5 h-3.5 ${
                isCurrentViewLoading || overviewLoading ? "animate-spin text-[#00685f]" : ""
              }`}
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
        {/* OVERVIEW CARDS SECTION (STRICTLY POPULATED FROM BACKEND)             */}
        {/* =================================================================== */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-3.5">
          {/* Card 1: Gross Captured Revenue */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.06)] flex flex-col justify-between">
            <div className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Gross Captured
            </div>
            {overviewLoading ? (
              <div className="h-7 w-24 bg-gray-200 animate-pulse rounded my-1" />
            ) : (
              <div className="text-[#141b2b] text-xl lg:text-2xl font-bold tracking-tight">
                {formatCurrency(overview.gross_captured_amount, overview.currency)}
              </div>
            )}
            <div className="text-[11px] mt-1 text-[#6d7a77] font-medium">
              Total funds captured
            </div>
          </div>

          {/* Card 2: Ticket Sales */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.06)] flex flex-col justify-between">
            <div className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Ticket Sales
            </div>
            {overviewLoading ? (
              <div className="h-7 w-24 bg-gray-200 animate-pulse rounded my-1" />
            ) : (
              <div className="text-[#141b2b] text-xl lg:text-2xl font-bold tracking-tight">
                {formatCurrency(overview.total_ticket_sales_amount, overview.currency)}
              </div>
            )}
            <div className="text-[11px] mt-1 text-[#6d7a77] font-medium">
              Gross ticket volume
            </div>
          </div>

          {/* Card 3: Platform Fees */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.06)] flex flex-col justify-between">
            <div className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Platform Fees
            </div>
            {overviewLoading ? (
              <div className="h-7 w-20 bg-gray-200 animate-pulse rounded my-1" />
            ) : (
              <div className="text-[#c2410c] text-xl lg:text-2xl font-bold tracking-tight">
                {formatCurrency(overview.total_platform_fees, overview.currency)}
              </div>
            )}
            <div className="text-[11px] mt-1 text-[#6d7a77] font-medium">
              Platform fee deduction
            </div>
          </div>

          {/* Card 4: Organizer Earnings (Highlighted in Teal/Green) */}
          <div className="bg-[#e6f4f2]/60 border border-[#00685f]/30 rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.06)] flex flex-col justify-between">
            <div className="text-[#00685f] text-[11px] font-bold uppercase tracking-wider mb-1">
              Organizer Earnings
            </div>
            {overviewLoading ? (
              <div className="h-7 w-24 bg-[#00685f]/20 animate-pulse rounded my-1" />
            ) : (
              <div className="text-[#00685f] text-xl lg:text-2xl font-extrabold tracking-tight">
                {formatCurrency(overview.organizer_earnings, overview.currency)}
              </div>
            )}
            <div className="text-[11px] mt-1 text-[#00685f] font-semibold">
              Net credited payable
            </div>
          </div>

          {/* Card 5: Tickets Sold */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.06)] flex flex-col justify-between">
            <div className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Tickets Sold
            </div>
            {overviewLoading ? (
              <div className="h-7 w-16 bg-gray-200 animate-pulse rounded my-1" />
            ) : (
              <div className="text-[#141b2b] text-xl lg:text-2xl font-bold tracking-tight">
                {overview.total_tickets_sold?.toLocaleString("en-IN") ?? 0}
              </div>
            )}
            <div className="text-[11px] mt-1 text-[#6d7a77] font-medium">
              Confirmed tickets
            </div>
          </div>

          {/* Card 6: Total Transactions */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.06)] flex flex-col justify-between">
            <div className="text-[#3d4947] text-[11px] font-semibold uppercase tracking-wider mb-1">
              Transactions
            </div>
            {overviewLoading ? (
              <div className="h-7 w-16 bg-gray-200 animate-pulse rounded my-1" />
            ) : (
              <div className="text-[#141b2b] text-xl lg:text-2xl font-bold tracking-tight">
                {overview.total_payments_count?.toLocaleString("en-IN") ?? 0}
              </div>
            )}
            <div className="text-[11px] mt-1 text-[#6d7a77] font-medium">
              {overview.total_events || 0} active {overview.total_events === 1 ? "event" : "events"}
            </div>
          </div>
        </div>

        {/* Attention Notice if refunds or unallocated payments exist */}
        {(overview.refund_required_count > 0 || overview.unallocated_captured_count > 0) && (
          <div className="bg-[#fff7ed] border border-[#fed7aa] rounded-xl px-4 py-3 flex flex-wrap items-center justify-between gap-3 text-xs">
            <div className="flex items-center gap-2 text-[#9a3412]">
              <svg className="w-4 h-4 shrink-0 text-[#c2410c]" fill="currentColor" viewBox="0 0 20 20">
                <path
                  fillRule="evenodd"
                  d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                  clipRule="evenodd"
                />
              </svg>
              <span>
                <strong>Attention:</strong>{" "}
                {overview.refund_required_count > 0 && (
                  <span>{overview.refund_required_count} transaction(s) require refund review. </span>
                )}
                {overview.unallocated_captured_count > 0 && (
                  <span>{overview.unallocated_captured_count} payment(s) captured without confirmed booking.</span>
                )}
              </span>
            </div>
            <button
              type="button"
              onClick={() => {
                setStatus(overview.refund_required_count > 0 ? "REFUND_REQUIRED" : "ALL");
                setAllocationStatus(overview.unallocated_captured_count > 0 ? "UNALLOCATED" : "ALL");
                setViewMode("ALL_PAYMENTS");
                setPaymentPage(1);
              }}
              className="text-[#c2410c] font-bold hover:underline cursor-pointer ml-auto"
            >
              Filter in All Transactions →
            </button>
          </div>
        )}

        {/* =================================================================== */}
        {/* UNIFIED FILTER BAR                                                  */}
        {/* =================================================================== */}
        <div className="bg-white border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.06)] space-y-3">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-12 gap-3 items-center">
            {/* Search Input (4 columns on lg) */}
            <div className="relative lg:col-span-4">
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
                placeholder={
                  viewMode === "BY_EVENT"
                    ? "Search event title..."
                    : "Search booking #, customer, txn ID..."
                }
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

            {/* Event Dropdown Filter (3 columns on lg) */}
            <div className="lg:col-span-3">
              <select
                value={selectedEventId}
                onChange={(e) => {
                  setSelectedEventId(e.target.value);
                  setEventPage(1);
                  setPaymentPage(1);
                }}
                disabled={loadingEventsList}
                className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-3 py-2 text-xs sm:text-sm text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40 cursor-pointer disabled:opacity-50"
              >
                <option value="">All Events</option>
                {organizerEvents.map((evt) => (
                  <option key={evt.id} value={evt.id}>
                    {evt.title}
                  </option>
                ))}
              </select>
            </div>

            {/* Status Dropdown (2 columns on lg) */}
            <div className="lg:col-span-2">
              <select
                value={status}
                onChange={(e) => {
                  setStatus(e.target.value);
                  setEventPage(1);
                  setPaymentPage(1);
                }}
                className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-3 py-2 text-xs sm:text-sm text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40 cursor-pointer"
              >
                {PAYMENT_STATUS_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>

            {/* Allocation Status Dropdown (3 columns on lg) */}
            <div className="lg:col-span-3">
              <select
                value={allocationStatus}
                onChange={(e) => {
                  setAllocationStatus(e.target.value);
                  setEventPage(1);
                  setPaymentPage(1);
                }}
                className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-3 py-2 text-xs sm:text-sm text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40 cursor-pointer"
              >
                {ALLOCATION_STATUS_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Date Range & Clear Filters Row */}
          <div className="flex flex-wrap items-center justify-between gap-3 pt-2 border-t border-[rgba(188,201,198,0.25)]">
            <div className="flex flex-wrap items-center gap-3">
              <div className="flex items-center gap-1.5">
                <span className="text-[11px] font-semibold text-[#6d7a77] uppercase">From:</span>
                <input
                  type="date"
                  value={fromDate}
                  onChange={(e) => {
                    setFromDate(e.target.value);
                    setEventPage(1);
                    setPaymentPage(1);
                  }}
                  className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-2.5 py-1 text-xs text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40"
                />
              </div>

              <div className="flex items-center gap-1.5">
                <span className="text-[11px] font-semibold text-[#6d7a77] uppercase">To:</span>
                <input
                  type="date"
                  value={toDate}
                  onChange={(e) => {
                    setToDate(e.target.value);
                    setEventPage(1);
                    setPaymentPage(1);
                  }}
                  className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-lg px-2.5 py-1 text-xs text-[#141b2b] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40"
                />
              </div>

              {hasActiveFilters && (
                <button
                  type="button"
                  onClick={handleResetFilters}
                  className="text-xs text-[#00685f] hover:text-[#00524b] font-bold hover:underline cursor-pointer ml-1 whitespace-nowrap"
                >
                  Clear all filters
                </button>
              )}
            </div>

            {/* Drill-down banner if single event filtered */}
            {selectedEventId && (
              <div className="flex items-center gap-2 bg-[#e6f4f2] text-[#00685f] px-3 py-1 rounded-lg border border-[#00685f]/20 text-xs">
                <span className="font-semibold">
                  Filtered by Event: {selectedEventTitle || selectedEventId}
                </span>
                <button
                  type="button"
                  onClick={handleClearEventFilter}
                  className="text-[#00685f] hover:text-[#93000a] font-bold cursor-pointer"
                  title="Remove event filter"
                >
                  ✕
                </button>
              </div>
            )}
          </div>
        </div>

        {/* =================================================================== */}
        {/* VIEW 1: BY EVENT (EVENT-WISE REVENUE TABLE)                         */}
        {/* =================================================================== */}
        {viewMode === "BY_EVENT" && (
          <div className="space-y-4">
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
                    onClick={() => setRefreshTrigger((k) => k + 1)}
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

            {/* Table Card */}
            <div className="bg-white border border-[#bcc9c6] rounded-xl overflow-hidden shadow-[0px_1px_1.5px_rgba(0,0,0,0.06)]">
              <div className="px-4 py-3 border-b border-[#dce2f7] flex items-center justify-between bg-[#f1f3ff]">
                <div className="text-xs text-[#141b2b] font-bold uppercase tracking-wider">
                  Event-Wise Revenue Breakdown
                </div>
                <div className="text-xs text-[#6d7a77] font-medium">
                  Showing {eventTotalRecords === 0 ? 0 : eventStartRecord}–{eventEndRecord} of{" "}
                  {eventTotalRecords} events
                </div>
              </div>

              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs min-w-[1050px]">
                  <thead className="bg-[#f9f9ff] text-[#6d7a77] uppercase tracking-wide border-b border-[#bcc9c6]">
                    <tr>
                      <th className="px-4 py-3 font-semibold">Event Name</th>
                      <th className="px-4 py-3 font-semibold text-center">Tickets Sold</th>
                      <th className="px-4 py-3 font-semibold text-right">Gross Captured</th>
                      <th className="px-4 py-3 font-semibold text-right">Platform Fees</th>
                      <th className="px-4 py-3 font-semibold text-right">Organizer Earnings</th>
                      <th className="px-4 py-3 font-semibold text-center">Payment Statuses</th>
                      <th className="px-4 py-3 font-semibold text-center">Refund Info</th>
                      <th className="px-4 py-3 font-semibold text-right">Action</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-[rgba(188,201,198,0.25)]">
                    {eventLoading ? (
                      Array.from({ length: 5 }).map((_, i) => (
                        <tr key={i} className="animate-pulse">
                          <td className="px-4 py-4">
                            <div className="h-4 bg-gray-200 rounded w-48 mb-1" />
                            <div className="h-3 bg-gray-100 rounded w-28" />
                          </td>
                          <td className="px-4 py-4 text-center">
                            <div className="h-4 bg-gray-200 rounded w-12 mx-auto" />
                          </td>
                          <td className="px-4 py-4 text-right">
                            <div className="h-4 bg-gray-200 rounded w-20 ml-auto" />
                          </td>
                          <td className="px-4 py-4 text-right">
                            <div className="h-4 bg-gray-200 rounded w-16 ml-auto" />
                          </td>
                          <td className="px-4 py-4 text-right">
                            <div className="h-4 bg-gray-200 rounded w-20 ml-auto" />
                          </td>
                          <td className="px-4 py-4 text-center">
                            <div className="h-4 bg-gray-200 rounded w-28 mx-auto" />
                          </td>
                          <td className="px-4 py-4 text-center">
                            <div className="h-4 bg-gray-200 rounded w-20 mx-auto" />
                          </td>
                          <td className="px-4 py-4 text-right">
                            <div className="h-6 bg-gray-200 rounded w-24 ml-auto" />
                          </td>
                        </tr>
                      ))
                    ) : eventSummaries.length === 0 ? (
                      <tr>
                        <td colSpan={8} className="px-4 py-12 text-center text-[#6d7a77]">
                          <div className="flex flex-col items-center justify-center gap-2">
                            <svg
                              className="w-8 h-8 text-[#bcc9c6]"
                              fill="none"
                              viewBox="0 0 24 24"
                              stroke="currentColor"
                            >
                              <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                strokeWidth={1.5}
                                d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
                              />
                            </svg>
                            <span className="font-semibold text-sm text-[#141b2b]">
                              No event payment data found
                            </span>
                            <span className="text-xs text-[#6d7a77]">
                              {hasActiveFilters
                                ? "Try adjusting your search query or filters."
                                : "No ticket payments recorded yet for your events."}
                            </span>
                            {hasActiveFilters && (
                              <button
                                type="button"
                                onClick={handleResetFilters}
                                className="mt-2 text-xs text-[#00685f] font-bold hover:underline cursor-pointer"
                              >
                                Clear all filters
                              </button>
                            )}
                          </div>
                        </td>
                      </tr>
                    ) : (
                      eventSummaries.map((summary) => (
                        <tr
                          key={summary.event_id}
                          className="hover:bg-[#f9f9ff] transition-colors"
                        >
                          {/* Event Name */}
                          <td className="px-4 py-3.5">
                            <div className="font-bold text-sm text-[#141b2b]">
                              {summary.event_title || "Untitled Event"}
                            </div>
                            <div className="text-[11px] font-mono text-[#6d7a77] mt-0.5">
                              {summary.event_id}
                            </div>
                          </td>

                          {/* Tickets Sold */}
                          <td className="px-4 py-3.5 text-center">
                            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-bold bg-[#00685f]/10 text-[#00685f]">
                              {summary.tickets_sold?.toLocaleString("en-IN") ?? 0}
                            </span>
                          </td>

                          {/* Gross Captured */}
                          <td className="px-4 py-3.5 text-right font-semibold text-[#141b2b]">
                            {formatCurrency(summary.gross_captured_amount, summary.currency)}
                          </td>

                          {/* Platform Fees */}
                          <td className="px-4 py-3.5 text-right font-medium text-[#c2410c]">
                            {formatCurrency(summary.platform_fees, summary.currency)}
                          </td>

                          {/* Organizer Earnings */}
                          <td className="px-4 py-3.5 text-right font-bold text-[#00685f] text-sm">
                            {formatCurrency(summary.organizer_earnings, summary.currency)}
                          </td>

                          {/* Payment Status Counts */}
                          <td className="px-4 py-3.5 text-center">
                            <div className="inline-flex items-center gap-1.5 flex-wrap justify-center">
                              <span
                                className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-[#dcfce7] text-[#15803d]"
                                title="Successful Payments"
                              >
                                {summary.successfully_completed_payments_count ?? 0} paid
                              </span>
                              {(summary.pending_payments_count > 0 ||
                                summary.failed_payments_count > 0) && (
                                <>
                                  {summary.pending_payments_count > 0 && (
                                    <span
                                      className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-[#fef9c3] text-[#854d0e]"
                                      title="Pending Payments"
                                    >
                                      {summary.pending_payments_count} pend.
                                    </span>
                                  )}
                                  {summary.failed_payments_count > 0 && (
                                    <span
                                      className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-[#fee2e2] text-[#b91c1c]"
                                      title="Failed Payments"
                                    >
                                      {summary.failed_payments_count} fail
                                    </span>
                                  )}
                                </>
                              )}
                            </div>
                          </td>

                          {/* Refund Info */}
                          <td className="px-4 py-3.5 text-center">
                            {summary.refund_required_count > 0 ||
                            summary.payments_with_successful_refunds_count > 0 ||
                            summary.unallocated_captured_count > 0 ? (
                              <div className="inline-flex items-center gap-1.5 flex-wrap justify-center">
                                {summary.refund_required_count > 0 && (
                                  <span className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-[#ffedd5] text-[#c2410c]">
                                    {summary.refund_required_count} ref. req.
                                  </span>
                                )}
                                {summary.payments_with_successful_refunds_count > 0 && (
                                  <span className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-[#f3e8ff] text-[#7e22ce]">
                                    {summary.payments_with_successful_refunds_count} refunded
                                  </span>
                                )}
                                {summary.unallocated_captured_count > 0 && (
                                  <span className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-[#fef3c7] text-[#b45309]">
                                    {summary.unallocated_captured_count} unallocated
                                  </span>
                                )}
                              </div>
                            ) : (
                              <span className="text-[#6d7a77] text-[11px]">—</span>
                            )}
                          </td>

                          {/* Action Button */}
                          <td className="px-4 py-3.5 text-right">
                            <button
                              type="button"
                              onClick={() => handleDrillDownToEvent(summary)}
                              className="px-3 py-1 rounded-lg bg-[#00685f]/10 hover:bg-[#00685f] text-[#00685f] hover:text-white font-bold text-xs transition-colors cursor-pointer whitespace-nowrap"
                            >
                              Transactions →
                            </button>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>

              {/* Pagination Footer */}
              <div className="px-4 py-3 border-t border-[#dce2f7] bg-[#f9f9ff] flex flex-col sm:flex-row items-center justify-between gap-3">
                <div className="flex items-center gap-2 text-xs text-[#6d7a77]">
                  <span>Rows per page:</span>
                  <select
                    value={eventLimit}
                    onChange={(e) => {
                      setEventLimit(Number(e.target.value));
                      setEventPage(1);
                    }}
                    className="bg-white border border-[#bcc9c6] rounded px-2 py-1 text-xs text-[#141b2b] focus:outline-none"
                  >
                    {PAGE_SIZE_OPTIONS.map((size) => (
                      <option key={size} value={size}>
                        {size}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="flex items-center gap-2">
                  <span className="text-xs text-[#6d7a77]">
                    Page {eventPage} of {eventTotalPages}
                  </span>
                  <button
                    type="button"
                    onClick={() => setEventPage((p) => Math.max(1, p - 1))}
                    disabled={eventPage <= 1 || eventLoading}
                    className="px-2.5 py-1 rounded border border-[#bcc9c6] bg-white text-xs font-semibold text-[#141b2b] hover:bg-[#eef0fb] disabled:opacity-40 cursor-pointer"
                  >
                    Prev
                  </button>
                  <button
                    type="button"
                    onClick={() => setEventPage((p) => Math.min(eventTotalPages, p + 1))}
                    disabled={eventPage >= eventTotalPages || eventLoading}
                    className="px-2.5 py-1 rounded border border-[#bcc9c6] bg-white text-xs font-semibold text-[#141b2b] hover:bg-[#eef0fb] disabled:opacity-40 cursor-pointer"
                  >
                    Next
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* =================================================================== */}
        {/* VIEW 2: ALL TRANSACTIONS (TRANSACTION HISTORY TABLE)                */}
        {/* =================================================================== */}
        {viewMode === "ALL_PAYMENTS" && (
          <div className="space-y-4">
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
                    onClick={() => setRefreshTrigger((k) => k + 1)}
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

            {/* Table Card */}
            <div className="bg-white border border-[#bcc9c6] rounded-xl overflow-hidden shadow-[0px_1px_1.5px_rgba(0,0,0,0.06)]">
              <div className="px-4 py-3 border-b border-[#dce2f7] flex items-center justify-between bg-[#f1f3ff]">
                <div className="text-xs text-[#141b2b] font-bold uppercase tracking-wider">
                  Transaction History & Ledger Allocations
                </div>
                <div className="text-xs text-[#6d7a77] font-medium">
                  Showing {paymentTotalRecords === 0 ? 0 : paymentStartRecord}–{paymentEndRecord} of{" "}
                  {paymentTotalRecords} transactions
                </div>
              </div>

              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs min-w-[1100px]">
                  <thead className="bg-[#f9f9ff] text-[#6d7a77] uppercase tracking-wide border-b border-[#bcc9c6]">
                    <tr>
                      <th className="px-4 py-3 font-semibold">Booking #</th>
                      <th className="px-4 py-3 font-semibold">Customer</th>
                      <th className="px-4 py-3 font-semibold">Event</th>
                      <th className="px-4 py-3 font-semibold text-center">Tickets</th>
                      <th className="px-4 py-3 font-semibold text-right">Amount</th>
                      <th className="px-4 py-3 font-semibold text-center">Status</th>
                      <th className="px-4 py-3 font-semibold text-center">Allocation</th>
                      <th className="px-4 py-3 font-semibold text-center">Method</th>
                      <th className="px-4 py-3 font-semibold">Date</th>
                      <th className="px-4 py-3 font-semibold text-right">Action</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-[rgba(188,201,198,0.25)]">
                    {paymentsLoading ? (
                      Array.from({ length: 6 }).map((_, i) => (
                        <tr key={i} className="animate-pulse">
                          <td className="px-4 py-4">
                            <div className="h-4 bg-gray-200 rounded w-28" />
                          </td>
                          <td className="px-4 py-4">
                            <div className="h-4 bg-gray-200 rounded w-32 mb-1" />
                            <div className="h-3 bg-gray-100 rounded w-24" />
                          </td>
                          <td className="px-4 py-4">
                            <div className="h-4 bg-gray-200 rounded w-40" />
                          </td>
                          <td className="px-4 py-4 text-center">
                            <div className="h-4 bg-gray-200 rounded w-8 mx-auto" />
                          </td>
                          <td className="px-4 py-4 text-right">
                            <div className="h-4 bg-gray-200 rounded w-20 ml-auto" />
                          </td>
                          <td className="px-4 py-4 text-center">
                            <div className="h-4 bg-gray-200 rounded w-16 mx-auto" />
                          </td>
                          <td className="px-4 py-4 text-center">
                            <div className="h-4 bg-gray-200 rounded w-20 mx-auto" />
                          </td>
                          <td className="px-4 py-4 text-center">
                            <div className="h-4 bg-gray-200 rounded w-12 mx-auto" />
                          </td>
                          <td className="px-4 py-4">
                            <div className="h-4 bg-gray-200 rounded w-28" />
                          </td>
                          <td className="px-4 py-4 text-right">
                            <div className="h-6 bg-gray-200 rounded w-16 ml-auto" />
                          </td>
                        </tr>
                      ))
                    ) : payments.length === 0 ? (
                      <tr>
                        <td colSpan={10} className="px-4 py-12 text-center text-[#6d7a77]">
                          <div className="flex flex-col items-center justify-center gap-2">
                            <svg
                              className="w-8 h-8 text-[#bcc9c6]"
                              fill="none"
                              viewBox="0 0 24 24"
                              stroke="currentColor"
                            >
                              <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                strokeWidth={1.5}
                                d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z"
                              />
                            </svg>
                            <span className="font-semibold text-sm text-[#141b2b]">
                              No transactions found
                            </span>
                            <span className="text-xs text-[#6d7a77]">
                              {hasActiveFilters
                                ? "No payments match your active filters or search criteria."
                                : "No payment transactions have been recorded yet."}
                            </span>
                            {hasActiveFilters && (
                              <button
                                type="button"
                                onClick={handleResetFilters}
                                className="mt-2 text-xs text-[#00685f] font-bold hover:underline cursor-pointer"
                              >
                                Clear all filters
                              </button>
                            )}
                          </div>
                        </td>
                      </tr>
                    ) : (
                      payments.map((p) => {
                        const statusKey = p.status?.toUpperCase() || "PENDING";
                        const statusClass =
                          paymentStatusBadgeStyles[statusKey] ||
                          "bg-[#f3f4f6] text-[#4b5563] border border-[#e5e7eb]";

                        const allocationKey =
                          p.allocation_status?.toUpperCase() || "NOT_APPLICABLE";
                        const allocationClass =
                          allocationBadgeStyles[allocationKey] ||
                          "bg-[#f3f4f6] text-[#4b5563] border border-[#e5e7eb]";

                        return (
                          <tr
                            key={p.id}
                            className="hover:bg-[#f9f9ff] transition-colors cursor-pointer"
                            onClick={() => setActiveDrawerPayment(p)}
                          >
                            {/* Booking # */}
                            <td className="px-4 py-3.5">
                              <div className="flex items-center gap-1.5">
                                <span className="font-mono font-bold text-xs text-[#141b2b]">
                                  {p.booking_number || "—"}
                                </span>
                                {p.booking_number && (
                                  <button
                                    type="button"
                                    onClick={(e) => {
                                      e.stopPropagation();
                                      copyText(p.booking_number, "Booking Number");
                                    }}
                                    className="text-[#6d7a77] hover:text-[#00685f] cursor-pointer p-0.5 rounded"
                                    title={
                                      copiedBooking === p.booking_number
                                        ? "Copied!"
                                        : "Copy Booking Number"
                                    }
                                  >
                                    <svg
                                      className="w-3.5 h-3.5"
                                      fill="none"
                                      viewBox="0 0 24 24"
                                      stroke="currentColor"
                                    >
                                      <path
                                        strokeLinecap="round"
                                        strokeLinejoin="round"
                                        strokeWidth={2}
                                        d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
                                      />
                                    </svg>
                                  </button>
                                )}
                              </div>
                              {p.transaction_id && (
                                <div className="text-[10px] font-mono text-[#6d7a77] truncate max-w-[140px] mt-0.5">
                                  Txn: {p.transaction_id}
                                </div>
                              )}
                            </td>

                            {/* Customer */}
                            <td className="px-4 py-3.5">
                              <div className="font-semibold text-[#141b2b]">
                                {p.customer?.name || "Customer"}
                              </div>
                              <div className="text-[11px] text-[#6d7a77] truncate max-w-[160px]">
                                {p.customer?.email || "—"}
                              </div>
                            </td>

                            {/* Event */}
                            <td className="px-4 py-3.5">
                              <div className="font-medium text-[#141b2b] truncate max-w-[180px]">
                                {p.event?.title || "—"}
                              </div>
                            </td>

                            {/* Ticket Quantity */}
                            <td className="px-4 py-3.5 text-center">
                              <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-bold bg-[#00685f]/10 text-[#00685f]">
                                {p.ticket_quantity || 1}
                              </span>
                            </td>

                            {/* Payment Amount */}
                            <td className="px-4 py-3.5 text-right font-bold text-[#141b2b] text-sm">
                              {formatCurrency(p.amount, p.currency)}
                            </td>

                            {/* Payment Status */}
                            <td className="px-4 py-3.5 text-center">
                              <span
                                className={`inline-block px-2 py-0.5 text-[11px] font-bold rounded-full ${statusClass}`}
                              >
                                {statusKey}
                              </span>
                            </td>

                            {/* Allocation Status */}
                            <td className="px-4 py-3.5 text-center">
                              <span
                                className={`inline-block px-2 py-0.5 text-[11px] font-bold rounded-full ${allocationClass}`}
                              >
                                {allocationKey}
                              </span>
                            </td>

                            {/* Payment Method */}
                            <td className="px-4 py-3.5 text-center">
                              <span className="text-xs font-semibold text-[#3d4947] bg-[#f1f3ff] px-2 py-0.5 rounded border border-[#dce2f7]">
                                {p.method || "—"}
                              </span>
                            </td>

                            {/* Date */}
                            <td className="px-4 py-3.5 text-xs text-[#3d4947] whitespace-nowrap">
                              {formatDateTime(p.paid_at || p.created_at)}
                            </td>

                            {/* Action Button */}
                            <td className="px-4 py-3.5 text-right">
                              <button
                                type="button"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  setActiveDrawerPayment(p);
                                }}
                                className="px-2.5 py-1 rounded-lg border border-[#bcc9c6] hover:border-[#00685f] hover:bg-[#00685f]/5 text-xs font-semibold text-[#141b2b] transition-colors cursor-pointer"
                              >
                                Details
                              </button>
                            </td>
                          </tr>
                        );
                      })
                    )}
                  </tbody>
                </table>
              </div>

              {/* Pagination Footer */}
              <div className="px-4 py-3 border-t border-[#dce2f7] bg-[#f9f9ff] flex flex-col sm:flex-row items-center justify-between gap-3">
                <div className="flex items-center gap-2 text-xs text-[#6d7a77]">
                  <span>Rows per page:</span>
                  <select
                    value={paymentLimit}
                    onChange={(e) => {
                      setPaymentLimit(Number(e.target.value));
                      setPaymentPage(1);
                    }}
                    className="bg-white border border-[#bcc9c6] rounded px-2 py-1 text-xs text-[#141b2b] focus:outline-none"
                  >
                    {PAGE_SIZE_OPTIONS.map((size) => (
                      <option key={size} value={size}>
                        {size}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="flex items-center gap-2">
                  <span className="text-xs text-[#6d7a77]">
                    Page {paymentPage} of {paymentTotalPages}
                  </span>
                  <button
                    type="button"
                    onClick={() => setPaymentPage((p) => Math.max(1, p - 1))}
                    disabled={paymentPage <= 1 || paymentsLoading}
                    className="px-2.5 py-1 rounded border border-[#bcc9c6] bg-white text-xs font-semibold text-[#141b2b] hover:bg-[#eef0fb] disabled:opacity-40 cursor-pointer"
                  >
                    Prev
                  </button>
                  <button
                    type="button"
                    onClick={() =>
                      setPaymentPage((p) => Math.min(paymentTotalPages, p + 1))
                    }
                    disabled={paymentPage >= paymentTotalPages || paymentsLoading}
                    className="px-2.5 py-1 rounded border border-[#bcc9c6] bg-white text-xs font-semibold text-[#141b2b] hover:bg-[#eef0fb] disabled:opacity-40 cursor-pointer"
                  >
                    Next
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Slide-over Payment Detail Drawer */}
      {activeDrawerPayment && (
        <OrganizerPaymentDetailDrawer
          payment={activeDrawerPayment}
          onClose={() => setActiveDrawerPayment(null)}
        />
      )}
    </div>
  );
}
