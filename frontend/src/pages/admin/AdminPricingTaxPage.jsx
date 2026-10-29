import { useState, useEffect } from "react";
import {
  getPricingTaxCurrentAndTimeline,
  getPricingTaxHistory,
  publishImmediatePricingTax,
  scheduleFuturePricingTax,
  cancelScheduledPricingTax,
} from "../../api/pricingTaxApi";
import notify from "../../utils/toast";

export default function AdminPricingTaxPage() {
  const [activeConfig, setActiveConfig] = useState(null);
  const [timeline, setTimeline] = useState([]);
  const [history, setHistory] = useState([]);
  const [historyTotal, setHistoryTotal] = useState(0);
  const [historyPage, setHistoryPage] = useState(1);
  const [refreshIndex, setRefreshIndex] = useState(0);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [errorMsg, setErrorMsg] = useState("");
  const [successMsg, setSuccessMsg] = useState("");
  const [conflictOccurred, setConflictOccurred] = useState(false);

  // Form state
  const [formMode, setFormMode] = useState("immediate"); // "immediate" | "future"
  const [feeEnabled, setFeeEnabled] = useState(false);
  const [feePercent, setFeePercent] = useState("0.00");
  const [freeExempt, setFreeExempt] = useState(true);
  const [taxEnabled, setTaxEnabled] = useState(false);
  const [taxPercent, setTaxPercent] = useState("0.00");
  const [taxBasis, setTaxBasis] = useState("UNVALIDATED");
  const [taxValidated, setTaxValidated] = useState(false);
  const [taxLegalRef, setTaxLegalRef] = useState("");
  const [futureDate, setFutureDate] = useState("");
  const [replaceFuture, setReplaceFuture] = useState(false);
  const [changeReason, setChangeReason] = useState("");

  // Cancel modal state
  const [cancelModalConfig, setCancelModalConfig] = useState(null);
  const [cancelReason, setCancelReason] = useState("");

  const triggerRefresh = () => {
    setRefreshIndex((prev) => prev + 1);
  };

  useEffect(() => {
    let cancelled = false;

    const fetchData = async () => {
      try {
        setLoading(true);
        setErrorMsg("");
        setConflictOccurred(false);

        const res = await getPricingTaxCurrentAndTimeline();
        if (cancelled) return;

        if (res?.data) {
          setActiveConfig(res.data.active || null);
          setTimeline(res.data.timeline || []);

          if (res.data.active) {
            setFeeEnabled(res.data.active.platform_fee_enabled);
            setFeePercent(res.data.active.platform_fee_percent || "0.00");
            setFreeExempt(res.data.active.free_ticket_exempt);
            setTaxEnabled(res.data.active.tax_enabled);
            setTaxPercent(res.data.active.tax_percent || "0.00");
            setTaxBasis(res.data.active.tax_basis || "UNVALIDATED");
            setTaxValidated(res.data.active.tax_rules_validated);
            setTaxLegalRef(res.data.active.tax_legal_signoff_reference || "");
          }
        }

        const histRes = await getPricingTaxHistory({ page: historyPage, limit: 10 });
        if (cancelled) return;

        if (histRes?.data) {
          setHistory(histRes.data.history || []);
          setHistoryTotal(histRes.data.total || 0);
        }
      } catch (err) {
        if (!cancelled) {
          setErrorMsg(err?.response?.data?.message || err?.message || "Failed to load pricing data.");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    fetchData();

    return () => {
      cancelled = true;
    };
  }, [historyPage, refreshIndex]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setErrorMsg("");
    setSuccessMsg("");
    setConflictOccurred(false);

    if (!changeReason.trim()) {
      setErrorMsg("A change reason is mandatory for audit logging.");
      return;
    }

    if (taxEnabled && (!taxValidated || taxBasis === "UNVALIDATED" || !taxLegalRef.trim())) {
      setErrorMsg("Tax cannot be enabled without a validated statutory basis, confirmation, and legal sign-off reference.");
      return;
    }

    const expectedBaseVersion = activeConfig ? activeConfig.version : 1;
    setSubmitting(true);

    try {
      if (formMode === "immediate") {
        await publishImmediatePricingTax({
          expected_base_version: expectedBaseVersion,
          platform_fee_enabled: feeEnabled,
          platform_fee_percent: feePercent,
          tax_enabled: taxEnabled,
          tax_percent: taxPercent,
          tax_basis: taxBasis,
          tax_rules_validated: taxValidated,
          tax_legal_signoff_reference: taxLegalRef.trim() ? taxLegalRef.trim() : null,
          free_ticket_exempt: freeExempt,
          replace_future_schedules: replaceFuture,
          change_reason: changeReason.trim(),
        });
        const msg = "Configuration published immediately and activated successfully.";
        setSuccessMsg(msg);
        notify.success(msg);
      } else {
        if (!futureDate) {
          setErrorMsg("Please select an effective future date and time.");
          setSubmitting(false);
          return;
        }

        await scheduleFuturePricingTax({
          expected_base_version: expectedBaseVersion,
          platform_fee_enabled: feeEnabled,
          platform_fee_percent: feePercent,
          tax_enabled: taxEnabled,
          tax_percent: taxPercent,
          tax_basis: taxBasis,
          tax_rules_validated: taxValidated,
          tax_legal_signoff_reference: taxLegalRef.trim() ? taxLegalRef.trim() : null,
          free_ticket_exempt: freeExempt,
          effective_from: new Date(futureDate).toISOString(),
          change_reason: changeReason.trim(),
        });
        const msg = "Upcoming configuration scheduled successfully.";
        setSuccessMsg(msg);
        notify.success(msg);
      }

      setChangeReason("");
      triggerRefresh();
    } catch (err) {
      if (err?.response?.status === 409) {
        setConflictOccurred(true);
        const conflictMsg = "Version Conflict: Another administrator updated the configuration. Please reload the timeline.";
        setErrorMsg(conflictMsg);
        notify.error(conflictMsg);
      } else {
        const errText = err?.response?.data?.message || err?.message || "Failed to update configuration.";
        setErrorMsg(errText);
        notify.error(errText);
      }
    } finally {
      setSubmitting(false);
    }
  };

  const handleCancelScheduled = async () => {
    if (!cancelModalConfig || !cancelReason.trim()) {
      setErrorMsg("A cancellation reason is required.");
      return;
    }

    setSubmitting(true);
    try {
      await cancelScheduledPricingTax(cancelModalConfig.id, {
        expected_base_version: cancelModalConfig.version,
        reason: cancelReason.trim(),
      });
      const msg = "Scheduled configuration cancelled and timeline predecessor restored.";
      setSuccessMsg(msg);
      notify.success(msg);
      setCancelModalConfig(null);
      setCancelReason("");
      triggerRefresh();
    } catch (err) {
      if (err?.response?.status === 409) {
        setConflictOccurred(true);
        const conflictMsg = "Version Conflict: Configuration timeline was updated concurrently.";
        setErrorMsg(conflictMsg);
        notify.error(conflictMsg);
      } else {
        const errText = err?.response?.data?.message || err?.message || "Failed to cancel scheduled configuration.";
        setErrorMsg(errText);
        notify.error(errText);
      }
    } finally {
      setSubmitting(false);
    }
  };

  const scheduledConfigs = timeline.filter(
    (c) => c.status === "SCHEDULED" && new Date(c.effective_from) > new Date()
  );

  return (
    <div className="space-y-8 p-6 max-w-7xl mx-auto text-[#141b2b]">
      {/* Page Header */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-[#141b2b]">Pricing & Tax Configuration</h1>
          <p className="text-sm text-[#4f7a75] mt-1">
            Manage platform fee percentages, statutory taxes, effective-dated timelines, and audit history.
          </p>
        </div>
        {loading && (
          <div className="flex items-center gap-2 text-xs text-[#00685f] bg-[#f0f5f4] px-3 py-1.5 rounded-full">
            <div className="size-3 border-2 border-[#00685f]/30 border-t-[#00685f] rounded-full animate-spin" />
            <span>Refreshing timeline...</span>
          </div>
        )}
      </div>

      {/* Notifications */}
      {conflictOccurred && (
        <div className="p-4 rounded-xl bg-amber-50 border border-amber-300 text-amber-900 flex items-center justify-between shadow-xs">
          <div>
            <p className="font-semibold">Concurrent Modification Detected</p>
            <p className="text-sm mt-0.5">A newer configuration was published by another administrator. Please refresh before saving.</p>
          </div>
          <button
            onClick={triggerRefresh}
            className="px-4 py-2 bg-amber-600 text-white rounded-lg text-sm font-medium hover:bg-amber-700 transition"
          >
            Refresh Timeline
          </button>
        </div>
      )}

      {errorMsg && !conflictOccurred && (
        <div className="p-4 rounded-xl bg-red-50 border border-red-200 text-red-700 text-sm">
          {errorMsg}
        </div>
      )}

      {successMsg && (
        <div className="p-4 rounded-xl bg-emerald-50 border border-emerald-200 text-emerald-800 text-sm">
          {successMsg}
        </div>
      )}

      {/* Active Configuration Overview Card */}
      <div className="bg-white rounded-2xl border border-[#d8e5e3] p-6 shadow-xs">
        <div className="flex flex-wrap items-center justify-between gap-4 border-b border-[#e5eeec] pb-4">
          <div>
            <span className="text-xs font-semibold text-[#00685f] uppercase tracking-wider">Current Live Rate</span>
            <h2 className="text-xl font-bold text-[#141b2b] mt-0.5">
              Active Configuration (Version {activeConfig?.version ?? 1})
            </h2>
          </div>
          <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium bg-emerald-100 text-emerald-800">
            <span className="size-2 rounded-full bg-emerald-500 animate-pulse" />
            Active Live
          </span>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mt-6">
          <div className="p-4 rounded-xl bg-[#f7faf9] border border-[#e5eeec]">
            <span className="text-xs font-medium text-[#4f7a75]">Platform Fee</span>
            <div className="text-2xl font-bold text-[#141b2b] mt-1">
              {activeConfig?.platform_fee_enabled ? `${activeConfig.platform_fee_percent}%` : "0.00%"}
            </div>
            <div className="mt-2 text-xs">
              <span className={`px-2 py-0.5 rounded-full font-medium ${activeConfig?.platform_fee_enabled ? "bg-emerald-100 text-emerald-700" : "bg-gray-100 text-gray-600"}`}>
                {activeConfig?.platform_fee_enabled ? "Enabled" : "Disabled (0%)"}
              </span>
            </div>
          </div>

          <div className="p-4 rounded-xl bg-[#f7faf9] border border-[#e5eeec]">
            <span className="text-xs font-medium text-[#4f7a75]">Statutory Tax</span>
            <div className="text-2xl font-bold text-[#141b2b] mt-1">
              {activeConfig?.tax_enabled ? `${activeConfig.tax_percent}%` : "0.00%"}
            </div>
            <div className="mt-2 text-xs">
              <span className={`px-2 py-0.5 rounded-full font-medium ${activeConfig?.tax_enabled ? "bg-emerald-100 text-emerald-700" : "bg-gray-100 text-gray-600"}`}>
                {activeConfig?.tax_enabled ? "Enabled" : "Disabled (0%)"}
              </span>
            </div>
          </div>

          <div className="p-4 rounded-xl bg-[#f7faf9] border border-[#e5eeec]">
            <span className="text-xs font-medium text-[#4f7a75]">Tax Basis</span>
            <div className="text-sm font-semibold text-[#141b2b] mt-1 truncate">
              {activeConfig?.tax_basis || "UNVALIDATED"}
            </div>
            <p className="text-xs text-[#4f7a75] mt-1">
              {activeConfig?.tax_rules_validated ? "Rules Formally Validated" : "Awaiting Legal Determination"}
            </p>
          </div>

          <div className="p-4 rounded-xl bg-[#f7faf9] border border-[#e5eeec]">
            <span className="text-xs font-medium text-[#4f7a75]">Free Ticket Exemption</span>
            <div className="text-sm font-semibold text-[#141b2b] mt-1">
              {activeConfig?.free_ticket_exempt ? "Exempt by Default" : "Fees Apply"}
            </div>
            <p className="text-xs text-[#4f7a75] mt-1">
              Free tickets contribute ₹0 to paid subtotal
            </p>
          </div>
        </div>
      </div>

      {/* Upcoming Scheduled Configurations */}
      {scheduledConfigs.length > 0 && (
        <div className="bg-white rounded-2xl border border-[#d8e5e3] p-6 shadow-xs">
          <h3 className="text-lg font-bold text-[#141b2b] mb-4">Upcoming Scheduled Changes</h3>
          <div className="space-y-4">
            {scheduledConfigs.map((sched) => (
              <div
                key={sched.id}
                className="p-4 rounded-xl border border-blue-200 bg-blue-50/50 flex flex-wrap items-center justify-between gap-4"
              >
                <div>
                  <div className="flex items-center gap-2">
                    <span className="px-2 py-0.5 rounded-full text-xs font-semibold bg-blue-100 text-blue-800">
                      Version {sched.version} (SCHEDULED)
                    </span>
                    <span className="text-xs text-[#4f7a75]">
                      Takes effect: {new Date(sched.effective_from).toLocaleString()} ({new Date(sched.effective_from).toISOString().replace("T", " ").substring(0, 19)} UTC)
                    </span>
                  </div>
                  <div className="mt-2 text-sm text-[#141b2b] flex gap-4">
                    <span>Platform Fee: <strong>{sched.platform_fee_enabled ? `${sched.platform_fee_percent}%` : "Disabled (0%)"}</strong></span>
                    <span>Tax: <strong>{sched.tax_enabled ? `${sched.tax_percent}%` : "Disabled (0%)"}</strong></span>
                  </div>
                  <p className="text-xs text-[#4f7a75] mt-1">Reason: {sched.change_reason}</p>
                </div>

                <button
                  onClick={() => {
                    setCancelModalConfig(sched);
                    setCancelReason("");
                  }}
                  className="px-3 py-1.5 rounded-lg border border-red-300 text-red-700 text-xs font-medium hover:bg-red-50 transition"
                >
                  Cancel Schedule
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Update / Schedule Form */}
      <div className="bg-white rounded-2xl border border-[#d8e5e3] p-6 shadow-xs">
        <div className="flex items-center justify-between border-b border-[#e5eeec] pb-4 mb-6">
          <h3 className="text-lg font-bold text-[#141b2b]">Publish or Schedule Configuration</h3>

          {/* Mode Switcher */}
          <div className="inline-flex rounded-lg border border-[#bcc9c6] p-0.5 bg-[#f0f5f4]">
            <button
              type="button"
              onClick={() => setFormMode("immediate")}
              className={`px-3 py-1.5 rounded-md text-xs font-medium transition ${formMode === "immediate" ? "bg-white text-[#00685f] shadow-xs" : "text-[#4f7a75] hover:text-[#141b2b]"}`}
            >
              Publish Immediately
            </button>
            <button
              type="button"
              onClick={() => setFormMode("future")}
              className={`px-3 py-1.5 rounded-md text-xs font-medium transition ${formMode === "future" ? "bg-white text-[#00685f] shadow-xs" : "text-[#4f7a75] hover:text-[#141b2b]"}`}
            >
              Schedule for Future Date
            </button>
          </div>
        </div>

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Platform Fee Settings */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 p-4 rounded-xl bg-[#f7faf9] border border-[#e5eeec]">
            <div>
              <label className="flex items-center gap-2 cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={feeEnabled}
                  onChange={(e) => setFeeEnabled(e.target.checked)}
                  className="size-4 rounded text-[#00685f] focus:ring-[#00685f]"
                />
                <span className="text-sm font-semibold text-[#141b2b]">Enable Percentage Platform Fee</span>
              </label>
              <p className="text-xs text-[#4f7a75] mt-1 ml-6">
                Assessed on aggregate paid-ticket subtotal at checkout.
              </p>
            </div>

            <div>
              <label className="block text-xs font-medium text-[#4f7a75] mb-1">
                Platform Fee Percentage (%)
              </label>
              <input
                type="number"
                step="0.01"
                min="0.00"
                max="100.00"
                value={feePercent}
                onChange={(e) => setFeePercent(e.target.value)}
                disabled={!feeEnabled}
                className="w-full px-3 py-2 border border-[#bcc9c6] rounded-lg text-sm bg-white focus:outline-none focus:border-[#00685f] disabled:bg-gray-100 disabled:text-gray-400"
              />
            </div>
          </div>

          {/* Tax Settings */}
          <div className="p-4 rounded-xl bg-[#f7faf9] border border-[#e5eeec] space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <label className="flex items-center gap-2 cursor-pointer select-none">
                  <input
                    type="checkbox"
                    checked={taxEnabled}
                    onChange={(e) => setTaxEnabled(e.target.checked)}
                    className="size-4 rounded text-[#00685f] focus:ring-[#00685f]"
                  />
                  <span className="text-sm font-semibold text-[#141b2b]">Enable Tax Configuration</span>
                </label>
                <p className="text-xs text-[#4f7a75] mt-1 ml-6">
                  Requires statutory compliance validation and formal legal sign-off reference.
                </p>
              </div>

              <span className="text-xs px-2.5 py-1 rounded-full font-medium bg-amber-100 text-amber-800">
                Statutory Lock Active
              </span>
            </div>

            {taxEnabled && (
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 pt-3 border-t border-[#e5eeec]">
                <div>
                  <label className="block text-xs font-medium text-[#4f7a75] mb-1">Tax Percentage (%)</label>
                  <input
                    type="number"
                    step="0.01"
                    min="0.00"
                    max="100.00"
                    value={taxPercent}
                    onChange={(e) => setTaxPercent(e.target.value)}
                    className="w-full px-3 py-2 border border-[#bcc9c6] rounded-lg text-sm bg-white focus:outline-none focus:border-[#00685f]"
                  />
                </div>

                <div>
                  <label className="block text-xs font-medium text-[#4f7a75] mb-1">Tax Basis</label>
                  <select
                    value={taxBasis}
                    onChange={(e) => setTaxBasis(e.target.value)}
                    className="w-full px-3 py-2 border border-[#bcc9c6] rounded-lg text-sm bg-white focus:outline-none focus:border-[#00685f]"
                  >
                    <option value="UNVALIDATED">UNVALIDATED (Unselected)</option>
                    <option value="PLATFORM_FEE_ONLY">PLATFORM_FEE_ONLY (Intermediary Convenience Fee)</option>
                    <option value="SUBTOTAL_ONLY">SUBTOTAL_ONLY (Ticket Admission)</option>
                    <option value="TOTAL_BEFORE_TAX">TOTAL_BEFORE_TAX (Combined)</option>
                  </select>
                </div>

                <div>
                  <label className="block text-xs font-medium text-[#4f7a75] mb-1">Legal Sign-Off Reference</label>
                  <input
                    type="text"
                    placeholder="e.g. LEGAL-GST-MEMO-2026-01"
                    value={taxLegalRef}
                    onChange={(e) => setTaxLegalRef(e.target.value)}
                    className="w-full px-3 py-2 border border-[#bcc9c6] rounded-lg text-sm bg-white focus:outline-none focus:border-[#00685f]"
                  />
                </div>

                <div className="md:col-span-3">
                  <label className="flex items-center gap-2 cursor-pointer select-none text-xs text-[#141b2b]">
                    <input
                      type="checkbox"
                      checked={taxValidated}
                      onChange={(e) => setTaxValidated(e.target.checked)}
                      className="size-4 rounded text-[#00685f]"
                    />
                    <span>I certify that applicable GST supply classification, invoicing, credit notes, and place-of-supply rules have been legally signed off.</span>
                  </label>
                </div>
              </div>
            )}
          </div>

          {/* Scheduling and Replacement Controls */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <label className="flex items-center gap-2 cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={freeExempt}
                  onChange={(e) => setFreeExempt(e.target.checked)}
                  className="size-4 rounded text-[#00685f]"
                />
                <span className="text-sm font-medium text-[#141b2b]">Exempt Free Tickets by Default</span>
              </label>
              <p className="text-xs text-[#4f7a75] mt-1 ml-6">
                Free tickets never incur platform fees or taxes.
              </p>
            </div>

            {formMode === "future" ? (
              <div>
                <div className="flex items-center justify-between mb-1">
                  <label className="block text-xs font-medium text-[#4f7a75]">
                    Effective Start Date & Time
                  </label>
                  <span className="text-[11px] text-[#00685f]">
                    Timezone: {Intl.DateTimeFormat().resolvedOptions().timeZone}
                  </span>
                </div>
                <input
                  type="datetime-local"
                  value={futureDate}
                  onChange={(e) => setFutureDate(e.target.value)}
                  className="w-full px-3 py-2 border border-[#bcc9c6] rounded-lg text-sm bg-white focus:outline-none focus:border-[#00685f]"
                />
                {futureDate && (
                  <p className="text-[11px] text-[#4f7a75] mt-1">
                    Scheduled for: <span className="font-medium text-[#141b2b]">{new Date(futureDate).toLocaleString()}</span> (UTC: {new Date(futureDate).toISOString().replace("T", " ").substring(0, 19)} UTC)
                  </p>
                )}
              </div>
            ) : (
              <div>
                <label className="flex items-center gap-2 cursor-pointer select-none">
                  <input
                    type="checkbox"
                    checked={replaceFuture}
                    onChange={(e) => setReplaceFuture(e.target.checked)}
                    className="size-4 rounded text-[#00685f]"
                  />
                  <span className="text-sm font-medium text-[#141b2b]">Replace All Upcoming Schedules</span>
                </label>
                <p className="text-xs text-[#4f7a75] mt-1 ml-6">
                  Default is unchecked (preserves scheduled changes).
                </p>
              </div>
            )}
          </div>

          {/* Change Reason */}
          <div>
            <label className="block text-xs font-medium text-[#4f7a75] mb-1">
              Audit Change Reason <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              required
              placeholder="e.g. Q4 2026 fee revision approved by finance committee"
              value={changeReason}
              onChange={(e) => setChangeReason(e.target.value)}
              className="w-full px-3 py-2 border border-[#bcc9c6] rounded-lg text-sm bg-white focus:outline-none focus:border-[#00685f]"
            />
          </div>

          <button
            type="submit"
            disabled={submitting}
            className="px-6 py-2.5 bg-[#00685f] hover:bg-[#00524b] text-white rounded-lg text-sm font-semibold transition disabled:opacity-50"
          >
            {submitting ? "Processing..." : formMode === "immediate" ? "Publish Immediately" : "Schedule Configuration"}
          </button>
        </form>
      </div>

      {/* Audit History Table */}
      <div className="bg-white rounded-2xl border border-[#d8e5e3] p-6 shadow-xs">
        <h3 className="text-lg font-bold text-[#141b2b] mb-4">Configuration Audit Trail</h3>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs border-collapse">
            <thead>
              <tr className="border-b border-[#e5eeec] text-[#4f7a75] font-medium">
                <th className="py-3 px-3">Version</th>
                <th className="py-3 px-3">Status</th>
                <th className="py-3 px-3">Platform Fee</th>
                <th className="py-3 px-3">Tax</th>
                <th className="py-3 px-3">Effective Interval</th>
                <th className="py-3 px-3">Reason</th>
                <th className="py-3 px-3">Updated At</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#e5eeec]">
              {history.map((row) => (
                <tr key={row.id} className="hover:bg-[#f7faf9]">
                  <td className="py-3 px-3 font-semibold text-[#141b2b]">v{row.version}</td>
                  <td className="py-3 px-3">
                    <span className={`px-2 py-0.5 rounded-full font-medium ${
                      row.status === "ACTIVE"
                        ? "bg-emerald-100 text-emerald-800"
                        : row.status === "SCHEDULED"
                        ? "bg-blue-100 text-blue-800"
                        : row.status === "CANCELLED"
                        ? "bg-red-100 text-red-800"
                        : "bg-gray-100 text-gray-700"
                    }`}>
                      {row.status}
                    </span>
                  </td>
                  <td className="py-3 px-3">
                    {row.platform_fee_enabled ? `${row.platform_fee_percent}%` : "0% (Disabled)"}
                  </td>
                  <td className="py-3 px-3">
                    {row.tax_enabled ? `${row.tax_percent}%` : "0% (Disabled)"}
                  </td>
                  <td className="py-3 px-3 font-mono text-[11px] text-[#4f7a75]">
                    [{new Date(row.effective_from).toLocaleString()}, {row.effective_to ? new Date(row.effective_to).toLocaleString() : "∞"})
                  </td>
                  <td className="py-3 px-3 max-w-xs truncate" title={row.change_reason}>
                    {row.change_reason}
                  </td>
                  <td className="py-3 px-3 text-[#4f7a75]">
                    {new Date(row.updated_at).toLocaleString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        <div className="flex items-center justify-between mt-4 text-xs text-[#4f7a75]">
          <span>Total records: {historyTotal}</span>
          <div className="flex gap-2">
            <button
              onClick={() => setHistoryPage((p) => Math.max(p - 1, 1))}
              disabled={historyPage <= 1}
              className="px-3 py-1 rounded border border-[#bcc9c6] disabled:opacity-50"
            >
              Previous
            </button>
            <span className="px-2 py-1">Page {historyPage}</span>
            <button
              onClick={() => setHistoryPage((p) => p + 1)}
              disabled={historyPage * 10 >= historyTotal}
              className="px-3 py-1 rounded border border-[#bcc9c6] disabled:opacity-50"
            >
              Next
            </button>
          </div>
        </div>
      </div>

      {/* Cancellation Modal */}
      {cancelModalConfig && (
        <div className="fixed inset-0 z-50 bg-black/40 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl max-w-md w-full p-6 shadow-xl space-y-4">
            <h3 className="text-lg font-bold text-[#141b2b]">Cancel Scheduled Configuration</h3>
            <p className="text-sm text-[#4f7a75]">
              Cancelling Version {cancelModalConfig.version} (scheduled for {new Date(cancelModalConfig.effective_from).toLocaleString()}) will restore the preceding configuration interval.
            </p>
            <div>
              <label className="block text-xs font-medium text-[#4f7a75] mb-1">
                Cancellation Reason <span className="text-red-500">*</span>
              </label>
              <textarea
                rows={3}
                value={cancelReason}
                onChange={(e) => setCancelReason(e.target.value)}
                placeholder="Reason for aborting this scheduled rate..."
                className="w-full px-3 py-2 border border-[#bcc9c6] rounded-lg text-sm focus:outline-none focus:border-[#00685f]"
              />
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <button
                type="button"
                onClick={() => setCancelModalConfig(null)}
                className="px-4 py-2 border border-[#bcc9c6] rounded-lg text-sm font-medium text-[#4f7a75] hover:bg-gray-50"
              >
                Keep Schedule
              </button>
              <button
                type="button"
                onClick={handleCancelScheduled}
                disabled={submitting || !cancelReason.trim()}
                className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg text-sm font-medium disabled:opacity-50"
              >
                {submitting ? "Cancelling..." : "Confirm Cancellation"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
