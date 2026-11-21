import { useState, useEffect } from "react";
import {
  getOrganizerSettlementBalance,
  getOrganizerSettlements,
  requestOrganizerSettlement,
} from "../../api/organizerSettlementApi";
import RequestSettlementModal from "../../components/organizer/RequestSettlementModal";
import OrganizerSettlementDetailDrawer from "../../components/organizer/OrganizerSettlementDetailDrawer";
import notify from "../../utils/toast";
import {
  formatEventLifecycle,
  getLifecycleBadgeStyle,
  getEligibilityReasonMessage,
  getPayoutDisabledReason,
  isPayoutButtonDisabled,
} from "../../utils/settlementHelpers";

const STATUS_TABS = [
  { value: "ALL", label: "All Settlements" },
  { value: "PENDING", label: "Pending" },
  { value: "APPROVED", label: "Approved" },
  { value: "PROCESSING", label: "Processing" },
  { value: "COMPLETED", label: "Completed" },
  { value: "REJECTED", label: "Rejected" },
  { value: "FAILED", label: "Failed" },
];

const PAGE_SIZE_OPTIONS = [10, 25, 50];

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

export default function OrganizerSettlementsPage() {
  // Balance State
  const [balanceData, setBalanceData] = useState(null);
  const [balanceLoading, setBalanceLoading] = useState(true);
  const [balanceError, setBalanceError] = useState("");

  // Settlements History State
  const [settlements, setSettlements] = useState([]);
  const [historyLoading, setHistoryLoading] = useState(true);
  const [historyError, setHistoryError] = useState("");
  const [totalRecords, setTotalRecords] = useState(0);
  const [totalPages, setTotalPages] = useState(1);
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useState(10);
  const [selectedStatus, setSelectedStatus] = useState("ALL");

  // Per-Event Breakdown Expand/Collapse
  const [showEventBreakdown, setShowEventBreakdown] = useState(false);

  // Modals & Drawers
  const [showRequestModal, setShowRequestModal] = useState(false);
  const [preselectedEventId, setPreselectedEventId] = useState("");
  const [activeDrawerSettlement, setActiveDrawerSettlement] = useState(null);
  const [requestSubmitting, setRequestSubmitting] = useState(false);

  // Refresh counter
  const [refreshTrigger, setRefreshTrigger] = useState(0);

  // 1. Fetch Authoritative Balance Projection
  useEffect(() => {
    let cancelled = false;
    const timer = setTimeout(async () => {
      setBalanceLoading(true);
      setBalanceError("");
      try {
        const res = await getOrganizerSettlementBalance();
        if (cancelled) return;
        if (res.success && res.data) {
          setBalanceData(res.data);
        } else {
          throw new Error(res.message || "Failed to fetch balance projection");
        }
      } catch (err) {
        if (cancelled) return;
        console.error("Failed to load organizer settlement balance:", err);
        const msg = err.response?.data?.message || err.message || "Failed to load settlement balance.";
        setBalanceError(msg);
      } finally {
        if (!cancelled) {
          setBalanceLoading(false);
        }
      }
    }, 0);

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [refreshTrigger]);

  // 2. Fetch Settlement History List
  useEffect(() => {
    let cancelled = false;
    const timer = setTimeout(async () => {
      setHistoryLoading(true);
      setHistoryError("");
      try {
        const statusParam = selectedStatus !== "ALL" ? selectedStatus : null;
        const res = await getOrganizerSettlements(page, limit, statusParam);
        if (cancelled) return;
        if (res.success && res.data) {
          setSettlements(res.data.settlements || []);
          const total = res.data.total || 0;
          setTotalRecords(total);
          setTotalPages(Math.max(1, Math.ceil(total / limit)));
        } else {
          throw new Error(res.message || "Failed to fetch settlements");
        }
      } catch (err) {
        if (cancelled) return;
        console.error("Failed to load organizer settlements:", err);
        const msg = err.response?.data?.message || err.message || "Failed to load settlement history.";
        setHistoryError(msg);
        setSettlements([]);
        setTotalRecords(0);
        setTotalPages(1);
      } finally {
        if (!cancelled) {
          setHistoryLoading(false);
        }
      }
    }, 0);

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [page, limit, selectedStatus, refreshTrigger]);

  // Handle Submit Settlement Request
  const handleRequestSubmit = async (payload) => {
    setRequestSubmitting(true);
    try {
      const res = await requestOrganizerSettlement(payload);
      if (res.success) {
        notify.success("Settlement payout requested successfully!");
        setShowRequestModal(false);
        setPreselectedEventId("");
        // Trigger reload
        setRefreshTrigger((k) => k + 1);
      } else {
        throw new Error(res.message || "Failed to request settlement");
      }
    } catch (err) {
      console.error("Settlement request failed:", err);
      const msg = err.response?.data?.message || err.message || "Settlement request failed.";
      notify.error(msg);
    } finally {
      setRequestSubmitting(false);
    }
  };

  const handleOpenRequestModal = (eventId = "") => {
    setPreselectedEventId(eventId);
    setShowRequestModal(true);
  };

  const availBalanceNum = parseFloat(balanceData?.available_balance || 0);
  const pendingBalanceNum = parseFloat(balanceData?.pending_balance || 0);
  const isEligible = balanceData?.eligibility_status === "ELIGIBLE";
  const isPartiallyEligible = balanceData?.eligibility_status === "PARTIALLY_ELIGIBLE";

  // Backend provides eligibility_reasons (with fallback to ineligibility_reasons)
  const rawReasons = balanceData?.eligibility_reasons || balanceData?.ineligibility_reasons || [];
  const hasIneligibility =
    !balanceLoading &&
    Boolean(balanceData) &&
    (balanceData.eligibility_status === "INELIGIBLE" || availBalanceNum <= 0);

  // If no available balance and reasons list is empty, default to NO_AVAILABLE_BALANCE
  const eligibilityReasons =
    rawReasons.length > 0
      ? rawReasons
      : (availBalanceNum <= 0 ? ["NO_AVAILABLE_BALANCE"] : []);

  const eventBalances = balanceData?.event_balances || [];
  const isPayoutDisabled = isPayoutButtonDisabled(balanceLoading, balanceData);
  const payoutDisabledTooltip = getPayoutDisabledReason(balanceLoading, balanceData);

  return (
    <div className="flex-1 flex flex-col bg-[#f9f9ff] min-h-screen">
      {/* Top Header */}
      <div className="p-4 sm:p-6 bg-white border-b border-[#bcc9c6]/40 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-xl sm:text-2xl font-bold text-[#141b2b]">
            Settlements & Payouts
          </h1>
          <p className="text-xs sm:text-sm text-[#3d4947] mt-0.5">
            Double-entry ledger balances and verified disbursement payout requests.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={() => setRefreshTrigger((k) => k + 1)}
            disabled={balanceLoading || historyLoading}
            className="bg-white border border-[#bcc9c6] text-[#141b2b] text-xs sm:text-sm font-semibold px-3 py-2 rounded-lg flex items-center gap-1.5 hover:bg-[#f1f5f9] transition-colors cursor-pointer disabled:opacity-50"
            title="Refresh balance and settlements"
          >
            <svg
              className={`w-3.5 h-3.5 ${balanceLoading || historyLoading ? "animate-spin text-[#00685f]" : ""}`}
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

          <button
            type="button"
            onClick={() => handleOpenRequestModal("")}
            disabled={isPayoutDisabled}
            title={payoutDisabledTooltip}
            className="bg-[#00685f] hover:bg-[#005a52] text-white text-xs sm:text-sm font-semibold px-4 py-2 rounded-lg shadow-sm flex items-center gap-2 transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M12 5v14M5 12h14" />
            </svg>
            <span>Request Settlement</span>
          </button>
        </div>
      </div>

      {/* Main Body */}
      <div className="flex-1 p-4 sm:p-6 space-y-6 max-w-7xl w-full mx-auto">
        {/* Error Notification */}
        {balanceError && (
          <div className="p-4 bg-[#fee2e2] border border-[#fecaca] rounded-xl text-[#b91c1c] text-sm flex items-center justify-between">
            <span>{balanceError}</span>
            <button
              onClick={() => setRefreshTrigger((k) => k + 1)}
              className="text-xs font-bold underline ml-3"
            >
              Retry
            </button>
          </div>
        )}

        {/* Ineligibility Warning Alert */}
        {hasIneligibility && (
          <div className="bg-[#fffbeb] border border-[#fef3c7] rounded-xl p-4 text-xs text-[#92400e] space-y-2 shadow-xs">
            <div className="font-semibold text-[#b45309] flex items-center gap-2 text-sm">
              <svg width="18" height="18" viewBox="0 0 20 20" fill="currentColor">
                <path
                  fillRule="evenodd"
                  d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                  clipRule="evenodd"
                />
              </svg>
              Funds currently unavailable
            </div>
            {pendingBalanceNum > 0 && availBalanceNum <= 0 && (
              <p className="text-[#78350f] font-medium leading-relaxed">
                Your earnings from upcoming or ongoing events are held in pending reserves. These funds will become eligible for settlement after the respective events conclude, subject to applicable settlement requirements.
              </p>
            )}
            {eligibilityReasons.length > 0 && (
              <ul className="list-disc list-inside space-y-1 text-[#78350f] pl-1 font-medium">
                {eligibilityReasons.map((reason, idx) => (
                  <li key={idx}>{getEligibilityReasonMessage(reason, balanceData)}</li>
                ))}
              </ul>
            )}
          </div>
        )}

        {/* ================================================================= */}
        {/* BALANCE METRICS CARDS (7 AUTHORITATIVE METRICS)                   */}
        {/* ================================================================= */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {/* Card 1: Available for Payout (Hero Card) */}
          <div className="md:col-span-2 bg-gradient-to-br from-[#00685f] to-[#044a44] text-white rounded-2xl p-5 shadow-sm flex flex-col justify-between relative overflow-hidden">
            <div className="absolute right-0 top-0 translate-x-4 -translate-y-4 w-32 h-32 bg-white/10 rounded-full blur-2xl pointer-events-none" />
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-xs font-semibold uppercase tracking-wider text-teal-100">
                  Authoritative Available Balance
                </span>
                <span
                  className={`text-[11px] font-bold px-2 py-0.5 rounded-full uppercase tracking-wider ${
                    isEligible
                      ? "bg-teal-300/30 text-white border border-teal-200/40"
                      : isPartiallyEligible
                      ? "bg-amber-300/30 text-amber-100 border border-amber-200/40"
                      : "bg-red-400/30 text-red-100 border border-red-200/40"
                  }`}
                >
                  {balanceData?.eligibility_status || "PENDING"}
                </span>
              </div>
              <div className="text-3xl sm:text-4xl font-extrabold tracking-tight mt-1">
                {balanceLoading ? (
                  <div className="h-9 w-40 bg-white/20 animate-pulse rounded my-1" />
                ) : (
                  formatCurrency(balanceData?.available_balance || 0, "INR")
                )}
              </div>
              <p className="text-xs text-teal-100/90 mt-1.5 leading-relaxed">
                Eligible funds ready for bank disbursement after refund provisions and reserve holdbacks.
              </p>
            </div>

            <div className="pt-4 mt-4 border-t border-white/15 flex flex-col sm:flex-row sm:items-center justify-between gap-2 text-xs text-teal-100">
              <span>Account 2010 (Payable): <strong>{balanceLoading ? "..." : formatCurrency(balanceData?.organizer_payable || 0, "INR")}</strong></span>
              <div className="flex items-center gap-2">
                {isPayoutDisabled && !balanceLoading && (
                  <span className="text-[11px] text-teal-200/90 italic">
                    {pendingBalanceNum > 0 && availBalanceNum <= 0
                      ? "Held in reserves until events conclude"
                      : balanceData?.has_active_bank_account === false
                      ? "Bank account verification required"
                      : "Payout currently unavailable"}
                  </span>
                )}
                <button
                  type="button"
                  onClick={() => handleOpenRequestModal("")}
                  disabled={isPayoutDisabled}
                  title={payoutDisabledTooltip}
                  className="bg-white text-[#00685f] hover:bg-teal-50 font-bold px-3 py-1.5 rounded-lg transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  Request Payout
                </button>
              </div>
            </div>
          </div>

          {/* Card 2: Gross Earnings */}
          <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl p-4 shadow-2xs flex flex-col justify-between">
            <span className="text-[11px] font-bold text-[#3d4947] uppercase tracking-wider">
              Gross Earnings
            </span>
            <div className="text-xl sm:text-2xl font-bold text-[#141b2b] my-2">
              {balanceLoading ? (
                <div className="h-7 w-28 bg-gray-200 animate-pulse rounded" />
              ) : (
                formatCurrency(balanceData?.gross_earnings || 0, "INR")
              )}
            </div>
            <span className="text-[11px] text-[#64748b]">Total ticket booking revenue captured</span>
          </div>

          {/* Card 3: Refunds Deducted */}
          <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl p-4 shadow-2xs flex flex-col justify-between">
            <span className="text-[11px] font-bold text-[#b91c1c] uppercase tracking-wider">
              Refunds Deducted
            </span>
            <div className="text-xl sm:text-2xl font-bold text-[#b91c1c] my-2">
              {balanceLoading ? (
                <div className="h-7 w-28 bg-gray-200 animate-pulse rounded" />
              ) : (
                `-${formatCurrency(
                  balanceData?.refunds_deducted !== undefined
                    ? balanceData.refunds_deducted
                    : eventBalances.reduce((sum, eb) => sum + parseFloat(eb.refunds_deducted || 0), 0),
                  "INR"
                )}`
              )}
            </div>
            <span className="text-[11px] text-[#64748b]">Customer cancellations & event refunds</span>
          </div>

          {/* Card 4: Net Earnings */}
          <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl p-4 shadow-2xs flex flex-col justify-between">
            <span className="text-[11px] font-bold text-[#3d4947] uppercase tracking-wider">
              Net Earnings
            </span>
            <div className="text-xl sm:text-2xl font-bold text-[#0f766e] my-2">
              {balanceLoading ? (
                <div className="h-7 w-28 bg-gray-200 animate-pulse rounded" />
              ) : (
                formatCurrency(balanceData?.net_earnings || 0, "INR")
              )}
            </div>
            <span className="text-[11px] text-[#64748b]">Net ticket revenue less refunds</span>
          </div>

          {/* Card 5: Settled Amount */}
          <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl p-4 shadow-2xs flex flex-col justify-between">
            <span className="text-[11px] font-bold text-[#3d4947] uppercase tracking-wider">
              Settled / Disbursed
            </span>
            <div className="text-xl sm:text-2xl font-bold text-[#1e293b] my-2">
              {balanceLoading ? (
                <div className="h-7 w-28 bg-gray-200 animate-pulse rounded" />
              ) : (
                formatCurrency(balanceData?.settled_amount || 0, "INR")
              )}
            </div>
            <span className="text-[11px] text-[#64748b]">Successfully paid out to bank account</span>
          </div>

          {/* Card 6: In-Flight Settlement Holds */}
          <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl p-4 shadow-2xs flex flex-col justify-between">
            <span className="text-[11px] font-bold text-[#d97706] uppercase tracking-wider">
              In-Flight Holds
            </span>
            <div className="text-xl sm:text-2xl font-bold text-[#d97706] my-2">
              {balanceLoading ? (
                <div className="h-7 w-28 bg-gray-200 animate-pulse rounded" />
              ) : (
                formatCurrency(balanceData?.in_flight_settlement_hold || 0, "INR")
              )}
            </div>
            <span className="text-[11px] text-[#64748b]">Locked in pending/processing payouts</span>
          </div>

          {/* Card 7: Pending Balance (Unconcluded events) */}
          <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl p-4 shadow-2xs flex flex-col justify-between">
            <span className="text-[11px] font-bold text-[#64748b] uppercase tracking-wider">
              Pending Reserves
            </span>
            <div className="text-xl sm:text-2xl font-bold text-[#475569] my-2">
              {balanceLoading ? (
                <div className="h-7 w-28 bg-gray-200 animate-pulse rounded" />
              ) : (
                formatCurrency(balanceData?.pending_balance || 0, "INR")
              )}
            </div>
            <span className="text-[11px] text-[#64748b]">Revenue from upcoming or ongoing events</span>
          </div>
        </div>

        {/* ================================================================= */}
        {/* OPTIONAL EVENT BREAKDOWN ACCORDION                                */}
        {/* ================================================================= */}
        {eventBalances.length > 0 && (
          <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl p-4 sm:p-5 shadow-2xs">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-base font-bold text-[#141b2b]">
                  Per-Event Balance Breakdown
                </h2>
                <p className="text-xs text-[#64748b] mt-0.5">
                  Subledger balances partitioned across {eventBalances.length} events.
                </p>
              </div>
              <button
                type="button"
                onClick={() => setShowEventBreakdown(!showEventBreakdown)}
                className="text-xs font-semibold text-[#00685f] hover:text-[#005a52] flex items-center gap-1 cursor-pointer"
              >
                <span>{showEventBreakdown ? "Hide Details" : "Show Details"}</span>
                <svg
                  className={`w-4 h-4 transition-transform ${showEventBreakdown ? "rotate-180" : ""}`}
                  viewBox="0 0 20 20"
                  fill="currentColor"
                >
                  <path
                    fillRule="evenodd"
                    d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z"
                    clipRule="evenodd"
                  />
                </svg>
              </button>
            </div>

            {showEventBreakdown && (
              <div className="mt-4 overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr className="border-b border-[#e2e8f0] bg-[#f8fafc] text-[#64748b] uppercase tracking-wider font-semibold">
                      <th className="py-2.5 px-3">Event</th>
                      <th className="py-2.5 px-3">Lifecycle</th>
                      <th className="py-2.5 px-3 text-right">Gross</th>
                      <th className="py-2.5 px-3 text-right">Refunds</th>
                      <th className="py-2.5 px-3 text-right">Net Payable</th>
                      <th className="py-2.5 px-3 text-right">Settled</th>
                      <th className="py-2.5 px-3 text-right">Pending Reserve / Hold</th>
                      <th className="py-2.5 px-3 text-right">Available</th>
                      <th className="py-2.5 px-3 text-center">Action</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-[#f1f5f9]">
                    {eventBalances.map((eb) => {
                      const avail = parseFloat(eb.available_balance || 0);
                      const lifecycleKey = eb.lifecycle || eb.financial_lifecycle;
                      return (
                        <tr key={eb.event_id} className="hover:bg-[#f8fafc]">
                          <td className="py-3 px-3">
                            <div className="font-semibold text-[#1e293b]">{eb.event_title || "Untitled Event"}</div>
                            <div className="font-mono text-[10px] text-[#64748b]">{eb.event_id}</div>
                          </td>
                          <td className="py-3 px-3">
                            <span className={`inline-flex px-2 py-0.5 rounded text-[10px] font-semibold ${getLifecycleBadgeStyle(lifecycleKey)}`}>
                              {formatEventLifecycle(lifecycleKey)}
                            </span>
                          </td>
                          <td className="py-3 px-3 text-right font-medium">{formatCurrency(eb.gross_earnings)}</td>
                          <td className="py-3 px-3 text-right text-[#b91c1c]">{formatCurrency(eb.refunds_deducted)}</td>
                          <td className="py-3 px-3 text-right font-semibold">
                            {formatCurrency(eb.net_payable !== undefined ? eb.net_payable : eb.net_earnings)}
                          </td>
                          <td className="py-3 px-3 text-right text-[#64748b]">
                            {eb.settled_amount !== undefined && eb.settled_amount !== null
                              ? formatCurrency(eb.settled_amount)
                              : "—"}
                          </td>
                          <td className="py-3 px-3 text-right text-[#d97706]">
                            {eb.pending_balance !== undefined && eb.pending_balance !== null
                              ? formatCurrency(eb.pending_balance)
                              : (eb.in_flight_refund_hold !== undefined && eb.in_flight_refund_hold !== null
                                ? formatCurrency(eb.in_flight_refund_hold)
                                : "—")}
                          </td>
                          <td className="py-3 px-3 text-right font-bold text-[#0f766e]">{formatCurrency(eb.available_balance)}</td>
                          <td className="py-3 px-3 text-center">
                            <button
                              type="button"
                              onClick={() => handleOpenRequestModal(eb.event_id)}
                              disabled={avail <= 0 || !eb.is_eligible || balanceLoading || balanceData?.has_active_bank_account === false}
                              title={
                                avail <= 0
                                  ? "Earnings are held in pending reserve until event conclusion"
                                  : balanceData?.has_active_bank_account === false
                                  ? "Active bank account required"
                                  : !eb.is_eligible
                                  ? "Event not eligible for settlement"
                                  : `Request payout for ${eb.event_title || "event"}`
                              }
                              className="bg-[#00685f] hover:bg-[#005a52] text-white px-2.5 py-1 rounded text-[11px] font-semibold transition-colors disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
                            >
                              Settle
                            </button>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}

        {/* ================================================================= */}
        {/* SETTLEMENT HISTORY TABLE                                          */}
        {/* ================================================================= */}
        <div className="bg-white border border-[#bcc9c6]/50 rounded-2xl shadow-2xs overflow-hidden">
          {/* Table Header & Status Filter Tabs */}
          <div className="p-4 sm:p-5 border-b border-[#bcc9c6]/30 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div>
              <h2 className="text-base font-bold text-[#141b2b]">
                Settlement History
              </h2>
              <p className="text-xs text-[#64748b] mt-0.5">
                Past disbursement requests and real-time status transitions.
              </p>
            </div>

            {/* Status Pills */}
            <div className="flex flex-wrap items-center gap-1.5">
              {STATUS_TABS.map((tab) => (
                <button
                  key={tab.value}
                  type="button"
                  onClick={() => {
                    setSelectedStatus(tab.value);
                    setPage(1);
                  }}
                  className={`px-3 py-1 text-xs font-semibold rounded-lg transition-colors cursor-pointer ${
                    selectedStatus === tab.value
                      ? "bg-[#00685f] text-white shadow-2xs"
                      : "bg-[#f1f5f9] text-[#64748b] hover:bg-[#e2e8f0] hover:text-[#1e293b]"
                  }`}
                >
                  {tab.label}
                </button>
              ))}
            </div>
          </div>

          {/* Table Error */}
          {historyError && (
            <div className="p-4 bg-[#fee2e2] text-[#b91c1c] text-xs font-medium">
              {historyError}
            </div>
          )}

          {/* Table Content */}
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse text-xs">
              <thead>
                <tr className="border-b border-[#e2e8f0] bg-[#f8fafc] text-[#64748b] uppercase tracking-wider font-semibold">
                  <th className="py-3 px-4">Settlement ID</th>
                  <th className="py-3 px-4">Scope</th>
                  <th className="py-3 px-4 text-right">Amount</th>
                  <th className="py-3 px-4 text-center">Status</th>
                  <th className="py-3 px-4">Requested At</th>
                  <th className="py-3 px-4">Processed At</th>
                  <th className="py-3 px-4 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#f1f5f9]">
                {historyLoading ? (
                  Array.from({ length: 5 }).map((_, idx) => (
                    <tr key={idx} className="animate-pulse">
                      <td className="py-3.5 px-4"><div className="h-4 w-28 bg-gray-200 rounded" /></td>
                      <td className="py-3.5 px-4"><div className="h-4 w-20 bg-gray-200 rounded" /></td>
                      <td className="py-3.5 px-4 text-right"><div className="h-4 w-16 bg-gray-200 rounded ml-auto" /></td>
                      <td className="py-3.5 px-4 text-center"><div className="h-5 w-20 bg-gray-200 rounded-full mx-auto" /></td>
                      <td className="py-3.5 px-4"><div className="h-4 w-24 bg-gray-200 rounded" /></td>
                      <td className="py-3.5 px-4"><div className="h-4 w-24 bg-gray-200 rounded" /></td>
                      <td className="py-3.5 px-4 text-right"><div className="h-6 w-14 bg-gray-200 rounded ml-auto" /></td>
                    </tr>
                  ))
                ) : settlements.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="py-12 text-center text-[#64748b]">
                      <div className="max-w-xs mx-auto space-y-2">
                        <svg className="w-10 h-10 mx-auto text-[#cbd5e1]" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
                          <circle cx="12" cy="12" r="10" />
                          <path d="M16 16s-1.5-2-4-2-4 2-4 2M9 9h.01M15 9h.01" />
                        </svg>
                        <p className="font-semibold text-sm text-[#1e293b]">No settlement requests found</p>
                        <p className="text-xs">
                          {selectedStatus !== "ALL"
                            ? `No records found with status ${selectedStatus}.`
                            : "You have not requested any settlement disbursements yet."}
                        </p>
                      </div>
                    </td>
                  </tr>
                ) : (
                  settlements.map((s) => {
                    const statusKey = s.status?.toUpperCase() || "PENDING";
                    const badgeClass =
                      statusBadgeStyles[statusKey] || "bg-[#f3f4f6] text-[#4b5563] border border-[#e5e7eb]";
                    return (
                      <tr
                        key={s.id}
                        onClick={() => setActiveDrawerSettlement(s)}
                        className="hover:bg-[#f8fafc] transition-colors cursor-pointer group"
                      >
                        <td className="py-3.5 px-4 font-mono font-medium text-[#1e293b]">
                          {`${s.id.slice(0, 8)}...${s.id.slice(-4)}`}
                        </td>
                        <td className="py-3.5 px-4">
                          {s.event_id ? (
                            <span className="text-[#0f766e] font-medium" title={s.event_id}>
                              Event: {s.event_id.slice(0, 8)}...
                            </span>
                          ) : (
                            <span className="text-[#64748b]">General Account</span>
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
                        <td className="py-3.5 px-4 text-[#64748b] font-medium">
                          {formatDateTime(s.processed_at)}
                        </td>
                        <td className="py-3.5 px-4 text-right">
                          <button
                            type="button"
                            onClick={(e) => {
                              e.stopPropagation();
                              setActiveDrawerSettlement(s);
                            }}
                            className="text-[#00685f] hover:text-[#005a52] font-semibold hover:underline inline-flex items-center gap-1"
                          >
                            <span>Details</span>
                            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                              <path d="M9 18l6-6-6-6" />
                            </svg>
                          </button>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>

          {/* Table Footer with Pagination */}
          <div className="p-4 border-t border-[#e2e8f0] flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-[#64748b]">
            <div className="flex items-center gap-3">
              <span>
                Showing <strong>{settlements.length}</strong> of <strong>{totalRecords}</strong> requests
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
                disabled={page <= 1 || historyLoading}
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
                disabled={page >= totalPages || historyLoading}
                className="px-3 py-1.5 border border-[#cbd5e1] rounded-lg hover:bg-[#f1f5f9] transition-colors disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
              >
                Next
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Request Settlement Modal */}
      {showRequestModal && (
        <RequestSettlementModal
          balanceData={balanceData}
          preselectedEventId={preselectedEventId}
          isSubmitting={requestSubmitting}
          onSubmit={handleRequestSubmit}
          onClose={() => {
            setShowRequestModal(false);
            setPreselectedEventId("");
          }}
        />
      )}

      {/* Settlement Detail Drawer */}
      {activeDrawerSettlement && (
        <OrganizerSettlementDetailDrawer
          settlement={activeDrawerSettlement}
          onClose={() => setActiveDrawerSettlement(null)}
        />
      )}
    </div>
  );
}
