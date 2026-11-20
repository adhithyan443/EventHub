import { useState, useEffect } from "react";
import { getAdminLedgerJournals } from "../../api/adminLedgerApi";
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

export default function AdminLedgerPage() {
  const [journals, setJournals] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [totalRecords, setTotalRecords] = useState(0);
  const [totalPages, setTotalPages] = useState(1);

  // Filters & Pagination State
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useState(10);
  const [transactionType, setTransactionType] = useState("ALL");
  const [statusFilter, setStatusFilter] = useState("ALL");
  const [searchInput, setSearchInput] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [fromDate, setFromDate] = useState("");
  const [toDate, setToDate] = useState("");
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

  // Fetch journals
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
  }, [page, limit, transactionType, statusFilter, debouncedSearch, fromDate, toDate, refreshKey]);

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
    fromDate !== "" ||
    toDate !== "";

  return (
    <div className="p-8 space-y-6">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-black text-[#141b2b] tracking-tight">Financial Ledger</h1>
          <p className="text-xs text-[#6d7a77] mt-1">
            Authoritative double-entry general ledger journal audit trail and account line items
          </p>
        </div>
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={() => setRefreshKey((k) => k + 1)}
            disabled={loading}
            className="inline-flex items-center gap-2 px-3.5 py-2 text-xs font-semibold rounded-lg bg-white border border-[#bcc9c6] text-[#141b2b] hover:bg-[#eef0fb] transition-colors cursor-pointer shadow-xs disabled:opacity-50"
          >
            <svg
              className={`w-3.5 h-3.5 ${loading ? "animate-spin" : ""}`}
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

      {/* Filter Toolbar */}
      <div className="bg-white border border-[#bcc9c6] rounded-xl p-4 shadow-xs space-y-3">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
          {/* Search Input */}
          <div className="relative">
            <span className="text-[11px] font-semibold text-[#6d7a77] uppercase tracking-wider block mb-1">
              Search
            </span>
            <div className="relative">
              <input
                type="text"
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                placeholder="Reference ID, Journal ID, description..."
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
          </div>

          {/* Transaction Type Filter */}
          <div>
            <span className="text-[11px] font-semibold text-[#6d7a77] uppercase tracking-wider block mb-1">
              Transaction Type
            </span>
            <select
              value={transactionType}
              onChange={(e) => {
                setTransactionType(e.target.value);
                setPage(1);
              }}
              className="w-full text-xs bg-[#f9f9ff] border border-[#dce2f7] rounded-lg px-3 py-2 text-[#141b2b] focus:outline-hidden focus:border-[#0052cc] cursor-pointer"
            >
              {TRANSACTION_TYPE_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>

          {/* Status Filter */}
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
              className="w-full text-xs bg-[#f9f9ff] border border-[#dce2f7] rounded-lg px-3 py-2 text-[#141b2b] focus:outline-hidden focus:border-[#0052cc] cursor-pointer"
            >
              {STATUS_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>

          {/* Date Range Inputs */}
          <div>
            <span className="text-[11px] font-semibold text-[#6d7a77] uppercase tracking-wider block mb-1">
              Date Range (Posted)
            </span>
            <div className="flex items-center gap-1.5">
              <input
                type="date"
                value={fromDate}
                onChange={(e) => {
                  setFromDate(e.target.value);
                  setPage(1);
                }}
                className="w-1/2 text-xs bg-[#f9f9ff] border border-[#dce2f7] rounded-lg px-2 py-2 text-[#141b2b] focus:outline-hidden focus:border-[#0052cc]"
                title="From Date"
              />
              <span className="text-[#6d7a77] text-xs font-bold">—</span>
              <input
                type="date"
                value={toDate}
                onChange={(e) => {
                  setToDate(e.target.value);
                  setPage(1);
                }}
                className="w-1/2 text-xs bg-[#f9f9ff] border border-[#dce2f7] rounded-lg px-2 py-2 text-[#141b2b] focus:outline-hidden focus:border-[#0052cc]"
                title="To Date"
              />
            </div>
          </div>
        </div>

        {/* Filter status row & Clear */}
        <div className="flex items-center justify-between text-xs pt-1 border-t border-[#eef0fb]">
          <span className="text-[#6d7a77]">
            Showing <span className="font-semibold text-[#141b2b]">{journals.length}</span> of{" "}
            <span className="font-semibold text-[#141b2b]">{totalRecords}</span> journal records
          </span>
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

      {/* Error Banner */}
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

        {/* Server-side Pagination Footer */}
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
