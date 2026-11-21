import { useState, useEffect } from "react";
import {
  getAdminSettlements,
  approveAdminSettlement,
  rejectAdminSettlement,
  retryAdminSettlement,
} from "../../api/adminSettlementApi";
import AdminSettlementDetailDrawer from "../../components/admin/AdminSettlementDetailDrawer";
import ApproveSettlementModal from "../../components/admin/ApproveSettlementModal";
import RejectSettlementModal from "../../components/admin/RejectSettlementModal";
import RetrySettlementModal from "../../components/admin/RetrySettlementModal";
import notify from "../../utils/toast";

const STATUS_FILTERS = [
  { value: "ALL", label: "All Statuses" },
  { value: "PENDING", label: "Pending Review" },
  { value: "APPROVED", label: "Approved" },
  { value: "PROCESSING", label: "Processing" },
  { value: "COMPLETED", label: "Completed" },
  { value: "REJECTED", label: "Rejected" },
  { value: "FAILED", label: "Failed" },
];

const PAGE_SIZE_OPTIONS = [10, 25, 50, 100];

const statusBadgeStyles = {
  PENDING: "bg-[#fef9c3] text-[#854d0e] border border-[#fef08a]",
  APPROVED: "bg-[#eff6ff] text-[#1d4ed8] border border-[#bfdbfe]",
  PROCESSING: "bg-[#e0e7ff] text-[#4338ca] border border-[#c7d2fe]",
  COMPLETED: "bg-[#dcfce7] text-[#15803d] border border-[#bbf7d0]",
  REJECTED: "bg-[#fee2e2] text-[#b91c1c] border border-[#fecaca]",
  FAILED: "bg-[#fff1f2] text-[#be123c] border border-[#fecdd3]",
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

export default function AdminSettlementsPage() {
  const [settlements, setSettlements] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [totalRecords, setTotalRecords] = useState(0);
  const [totalPages, setTotalPages] = useState(1);

  // Filters & Pagination
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useState(10);
  const [statusFilter, setStatusFilter] = useState("ALL");
  const [organizerIdFilter, setOrganizerIdFilter] = useState("");
  const [debouncedOrganizerId, setDebouncedOrganizerId] = useState("");
  const [refreshTrigger, setRefreshTrigger] = useState(0);

  // Modals & Drawer State
  const [activeDrawerSettlement, setActiveDrawerSettlement] = useState(null);
  const [settlementToApprove, setSettlementToApprove] = useState(null);
  const [settlementToReject, setSettlementToReject] = useState(null);
  const [settlementToRetry, setSettlementToRetry] = useState(null);
  const [actionLoading, setActionLoading] = useState(false);

  // Debounce organizer UUID input
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedOrganizerId(organizerIdFilter.trim());
      setPage(1);
    }, 400);
    return () => clearTimeout(timer);
  }, [organizerIdFilter]);

  // Fetch Settlements
  useEffect(() => {
    let cancelled = false;
    const timer = setTimeout(async () => {
      setLoading(true);
      setError("");
      try {
        const statusParam = statusFilter !== "ALL" ? statusFilter : null;
        const organizerParam = debouncedOrganizerId || null;
        const res = await getAdminSettlements(page, limit, statusParam, organizerParam);
        if (cancelled) return;
        if (res.success && res.data) {
          setSettlements(res.data.settlements || []);
          const total = res.data.total || 0;
          setTotalRecords(total);
          setTotalPages(Math.max(1, Math.ceil(total / limit)));
        } else {
          throw new Error(res.message || "Failed to load settlements");
        }
      } catch (err) {
        if (cancelled) return;
        console.error("Admin failed to load settlements:", err);
        const msg = err.response?.data?.message || err.message || "Failed to load settlements.";
        setError(msg);
        setSettlements([]);
        setTotalRecords(0);
        setTotalPages(1);
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }, 0);

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [page, limit, statusFilter, debouncedOrganizerId, refreshTrigger]);

  // Action: Approve
  const handleConfirmApprove = async () => {
    if (!settlementToApprove) return;
    setActionLoading(true);
    try {
      const res = await approveAdminSettlement(settlementToApprove.id);
      if (res.success) {
        notify.success("Settlement approved and payout initiated successfully!");
        setSettlementToApprove(null);
        if (activeDrawerSettlement?.id === settlementToApprove.id) {
          setActiveDrawerSettlement(null);
        }
        setRefreshTrigger((k) => k + 1);
      } else {
        throw new Error(res.message || "Failed to approve settlement");
      }
    } catch (err) {
      console.error("Approve failed:", err);
      const msg = err.response?.data?.message || err.message || "Failed to approve settlement.";
      notify.error(msg);
    } finally {
      setActionLoading(false);
    }
  };

  // Action: Reject
  const handleConfirmReject = async (reason) => {
    if (!settlementToReject || actionLoading) return;
    setActionLoading(true);
    try {
      const res = await rejectAdminSettlement(settlementToReject.id, reason);
      if (res.success) {
        notify.success("Settlement rejected and hold released back to organizer.");
        setSettlementToReject(null);
        if (activeDrawerSettlement?.id === settlementToReject.id) {
          setActiveDrawerSettlement(null);
        }
        setRefreshTrigger((k) => k + 1);
      } else {
        throw new Error(res.message || "Failed to reject settlement");
      }
    } catch (err) {
      console.error("Reject failed:", err);
      const msg = err.response?.data?.message || err.message || "Failed to reject settlement.";
      notify.error(msg);
      throw err;
    } finally {
      setActionLoading(false);
    }
  };

  // Action: Retry
  const handleConfirmRetry = async () => {
    if (!settlementToRetry) return;
    setActionLoading(true);
    try {
      const res = await retryAdminSettlement(settlementToRetry.id);
      if (res.success) {
        notify.success("Payout retry initiated successfully!");
        setSettlementToRetry(null);
        if (activeDrawerSettlement?.id === settlementToRetry.id) {
          setActiveDrawerSettlement(null);
        }
        setRefreshTrigger((k) => k + 1);
      } else {
        throw new Error(res.message || "Failed to retry settlement");
      }
    } catch (err) {
      console.error("Retry failed:", err);
      const msg = err.response?.data?.message || err.message || "Failed to retry payout.";
      notify.error(msg);
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <div className="flex-1 flex flex-col bg-[#f8fafc] min-h-screen">
      {/* Top Bar Header */}
      <div className="p-4 sm:p-6 bg-white border-b border-[#e2e8f0] flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-xl sm:text-2xl font-bold text-[#141b2b]">
            Settlement Disbursements
          </h1>
          <p className="text-xs sm:text-sm text-[#64748b] mt-0.5">
            Review organizer payout requests, approve disbursements, and audit ledger settlement journals.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={() => setRefreshTrigger((k) => k + 1)}
            disabled={loading}
            className="bg-white border border-[#cbd5e1] text-[#1e293b] text-xs sm:text-sm font-semibold px-3 py-2 rounded-lg flex items-center gap-1.5 hover:bg-[#f1f5f9] transition-colors cursor-pointer disabled:opacity-50"
            title="Refresh settlements list"
          >
            <svg
              className={`w-3.5 h-3.5 ${loading ? "animate-spin text-[#00685f]" : ""}`}
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

      {/* Main Body */}
      <div className="flex-1 p-4 sm:p-6 space-y-5 max-w-7xl w-full mx-auto">
        {/* Error Alert */}
        {error && (
          <div className="p-4 bg-[#fee2e2] border border-[#fecaca] rounded-xl text-[#b91c1c] text-sm flex items-center justify-between">
            <span>{error}</span>
            <button
              onClick={() => setRefreshTrigger((k) => k + 1)}
              className="text-xs font-bold underline ml-3"
            >
              Retry
            </button>
          </div>
        )}

        {/* Filter Toolbar */}
        <div className="bg-white border border-[#e2e8f0] rounded-xl p-4 shadow-2xs flex flex-col md:flex-row md:items-center justify-between gap-4">
          {/* Status Filters */}
          <div className="flex flex-wrap items-center gap-1.5">
            {STATUS_FILTERS.map((f) => (
              <button
                key={f.value}
                type="button"
                onClick={() => {
                  setStatusFilter(f.value);
                  setPage(1);
                }}
                className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-colors cursor-pointer ${
                  statusFilter === f.value
                    ? "bg-[#00685f] text-white shadow-2xs"
                    : "bg-[#f1f5f9] text-[#64748b] hover:bg-[#e2e8f0] hover:text-[#1e293b]"
                }`}
              >
                {f.label}
              </button>
            ))}
          </div>

          {/* Organizer UUID Search Filter */}
          <div className="flex items-center gap-2">
            <div className="relative w-full sm:w-64">
              <input
                type="text"
                value={organizerIdFilter}
                onChange={(e) => setOrganizerIdFilter(e.target.value)}
                placeholder="Filter by Organizer UUID..."
                className="w-full pl-8 pr-3 py-1.5 border border-[#cbd5e1] rounded-lg text-xs text-[#1e293b] placeholder:text-[#94a3b8] focus:outline-none focus:ring-1 focus:ring-[#00685f] focus:border-[#00685f]"
              />
              <svg
                className="w-4 h-4 text-[#94a3b8] absolute left-2.5 top-1/2 -translate-y-1/2"
                viewBox="0 0 20 20"
                fill="currentColor"
              >
                <path
                  fillRule="evenodd"
                  d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z"
                  clipRule="evenodd"
                />
              </svg>
            </div>
            {organizerIdFilter && (
              <button
                type="button"
                onClick={() => setOrganizerIdFilter("")}
                className="text-xs text-[#64748b] hover:text-[#1e293b] px-2 py-1"
              >
                Clear
              </button>
            )}
          </div>
        </div>

        {/* Settlements Table */}
        <div className="bg-white border border-[#e2e8f0] rounded-2xl shadow-2xs overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse text-xs">
              <thead>
                <tr className="border-b border-[#e2e8f0] bg-[#f8fafc] text-[#64748b] uppercase tracking-wider font-semibold">
                  <th className="py-3 px-4">Settlement ID</th>
                  <th className="py-3 px-4">Organizer</th>
                  <th className="py-3 px-4">Scope</th>
                  <th className="py-3 px-4 text-right">Amount</th>
                  <th className="py-3 px-4 text-center">Status</th>
                  <th className="py-3 px-4">Requested At</th>
                  <th className="py-3 px-4">Payout Ref / Reason</th>
                  <th className="py-3 px-4 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#f1f5f9]">
                {loading ? (
                  Array.from({ length: 6 }).map((_, idx) => (
                    <tr key={idx} className="animate-pulse">
                      <td className="py-3.5 px-4"><div className="h-4 w-24 bg-gray-200 rounded" /></td>
                      <td className="py-3.5 px-4"><div className="h-4 w-24 bg-gray-200 rounded" /></td>
                      <td className="py-3.5 px-4"><div className="h-4 w-20 bg-gray-200 rounded" /></td>
                      <td className="py-3.5 px-4 text-right"><div className="h-4 w-16 bg-gray-200 rounded ml-auto" /></td>
                      <td className="py-3.5 px-4 text-center"><div className="h-5 w-20 bg-gray-200 rounded-full mx-auto" /></td>
                      <td className="py-3.5 px-4"><div className="h-4 w-24 bg-gray-200 rounded" /></td>
                      <td className="py-3.5 px-4"><div className="h-4 w-28 bg-gray-200 rounded" /></td>
                      <td className="py-3.5 px-4 text-right"><div className="h-6 w-24 bg-gray-200 rounded ml-auto" /></td>
                    </tr>
                  ))
                ) : settlements.length === 0 ? (
                  <tr>
                    <td colSpan={8} className="py-12 text-center text-[#64748b]">
                      <div className="max-w-xs mx-auto space-y-2">
                        <svg className="w-10 h-10 mx-auto text-[#cbd5e1]" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
                          <rect x="2" y="5" width="20" height="14" rx="2" />
                          <line x1="2" y1="10" x2="22" y2="10" />
                        </svg>
                        <p className="font-semibold text-sm text-[#1e293b]">No settlement requests found</p>
                        <p className="text-xs">
                          {statusFilter !== "ALL" || debouncedOrganizerId
                            ? "Try adjusting your filter or search criteria."
                            : "No organizers have requested payouts yet."}
                        </p>
                      </div>
                    </td>
                  </tr>
                ) : (
                  settlements.map((s) => {
                    const statusKey = s.status?.toUpperCase() || "PENDING";
                    const badgeClass =
                      statusBadgeStyles[statusKey] || "bg-[#f3f4f6] text-[#4b5563] border border-[#e5e7eb]";
                    const isPending = statusKey === "PENDING";
                    const isFailed = statusKey === "FAILED";

                    return (
                      <tr
                        key={s.id}
                        onClick={() => setActiveDrawerSettlement(s)}
                        className="hover:bg-[#f8fafc] transition-colors cursor-pointer group"
                      >
                        <td className="py-3.5 px-4 font-mono font-medium text-[#1e293b]">
                          {`${s.id.slice(0, 8)}...${s.id.slice(-4)}`}
                        </td>
                        <td className="py-3.5 px-4 font-mono text-[#64748b]">
                          {s.organizer_id ? `${s.organizer_id.slice(0, 8)}...` : "—"}
                        </td>
                        <td className="py-3.5 px-4">
                          {s.event_id ? (
                            <span className="text-[#0f766e] font-medium" title={s.event_id}>
                              Event: {s.event_id.slice(0, 8)}...
                            </span>
                          ) : (
                            <span className="text-[#64748b]">General</span>
                          )}
                        </td>
                        <td className="py-3.5 px-4 text-right font-bold text-[#141b2b]">
                          {formatCurrency(s.amount, s.currency)}
                        </td>
                        <td className="py-3.5 px-4 text-center">
                          <span className={`inline-flex px-2.5 py-0.5 rounded-full text-[11px] font-semibold ${badgeClass}`}>
                            {statusKey}
                          </span>
                        </td>
                        <td className="py-3.5 px-4 text-[#64748b] font-medium">
                          {formatDateTime(s.created_at)}
                        </td>
                        <td className="py-3.5 px-4 max-w-[160px] truncate">
                          {s.payout_reference ? (
                            <span className="font-mono text-[#0f766e] font-medium" title={s.payout_reference}>
                              {s.payout_reference}
                            </span>
                          ) : s.rejection_reason ? (
                            <span className="text-[#b91c1c] truncate block" title={s.rejection_reason}>
                              Rej: {s.rejection_reason}
                            </span>
                          ) : s.failure_reason ? (
                            <span className="text-[#b91c1c] truncate block" title={s.failure_reason}>
                              Fail: {s.failure_reason}
                            </span>
                          ) : (
                            <span className="text-[#94a3b8]">—</span>
                          )}
                        </td>
                        <td className="py-3.5 px-4 text-right">
                          <div className="flex items-center justify-end gap-1.5" onClick={(e) => e.stopPropagation()}>
                            {isPending && (
                              <>
                                <button
                                  type="button"
                                  onClick={() => setSettlementToApprove(s)}
                                  className="bg-[#00685f] hover:bg-[#005a52] text-white px-2 py-1 rounded text-[11px] font-semibold transition-colors cursor-pointer"
                                  title="Approve Settlement"
                                >
                                  Approve
                                </button>
                                <button
                                  type="button"
                                  onClick={() => setSettlementToReject(s)}
                                  className="border border-[#fca5a5] text-[#b91c1c] hover:bg-[#fef2f2] px-2 py-1 rounded text-[11px] font-semibold transition-colors cursor-pointer"
                                  title="Reject Settlement"
                                >
                                  Reject
                                </button>
                              </>
                            )}
                            {isFailed && (
                              <button
                                type="button"
                                onClick={() => setSettlementToRetry(s)}
                                className="bg-[#2563eb] hover:bg-[#1d4ed8] text-white px-2 py-1 rounded text-[11px] font-semibold transition-colors cursor-pointer"
                                title="Retry Payout"
                              >
                                Retry
                              </button>
                            )}
                            <button
                              type="button"
                              onClick={() => setActiveDrawerSettlement(s)}
                              className="text-[#64748b] hover:text-[#1e293b] p-1 rounded transition-colors"
                              title="View Details"
                            >
                              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                                <circle cx="12" cy="12" r="10" />
                                <line x1="12" y1="16" x2="12" y2="12" />
                                <line x1="12" y1="8" x2="12.01" y2="8" />
                              </svg>
                            </button>
                          </div>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>

          {/* Pagination Footer */}
          <div className="p-4 border-t border-[#e2e8f0] flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-[#64748b]">
            <div className="flex items-center gap-3">
              <span>
                Showing <strong>{settlements.length}</strong> of <strong>{totalRecords}</strong> records
              </span>
              <div className="flex items-center gap-1.5">
                <span>Per page:</span>
                <select
                  value={limit}
                  onChange={(e) => {
                    setLimit(Number(e.target.value));
                    setPage(1);
                  }}
                  className="border border-[#cbd5e1] rounded px-2 py-1 bg-white text-[#1e293b]"
                >
                  {PAGE_SIZE_OPTIONS.map((opt) => (
                    <option key={opt} value={opt}>
                      {opt}
                    </option>
                  ))}
                </select>
              </div>
            </div>

            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1 || loading}
                className="px-3 py-1.5 border border-[#cbd5e1] rounded-lg hover:bg-[#f1f5f9] transition-colors disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
              >
                Previous
              </button>
              <span className="font-medium text-[#1e293b]">
                Page {page} of {totalPages}
              </span>
              <button
                type="button"
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages || loading}
                className="px-3 py-1.5 border border-[#cbd5e1] rounded-lg hover:bg-[#f1f5f9] transition-colors disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
              >
                Next
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Approve Modal */}
      {settlementToApprove && (
        <ApproveSettlementModal
          settlement={settlementToApprove}
          actionLoading={actionLoading}
          onConfirm={handleConfirmApprove}
          onCancel={() => setSettlementToApprove(null)}
        />
      )}

      {/* Reject Modal */}
      {settlementToReject && (
        <RejectSettlementModal
          settlement={settlementToReject}
          actionLoading={actionLoading}
          onConfirm={handleConfirmReject}
          onCancel={() => setSettlementToReject(null)}
        />
      )}

      {/* Retry Modal */}
      {settlementToRetry && (
        <RetrySettlementModal
          settlement={settlementToRetry}
          actionLoading={actionLoading}
          onConfirm={handleConfirmRetry}
          onCancel={() => setSettlementToRetry(null)}
        />
      )}

      {/* Settlement Detail Drawer */}
      {activeDrawerSettlement && (
        <AdminSettlementDetailDrawer
          settlement={activeDrawerSettlement}
          onClose={() => setActiveDrawerSettlement(null)}
          onApprove={(s) => setSettlementToApprove(s)}
          onReject={(s) => setSettlementToReject(s)}
          onRetry={(s) => setSettlementToRetry(s)}
        />
      )}
    </div>
  );
}
