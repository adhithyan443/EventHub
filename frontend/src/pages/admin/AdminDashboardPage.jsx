import { useState, useEffect, useCallback, useMemo } from "react";
import { Link, useNavigate } from "react-router-dom";
import RevenueChart from "../../components/admin/RevenueChart.jsx";
import { getAdminDashboard } from "../../api/adminDashboardApi.js";
import { getOrganizerApplications } from "../../api/adminOrganizerApi.js";
import { getAdminEvents } from "../../api/adminEventApi.js";
import { formatCurrency, formatRelativeTime } from "../../utils/formatters.js";

export default function AdminDashboardPage() {
  const navigate = useNavigate();
  const [chartMode, setChartMode] = useState("monthly");

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [dashboardData, setDashboardData] = useState(null);
  const [pendingApplications, setPendingApplications] = useState([]);
  const [pendingAppsCount, setPendingAppsCount] = useState(0);
  const [eventStats, setEventStats] = useState(null);
  const [refreshKey, setRefreshKey] = useState(0);

  const handleRefresh = useCallback(() => {
    setLoading(true);
    setRefreshKey((prev) => prev + 1);
  }, []);

  useEffect(() => {
    let cancelled = false;

    const loadData = async () => {
      setError("");

      try {
        const [dashRes, appsRes, eventsRes] = await Promise.allSettled([
          getAdminDashboard(),
          getOrganizerApplications({ page: 1, limit: 3, status: "PENDING" }),
          getAdminEvents({ page: 1, limit: 1 }),
        ]);

        if (cancelled) return;

        if (dashRes.status === "fulfilled" && dashRes.value?.data) {
          setDashboardData(dashRes.value.data);
        } else {
          const errMsg =
            dashRes.reason?.response?.data?.message ||
            dashRes.reason?.message ||
            "Failed to load dashboard metrics.";
          setError(errMsg);
        }

        if (appsRes.status === "fulfilled" && appsRes.value?.data) {
          const apps = Array.isArray(appsRes.value.data.applications)
            ? appsRes.value.data.applications
            : Array.isArray(appsRes.value.data)
            ? appsRes.value.data
            : [];
          setPendingApplications(apps);
          setPendingAppsCount(
            typeof appsRes.value.data.total_records === "number"
              ? appsRes.value.data.total_records
              : typeof appsRes.value.data.total === "number"
              ? appsRes.value.data.total
              : apps.length
          );
        }

        if (eventsRes.status === "fulfilled" && eventsRes.value?.data?.status_counts) {
          setEventStats(eventsRes.value.data.status_counts);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err?.message || "An unexpected error occurred while loading dashboard.");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    loadData();

    return () => {
      cancelled = true;
    };
  }, [refreshKey]);

  const platform = useMemo(() => dashboardData?.platform_overview || {}, [dashboardData]);
  const financial = useMemo(() => dashboardData?.financial_overview || {}, [dashboardData]);
  const currency = financial.currency || "INR";

  // Subtitle date range for the Platform Revenue chart
  const revenueChartSubtitle = useMemo(() => {
    const trends = dashboardData?.trends?.revenue_trends;
    if (!Array.isArray(trends) || trends.length === 0) {
      return "Past 12 Months";
    }
    const firstPeriod = trends[0]?.period || "";
    const lastPeriod = trends[trends.length - 1]?.period || "";
    if (firstPeriod && lastPeriod) {
      return `${firstPeriod} – ${lastPeriod}`;
    }
    return "Monthly Overview";
  }, [dashboardData]);

  // Derived event statistics (using status_counts from admin events API or falling back to platform overview)
  const eventBreakdownItems = useMemo(() => {
    if (eventStats) {
      return [
        { label: "Published", count: eventStats.published ?? 0, color: "bg-[#00685f]" },
        { label: "Draft", count: eventStats.draft ?? 0, color: "bg-[#bcc9c6]" },
        { label: "Ongoing", count: eventStats.ongoing ?? 0, color: "bg-[#5c6bc0]" },
        { label: "Completed", count: eventStats.completed ?? 0, color: "bg-[#00685f]/60" },
        { label: "Cancelled", count: eventStats.cancelled ?? 0, color: "bg-[#ef4444]" },
      ];
    }
    return [
      { label: "Active Events", count: platform.active_events ?? 0, color: "bg-[#00685f]" },
      { label: "Total Events", count: platform.total_events ?? 0, color: "bg-[#5c6bc0]" },
    ];
  }, [eventStats, platform]);

  const platformMetricCards = [
    {
      label: "Total Users",
      value: (platform.total_users ?? 0).toLocaleString("en-IN"),
      description: "Registered platform users",
      icon: (
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
          <path
            d="M13 10C14.66 10 15.99 8.66 15.99 7C15.99 5.34 14.66 4 13 4C11.34 4 10 5.34 10 7C10 8.66 11.34 10 13 10ZM5 10C6.66 10 7.99 8.66 7.99 7C7.99 5.34 6.66 4 5 4C3.34 4 2 5.34 2 7C2 8.66 3.34 10 5 10ZM5 12C3.34 12 0 12.84 0 14.5V16H10V14.5C10 12.84 6.66 12 5 12ZM13 12C12.79 12 12.55 12.01 12.3 12.03C13.19 12.68 13.83 13.55 13.83 14.5V16H18V14.5C18 12.84 14.66 12 13 12Z"
            fill="#5c6bc0"
          />
        </svg>
      ),
      bg: "bg-[#e8eaf6]",
    },
    {
      label: "Total Organizers",
      value: (platform.total_organizers ?? 0).toLocaleString("en-IN"),
      description: "Approved organizers",
      icon: (
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
          <path
            d="M10 2C7.79 2 6 3.79 6 6C6 8.21 7.79 10 10 10C12.21 10 14 8.21 14 6C14 3.79 12.21 2 10 2ZM10 12C7.33 12 2 13.34 2 16V18H18V16C18 13.34 12.67 12 10 12Z"
            fill="#5c6bc0"
          />
        </svg>
      ),
      bg: "bg-[#e8eaf6]",
    },
    {
      label: "Total Events",
      value: (platform.total_events ?? 0).toLocaleString("en-IN"),
      description: `${(platform.active_events ?? 0).toLocaleString("en-IN")} published & active`,
      icon: (
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
          <path
            d="M17 2H16V0H14V2H6V0H4V2H3C1.9 2 1 2.9 1 4V18C1 19.1 1.9 20 3 20H17C18.1 20 19 19.1 19 18V4C19 2.9 18.1 2 17 2ZM17 18H3V7H17V18ZM3 5H17V7H3V5Z"
            fill="#5c6bc0"
          />
        </svg>
      ),
      bg: "bg-[#e8eaf6]",
    },
    {
      label: "Active Events",
      value: (platform.active_events ?? 0).toLocaleString("en-IN"),
      description: "Live on marketplace",
      icon: (
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
          <circle cx="10" cy="10" r="8" fill="#00685f" />
          <path
            d="M8 13.5L5 10.5L6.41 9.09L8 10.67L13.59 5.08L15 6.5L8 13.5Z"
            fill="white"
          />
        </svg>
      ),
      bg: "bg-[#dcfce7]",
    },
    {
      label: "Total Bookings",
      value: (platform.total_bookings ?? 0).toLocaleString("en-IN"),
      description: "Total reservations recorded",
      icon: (
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
          <path
            d="M18 2H2C1.1 2 0 2.9 0 4V14C0 15.1 1.1 16 2 16H18C18.9 16 20 15.1 20 14V4C20 2.9 18.9 2 18 2ZM18 14H2V8H18V14ZM18 6H2V4H18V6Z"
            fill="#5c6bc0"
          />
        </svg>
      ),
      bg: "bg-[#e8eaf6]",
    },
    {
      label: "Tickets Sold",
      value: (platform.tickets_sold ?? 0).toLocaleString("en-IN"),
      description: "Confirmed ticket admissions",
      icon: (
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
          <path
            d="M20 10C20 8.9 19.1 8 18 8V6C18 4.9 17.1 4 16 4H4C2.9 4 2 4.9 2 6V8C0.9 8 0 8.9 0 10C0 11.1 0.9 12 2 12V14C2 15.1 2.9 16 4 16H16C17.1 16 18 15.1 18 14V12C19.1 12 20 11.1 20 10Z"
            fill="#5c6bc0"
          />
        </svg>
      ),
      bg: "bg-[#e8eaf6]",
    },
  ];

  const financialMetricCards = [
    {
      label: "Gross Captured Amount",
      value: formatCurrency(financial.gross_captured_amount, currency),
      description: "Total gateway payment volume",
      icon: (
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
          <circle cx="10" cy="10" r="8" fill="#00685f" />
          <path
            d="M10 5V15M7.5 7.5H11.5C12.33 7.5 13 8.17 13 9C13 9.83 12.33 10.5 11.5 10.5H8.5C7.67 10.5 7 11.17 7 12C7 12.83 7.67 13.5 8.5 13.5H12.5"
            stroke="white"
            strokeWidth="1.5"
            strokeLinecap="round"
          />
        </svg>
      ),
      bg: "bg-[#dcfce7]",
    },
    {
      label: "Platform Fee Revenue",
      value: formatCurrency(financial.platform_fee_revenue, currency),
      description: "Account 4010 (Net Revenue)",
      icon: (
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
          <path
            d="M10 2L3 7V17L10 19L17 17V7L10 2ZM10 11.5C8.62 11.5 7.5 10.38 7.5 9C7.5 7.62 8.62 6.5 10 6.5C11.38 6.5 12.5 7.62 12.5 9C12.5 10.38 11.38 11.5 10 11.5Z"
            fill="#0f766e"
          />
        </svg>
      ),
      bg: "bg-[#ccfbf1]",
    },
    {
      label: "Organizer Payable",
      value: formatCurrency(financial.organizer_payable_liability, currency),
      description: "Account 2010 (Unsettled Liability)",
      icon: (
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
          <path
            d="M4 4H16V16H4V4ZM2 2V18H18V2H2ZM6 7H14V9H6V7ZM6 11H14V13H6V11Z"
            fill="#d97706"
          />
        </svg>
      ),
      bg: "bg-[#fef3c7]",
    },
    {
      label: "Tax Liability",
      value: formatCurrency(financial.tax_liability, currency),
      description: "Account 2030 (Tax Payable)",
      icon: (
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
          <path
            d="M10 2C5.58 2 2 5.58 2 10C2 14.42 5.58 18 10 18C14.42 18 18 14.42 18 10C18 5.58 14.42 2 10 2ZM11 14H9V12H11V14ZM11 10H9V6H11V10Z"
            fill="#475569"
          />
        </svg>
      ),
      bg: "bg-[#f1f5f9]",
    },
    {
      label: "Total Refunds",
      value: formatCurrency(financial.total_refunds, currency),
      description: "Processed customer & event refunds",
      icon: (
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
          <circle cx="10" cy="10" r="8" fill="#ffdad6" />
          <path
            d="M13.5 7.41L12.09 6L10 8.09L7.91 6L6.5 7.41L8.59 9.5L6.5 11.59L7.91 13L10 10.91L12.09 13L13.5 11.59L11.41 9.5L13.5 7.41Z"
            fill="#93000a"
          />
        </svg>
      ),
      bg: "bg-[#ffdad6]",
    },
    {
      label: "Completed Settlements",
      value: formatCurrency(financial.completed_settlements, currency),
      description: "Settled payouts transferred",
      icon: (
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
          <circle cx="10" cy="10" r="8" fill="#e0e7ff" />
          <path
            d="M6 10L9 13L14 7"
            stroke="#4338ca"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </svg>
      ),
      bg: "bg-[#e0e7ff]",
    },
    {
      label: "Pending Settlements",
      value: formatCurrency(financial.pending_settlements, currency),
      description: "In-flight payout holds",
      icon: (
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
          <circle cx="10" cy="10" r="8" fill="#fef3c7" />
          <path
            d="M10 6V10L13 12"
            stroke="#b45309"
            strokeWidth="1.8"
            strokeLinecap="round"
          />
        </svg>
      ),
      bg: "bg-[#fef3c7]",
    },
  ];

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-white w-full">
      {/* Page Header */}
      <div className="px-4 sm:px-6 py-4 sm:py-5 border-b border-[#dce2f7] shrink-0 flex items-center justify-between">
        <div>
          <h1 className="text-[#141b2b] text-xl sm:text-2xl font-bold leading-8 tracking-tight">
            Admin Dashboard
          </h1>
          <p className="text-[#3d4947] text-xs sm:text-sm mt-0.5">
            Real-time platform activity and authoritative financial ledger overview.
          </p>
        </div>
        <div>
          <button
            type="button"
            onClick={handleRefresh}
            disabled={loading}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold text-[#00685f] bg-[#f0fdfa] border border-[#00685f]/30 rounded-lg hover:bg-[#ccfbf1] transition-colors disabled:opacity-50 cursor-pointer"
          >
            <svg
              className={`w-3.5 h-3.5 ${loading ? "animate-spin" : ""}`}
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
            >
              <path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67" />
            </svg>
            Refresh
          </button>
        </div>
      </div>

      {/* Main Scrollable View */}
      <div className="flex-1 overflow-y-auto px-4 sm:px-6 py-4 sm:py-6 space-y-6">
        {/* Error Alert with Retry */}
        {error && (
          <div className="bg-[#fff1f2] border border-[#fecdd3] rounded-xl p-4 flex items-center justify-between text-sm text-[#9f1239]">
            <div className="flex items-center gap-2">
              <svg className="w-5 h-5 shrink-0 text-[#e11d48]" viewBox="0 0 20 20" fill="currentColor">
                <path
                  fillRule="evenodd"
                  d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
                  clipRule="evenodd"
                />
              </svg>
              <span>{error}</span>
            </div>
            <button
              type="button"
              onClick={handleRefresh}
              className="px-3 py-1 bg-white border border-[#fecdd3] text-xs font-semibold rounded-lg hover:bg-[#ffe4e6] transition-colors cursor-pointer"
            >
              Retry
            </button>
          </div>
        )}

        {/* Loading Skeleton */}
        {loading && !dashboardData && (
          <div className="space-y-6">
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-4 animate-pulse">
              {[...Array(6)].map((_, i) => (
                <div key={i} className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 h-28" />
              ))}
            </div>
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 animate-pulse">
              <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-5 h-64" />
              <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-5 h-64" />
            </div>
          </div>
        )}

        {/* Loaded Content */}
        {dashboardData && (
          <>
            {/* Section 1: Platform Overview Cards */}
            <section>
              <h2 className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider mb-3">
                Platform Volume Overview
              </h2>
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-4">
                {platformMetricCards.map((card, i) => (
                  <div
                    key={i}
                    className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)] hover:border-[#00685f]/40 transition-colors"
                  >
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-[#3d4947] text-xs font-medium truncate" title={card.label}>
                        {card.label}
                      </span>
                      <span className={`${card.bg} flex items-center justify-center rounded-lg w-7 h-7 shrink-0`}>
                        {card.icon}
                      </span>
                    </div>
                    <div className="text-[#141b2b] text-xl sm:text-2xl font-bold leading-7 tracking-tight">
                      {card.value}
                    </div>
                    <div className="text-[#6d7a77] text-[11px] mt-1 truncate" title={card.description}>
                      {card.description}
                    </div>
                  </div>
                ))}
              </div>
            </section>

            {/* Section 2: Financial Ledger Overview Cards */}
            <section>
              <h2 className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider mb-3">
                Financial Ledger Overview ({currency})
              </h2>
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                {financialMetricCards.map((card, i) => (
                  <div
                    key={i}
                    className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)] hover:border-[#00685f]/40 transition-colors"
                  >
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-[#3d4947] text-xs font-medium truncate" title={card.label}>
                        {card.label}
                      </span>
                      <span className={`${card.bg} flex items-center justify-center rounded-lg w-7 h-7 shrink-0`}>
                        {card.icon}
                      </span>
                    </div>
                    <div className="text-[#141b2b] text-xl font-bold leading-7 tracking-tight">
                      {card.value}
                    </div>
                    <div className="text-[#6d7a77] text-[11px] mt-1 truncate" title={card.description}>
                      {card.description}
                    </div>
                  </div>
                ))}
              </div>
            </section>

            {/* Middle Row: Pending Applications + Platform Revenue Chart */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
              {/* Pending Applications Card */}
              <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-5 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)] flex flex-col justify-between">
                <div>
                  <div className="flex items-center justify-between mb-4">
                    <div className="flex items-center gap-2">
                      <h2 className="text-[#141b2b] text-base font-semibold">
                        Pending Applications
                      </h2>
                      <span className="bg-[#fef3c7] text-[#92400e] text-xs font-bold px-2 py-0.5 rounded-full">
                        {pendingAppsCount}
                      </span>
                    </div>
                    <Link
                      to="/admin/applications"
                      className="text-[#00685f] text-xs font-semibold hover:underline"
                    >
                      View All
                    </Link>
                  </div>

                  {pendingApplications.length === 0 ? (
                    <div className="py-8 text-center text-xs text-[#6d7a77]">
                      No pending organizer applications awaiting review.
                    </div>
                  ) : (
                    <div className="space-y-2.5">
                      {pendingApplications.map((app, i) => (
                        <div
                          key={app.id || i}
                          onClick={() => navigate("/admin/applications")}
                          className="flex items-center justify-between bg-white border border-[#dce2f7] rounded-lg px-4 py-3 hover:bg-[#f1f3ff] transition-colors cursor-pointer"
                        >
                          <div className="flex items-center gap-3">
                            <div className="w-8 h-8 bg-[#e8eaf6] rounded-full flex items-center justify-center text-[#5c6bc0] text-xs font-bold uppercase">
                              {(app.business_name || app.name || "A")[0]}
                            </div>
                            <div>
                              <div className="text-[#141b2b] text-sm font-medium">
                                {app.business_name || app.name || "Organizer Applicant"}
                              </div>
                              <div className="text-[#6d7a77] text-xs">
                                {app.created_at ? formatRelativeTime(app.created_at) : "Pending"}
                              </div>
                            </div>
                          </div>
                          <svg width="6" height="10" viewBox="0 0 6 10" fill="none">
                            <path
                              d="M1 1L5 5L1 9"
                              stroke="#3d4947"
                              strokeWidth="1.5"
                              strokeLinecap="round"
                              strokeLinejoin="round"
                            />
                          </svg>
                        </div>
                      ))}
                    </div>
                  )}
                </div>

                <button
                  type="button"
                  onClick={() => navigate("/admin/applications")}
                  className="w-full mt-4 border border-[#bcc9c6] bg-white rounded-lg py-2 text-[#141b2b] text-sm font-semibold hover:bg-[#eef0fb] transition-colors cursor-pointer"
                >
                  Review Applications
                </button>
              </div>

              {/* Platform Revenue Area Chart */}
              <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-5 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)] flex flex-col justify-between">
                <div className="flex items-center justify-between mb-2">
                  <div>
                    <h2 className="text-[#141b2b] text-base font-semibold">
                      Platform Fee Revenue
                    </h2>
                    <p className="text-[#6d7a77] text-xs">{revenueChartSubtitle}</p>
                  </div>
                  <div className="flex gap-1 bg-white p-0.5 border border-[#bcc9c6] rounded-lg">
                    <button
                      type="button"
                      onClick={() => setChartMode("monthly")}
                      className={`text-xs font-semibold px-3 py-1 rounded-md transition-colors cursor-pointer ${
                        chartMode === "monthly"
                          ? "bg-[#00685f] text-white"
                          : "text-[#3d4947] hover:text-[#141b2b]"
                      }`}
                    >
                      Monthly
                    </button>
                    <button
                      type="button"
                      onClick={() => setChartMode("quarterly")}
                      className={`text-xs font-semibold px-3 py-1 rounded-md transition-colors cursor-pointer ${
                        chartMode === "quarterly"
                          ? "bg-[#00685f] text-white"
                          : "text-[#3d4947] hover:text-[#141b2b]"
                      }`}
                    >
                      Quarterly
                    </button>
                  </div>
                </div>

                <div className="w-full mt-2">
                  <RevenueChart
                    trends={dashboardData.trends?.revenue_trends || []}
                    mode={chartMode}
                  />
                </div>
              </div>
            </div>

            {/* Bottom Row: Payment Breakdown & Event Statistics + Recent Bookings */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
              {/* Payment Status Breakdown & Event Statistics */}
              <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-5 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)] space-y-6">
                {/* Payment Breakdown */}
                <div>
                  <div className="flex items-center justify-between mb-3">
                    <h2 className="text-[#141b2b] text-base font-semibold">
                      Payment Status Breakdown
                    </h2>
                    <Link
                      to="/admin/payments"
                      className="text-[#00685f] text-xs font-semibold hover:underline"
                    >
                      All Payments
                    </Link>
                  </div>
                  <div className="space-y-2.5">
                    {Array.isArray(dashboardData.payment_status_breakdown) &&
                    dashboardData.payment_status_breakdown.length > 0 ? (
                      dashboardData.payment_status_breakdown.map((item, idx) => {
                        const statusColors = {
                          SUCCESS: "bg-[#dcfce7] text-[#166534] border-[#bbf7d0]",
                          PENDING: "bg-[#fef3c7] text-[#92400e] border-[#fde68a]",
                          FAILED: "bg-[#fee2e2] text-[#991b1b] border-[#fecaca]",
                          REFUND_REQUIRED: "bg-[#f3e8ff] text-[#6b21a8] border-[#e9d5ff]",
                        };
                        const badgeStyle =
                          statusColors[item.status] || "bg-[#f1f5f9] text-[#475569] border-[#e2e8f0]";

                        return (
                          <div
                            key={idx}
                            className="flex items-center justify-between bg-white border border-[#dce2f7] rounded-lg px-3.5 py-2 text-xs"
                          >
                            <div className="flex items-center gap-2">
                              <span
                                className={`px-2 py-0.5 rounded-full font-bold text-[10px] border ${badgeStyle}`}
                              >
                                {item.status}
                              </span>
                              <span className="text-[#6d7a77]">
                                ({item.count} {item.count === 1 ? "payment" : "payments"})
                              </span>
                            </div>
                            <span className="font-semibold text-[#141b2b]">
                              {formatCurrency(item.amount, currency)}
                            </span>
                          </div>
                        );
                      })
                    ) : (
                      <div className="text-center py-3 text-xs text-[#6d7a77]">
                        No payment breakdown data available.
                      </div>
                    )}
                  </div>
                </div>

                {/* Event Statistics */}
                <div className="pt-4 border-t border-[rgba(188,201,198,0.3)]">
                  <div className="flex items-center justify-between mb-3">
                    <h2 className="text-[#141b2b] text-base font-semibold">
                      Event Statistics
                    </h2>
                    <Link
                      to="/admin/events"
                      className="text-[#00685f] text-xs font-semibold hover:underline"
                    >
                      Manage Events
                    </Link>
                  </div>
                  <div className="space-y-2">
                    {eventBreakdownItems.map((item, i) => (
                      <div
                        key={i}
                        className="flex items-center justify-between py-1 border-b border-[rgba(188,201,198,0.2)] last:border-0"
                      >
                        <div className="flex items-center gap-2.5">
                          <span className={`w-2.5 h-2.5 rounded-full ${item.color}`} />
                          <span className="text-[#3d4947] text-xs font-medium">
                            {item.label}
                          </span>
                        </div>
                        <span className="text-[#141b2b] text-xs font-bold">
                          {typeof item.count === "number" ? item.count.toLocaleString("en-IN") : item.count}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>

              {/* Recent Bookings Table */}
              <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-5 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)] flex flex-col justify-between">
                <div>
                  <div className="flex items-center justify-between mb-4">
                    <h2 className="text-[#141b2b] text-base font-semibold">
                      Recent Bookings
                    </h2>
                    <button
                      type="button"
                      onClick={() => navigate("/admin/payments")}
                      className="text-[#00685f] text-xs font-semibold hover:underline cursor-pointer"
                    >
                      View All
                    </button>
                  </div>

                  {Array.isArray(dashboardData.recent_bookings) &&
                  dashboardData.recent_bookings.length > 0 ? (
                    <div className="overflow-x-auto">
                      <table className="w-full text-xs">
                        <thead>
                          <tr className="text-[#6d7a77] uppercase tracking-wide border-b border-[rgba(188,201,198,0.4)]">
                            <th className="text-left pb-2 font-medium">Booking #</th>
                            <th className="text-left pb-2 font-medium">Customer</th>
                            <th className="text-left pb-2 font-medium">Event</th>
                            <th className="text-left pb-2 font-medium">Amount</th>
                            <th className="text-left pb-2 font-medium">Status</th>
                          </tr>
                        </thead>
                        <tbody>
                          {dashboardData.recent_bookings.map((b, i) => {
                            const isPaid =
                              b.payment_status === "SUCCESS" || b.status === "CONFIRMED";
                            return (
                              <tr
                                key={b.id || i}
                                className="border-t border-[rgba(188,201,198,0.25)] hover:bg-white/60 transition-colors"
                              >
                                <td className="py-2.5 font-mono text-[#141b2b] font-medium truncate max-w-[100px]">
                                  {b.booking_number || b.id}
                                </td>
                                <td className="py-2.5 text-[#3d4947] font-medium truncate max-w-[110px]" title={b.customer_name || b.customer_email}>
                                  {b.customer_name || b.customer_email || "Customer"}
                                </td>
                                <td className="py-2.5 text-[#3d4947] truncate max-w-[120px]" title={b.event_title}>
                                  {b.event_title || "—"}
                                </td>
                                <td className="py-2.5 text-[#141b2b] font-semibold">
                                  {formatCurrency(b.amount, currency)}
                                </td>
                                <td className="py-2.5">
                                  <span
                                    className={`px-2 py-0.5 rounded-full text-[10px] font-bold ${
                                      isPaid
                                        ? "bg-[#dcfce7] text-[#166534]"
                                        : "bg-[#fef3c7] text-[#92400e]"
                                    }`}
                                  >
                                    {b.payment_status || b.status}
                                  </span>
                                </td>
                              </tr>
                            );
                          })}
                        </tbody>
                      </table>
                    </div>
                  ) : (
                    <div className="py-12 text-center text-xs text-[#6d7a77]">
                      No recent bookings recorded yet.
                    </div>
                  )}
                </div>

                <div className="mt-4 pt-3 border-t border-[rgba(188,201,198,0.3)] flex justify-end">
                  <span className="text-[11px] text-[#6d7a77]">
                    Showing latest {dashboardData.recent_bookings?.length || 0} bookings
                  </span>
                </div>
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
