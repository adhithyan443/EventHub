import { useState, useEffect, useCallback } from "react";
import { getAdminLedgerJournalById } from "../../api/adminLedgerApi";
import notify from "../../utils/toast";

const transactionTypeBadgeStyles = {
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
    second: "2-digit",
  });
};

export default function LedgerJournalDetailDrawer({
  journalId,
  isOpen,
  onClose,
  onSelectJournal,
}) {
  const [detail, setDetail] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [copiedField, setCopiedField] = useState(null);

  const fetchJournalDetail = useCallback(async (id, isSubscribed = () => true) => {
    if (!id) return;
    setLoading(true);
    setError("");
    try {
      const res = await getAdminLedgerJournalById(id);
      if (!isSubscribed()) return;
      if (res?.data) {
        setDetail(res.data);
      } else {
        setDetail(null);
        setError("Journal detail not found.");
      }
    } catch (err) {
      if (!isSubscribed()) return;
      console.error("Failed to fetch journal details:", err);
      const msg =
        err.response?.data?.message || err.message || "Failed to load journal details.";
      setError(msg);
      setDetail(null);
    } finally {
      if (isSubscribed()) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    if (!isOpen || !journalId) {
      return;
    }

    const timer = setTimeout(() => {
      fetchJournalDetail(journalId, () => !cancelled);
    }, 0);

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [isOpen, journalId, fetchJournalDetail]);

  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.key === "Escape" && isOpen) {
        onClose();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const copyToClipboard = async (text, fieldName) => {
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
      setCopiedField(fieldName);
      notify.info(`Copied ${fieldName} to clipboard`);
      setTimeout(() => {
        setCopiedField(null);
      }, 2000);
    } catch {
      notify.error("Failed to copy to clipboard");
    }
  };

  const txBadge =
    transactionTypeBadgeStyles[detail?.transaction_type?.toUpperCase()] ||
    "bg-[#f3f4f6] text-[#4b5563] border border-[#e5e7eb]";

  const statusBadge =
    statusBadgeStyles[detail?.status?.toUpperCase()] ||
    "bg-[#f3f4f6] text-[#4b5563] border border-[#e5e7eb]";

  return (
    <div className="fixed inset-0 z-50 overflow-hidden" role="dialog" aria-modal="true">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/40 backdrop-blur-xs transition-opacity"
        onClick={onClose}
      />

      <div className="fixed inset-y-0 right-0 max-w-full flex pl-10">
        <div className="w-screen max-w-2xl bg-white shadow-2xl flex flex-col">
          {/* Drawer Header */}
          <div className="px-6 py-5 border-b border-[#dce2f7] bg-[#f9f9ff] flex items-center justify-between">
            <div>
              <div className="flex items-center gap-2 flex-wrap">
                <h2 className="text-[#141b2b] text-lg font-bold">General Ledger Journal</h2>
                {detail && (
                  <>
                    <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${txBadge}`}>
                      {detail.transaction_type}
                    </span>
                    <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${statusBadge}`}>
                      {detail.status}
                    </span>
                    {detail.is_balanced ? (
                      <span className="text-[10px] font-bold px-2 py-0.5 rounded-full bg-[#dcfce7] text-[#15803d] border border-[#bbf7d0] flex items-center gap-1">
                        <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                        </svg>
                        BALANCED
                      </span>
                    ) : (
                      <span className="text-[10px] font-bold px-2 py-0.5 rounded-full bg-[#fee2e2] text-[#b91c1c] border border-[#fecaca]">
                        UNBALANCED
                      </span>
                    )}
                    {detail.is_reversed && (
                      <span className="text-[10px] font-bold px-2 py-0.5 rounded-full bg-[#fee2e2] text-[#991b1b] border border-[#fca5a5]">
                        REVERSED
                      </span>
                    )}
                    {detail.transaction_type === "REVERSAL" && (
                      <span className="text-[10px] font-bold px-2 py-0.5 rounded-full bg-[#fae8ff] text-[#86198f] border border-[#f5d0fe]">
                        COMPENSATING REVERSAL
                      </span>
                    )}
                  </>
                )}
              </div>
              <p className="text-[#6d7a77] text-xs font-mono mt-1 select-all flex items-center gap-2">
                <span>{journalId}</span>
                <button
                  type="button"
                  onClick={() => copyToClipboard(journalId, "Journal ID")}
                  className="text-[#0052cc] hover:underline text-[11px] cursor-pointer"
                >
                  {copiedField === "Journal ID" ? "Copied!" : "Copy"}
                </button>
              </p>
            </div>
            <button
              type="button"
              onClick={onClose}
              className="p-1.5 rounded-lg text-[#6d7a77] hover:text-[#141b2b] hover:bg-[#eef0fb] transition-colors cursor-pointer"
              title="Close drawer"
            >
              <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                <path
                  fillRule="evenodd"
                  d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                  clipRule="evenodd"
                />
              </svg>
            </button>
          </div>

          {/* Drawer Body */}
          <div className="flex-1 overflow-y-auto p-6 space-y-6">
            {loading && (
              <div className="space-y-4 py-8 animate-pulse">
                <div className="h-20 bg-gray-100 rounded-xl" />
                <div className="h-40 bg-gray-100 rounded-xl" />
                <div className="h-48 bg-gray-100 rounded-xl" />
              </div>
            )}

            {!loading && error && (
              <div className="p-6 text-center bg-[#fee2e2]/40 border border-[#fecaca] rounded-xl text-[#b91c1c]">
                <p className="font-semibold text-sm">{error}</p>
                <button
                  type="button"
                  onClick={() => fetchJournalDetail(journalId)}
                  className="mt-3 px-4 py-1.5 text-xs font-semibold rounded-lg bg-[#b91c1c] text-white hover:bg-[#991b1b] transition-colors cursor-pointer"
                >
                  Retry Loading
                </button>
              </div>
            )}

            {!loading && !error && detail && (
              <>
                {/* Reversal Alert Banner */}
                {detail.is_reversed && (
                  <div className="p-4 rounded-xl bg-[#fee2e2]/60 border border-[#fca5a5] flex items-start gap-3">
                    <svg className="w-5 h-5 text-[#b91c1c] mt-0.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                    </svg>
                    <div className="text-xs text-[#7f1d1d]">
                      <p className="font-bold">This Journal Entry Has Been Reversed</p>
                      <p className="mt-0.5">
                        Compensating reversal journal was posted at {formatDateTime(detail.reversed_at)}.
                        {detail.reversed_by_journal_id && (
                          <span className="block mt-1">
                            Reversed by:{" "}
                            {onSelectJournal ? (
                              <button
                                type="button"
                                onClick={() => onSelectJournal(detail.reversed_by_journal_id)}
                                className="font-mono underline text-[#0052cc] hover:text-[#0747a6] cursor-pointer"
                              >
                                {detail.reversed_by_journal_id}
                              </button>
                            ) : (
                              <span className="font-mono font-medium">{detail.reversed_by_journal_id}</span>
                            )}
                          </span>
                        )}
                      </p>
                    </div>
                  </div>
                )}

                {detail.transaction_type === "REVERSAL" && detail.reversal_reference_id && (
                  <div className="p-4 rounded-xl bg-[#fae8ff]/60 border border-[#f5d0fe] flex items-start gap-3">
                    <svg className="w-5 h-5 text-[#86198f] mt-0.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                    </svg>
                    <div className="text-xs text-[#701a75]">
                      <p className="font-bold">Compensating Reversal Journal</p>
                      <p className="mt-0.5">
                        This journal entry reverses original journal:{" "}
                        {onSelectJournal ? (
                          <button
                            type="button"
                            onClick={() => onSelectJournal(detail.reversal_reference_id)}
                            className="font-mono underline text-[#0052cc] hover:text-[#0747a6] cursor-pointer"
                          >
                            {detail.reversal_reference_id}
                          </button>
                        ) : (
                          <span className="font-mono font-medium">{detail.reversal_reference_id}</span>
                        )}
                      </p>
                    </div>
                  </div>
                )}

                {/* Journal Metadata Cards */}
                <div className="grid grid-cols-2 gap-4">
                  <div className="bg-[#f9f9ff] border border-[#dce2f7] rounded-xl p-4">
                    <span className="text-[11px] font-semibold text-[#6d7a77] uppercase tracking-wider block">
                      Financial Event Type
                    </span>
                    <span className="text-xs font-bold text-[#141b2b] mt-1 block">
                      {detail.financial_event_type || "—"}
                    </span>
                  </div>

                  <div className="bg-[#f9f9ff] border border-[#dce2f7] rounded-xl p-4">
                    <span className="text-[11px] font-semibold text-[#6d7a77] uppercase tracking-wider block">
                      Posted / Created At
                    </span>
                    <span className="text-xs font-semibold text-[#141b2b] mt-1 block">
                      {formatDateTime(detail.posted_at || detail.created_at)}
                    </span>
                  </div>

                  <div className="bg-[#f9f9ff] border border-[#dce2f7] rounded-xl p-4">
                    <span className="text-[11px] font-semibold text-[#6d7a77] uppercase tracking-wider block">
                      Source Reference ID ({detail.source_type || "REF"})
                    </span>
                    <div className="flex items-center gap-2 mt-1">
                      <span className="text-xs font-mono font-medium text-[#141b2b] truncate select-all">
                        {detail.source_id || "—"}
                      </span>
                      {detail.source_id && (
                        <button
                          type="button"
                          onClick={() => copyToClipboard(detail.source_id, "Reference ID")}
                          className="text-[11px] text-[#0052cc] hover:underline cursor-pointer"
                        >
                          {copiedField === "Reference ID" ? "Copied!" : "Copy"}
                        </button>
                      )}
                    </div>
                  </div>

                  <div className="bg-[#f9f9ff] border border-[#dce2f7] rounded-xl p-4">
                    <span className="text-[11px] font-semibold text-[#6d7a77] uppercase tracking-wider block">
                      Idempotency Key
                    </span>
                    <div className="flex items-center gap-2 mt-1">
                      <span className="text-xs font-mono font-medium text-[#141b2b] truncate select-all">
                        {detail.idempotency_key || "—"}
                      </span>
                      {detail.idempotency_key && (
                        <button
                          type="button"
                          onClick={() => copyToClipboard(detail.idempotency_key, "Idempotency Key")}
                          className="text-[11px] text-[#0052cc] hover:underline cursor-pointer"
                        >
                          {copiedField === "Idempotency Key" ? "Copied!" : "Copy"}
                        </button>
                      )}
                    </div>
                  </div>
                </div>

                {/* Description */}
                {detail.description && (
                  <div className="bg-[#f9f9ff] border border-[#dce2f7] rounded-xl p-4">
                    <span className="text-[11px] font-semibold text-[#6d7a77] uppercase tracking-wider block mb-1">
                      Transaction Description
                    </span>
                    <p className="text-xs text-[#141b2b] leading-relaxed">
                      {detail.description}
                    </p>
                  </div>
                )}

                {/* Double-Entry Ledger Lines Table */}
                <div>
                  <div className="flex items-center justify-between mb-2">
                    <h3 className="text-xs font-bold uppercase tracking-wider text-[#141b2b]">
                      Ledger Line Items (Double-Entry Distribution)
                    </h3>
                    <span className="text-[11px] text-[#6d7a77]">
                      {detail.entries?.length || 0} line item{detail.entries?.length === 1 ? "" : "s"}
                    </span>
                  </div>

                  <div className="border border-[#bcc9c6] rounded-xl overflow-hidden bg-white shadow-xs">
                    <table className="w-full text-left text-xs">
                      <thead className="bg-[#f9f9ff] border-b border-[#dce2f7] text-[#6d7a77] font-semibold">
                        <tr>
                          <th className="py-2.5 px-3">Account</th>
                          <th className="py-2.5 px-3">Type</th>
                          <th className="py-2.5 px-3">Entry</th>
                          <th className="py-2.5 px-3 text-right">Debit</th>
                          <th className="py-2.5 px-3 text-right">Credit</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-[#eef0fb]">
                        {detail.entries && detail.entries.length > 0 ? (
                          detail.entries.map((entry) => {
                            const isDebit = entry.entry_type === "DEBIT";
                            return (
                              <tr key={entry.id} className="hover:bg-[#f9f9ff] transition-colors">
                                <td className="py-2.5 px-3">
                                  <div className="font-semibold text-[#141b2b] flex items-center gap-1.5">
                                    <span className="font-mono text-[11px] px-1.5 py-0.5 rounded bg-gray-100 text-[#4b5563]">
                                      {entry.account_code}
                                    </span>
                                    <span>{entry.account_name}</span>
                                  </div>
                                </td>
                                <td className="py-2.5 px-3 text-[#6d7a77] font-mono text-[11px]">
                                  {entry.account_type || "—"}
                                </td>
                                <td className="py-2.5 px-3">
                                  <span
                                    className={`px-1.5 py-0.5 rounded text-[10px] font-bold ${
                                      isDebit
                                        ? "bg-[#e0f2fe] text-[#0369a1]"
                                        : "bg-[#fae8ff] text-[#86198f]"
                                    }`}
                                  >
                                    {entry.entry_type}
                                  </span>
                                </td>
                                <td className="py-2.5 px-3 text-right font-mono font-medium text-[#141b2b]">
                                  {isDebit ? formatCurrency(entry.amount, entry.currency) : "—"}
                                </td>
                                <td className="py-2.5 px-3 text-right font-mono font-medium text-[#141b2b]">
                                  {!isDebit ? formatCurrency(entry.amount, entry.currency) : "—"}
                                </td>
                              </tr>
                            );
                          })
                        ) : (
                          <tr>
                            <td colSpan="5" className="py-4 text-center text-[#6d7a77]">
                              No ledger entries found for this journal.
                            </td>
                          </tr>
                        )}
                      </tbody>
                      <tfoot className="bg-[#f9f9ff] border-t-2 border-[#dce2f7] font-bold text-xs text-[#141b2b]">
                        <tr>
                          <td colSpan="3" className="py-3 px-3 uppercase tracking-wider text-[#6d7a77]">
                            Total ({detail.currency || "INR"})
                          </td>
                          <td className="py-3 px-3 text-right font-mono text-[#0369a1]">
                            {formatCurrency(detail.total_debit, detail.currency)}
                          </td>
                          <td className="py-3 px-3 text-right font-mono text-[#86198f]">
                            {formatCurrency(detail.total_credit, detail.currency)}
                          </td>
                        </tr>
                      </tfoot>
                    </table>
                  </div>

                  {/* Balanced status footer banner */}
                  <div className="mt-3 flex items-center justify-between text-xs px-2 text-[#6d7a77]">
                    <span className="flex items-center gap-1.5">
                      <span className={`w-2 h-2 rounded-full ${detail.is_balanced ? "bg-[#15803d]" : "bg-[#b91c1c]"}`} />
                      {detail.is_balanced
                        ? "Total Debit equals Total Credit. General Ledger is balanced."
                        : "Debit and Credit totals do not balance! Requires financial investigation."}
                    </span>
                    <span className="font-mono text-[11px]">
                      Δ {formatCurrency(Math.abs((parseFloat(detail.total_debit) || 0) - (parseFloat(detail.total_credit) || 0)), detail.currency)}
                    </span>
                  </div>
                </div>
              </>
            )}
          </div>

          {/* Drawer Footer */}
          <div className="px-6 py-4 border-t border-[#dce2f7] bg-[#f9f9ff] flex justify-end">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-xs font-semibold rounded-lg border border-[#bcc9c6] text-[#141b2b] hover:bg-[#eef0fb] transition-colors cursor-pointer"
            >
              Close
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
