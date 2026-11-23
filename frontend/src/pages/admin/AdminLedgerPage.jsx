import { useState, useEffect, useMemo, useCallback } from "react";
import { getAdminLedgerJournals, getAdminFinancialOverview } from "../../api/adminLedgerApi";
import LedgerJournalDetailDrawer from "../../components/admin/LedgerJournalDetailDrawer";
import notify from "../../utils/toast";

const TRANSACTION_TYPE_OPTIONS = [
  { value: "ALL", label: "All Transaction Types" },
  { value: "PAYMENT", label: "Payment (Capture)" },
  { value: "REFUND", label: "Refund" },
  { value: "PAYOUT", label: "Payout / Settlement" },
  { value: "ADJUSTMENT", label: "Adjustment" },
  { value: "REVERSAL", label: "Reversal" },
];

const STATUS_OPTIONS = [
  { value: "ALL", label: "All Statuses" },
  { value: "POSTED", label: "Posted" },
  { value: "DRAFT", label: "Draft" },
];

const SETTLEMENT_STATUS_OPTIONS = [
  { value: "ALL", label: "All Settlement Statuses" },
  { value: "PENDING", label: "Pending" },
  { value: "APPROVED", label: "Approved" },
  { value: "PROCESSING", label: "Processing" },
  { value: "PAID", label: "Paid / Disbursed" },
  { value: "REJECTED", label: "Rejected" },
  { value: "FAILED", label: "Failed" },
];

const DATE_PRESETS = [
  { id: "ALL", label: "All Time" },
  { id: "TODAY", label: "Today" },
  { id: "LAST_7_DAYS", label: "Last 7 Days" },
  { id: "THIS_MONTH", label: "This Month" },
  { id: "LAST_30_DAYS", label: "Last 30 Days" },
  { id: "YTD", label: "Year to Date" },
];

const PAGE_SIZE_OPTIONS = [10, 25, 50, 100];

const txBadgeStyles = {
  PAYMENT: "bg-[#dcfce7] text-[#15803d] border border-[#bbf7d0]",
  REFUND: "bg-[#fae8ff] text-[#86198f] border border-[#f5d0fe]",
  PAYOUT: "bg-[#e0f2fe] text-[#0369a1] border border-[#bae6fd]",
  ADJUSTMENT: "bg-[#fef9c3] text-[#854d0e] border border-[#fef08a]",
  REVERSAL: "bg-[#f3e8ff] text-[#7e22ce] border border-[#e9d5ff]",
};

const statusBadgeStyles = {
  POSTED: "bg-[#dcfce7] text-[#15803d] border border-[#bbf7d0]",
  DRAFT: "bg-[#fef9c3] text-[#854d0e] border border-[#fef08a]",
};

const lifecycleBadgeStyles = {
  COMPLETED: "bg-emerald-100 text-emerald-800 border-emerald-300",
  UPCOMING: "bg-blue-100 text-blue-800 border-blue-300",
  ONGOING: "bg-amber-100 text-amber-800 border-amber-300",
  CANCELLED: "bg-red-100 text-red-800 border-red-300",
  DRAFT: "bg-gray-100 text-gray-700 border-gray-300",
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

const getDatePresetRange = (presetId) => {
  const now = new Date();
  const todayStr = now.toISOString().split("T")[0];

  switch (presetId) {
    case "TODAY":
      return { from: todayStr, to: todayStr };
    case "LAST_7_DAYS": {
      const d = new Date();
      d.setDate(d.getDate() - 7);
      return { from: d.toISOString().split("T")[0], to: todayStr };
    }
    case "THIS_MONTH": {
      const start = new Date(now.getFullYear(), now.getMonth(), 1);
      return { from: start.toISOString().split("T")[0], to: todayStr };
    }
    case "LAST_30_DAYS": {
      const d = new Date();
      d.setDate(d.getDate() - 30);
      return { from: d.toISOString().split("T")[0], to: todayStr };
    }
    case "YTD": {
      const start = new Date(now.getFullYear(), 0, 1);
      return { from: start.toISOString().split("T")[0], to: todayStr };
    }
    default:
      return { from: "", to: "" };
  }
};

export default function AdminLedgerPage() {
  // Active Tab
  const [activeTab, setActiveTab] = useState("journals"); // "journals" | "events" | "organizers"

  // Overview Data & Loading
  const [overviewData, setOverviewData] = useState(null);
  const [overviewLoading, setOverviewLoading] = useState(true);
  const [overviewError, setOverviewError] = useState("");

  // Journals Data & Loading
  const [journals, setJournals] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [totalRecords, setTotalRecords] = useState(0);
  const [totalPages, setTotalPages] = useState(1);

  // Filters State
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useState(10);
  const [transactionType, setTransactionType] = useState("ALL");
  const [statusFilter, setStatusFilter] = useState("ALL");
  const [settlementStatus, setSettlementStatus] = useState("ALL");
  const [selectedEventId, setSelectedEventId] = useState("");
  const [selectedOrganizerId, setSelectedOrganizerId] = useState("");
  const [activePreset, setActivePreset] = useState("ALL");
  const [fromDate, setFromDate] = useState("");
  const [toDate, setToDate] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [refreshKey, setRefreshKey] = useState(0);

  // Drawer State
  const [selectedJournalId, setSelectedJournalId] = useState(null);
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);

  // Debounce search input
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(searchInput.trim());
      setPage(1);
    }, 350);
    return () => clearTimeout(timer);
  }, [searchInput]);

  // Handle Preset Selection
  const handleSelectPreset = (presetId) => {
    setActivePreset(presetId);
    const range = getDatePresetRange(presetId);
    setFromDate(range.from);
    setToDate(range.to);
    setPage(1);
  };

  // Fetch Financial Overview
  const loadOverview = useCallback(async () => {
    setOverviewLoading(true);
    setOverviewError("");
    try {
      const res = await getAdminFinancialOverview({
        from: fromDate,
        to: toDate,
        eventId: selectedEventId,
        organizerId: selectedOrganizerId,
        transactionType,
        settlementStatus,
      });

      if (res?.data) {
        setOverviewData(res.data);
      } else {
        setOverviewData(null);
      }
    } catch (err) {
      console.error("Failed to fetch financial overview:", err);
      const msg =
        err.response?.data?.message || err.message || "Failed to load financial overview metrics.";
      setOverviewError(msg);
      setOverviewData(null);
    } finally {
      setOverviewLoading(false);
    }
  }, [fromDate, toDate, selectedEventId, selectedOrganizerId, transactionType, settlementStatus]);

  // Trigger Overview fetch
  useEffect(() => {
    const timer = setTimeout(() => {
      loadOverview();
    }, 0);
    return () => clearTimeout(timer);
  }, [loadOverview, refreshKey]);

  // Fetch Journals
  useEffect(() => {
    let cancelled = false;

    const loadJournals = async () => {
      setLoading(true);
      setError("");
      try {
        const res = await getAdminLedgerJournals({
          page,
          limit,
          transactionType,
          status: statusFilter,
          search: debouncedSearch,
          from: fromDate,
          to: toDate,
          eventId: selectedEventId,
          organizerId: selectedOrganizerId,
          settlementStatus,
        });

        if (cancelled) return;

        if (res?.data) {
          setJournals(Array.isArray(res.data.journals) ? res.data.journals : []);
          setTotalRecords(typeof res.data.total_records === "number" ? res.data.total_records : 0);
          setTotalPages(typeof res.data.total_pages === "number" ? Math.max(1, res.data.total_pages) : 1);
        } else {
          setJournals([]);
          setTotalRecords(0);
          setTotalPages(1);
        }
      } catch (err) {
        if (cancelled) return;
        console.error("Failed to fetch ledger journals:", err);
        const msg =
          err.response?.data?.message || err.message || "Failed to load financial ledger journals.";
        setError(msg);
        setJournals([]);
        setTotalRecords(0);
        setTotalPages(1);
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    const timer = setTimeout(() => {
      loadJournals();
    }, 0);

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [
    page,
    limit,
    transactionType,
    statusFilter,
    debouncedSearch,
    fromDate,
    toDate,
    selectedEventId,
    selectedOrganizerId,
    settlementStatus,
    refreshKey,
  ]);

  const handleOpenDrawer = (journalId) => {
    setSelectedJournalId(journalId);
    setIsDrawerOpen(true);
  };

  const handleCloseDrawer = () => {
    setIsDrawerOpen(false);
  };

  const handleResetFilters = () => {
    setSearchInput("");
    setDebouncedSearch("");
    setTransactionType("ALL");
    setStatusFilter("ALL");
    setSettlementStatus("ALL");
    setSelectedEventId("");
    setSelectedOrganizerId("");
    setActivePreset("ALL");
    setFromDate("");
    setToDate("");
    setPage(1);
  };

  const copyText = async (text, label) => {
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
      notify.info(`Copied ${label} to clipboard`);
    } catch {
      notify.error("Failed to copy");
    }
  };

  const hasActiveFilters =
    debouncedSearch !== "" ||
    transactionType !== "ALL" ||
    statusFilter !== "ALL" ||
    settlementStatus !== "ALL" ||
    selectedEventId !== "" ||
    selectedOrganizerId !== "" ||
    fromDate !== "" ||
    toDate !== "" ||
    activePreset !== "ALL";

  // Distinct Events and Organizers for dropdowns
  const availableEvents = useMemo(() => {
    if (!overviewData?.breakdown_by_event) return [];
    return overviewData.breakdown_by_event.map((ev) => ({
      id: ev.event_id,
      title: ev.event_title,
    }));
  }, [overviewData]);

  const availableOrganizers = useMemo(() => {
    if (!overviewData?.breakdown_by_organizer) return [];
    return overviewData.breakdown_by_organizer.map((org) => ({
      id: org.organizer_id,
      name: org.organizer_name,
      email: org.organizer_email,
    }));
  }, [overviewData]);

  const metrics = overviewData?.metrics || {};

  return (
    <div className="p-8 space-y-6 max-w-7xl mx-auto">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-black text-[#141b2b] tracking-tight">Financial Overview & Accounting</h1>
          <p className="text-xs text-[#6d7a77] mt-1">
            Authoritative platform revenue, organizer earnings, tax liabilities, and double-entry ledger audit trail
          </p>
        </div>
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={() => setRefreshKey((k) => k + 1)}
            disabled={loading || overviewLoading}
            className="inline-flex items-center gap-2 px-3.5 py-2 text-xs font-semibold rounded-lg bg-white border border-[#bcc9c6] text-[#141b2b] hover:bg-[#eef0fb] transition-colors cursor-pointer shadow-xs disabled:opacity-50"
          >
            <svg
              className={`w-3.5 h-3.5 ${loading || overviewLoading ? "animate-spin" : ""}`}
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
              />
            </svg>
            Refresh
          </button>
        </div>
      </div>

      {/* 10 Authoritative Financial Overview Cards */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-xs font-bold uppercase tracking-wider text-[#6d7a77]">Platform Financial Metrics</h2>
          {overviewData?.metrics?.currency && (
            <span className="text-[11px] font-medium text-[#6d7a77] bg-gray-100 px-2 py-0.5 rounded">
              Base Currency: {overviewData.metrics.currency}
            </span>
          )}
        </div>

        {overviewError && (
          <div className="p-3 rounded-lg bg-[#fee2e2]/60 border border-[#fca5a5] text-xs text-[#991b1b]">
            {overviewError}
          </div>
        )}

        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-3.5">
          {/* Card 1: Total Gross Collections */}
          <div className="bg-white border border-[#bcc9c6] rounded-xl p-4 shadow-xs relative overflow-hidden flex flex-col justify-between">
            <div className="flex items-center justify-between text-[#6d7a77]">
              <span className="text-[11px] font-bold uppercase tracking-wider">Gross Collections</span>
              <span className="p-1.5 rounded-lg bg-blue-50 text-blue-600">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </span>
            </div>
            <div className="mt-2">
              <div className="text-lg font-black text-[#141b2b]">
                {overviewLoading ? <span className="text-gray-300">...</span> : formatCurrency(metrics.total_gross_collections, metrics.currency)}
              </div>
              <p className="text-[10px] text-[#6d7a77] mt-0.5">Total collected from customers</p>
            </div>
          </div>

          {/* Card 2: Total Refunds */}
          <div className="bg-white border border-[#bcc9c6] rounded-xl p-4 shadow-xs relative overflow-hidden flex flex-col justify-between">
            <div className="flex items-center justify-between text-[#6d7a77]">
              <span className="text-[11px] font-bold uppercase tracking-wider">Total Refunds</span>
              <span className="p-1.5 rounded-lg bg-rose-50 text-rose-600">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 15v-1a4 4 0 00-4-4H8m0 0l3 3m-3-3l3-3m9 14V5a2 2 0 00-2-2H6a2 2 0 00-2 2v16l4-2 4 2 4-2 4 2z" />
                </svg>
              </span>
            </div>
            <div className="mt-2">
              <div className="text-lg font-black text-rose-600">
                {overviewLoading ? <span className="text-gray-300">...</span> : formatCurrency(metrics.total_refunds, metrics.currency)}
              </div>
              <p className="text-[10px] text-[#6d7a77] mt-0.5">Refunded to customers</p>
            </div>
          </div>

          {/* Card 3: Net Collections */}
          <div className="bg-white border border-[#bcc9c6] rounded-xl p-4 shadow-xs relative overflow-hidden flex flex-col justify-between">
            <div className="flex items-center justify-between text-[#6d7a77]">
              <span className="text-[11px] font-bold uppercase tracking-wider">Net Collections</span>
              <span className="p-1.5 rounded-lg bg-emerald-50 text-emerald-600">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </span>
            </div>
            <div className="mt-2">
              <div className="text-lg font-black text-[#141b2b]">
                {overviewLoading ? <span className="text-gray-300">...</span> : formatCurrency(metrics.net_collections, metrics.currency)}
              </div>
              <p className="text-[10px] text-[#6d7a77] mt-0.5">Gross collections minus refunds</p>
            </div>
          </div>

          {/* Card 4: Platform Revenue */}
          <div className="bg-white border border-[#bcc9c6] rounded-xl p-4 shadow-xs relative overflow-hidden flex flex-col justify-between">
            <div className="flex items-center justify-between text-[#6d7a77]">
              <span className="text-[11px] font-bold uppercase tracking-wider">Platform Revenue</span>
              <span className="p-1.5 rounded-lg bg-indigo-50 text-indigo-600">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
                </svg>
              </span>
            </div>
            <div className="mt-2">
              <div className="text-lg font-black text-indigo-600">
                {overviewLoading ? <span className="text-gray-300">...</span> : formatCurrency(metrics.platform_revenue, metrics.currency)}
              </div>
              <p className="text-[10px] text-[#6d7a77] mt-0.5">Account 4010 platform fees</p>
            </div>
          </div>

          {/* Card 5: Tax Liability */}
          <div className="bg-white border border-[#bcc9c6] rounded-xl p-4 shadow-xs relative overflow-hidden flex flex-col justify-between">
            <div className="flex items-center justify-between text-[#6d7a77]">
              <span className="text-[11px] font-bold uppercase tracking-wider">Tax Liability</span>
              <span className="p-1.5 rounded-lg bg-amber-50 text-amber-600">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 14l6-6m-5.5.5h.01m4.99 5h.01M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16l3.5-2 3.5 2 3.5-2 3.5 2z" />
                </svg>
              </span>
            </div>
            <div className="mt-2">
              <div className="text-lg font-black text-[#141b2b]">
                {overviewLoading ? <span className="text-gray-300">...</span> : formatCurrency(metrics.tax_liability, metrics.currency)}
              </div>
              <p className="text-[10px] text-[#6d7a77] mt-0.5">Account 2030 taxes collected</p>
            </div>
          </div>

          {/* Card 6: Organizer Earnings */}
          <div className="bg-white border border-[#bcc9c6] rounded-xl p-4 shadow-xs relative overflow-hidden flex flex-col justify-between">
            <div className="flex items-center justify-between text-[#6d7a77]">
              <span className="text-[11px] font-bold uppercase tracking-wider">Organizer Earnings</span>
              <span className="p-1.5 rounded-lg bg-teal-50 text-teal-600">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                </svg>
              </span>
            </div>
            <div className="mt-2">
              <div className="text-lg font-black text-[#141b2b]">
                {overviewLoading ? <span className="text-gray-300">...</span> : formatCurrency(metrics.organizer_earnings, metrics.currency)}
              </div>
              <p className="text-[10px] text-[#6d7a77] mt-0.5">Account 2010 owed to organizers</p>
            </div>
          </div>

          {/* Card 7: Organizer Payouts */}
          <div className="bg-white border border-[#bcc9c6] rounded-xl p-4 shadow-xs relative overflow-hidden flex flex-col justify-between">
            <div className="flex items-center justify-between text-[#6d7a77]">
              <span className="text-[11px] font-bold uppercase tracking-wider">Organizer Payouts</span>
              <span className="p-1.5 rounded-lg bg-sky-50 text-sky-600">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
              </span>
            </div>
            <div className="mt-2">
              <div className="text-lg font-black text-[#141b2b]">
                {overviewLoading ? <span className="text-gray-300">...</span> : formatCurrency(metrics.organizer_payouts, metrics.currency)}
              </div>
              <p className="text-[10px] text-[#6d7a77] mt-0.5">Account 1020 settled / disbursed</p>
            </div>
          </div>

          {/* Card 8: Pending Settlements */}
          <div className="bg-white border border-[#bcc9c6] rounded-xl p-4 shadow-xs relative overflow-hidden flex flex-col justify-between">
            <div className="flex items-center justify-between text-[#6d7a77]">
              <span className="text-[11px] font-bold uppercase tracking-wider">Pending Settlements</span>
              <span className="p-1.5 rounded-lg bg-orange-50 text-orange-600">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </span>
            </div>
            <div className="mt-2">
              <div className="text-lg font-black text-orange-600">
                {overviewLoading ? <span className="text-gray-300">...</span> : formatCurrency(metrics.pending_settlements, metrics.currency)}
              </div>
              <p className="text-[10px] text-[#6d7a77] mt-0.5">In-flight settlement requests</p>
            </div>
          </div>

          {/* Card 9: Payment Gateway Fees */}
          <div className="bg-white border border-[#bcc9c6] rounded-xl p-4 shadow-xs relative overflow-hidden flex flex-col justify-between">
            <div className="flex items-center justify-between text-[#6d7a77]">
              <span className="text-[11px] font-bold uppercase tracking-wider">Gateway Fees</span>
              <span className="p-1.5 rounded-lg bg-purple-50 text-purple-600">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
                </svg>
              </span>
            </div>
            <div className="mt-2">
              <div className="text-lg font-black text-[#141b2b]">
                {overviewLoading ? <span className="text-gray-300">...</span> : formatCurrency(metrics.payment_gateway_fees, metrics.currency)}
              </div>
              <p className="text-[10px] text-[#6d7a77] mt-0.5">Account 5010 gateway cost</p>
            </div>
          </div>

          {/* Card 10: Available Platform Balance */}
          <div className="bg-white border border-[#0052cc]/30 rounded-xl p-4 shadow-xs relative overflow-hidden flex flex-col justify-between bg-gradient-to-br from-white to-[#0052cc]/5">
            <div className="flex items-center justify-between text-[#0052cc]">
              <span className="text-[11px] font-bold uppercase tracking-wider">Platform Net Profit</span>
              <span className="p-1.5 rounded-lg bg-[#0052cc]/10 text-[#0052cc]">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </span>
            </div>
            <div className="mt-2">
              <div className="text-lg font-black text-[#0052cc]">
                {overviewLoading ? <span className="text-gray-300">...</span> : formatCurrency(metrics.available_platform_balance, metrics.currency)}
              </div>
              <p className="text-[10px] text-[#6d7a77] mt-0.5">Revenue minus gateway fees</p>
            </div>
          </div>
        </div>
      </div>

      {/* Comprehensive Filter Toolbar */}
      <div className="bg-white border border-[#bcc9c6] rounded-xl p-4 shadow-xs space-y-4">
        {/* Date Presets Row */}
        <div className="flex flex-wrap items-center gap-1.5 pb-2 border-b border-[#eef0fb]">
          <span className="text-[11px] font-semibold text-[#6d7a77] uppercase tracking-wider mr-2">
            Period:
          </span>
          {DATE_PRESETS.map((p) => (
            <button
              key={p.id}
              type="button"
              onClick={() => handleSelectPreset(p.id)}
              className={`px-2.5 py-1 text-xs font-semibold rounded-lg transition-colors cursor-pointer ${
                activePreset === p.id
                  ? "bg-[#0052cc] text-white shadow-xs"
                  : "bg-[#f9f9ff] text-[#141b2b] hover:bg-[#eef0fb] border border-[#dce2f7]"
              }`}
            >
              {p.label}
            </button>
          ))}
        </div>

        {/* Filter Inputs Grid */}
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-7 gap-3">
          {/* Custom Date Range From */}
          <div>
            <span className="text-[11px] font-semibold text-[#6d7a77] uppercase tracking-wider block mb-1">
              From Date
            </span>
            <input
              type="date"
              value={fromDate}
              onChange={(e) => {
                setFromDate(e.target.value);
                setActivePreset("CUSTOM");
                setPage(1);
              }}
              className="w-full text-xs bg-[#f9f9ff] border border-[#dce2f7] rounded-lg px-2.5 py-2 text-[#141b2b] focus:outline-hidden focus:border-[#0052cc]"
            />
          </div>

          {/* Custom Date Range To */}
          <div>
            <span className="text-[11px] font-semibold text-[#6d7a77] uppercase tracking-wider block mb-1">
              To Date
            </span>
            <input
              type="date"
              value={toDate}
              onChange={(e) => {
                setToDate(e.target.value);
                setActivePreset("CUSTOM");
                setPage(1);
              }}
              className="w-full text-xs bg-[#f9f9ff] border border-[#dce2f7] rounded-lg px-2.5 py-2 text-[#141b2b] focus:outline-hidden focus:border-[#0052cc]"
            />
          </div>

          {/* Event Filter */}
          <div>
            <span className="text-[11px] font-semibold text-[#6d7a77] uppercase tracking-wider block mb-1">
              Event
            </span>
            <select
              value={selectedEventId}
              onChange={(e) => {
                setSelectedEventId(e.target.value);
                setPage(1);
              }}
              className="w-full text-xs bg-[#f9f9ff] border border-[#dce2f7] rounded-lg px-2.5 py-2 text-[#141b2b] focus:outline-hidden focus:border-[#0052cc] cursor-pointer"
            >
              <option value="">All Events</option>
              {availableEvents.map((ev) => (
                <option key={ev.id} value={ev.id}>
                  {ev.title}
                </option>
              ))}
            </select>
          </div>

          {/* Organizer Filter */}
          <div>
            <span className="text-[11px] font-semibold text-[#6d7a77] uppercase tracking-wider block mb-1">
              Organizer
            </span>
            <select
              value={selectedOrganizerId}
              onChange={(e) => {
                setSelectedOrganizerId(e.target.value);
                setPage(1);
              }}
              className="w-full text-xs bg-[#f9f9ff] border border-[#dce2f7] rounded-lg px-2.5 py-2 text-[#141b2b] focus:outline-hidden focus:border-[#0052cc] cursor-pointer"
            >
              <option value="">All Organizers</option>
              {availableOrganizers.map((org) => (
                <option key={org.id} value={org.id}>
                  {org.name || org.email}
                </option>
              ))}
            </select>
          </div>

          {/* Transaction Type Filter */}
          <div>
            <span className="text-[11px] font-semibold text-[#6d7a77] uppercase tracking-wider block mb-1">
              Tx Type
            </span>
            <select
              value={transactionType}
              onChange={(e) => {
                setTransactionType(e.target.value);
                setPage(1);
              }}
              className="w-full text-xs bg-[#f9f9ff] border border-[#dce2f7] rounded-lg px-2.5 py-2 text-[#141b2b] focus:outline-hidden focus:border-[#0052cc] cursor-pointer"
            >
              {TRANSACTION_TYPE_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>

          {/* Settlement Status Filter */}
          <div>
            <span className="text-[11px] font-semibold text-[#6d7a77] uppercase tracking-wider block mb-1">
              Settlement Status
            </span>
            <select
              value={settlementStatus}
              onChange={(e) => {
                setSettlementStatus(e.target.value);
                setPage(1);
              }}
              className="w-full text-xs bg-[#f9f9ff] border border-[#dce2f7] rounded-lg px-2.5 py-2 text-[#141b2b] focus:outline-hidden focus:border-[#0052cc] cursor-pointer"
            >
              {SETTLEMENT_STATUS_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>

          {/* Journal Status Filter */}
          <div>
            <span className="text-[11px] font-semibold text-[#6d7a77] uppercase tracking-wider block mb-1">
              Journal Status
            </span>
            <select
              value={statusFilter}
              onChange={(e) => {
                setStatusFilter(e.target.value);
                setPage(1);
              }}
              className="w-full text-xs bg-[#f9f9ff] border border-[#dce2f7] rounded-lg px-2.5 py-2 text-[#141b2b] focus:outline-hidden focus:border-[#0052cc] cursor-pointer"
            >
              {STATUS_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>
        </div>

        {/* Filter status row & Clear */}
        <div className="flex items-center justify-between text-xs pt-1 border-t border-[#eef0fb]">
          <div className="flex items-center gap-2">
            <span className="text-[#6d7a77]">
              Active Filters: {hasActiveFilters ? "Applied" : "None"}
            </span>
          </div>
          {hasActiveFilters && (
            <button
              type="button"
              onClick={handleResetFilters}
              className="text-[#0052cc] hover:underline font-semibold text-xs cursor-pointer"
            >
              Clear all filters
            </button>
          )}
        </div>
      </div>

      {/* Tabs Navigation */}
      <div className="flex border-b border-[#bcc9c6] gap-2">
        <button
          type="button"
          onClick={() => setActiveTab("journals")}
          className={`py-2.5 px-4 text-xs font-bold border-b-2 transition-colors cursor-pointer flex items-center gap-2 ${
            activeTab === "journals"
              ? "border-[#0052cc] text-[#0052cc]"
              : "border-transparent text-[#6d7a77] hover:text-[#141b2b]"
          }`}
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          General Ledger Journals ({totalRecords})
        </button>
        <button
          type="button"
          onClick={() => setActiveTab("events")}
          className={`py-2.5 px-4 text-xs font-bold border-b-2 transition-colors cursor-pointer flex items-center gap-2 ${
            activeTab === "events"
              ? "border-[#0052cc] text-[#0052cc]"
              : "border-transparent text-[#6d7a77] hover:text-[#141b2b]"
          }`}
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          Per-Event Breakdown ({overviewData?.breakdown_by_event?.length || 0})
        </button>
        <button
          type="button"
          onClick={() => setActiveTab("organizers")}
          className={`py-2.5 px-4 text-xs font-bold border-b-2 transition-colors cursor-pointer flex items-center gap-2 ${
            activeTab === "organizers"
              ? "border-[#0052cc] text-[#0052cc]"
              : "border-transparent text-[#6d7a77] hover:text-[#141b2b]"
          }`}
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
          </svg>
          Per-Organizer Breakdown ({overviewData?.breakdown_by_organizer?.length || 0})
        </button>
      </div>

      {/* Tab 1: General Ledger Journals */}
      {activeTab === "journals" && (
        <div className="space-y-4">
          {/* Search bar inside journals view */}
          <div className="flex items-center justify-between gap-4">
            <div className="relative w-full max-w-md">
              <input
                type="text"
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                placeholder="Search Reference ID, Journal ID, description..."
                className="w-full text-xs bg-[#f9f9ff] border border-[#dce2f7] rounded-lg pl-8 pr-3 py-2 text-[#141b2b] focus:outline-hidden focus:border-[#0052cc]"
              />
              <svg
                className="w-4 h-4 text-[#6d7a77] absolute left-2.5 top-2.5 pointer-events-none"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                />
              </svg>
            </div>
            <div className="text-xs text-[#6d7a77]">
              Showing <span className="font-semibold text-[#141b2b]">{journals.length}</span> of{" "}
              <span className="font-semibold text-[#141b2b]">{totalRecords}</span> entries
            </div>
          </div>

          {/* Journals Error Banner */}
          {error && (
            <div className="p-4 rounded-xl bg-[#fee2e2]/60 border border-[#fca5a5] flex items-center justify-between">
              <div className="flex items-center gap-2 text-xs text-[#991b1b] font-medium">
                <svg className="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <span>{error}</span>
              </div>
              <button
                type="button"
                onClick={() => setRefreshKey((k) => k + 1)}
                className="text-xs font-semibold px-2.5 py-1 bg-[#b91c1c] text-white rounded-md hover:bg-[#991b1b] transition-colors cursor-pointer"
              >
                Retry
              </button>
            </div>
          )}

          {/* Journals Table */}
          <div className="bg-white border border-[#bcc9c6] rounded-xl overflow-hidden shadow-xs">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs">
                <thead className="bg-[#f9f9ff] border-b border-[#dce2f7] text-[#6d7a77] font-semibold">
                  <tr>
                    <th className="py-3 px-4">Journal ID</th>
                    <th className="py-3 px-4">Date / Time</th>
                    <th className="py-3 px-4">Type</th>
                    <th className="py-3 px-4">Reference ID</th>
                    <th className="py-3 px-4">Description</th>
                    <th className="py-3 px-4 text-right">Debit / Credit</th>
                    <th className="py-3 px-4 text-center">Status</th>
                    <th className="py-3 px-4 text-center">Action</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#eef0fb]">
                  {loading ? (
                    Array.from({ length: 5 }).map((_, i) => (
                      <tr key={i} className="animate-pulse">
                        <td className="py-3.5 px-4"><div className="h-4 bg-gray-100 rounded w-24" /></td>
                        <td className="py-3.5 px-4"><div className="h-4 bg-gray-100 rounded w-28" /></td>
                        <td className="py-3.5 px-4"><div className="h-4 bg-gray-100 rounded w-16" /></td>
                        <td className="py-3.5 px-4"><div className="h-4 bg-gray-100 rounded w-24" /></td>
                        <td className="py-3.5 px-4"><div className="h-4 bg-gray-100 rounded w-36" /></td>
                        <td className="py-3.5 px-4 text-right"><div className="h-4 bg-gray-100 rounded w-20 ml-auto" /></td>
                        <td className="py-3.5 px-4 text-center"><div className="h-4 bg-gray-100 rounded w-16 mx-auto" /></td>
                        <td className="py-3.5 px-4 text-center"><div className="h-4 bg-gray-100 rounded w-16 mx-auto" /></td>
                      </tr>
                    ))
                  ) : journals.length > 0 ? (
                    journals.map((journal) => {
                      const txBadge =
                        txBadgeStyles[journal.transaction_type?.toUpperCase()] ||
                        "bg-[#f3f4f6] text-[#4b5563] border border-[#e5e7eb]";
                      const statusBadge =
                        statusBadgeStyles[journal.status?.toUpperCase()] ||
                        "bg-[#f3f4f6] text-[#4b5563] border border-[#e5e7eb]";

                      return (
                        <tr
                          key={journal.id}
                          onClick={() => handleOpenDrawer(journal.id)}
                          className="hover:bg-[#f9f9ff] transition-colors cursor-pointer group"
                        >
                          <td className="py-3 px-4 font-mono text-[11px] text-[#141b2b] whitespace-nowrap">
                            <div className="flex items-center gap-1.5">
                              <span title={journal.id}>{journal.id.substring(0, 8)}...</span>
                              <button
                                type="button"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  copyText(journal.id, "Journal ID");
                                }}
                                className="opacity-0 group-hover:opacity-100 text-[#0052cc] hover:underline cursor-pointer transition-opacity"
                                title="Copy full UUID"
                              >
                                <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                                </svg>
                              </button>
                            </div>
                          </td>
                          <td className="py-3 px-4 text-[#6d7a77] whitespace-nowrap">
                            {formatDateTime(journal.posted_at || journal.created_at)}
                          </td>
                          <td className="py-3 px-4 whitespace-nowrap">
                            <div className="flex items-center gap-1">
                              <span className={`px-2 py-0.5 rounded-full text-[10px] font-bold ${txBadge}`}>
                                {journal.transaction_type}
                              </span>
                              {journal.is_reversed && (
                                <span
                                  className="px-1.5 py-0.5 rounded-full text-[9px] font-bold bg-[#fee2e2] text-[#991b1b] border border-[#fca5a5]"
                                  title="Reversed by another journal"
                                >
                                  REV
                                </span>
                              )}
                            </div>
                          </td>
                          <td className="py-3 px-4 font-mono text-[11px] text-[#141b2b] whitespace-nowrap">
                            <div className="flex items-center gap-1.5">
                              <span>{journal.source_id || "—"}</span>
                              {journal.source_id && (
                                <button
                                  type="button"
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    copyText(journal.source_id, "Reference ID");
                                  }}
                                  className="opacity-0 group-hover:opacity-100 text-[#0052cc] hover:underline cursor-pointer transition-opacity"
                                  title="Copy Reference ID"
                                >
                                  <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                                  </svg>
                                </button>
                              )}
                            </div>
                          </td>
                          <td className="py-3 px-4 text-[#141b2b] max-w-xs truncate" title={journal.description}>
                            {journal.description || "—"}
                          </td>
                          <td className="py-3 px-4 text-right font-mono font-medium text-[#141b2b] whitespace-nowrap">
                            {formatCurrency(journal.total_debit, journal.currency)}
                          </td>
                          <td className="py-3 px-4 text-center whitespace-nowrap">
                            <div className="flex flex-col items-center gap-1">
                              <span className={`px-2 py-0.5 rounded-full text-[10px] font-bold ${statusBadge}`}>
                                {journal.status}
                              </span>
                              {journal.is_balanced ? (
                                <span className="text-[9px] font-semibold text-[#15803d]">Balanced</span>
                              ) : (
                                <span className="text-[9px] font-semibold text-[#b91c1c]">Unbalanced</span>
                              )}
                            </div>
                          </td>
                          <td className="py-3 px-4 text-center whitespace-nowrap">
                            <button
                              type="button"
                              onClick={(e) => {
                                e.stopPropagation();
                                handleOpenDrawer(journal.id);
                              }}
                              className="px-2.5 py-1 text-[11px] font-semibold rounded bg-[#eef0fb] text-[#0052cc] hover:bg-[#dce2f7] transition-colors cursor-pointer"
                            >
                              View Entries
                            </button>
                          </td>
                        </tr>
                      );
                    })
                  ) : (
                    <tr>
                      <td colSpan="8" className="py-12 text-center text-[#6d7a77]">
                        <div className="flex flex-col items-center justify-center space-y-2">
                          <svg className="w-8 h-8 text-[#bcc9c6]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                          </svg>
                          <p className="font-semibold text-sm text-[#141b2b]">
                            {hasActiveFilters ? "No matching journal entries found" : "No general ledger journals recorded"}
                          </p>
                          <p className="text-xs text-[#6d7a77] max-w-sm">
                            {hasActiveFilters
                              ? "Try adjusting your filters, search keyword, or date range to find journal transactions."
                              : "Financial transactions will appear here automatically when payments, refunds, and settlements are captured."}
                          </p>
                          {hasActiveFilters && (
                            <button
                              type="button"
                              onClick={handleResetFilters}
                              className="mt-2 px-3 py-1 text-xs font-semibold rounded-lg bg-[#0052cc] text-white hover:bg-[#0747a6] transition-colors cursor-pointer"
                            >
                              Clear Filters
                            </button>
                          )}
                        </div>
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>

            {/* Pagination Footer */}
            <div className="bg-[#f9f9ff] border-t border-[#dce2f7] px-4 py-3 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 text-xs">
              <div className="flex items-center gap-2">
                <span className="text-[#6d7a77]">Rows per page:</span>
                <select
                  value={limit}
                  onChange={(e) => {
                    setLimit(Number(e.target.value));
                    setPage(1);
                  }}
                  className="bg-white border border-[#dce2f7] rounded-md px-2 py-1 text-xs text-[#141b2b] focus:outline-hidden cursor-pointer"
                >
                  {PAGE_SIZE_OPTIONS.map((size) => (
                    <option key={size} value={size}>
                      {size}
                    </option>
                  ))}
                </select>
              </div>

              <div className="flex items-center gap-4">
                <span className="text-[#6d7a77]">
                  Page <span className="font-semibold text-[#141b2b]">{page}</span> of{" "}
                  <span className="font-semibold text-[#141b2b]">{totalPages}</span>
                </span>

                <div className="flex items-center gap-1">
                  <button
                    type="button"
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                    disabled={page <= 1 || loading}
                    className="p-1.5 rounded-md border border-[#dce2f7] bg-white text-[#141b2b] hover:bg-[#eef0fb] disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer transition-colors"
                    title="Previous page"
                  >
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
                    </svg>
                  </button>

                  <button
                    type="button"
                    onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                    disabled={page >= totalPages || loading}
                    className="p-1.5 rounded-md border border-[#dce2f7] bg-white text-[#141b2b] hover:bg-[#eef0fb] disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer transition-colors"
                    title="Next page"
                  >
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                    </svg>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Tab 2: Per-Event Breakdown */}
      {activeTab === "events" && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <span className="text-xs text-[#6d7a77]">
              Financial activity grouped by event. Shows collections, organizer net payable, platform commission, taxes, and reserves.
            </span>
          </div>

          <div className="bg-white border border-[#bcc9c6] rounded-xl overflow-hidden shadow-xs">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs">
                <thead className="bg-[#f9f9ff] border-b border-[#dce2f7] text-[#6d7a77] font-semibold">
                  <tr>
                    <th className="py-3 px-4">Event Details</th>
                    <th className="py-3 px-4">Organizer</th>
                    <th className="py-3 px-4 text-center">Lifecycle</th>
                    <th className="py-3 px-4 text-right">Gross Captured</th>
                    <th className="py-3 px-4 text-right">Refunds</th>
                    <th className="py-3 px-4 text-right">Net Captured</th>
                    <th className="py-3 px-4 text-right">Org Payable</th>
                    <th className="py-3 px-4 text-right">Platform Fee</th>
                    <th className="py-3 px-4 text-right">Tax Collected</th>
                    <th className="py-3 px-4 text-right">Settled</th>
                    <th className="py-3 px-4 text-right">Pending Reserve</th>
                    <th className="py-3 px-4 text-center">Action</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#eef0fb]">
                  {overviewLoading ? (
                    Array.from({ length: 4 }).map((_, i) => (
                      <tr key={i} className="animate-pulse">
                        <td className="py-3.5 px-4"><div className="h-4 bg-gray-100 rounded w-32" /></td>
                        <td className="py-3.5 px-4"><div className="h-4 bg-gray-100 rounded w-24" /></td>
                        <td className="py-3.5 px-4 text-center"><div className="h-4 bg-gray-100 rounded w-16 mx-auto" /></td>
                        <td className="py-3.5 px-4 text-right"><div className="h-4 bg-gray-100 rounded w-16 ml-auto" /></td>
                        <td className="py-3.5 px-4 text-right"><div className="h-4 bg-gray-100 rounded w-16 ml-auto" /></td>
                        <td className="py-3.5 px-4 text-right"><div className="h-4 bg-gray-100 rounded w-16 ml-auto" /></td>
                        <td className="py-3.5 px-4 text-right"><div className="h-4 bg-gray-100 rounded w-16 ml-auto" /></td>
                        <td className="py-3.5 px-4 text-right"><div className="h-4 bg-gray-100 rounded w-16 ml-auto" /></td>
                        <td className="py-3.5 px-4 text-right"><div className="h-4 bg-gray-100 rounded w-16 ml-auto" /></td>
                        <td className="py-3.5 px-4 text-right"><div className="h-4 bg-gray-100 rounded w-16 ml-auto" /></td>
                        <td className="py-3.5 px-4 text-right"><div className="h-4 bg-gray-100 rounded w-16 ml-auto" /></td>
                        <td className="py-3.5 px-4 text-center"><div className="h-4 bg-gray-100 rounded w-16 mx-auto" /></td>
                      </tr>
                    ))
                  ) : overviewData?.breakdown_by_event && overviewData.breakdown_by_event.length > 0 ? (
                    overviewData.breakdown_by_event.map((item) => {
                      const lifecycleBadge =
                        lifecycleBadgeStyles[item.financial_lifecycle?.toUpperCase()] ||
                        "bg-gray-100 text-gray-700 border-gray-300";

                      return (
                        <tr key={item.event_id} className="hover:bg-[#f9f9ff] transition-colors">
                          <td className="py-3 px-4">
                            <div className="font-semibold text-[#141b2b]">{item.event_title}</div>
                            <div className="font-mono text-[10px] text-[#6d7a77]">{item.event_id.substring(0, 8)}...</div>
                          </td>
                          <td className="py-3 px-4 text-[#141b2b]">
                            <div>{item.organizer_name}</div>
                          </td>
                          <td className="py-3 px-4 text-center whitespace-nowrap">
                            <span className={`px-2 py-0.5 rounded-full text-[10px] font-bold border ${lifecycleBadge}`}>
                              {item.financial_lifecycle}
                            </span>
                          </td>
                          <td className="py-3 px-4 text-right font-mono font-medium text-[#141b2b] whitespace-nowrap">
                            {formatCurrency(item.gross_captured, item.currency)}
                          </td>
                          <td className="py-3 px-4 text-right font-mono font-medium text-rose-600 whitespace-nowrap">
                            {formatCurrency(item.refunds_amount, item.currency)}
                          </td>
                          <td className="py-3 px-4 text-right font-mono font-semibold text-[#141b2b] whitespace-nowrap">
                            {formatCurrency(item.net_captured, item.currency)}
                          </td>
                          <td className="py-3 px-4 text-right font-mono font-medium text-teal-700 whitespace-nowrap">
                            {formatCurrency(item.net_organizer_payable, item.currency)}
                          </td>
                          <td className="py-3 px-4 text-right font-mono font-medium text-indigo-600 whitespace-nowrap">
                            {formatCurrency(item.platform_fee, item.currency)}
                          </td>
                          <td className="py-3 px-4 text-right font-mono font-medium text-amber-700 whitespace-nowrap">
                            {formatCurrency(item.tax_collected, item.currency)}
                          </td>
                          <td className="py-3 px-4 text-right font-mono font-medium text-sky-700 whitespace-nowrap">
                            {formatCurrency(item.settled_amount, item.currency)}
                          </td>
                          <td className="py-3 px-4 text-right font-mono font-medium text-orange-600 whitespace-nowrap">
                            {formatCurrency(item.pending_reserve, item.currency)}
                          </td>
                          <td className="py-3 px-4 text-center whitespace-nowrap">
                            <button
                              type="button"
                              onClick={() => {
                                setSelectedEventId(item.event_id);
                                setActiveTab("journals");
                                setPage(1);
                              }}
                              className="px-2.5 py-1 text-[11px] font-semibold rounded bg-[#eef0fb] text-[#0052cc] hover:bg-[#dce2f7] transition-colors cursor-pointer"
                              title="Filter ledger journals by this event"
                            >
                              Drill Down
                            </button>
                          </td>
                        </tr>
                      );
                    })
                  ) : (
                    <tr>
                      <td colSpan="12" className="py-10 text-center text-[#6d7a77]">
                        No event financial breakdowns recorded for the selected filter criteria.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}

      {/* Tab 3: Per-Organizer Breakdown */}
      {activeTab === "organizers" && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <span className="text-xs text-[#6d7a77]">
              Financial balances grouped by organizer. Tracks total earnings, refund deductions, payouts disbursed, and pending reserves.
            </span>
          </div>

          <div className="bg-white border border-[#bcc9c6] rounded-xl overflow-hidden shadow-xs">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs">
                <thead className="bg-[#f9f9ff] border-b border-[#dce2f7] text-[#6d7a77] font-semibold">
                  <tr>
                    <th className="py-3 px-4">Organizer</th>
                    <th className="py-3 px-4">Email</th>
                    <th className="py-3 px-4 text-center">Events</th>
                    <th className="py-3 px-4 text-right">Gross Earnings</th>
                    <th className="py-3 px-4 text-right">Refunds Deducted</th>
                    <th className="py-3 px-4 text-right">Net Earnings</th>
                    <th className="py-3 px-4 text-right">Settled Amount</th>
                    <th className="py-3 px-4 text-right">Pending Reserves</th>
                    <th className="py-3 px-4 text-right font-bold text-[#0052cc]">Available Balance</th>
                    <th className="py-3 px-4 text-center">Action</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#eef0fb]">
                  {overviewLoading ? (
                    Array.from({ length: 4 }).map((_, i) => (
                      <tr key={i} className="animate-pulse">
                        <td className="py-3.5 px-4"><div className="h-4 bg-gray-100 rounded w-28" /></td>
                        <td className="py-3.5 px-4"><div className="h-4 bg-gray-100 rounded w-36" /></td>
                        <td className="py-3.5 px-4 text-center"><div className="h-4 bg-gray-100 rounded w-10 mx-auto" /></td>
                        <td className="py-3.5 px-4 text-right"><div className="h-4 bg-gray-100 rounded w-16 ml-auto" /></td>
                        <td className="py-3.5 px-4 text-right"><div className="h-4 bg-gray-100 rounded w-16 ml-auto" /></td>
                        <td className="py-3.5 px-4 text-right"><div className="h-4 bg-gray-100 rounded w-16 ml-auto" /></td>
                        <td className="py-3.5 px-4 text-right"><div className="h-4 bg-gray-100 rounded w-16 ml-auto" /></td>
                        <td className="py-3.5 px-4 text-right"><div className="h-4 bg-gray-100 rounded w-16 ml-auto" /></td>
                        <td className="py-3.5 px-4 text-right"><div className="h-4 bg-gray-100 rounded w-16 ml-auto" /></td>
                        <td className="py-3.5 px-4 text-center"><div className="h-4 bg-gray-100 rounded w-16 mx-auto" /></td>
                      </tr>
                    ))
                  ) : overviewData?.breakdown_by_organizer && overviewData.breakdown_by_organizer.length > 0 ? (
                    overviewData.breakdown_by_organizer.map((item) => (
                      <tr key={item.organizer_id} className="hover:bg-[#f9f9ff] transition-colors">
                        <td className="py-3 px-4">
                          <div className="font-semibold text-[#141b2b]">{item.organizer_name}</div>
                          <div className="font-mono text-[10px] text-[#6d7a77]">{item.organizer_id.substring(0, 8)}...</div>
                        </td>
                        <td className="py-3 px-4 text-[#6d7a77] whitespace-nowrap">
                          {item.organizer_email || "—"}
                        </td>
                        <td className="py-3 px-4 text-center font-bold text-[#141b2b]">
                          {item.events_count}
                        </td>
                        <td className="py-3 px-4 text-right font-mono font-medium text-[#141b2b] whitespace-nowrap">
                          {formatCurrency(item.gross_earnings, item.currency)}
                        </td>
                        <td className="py-3 px-4 text-right font-mono font-medium text-rose-600 whitespace-nowrap">
                          {formatCurrency(item.refunds_deducted, item.currency)}
                        </td>
                        <td className="py-3 px-4 text-right font-mono font-semibold text-[#141b2b] whitespace-nowrap">
                          {formatCurrency(item.net_earnings, item.currency)}
                        </td>
                        <td className="py-3 px-4 text-right font-mono font-medium text-sky-700 whitespace-nowrap">
                          {formatCurrency(item.settled_amount, item.currency)}
                        </td>
                        <td className="py-3 px-4 text-right font-mono font-medium text-orange-600 whitespace-nowrap">
                          {formatCurrency(item.pending_reserves, item.currency)}
                        </td>
                        <td className="py-3 px-4 text-right font-mono font-bold text-emerald-700 whitespace-nowrap">
                          {formatCurrency(item.available_balance, item.currency)}
                        </td>
                        <td className="py-3 px-4 text-center whitespace-nowrap">
                          <button
                            type="button"
                            onClick={() => {
                              setSelectedOrganizerId(item.organizer_id);
                              setActiveTab("journals");
                              setPage(1);
                            }}
                            className="px-2.5 py-1 text-[11px] font-semibold rounded bg-[#eef0fb] text-[#0052cc] hover:bg-[#dce2f7] transition-colors cursor-pointer"
                            title="Filter ledger journals by this organizer"
                          >
                            Drill Down
                          </button>
                        </td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td colSpan="10" className="py-10 text-center text-[#6d7a77]">
                        No organizer financial breakdowns recorded for the selected filter criteria.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}

      {/* Slide-over Detail Drawer */}
      <LedgerJournalDetailDrawer
        journalId={selectedJournalId}
        isOpen={isDrawerOpen}
        onClose={handleCloseDrawer}
        onSelectJournal={(id) => {
          setSelectedJournalId(id);
          setIsDrawerOpen(true);
        }}
      />
    </div>
  );
}
