import { useState, useEffect, useCallback, useRef } from "react";
import notify from "../../utils/toast";
import { formatDateTime } from "../../utils/formatters";
import {
  getSystemHealth,
  getSystemLogs,
} from "../../api/adminSystemMonitoringApi";

const LOG_LEVELS = [
  { value: "ALL", label: "All Levels" },
  { value: "INFO", label: "INFO" },
  { value: "WARN", label: "WARN" },
  { value: "ERROR", label: "ERROR" },
  { value: "DEBUG", label: "DEBUG" },
];

const AUTO_REFRESH_INTERVALS = [
  { value: 0, label: "Off" },
  { value: 5000, label: "5s" },
  { value: 10000, label: "10s" },
  { value: 30000, label: "30s" },
];

const PAGE_SIZE_OPTIONS = [25, 50, 100, 200];

// Color badges for log levels
const levelBadgeStyles = {
  INFO: "bg-[#e6f4f2] text-[#00685f] border-[#00685f]/30",
  WARN: "bg-[#fef9c3] text-[#854d0e] border-[#fef08a]",
  ERROR: "bg-[#fee2e2] text-[#b91c1c] border-[#fecaca]",
  DEBUG: "bg-[#f3f4f6] text-[#4b5563] border-[#e5e7eb]",
};

export default function AdminSystemMonitoringPage() {
  const [activeTab, setActiveTab] = useState("health"); // "health" | "logs"

  // Health State
  const [healthData, setHealthData] = useState(null);
  const [healthLoading, setHealthLoading] = useState(true);
  const [healthError, setHealthError] = useState(null);
  const [healthLastChecked, setHealthLastChecked] = useState(null);

  // Logs Filter & Pagination State
  const [logs, setLogs] = useState([]);
  const [logsLoading, setLogsLoading] = useState(false);
  const [logsError, setLogsError] = useState(null);
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [selectedLevel, setSelectedLevel] = useState("ALL");
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useState(50);
  const [totalRecords, setTotalRecords] = useState(0);
  const [totalPages, setTotalPages] = useState(1);
  const [expandedLogIdx, setExpandedLogIdx] = useState(null);

  // Auto-refresh state
  const [autoRefreshInterval, setAutoRefreshInterval] = useState(0);
  const [copiedCode, setCopiedCode] = useState(null);
  const autoRefreshTimerRef = useRef(null);
  const isFetchingRef = useRef(false);
  const isMountedRef = useRef(true);

  // Track component mount status
  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
      if (autoRefreshTimerRef.current) {
        clearInterval(autoRefreshTimerRef.current);
        autoRefreshTimerRef.current = null;
      }
    };
  }, []);

  // Debounce search query
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 350);
    return () => clearTimeout(timer);
  }, [search]);

  // Fetch System Health
  const fetchHealth = useCallback(async (showToast = false) => {
    setHealthLoading(true);
    setHealthError(null);
    try {
      const res = await getSystemHealth();
      if (!isMountedRef.current) return;
      setHealthData(res);
      setHealthLastChecked(new Date());
      if (showToast) {
        if (res.success && res.data?.status === "ok") {
          notify.success("System health check verified operational");
        } else {
          notify.warn("System health check returned warning or error");
        }
      }
    } catch (err) {
      if (!isMountedRef.current) return;
      console.error("Health check error:", err);
      setHealthError(err.message || "Failed to reach backend health endpoint");
      if (showToast) notify.error("System health check failed");
    } finally {
      if (isMountedRef.current) {
        setHealthLoading(false);
      }
    }
  }, []);

  // Fetch Application Logs
  const fetchLogs = useCallback(
    async (isBackground = false) => {
      // Guard against concurrent overlapping requests
      if (isFetchingRef.current) {
        return;
      }
      isFetchingRef.current = true;

      if (!isBackground) {
        setLogsLoading(true);
        setLogsError(null);
      }
      try {
        const res = await getSystemLogs({
          page,
          limit,
          level: selectedLevel,
          search: debouncedSearch,
        });

        if (!isMountedRef.current) return;

        const logList = res?.data?.logs || res?.logs || [];
        const paginationInfo = res?.data?.pagination || res?.pagination || {};

        setLogs(logList);
        setTotalRecords(paginationInfo.total_records ?? logList.length);
        setTotalPages(Math.max(1, paginationInfo.total_pages ?? Math.ceil(logList.length / limit)));
      } catch (err) {
        if (!isMountedRef.current) return;
        console.error("Failed to load logs:", err);
        const errMsg =
          err?.response?.data?.message ||
          err?.response?.data?.error ||
          err?.message ||
          "Failed to load application logs.";
        setLogsError(errMsg);
      } finally {
        isFetchingRef.current = false;
        if (!isBackground && isMountedRef.current) {
          setLogsLoading(false);
        }
      }
    },
    [page, limit, selectedLevel, debouncedSearch]
  );

  // Initial load
  useEffect(() => {
    fetchHealth();
  }, [fetchHealth]);

  useEffect(() => {
    if (activeTab === "logs") {
      fetchLogs();
    }
  }, [activeTab, fetchLogs]);

  // Auto-refresh timer for logs
  useEffect(() => {
    if (autoRefreshTimerRef.current) {
      clearInterval(autoRefreshTimerRef.current);
      autoRefreshTimerRef.current = null;
    }

    if (autoRefreshInterval > 0 && activeTab === "logs") {
      autoRefreshTimerRef.current = setInterval(() => {
        fetchLogs(true);
      }, autoRefreshInterval);
    }

    return () => {
      if (autoRefreshTimerRef.current) {
        clearInterval(autoRefreshTimerRef.current);
      }
    };
  }, [autoRefreshInterval, activeTab, fetchLogs]);

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
    setSelectedLevel("ALL");
    setPage(1);
    notify.info("Log filters cleared");
  };

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-white w-full">
      {/* Header */}
      <div className="px-4 sm:px-6 py-4 border-b border-[#dce2f7] flex flex-col md:flex-row md:items-center justify-between gap-4 shrink-0 bg-white">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-[#141b2b] text-xl font-bold tracking-tight">
              System Monitoring
            </h1>
            <span className="bg-[#e6f4f2] text-[#00685f] text-xs font-semibold px-2.5 py-0.5 rounded-full border border-[#00685f]/20">
              Platform Diagnostics
            </span>
          </div>
          <p className="text-[#3d4947] text-xs sm:text-sm mt-0.5">
            Monitor real-time backend health, availability, and structured application logs.
          </p>
        </div>

        {/* Global Tab Switcher & Quick Actions */}
        <div className="flex items-center gap-2 self-start md:self-auto">
          <div className="bg-[#f1f3ff] p-1 rounded-xl flex items-center border border-[#bcc9c6]/40">
            <button
              type="button"
              onClick={() => setActiveTab("health")}
              className={`px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all cursor-pointer flex items-center gap-1.5 ${
                activeTab === "health"
                  ? "bg-white text-[#00685f] shadow-xs"
                  : "text-[#3d4947] hover:text-[#141b2b]"
              }`}
            >
              <span className={`w-2 h-2 rounded-full ${healthData?.success && healthData?.data?.status === "ok" ? "bg-emerald-500" : "bg-amber-500"}`} />
              System Health
            </button>
            <button
              type="button"
              onClick={() => setActiveTab("logs")}
              className={`px-3.5 py-1.5 rounded-lg text-xs font-bold transition-all cursor-pointer flex items-center gap-1.5 ${
                activeTab === "logs"
                  ? "bg-white text-[#00685f] shadow-xs"
                  : "text-[#3d4947] hover:text-[#141b2b]"
              }`}
            >
              <svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 6h16M4 12h16M4 18h7" />
              </svg>
              Application Logs
            </button>
          </div>

          <button
            type="button"
            onClick={() => (activeTab === "health" ? fetchHealth(true) : fetchLogs())}
            disabled={healthLoading || logsLoading}
            className="bg-[#f9f9ff] border border-[#bcc9c6] text-[#141b2b] text-xs sm:text-sm font-semibold px-3 py-1.5 rounded-lg flex items-center gap-1.5 hover:bg-[#eef0fb] transition-colors cursor-pointer shrink-0"
            title="Refresh current monitoring tab"
          >
            <svg
              className={`w-3.5 h-3.5 ${healthLoading || logsLoading ? "animate-spin text-[#00685f]" : ""}`}
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
      <div className="flex-1 overflow-y-auto p-4 sm:p-6 bg-[#f9f9ff]/40">
        {/* ======================= TAB A: SYSTEM HEALTH ======================= */}
        {activeTab === "health" && (
          <div className="max-w-5xl mx-auto space-y-6">
            {/* Primary Status Banner */}
            <div className="bg-white border border-[#dce2f7] rounded-2xl p-6 shadow-xs flex flex-col md:flex-row md:items-center justify-between gap-6">
              <div className="flex items-start sm:items-center gap-4">
                <div
                  className={`w-14 h-14 rounded-2xl flex items-center justify-center shrink-0 ${
                    healthData?.success && healthData?.data?.status === "ok"
                      ? "bg-emerald-50 text-emerald-600 border border-emerald-200"
                      : "bg-amber-50 text-amber-600 border border-amber-200"
                  }`}
                >
                  {healthData?.success && healthData?.data?.status === "ok" ? (
                    <svg className="w-8 h-8" viewBox="0 0 20 20" fill="currentColor">
                      <path
                        fillRule="evenodd"
                        d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                        clipRule="evenodd"
                      />
                    </svg>
                  ) : (
                    <svg className="w-8 h-8" viewBox="0 0 20 20" fill="currentColor">
                      <path
                        fillRule="evenodd"
                        d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                        clipRule="evenodd"
                      />
                    </svg>
                  )}
                </div>

                <div>
                  <div className="flex items-center gap-2">
                    <h2 className="text-[#141b2b] text-xl font-bold">
                      {healthData?.success && healthData?.data?.status === "ok"
                        ? "System Operational"
                        : healthLoading
                        ? "Checking System Health..."
                        : "System Degraded or Unreachable"}
                    </h2>
                    <span
                      className={`text-xs font-bold px-2 py-0.5 rounded-full uppercase tracking-wider ${
                        healthData?.success && healthData?.data?.status === "ok"
                          ? "bg-emerald-100 text-emerald-800"
                          : "bg-amber-100 text-amber-800"
                      }`}
                    >
                      {healthData?.data?.status || (healthLoading ? "Checking" : "Error")}
                    </span>
                  </div>
                  <p className="text-xs sm:text-sm text-[#3d4947] mt-1">
                    {healthData?.success
                      ? "Backend service responded successfully to the health check probe."
                      : "Unable to confirm backend availability. Check server logs or network."}
                  </p>
                </div>
              </div>

              <div className="flex flex-col sm:flex-row items-start sm:items-center gap-3 shrink-0">
                <button
                  type="button"
                  onClick={() => fetchHealth(true)}
                  disabled={healthLoading}
                  className="bg-[#00685f] text-white text-xs sm:text-sm font-semibold px-4 py-2 rounded-xl hover:bg-[#005a52] transition-colors cursor-pointer flex items-center gap-2 shadow-xs"
                >
                  <svg
                    className={`w-4 h-4 ${healthLoading ? "animate-spin" : ""}`}
                    viewBox="0 0 20 20"
                    fill="currentColor"
                  >
                    <path
                      fillRule="evenodd"
                      d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z"
                      clipRule="evenodd"
                    />
                  </svg>
                  <span>Check Health Now</span>
                </button>
              </div>
            </div>

            {/* Health Indicators Grid */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
              {/* Overall Status */}
              <div className="bg-white border border-[#dce2f7] rounded-xl p-4 shadow-xs">
                <span className="text-[11px] font-bold uppercase tracking-wider text-[#707977] block">
                  Overall Status
                </span>
                <div className="flex items-center gap-2 mt-2">
                  <span
                    className={`w-3 h-3 rounded-full ${
                      healthData?.data?.status === "ok" ? "bg-emerald-500" : "bg-red-500"
                    }`}
                  />
                  <span className="text-base font-bold text-[#141b2b]">
                    {healthData?.data?.status === "ok" ? "Healthy (OK)" : "Unhealthy"}
                  </span>
                </div>
                <span className="text-xs text-[#707977] mt-1 block">
                  Reported by backend
                </span>
              </div>

              {/* Backend Availability */}
              <div className="bg-white border border-[#dce2f7] rounded-xl p-4 shadow-xs">
                <span className="text-[11px] font-bold uppercase tracking-wider text-[#707977] block">
                  Backend Service
                </span>
                <div className="flex items-center gap-2 mt-2">
                  <span className="text-base font-bold text-[#00685f]">
                    {healthData?.success ? "Online & Reachable" : "Unreachable"}
                  </span>
                </div>
                <span className="text-xs text-[#707977] mt-1 block font-mono">
                  Probe latency: {healthData?.latency ?? "—"} ms
                </span>
              </div>

              {/* Database Connectivity */}
              <div className="bg-white border border-[#dce2f7] rounded-xl p-4 shadow-xs">
                <span className="text-[11px] font-bold uppercase tracking-wider text-[#707977] block">
                  Database Connectivity
                </span>
                <div className="flex items-center gap-2 mt-2">
                  <span className="text-sm font-semibold text-[#5c6462]">
                    {healthData?.data?.database || "Not reported by health endpoint"}
                  </span>
                </div>
                <span className="text-xs text-[#707977] mt-1 block">
                  Strictly per health endpoint contract
                </span>
              </div>

              {/* Last Checked Timestamp */}
              <div className="bg-white border border-[#dce2f7] rounded-xl p-4 shadow-xs">
                <span className="text-[11px] font-bold uppercase tracking-wider text-[#707977] block">
                  Last Checked
                </span>
                <div className="flex items-center gap-2 mt-2">
                  <span className="text-sm font-semibold text-[#141b2b]">
                    {healthLastChecked ? formatDateTime(healthLastChecked) : "Checking..."}
                  </span>
                </div>
                <span className="text-xs text-[#707977] mt-1 block">
                  Click refresh to update
                </span>
              </div>
            </div>

            {/* Raw Health Endpoint Payload Details */}
            <div className="bg-white border border-[#dce2f7] rounded-2xl p-6 shadow-xs space-y-4">
              <div className="flex items-center justify-between border-b border-[#dce2f7] pb-3">
                <div>
                  <h3 className="text-sm font-bold text-[#141b2b] uppercase tracking-wider">
                    Backend Health Endpoint Payload
                  </h3>
                  <p className="text-xs text-[#707977] mt-0.5">
                    Exact JSON response returned by <code className="text-[#00685f]">GET /health</code>
                  </p>
                </div>
                {healthData && (
                  <button
                    type="button"
                    onClick={() => copyToClipboard(JSON.stringify(healthData.data, null, 2), "Health payload")}
                    className="text-xs font-semibold text-[#00685f] hover:underline cursor-pointer"
                  >
                    Copy JSON
                  </button>
                )}
              </div>

              {healthError ? (
                <div className="p-4 bg-red-50 border border-red-200 rounded-xl text-red-700 text-xs">
                  <p className="font-bold">Error querying health endpoint:</p>
                  <p className="mt-1 font-mono">{healthError}</p>
                </div>
              ) : healthLoading ? (
                <div className="py-8 text-center text-xs text-[#707977]">
                  <div className="w-6 h-6 border-2 border-[#00685f] border-t-transparent rounded-full animate-spin mx-auto mb-2" />
                  Pinging backend health endpoint...
                </div>
              ) : (
                <pre className="bg-[#141b2b] text-[#eef0fb] p-4 rounded-xl text-xs font-mono overflow-x-auto">
                  {JSON.stringify(healthData?.data || {}, null, 2)}
                </pre>
              )}
            </div>
          </div>
        )}

        {/* ======================= TAB B: APPLICATION LOGS ======================= */}
        {activeTab === "logs" && (
          <div className="space-y-4">
            {/* Filter and Control Bar */}
            <div className="bg-white border border-[#dce2f7] rounded-2xl p-4 shadow-xs flex flex-col lg:flex-row lg:items-center justify-between gap-4">
              {/* Search and Filters */}
              <div className="flex flex-wrap items-center gap-3 flex-1">
                {/* Search */}
                <div className="relative min-w-[240px] flex-1 max-w-md">
                  <span className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-gray-400">
                    <svg className="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
                      <path
                        fillRule="evenodd"
                        d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z"
                        clipRule="evenodd"
                      />
                    </svg>
                  </span>
                  <input
                    type="text"
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    placeholder="Search logs by message, request ID, keyword..."
                    className="w-full pl-9 pr-8 py-2 text-xs sm:text-sm bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl focus:outline-none focus:border-[#00685f] text-[#141b2b]"
                  />
                  {search && (
                    <button
                      type="button"
                      onClick={() => setSearch("")}
                      className="absolute inset-y-0 right-0 pr-2.5 flex items-center text-gray-400 hover:text-gray-600"
                    >
                      <svg className="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
                        <path
                          fillRule="evenodd"
                          d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                          clipRule="evenodd"
                        />
                      </svg>
                    </button>
                  )}
                </div>

                {/* Level Filter */}
                <select
                  value={selectedLevel}
                  onChange={(e) => {
                    setSelectedLevel(e.target.value);
                    setPage(1);
                  }}
                  className="bg-[#f9f9ff] border border-[#bcc9c6] text-[#141b2b] text-xs sm:text-sm rounded-xl px-3 py-2 font-medium focus:outline-none focus:border-[#00685f] cursor-pointer"
                >
                  {LOG_LEVELS.map((lvl) => (
                    <option key={lvl.value} value={lvl.value}>
                      {lvl.label}
                    </option>
                  ))}
                </select>

                {(search || selectedLevel !== "ALL") && (
                  <button
                    type="button"
                    onClick={handleResetFilters}
                    className="text-xs text-[#ba1a1a] font-semibold hover:underline cursor-pointer px-2 py-1"
                  >
                    Reset Filters
                  </button>
                )}
              </div>

              {/* Auto Refresh & Controls */}
              <div className="flex items-center gap-3 shrink-0">
                <div className="flex items-center gap-2 text-xs text-[#3d4947]">
                  <span className="font-medium">Auto Refresh:</span>
                  <select
                    value={autoRefreshInterval}
                    onChange={(e) => setAutoRefreshInterval(Number(e.target.value))}
                    className="bg-[#f9f9ff] border border-[#bcc9c6] text-[#141b2b] text-xs rounded-lg px-2 py-1 focus:outline-none focus:border-[#00685f] cursor-pointer"
                  >
                    {AUTO_REFRESH_INTERVALS.map((opt) => (
                      <option key={opt.value} value={opt.value}>
                        {opt.label}
                      </option>
                    ))}
                  </select>
                  {autoRefreshInterval > 0 && (
                    <span className="relative flex h-2 w-2">
                      <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-[#00685f] opacity-75" />
                      <span className="relative inline-flex rounded-full h-2 w-2 bg-[#00685f]" />
                    </span>
                  )}
                </div>

                <button
                  type="button"
                  onClick={() => fetchLogs()}
                  disabled={logsLoading}
                  className="bg-[#00685f] text-white text-xs font-semibold px-3 py-1.5 rounded-lg flex items-center gap-1.5 hover:bg-[#005a52] transition-colors cursor-pointer"
                >
                  <svg
                    className={`w-3.5 h-3.5 ${logsLoading ? "animate-spin" : ""}`}
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

            {/* Error Banner */}
            {logsError && (
              <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-xl text-xs flex items-center justify-between">
                <span>{logsError}</span>
                <button
                  type="button"
                  onClick={() => fetchLogs()}
                  className="font-bold underline hover:text-red-900 cursor-pointer"
                >
                  Retry
                </button>
              </div>
            )}

            {/* Logs Table / List Container */}
            <div className="bg-white border border-[#dce2f7] rounded-2xl shadow-xs overflow-hidden flex flex-col">
              {/* Table header bar */}
              <div className="px-4 py-3 bg-[#f9f9ff] border-b border-[#dce2f7] flex items-center justify-between text-xs text-[#3d4947]">
                <div className="font-semibold text-[#141b2b] flex items-center gap-2">
                  <span>Showing {logs.length} entries</span>
                  <span className="text-[#707977]">({totalRecords} total records matched)</span>
                </div>
                <div className="flex items-center gap-2">
                  <span>Page Size:</span>
                  <select
                    value={limit}
                    onChange={(e) => {
                      setLimit(Number(e.target.value));
                      setPage(1);
                    }}
                    className="bg-white border border-[#bcc9c6] text-[#141b2b] rounded px-1.5 py-0.5 text-xs font-semibold focus:outline-none"
                  >
                    {PAGE_SIZE_OPTIONS.map((sz) => (
                      <option key={sz} value={sz}>
                        {sz}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              {/* Log List View */}
              <div className="divide-y divide-[#f1f3ff] overflow-x-auto">
                {logsLoading && logs.length === 0 ? (
                  <div className="py-16 text-center text-xs text-[#707977]">
                    <div className="w-6 h-6 border-2 border-[#00685f] border-t-transparent rounded-full animate-spin mx-auto mb-2" />
                    Streaming application logs...
                  </div>
                ) : logs.length === 0 ? (
                  <div className="py-16 text-center text-xs text-[#707977]">
                    <svg className="w-10 h-10 mx-auto mb-2 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                    </svg>
                    <p className="font-bold text-[#141b2b]">No application logs found</p>
                    <p className="mt-0.5">Try clearing your search query or switching log level filters.</p>
                  </div>
                ) : (
                  logs.map((log, index) => {
                    const level = log.level?.toUpperCase() || "INFO";
                    const isExpanded = expandedLogIdx === index;
                    const badgeClass = levelBadgeStyles[level] || levelBadgeStyles.INFO;
                    const hasFields = log.fields && Object.keys(log.fields).length > 0;

                    return (
                      <div
                        key={`${log.timestamp}-${index}`}
                        className={`hover:bg-[#f9f9ff] transition-colors p-3 text-xs ${
                          isExpanded ? "bg-[#f9f9ff]" : ""
                        }`}
                      >
                        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                          {/* Left metadata */}
                          <div className="flex flex-wrap items-center gap-2">
                            {/* Level Badge */}
                            <span className={`text-[10px] font-bold px-2 py-0.5 rounded-md border ${badgeClass}`}>
                              {level}
                            </span>

                            {/* Timestamp */}
                            <span className="text-[#707977] font-mono text-[11px]">
                              {formatDateTime(log.timestamp)}
                            </span>

                            {/* Request ID */}
                            {log.request_id && (
                              <button
                                type="button"
                                onClick={() => copyToClipboard(log.request_id, "Request ID")}
                                title="Click to copy Request ID"
                                className="bg-[#f1f3ff] text-[#3d4947] hover:text-[#00685f] px-2 py-0.5 rounded-md font-mono text-[10px] flex items-center gap-1 border border-[#bcc9c6]/40 cursor-pointer"
                              >
                                <span>req: {log.request_id.slice(0, 8)}...</span>
                                {copiedCode === log.request_id ? (
                                  <span className="text-emerald-600 font-bold">✓</span>
                                ) : (
                                  <svg className="w-2.5 h-2.5 opacity-60" viewBox="0 0 20 20" fill="currentColor">
                                    <path d="M8 3a1 1 0 011-1h2a1 1 0 110 2H9a1 1 0 01-1-1z" />
                                    <path d="M6 3a2 2 0 00-2 2v11a2 2 0 002 2h8a2 2 0 002-2V5a2 2 0 00-2-2 3 3 0 01-3 3H9a3 3 0 01-3-3z" />
                                  </svg>
                                )}
                              </button>
                            )}
                          </div>

                          {/* Inspect button */}
                          {hasFields && (
                            <button
                              type="button"
                              onClick={() => setExpandedLogIdx(isExpanded ? null : index)}
                              className="text-[11px] font-semibold text-[#00685f] hover:underline cursor-pointer self-start sm:self-auto"
                            >
                              {isExpanded ? "Hide attributes ▲" : "View attributes ▼"}
                            </button>
                          )}
                        </div>

                        {/* Log message */}
                        <div className="mt-1 font-mono text-xs text-[#141b2b] break-words">
                          {log.message}
                        </div>

                        {/* Expanded details */}
                        {isExpanded && hasFields && (
                          <div className="mt-2.5 pt-2 border-t border-[#dce2f7]">
                            <span className="text-[10px] font-bold uppercase tracking-wider text-[#707977] block mb-1">
                              Structured Context Attributes
                            </span>
                            <pre className="bg-[#141b2b] text-[#eef0fb] p-3 rounded-xl text-[11px] font-mono overflow-x-auto">
                              {JSON.stringify(log.fields, null, 2)}
                            </pre>
                          </div>
                        )}
                      </div>
                    );
                  })
                )}
              </div>

              {/* Pagination Controls */}
              {totalPages > 1 && (
                <div className="px-4 py-3 bg-[#f9f9ff] border-t border-[#dce2f7] flex items-center justify-between text-xs text-[#3d4947]">
                  <div>
                    Page <span className="font-bold text-[#141b2b]">{page}</span> of{" "}
                    <span className="font-bold text-[#141b2b]">{totalPages}</span>
                  </div>

                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      disabled={page <= 1}
                      onClick={() => setPage((p) => Math.max(1, p - 1))}
                      className="px-3 py-1 bg-white border border-[#bcc9c6] rounded-lg font-semibold hover:bg-[#eef0fb] disabled:opacity-50 disabled:pointer-events-none cursor-pointer"
                    >
                      Previous
                    </button>
                    <button
                      type="button"
                      disabled={page >= totalPages}
                      onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                      className="px-3 py-1 bg-white border border-[#bcc9c6] rounded-lg font-semibold hover:bg-[#eef0fb] disabled:opacity-50 disabled:pointer-events-none cursor-pointer"
                    >
                      Next
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
