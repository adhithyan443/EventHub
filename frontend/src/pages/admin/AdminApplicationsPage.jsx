import { useEffect, useState } from "react";
import ReviewPanel from "../../components/admin/ReviewPanel";
import RejectModal from "../../components/admin/RejectModal";
import {
  getOrganizerApplications,
  getOrganizerApplicationById,
  approveOrganizerApplication,
  rejectOrganizerApplication,
} from "../../api/adminOrganizerApi";

const statusColors = {
  PENDING: "bg-[#fef3c7] text-[#92400e]",
  APPROVED: "bg-[#dcfce7] text-[#166534]",
  REJECTED: "bg-[#ffdad6] text-[#93000a]",
};

const PER_PAGE = 10;

export default function AdminApplicationsPage() {
  // Applications returned by the backend.
  const [apps, setApps] = useState([]);

  // Total number of applications returned by the backend.
  const [total, setTotal] = useState(0);

  // Current status tab.
  const [activeTab, setActiveTab] = useState("all");

  // Search value.
  const [search, setSearch] = useState("");

  // Application opened in the review drawer.
  const [reviewApp, setReviewApp] = useState(null);

  // Application selected for rejection.
  const [rejectTarget, setRejectTarget] = useState(null);

  // Current backend page.
  const [page, setPage] = useState(1);

  // Loading states.
  const [loading, setLoading] = useState(false);
  const [detailsLoading, setDetailsLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);

  // API error message.
  const [error, setError] = useState("");

  /*
   * Fetch applications whenever page or status changes.
   *
   * Pagination and status filtering are handled by the backend.
   */
  useEffect(() => {
    let cancelled = false;

    const loadApplications = async () => {
      try {
        setLoading(true);
        setError("");

        const response = await getOrganizerApplications({
          page,
          limit: PER_PAGE,
          status:
            activeTab === "all"
              ? ""
              : activeTab.toUpperCase(),
        });

        // Ignore the response if this effect has already been cleaned up.
        if (cancelled) {
          return;
        }

        setApps(response.data?.applications || []);
        setTotal(response.data?.total || 0);
      } catch (err) {
        // Ignore errors from an already-cancelled request.
        if (cancelled) {
          return;
        }

        console.error(
          "Failed to fetch organizer applications:",
          err
        );

        setError(
          err.response?.data?.message ||
          "Failed to load organizer applications."
        );

        setApps([]);
        setTotal(0);
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    loadApplications();

    return () => {
      cancelled = true;
    };
  }, [page, activeTab]);

  /*
   * Fetch complete application details before
   * opening the review panel.
   */
  const handleReview = async (application) => {
    try {
      setDetailsLoading(true);
      setError("");

      const response = await getOrganizerApplicationById(
        application.id
      );

      setReviewApp(response.data);
    } catch (err) {
      console.error(
        "Failed to fetch organizer application details:",
        err
      );

      setError(
        err.response?.data?.message ||
        "Failed to load application details."
      );
    } finally {
      setDetailsLoading(false);
    }
  };

  /*
   * Approve an organizer application.
   */
  const handleApprove = async (id) => {
    try {
      setActionLoading(true);
      setError("");

      await approveOrganizerApplication(id);

      setReviewApp(null);

      // Remove the approved application from the current list.
      setApps((prev) =>
        prev.filter((app) => app.id !== id)
      );

      setTotal((prev) => Math.max(0, prev - 1));
    } catch (err) {
      console.error(
        "Failed to approve organizer application:",
        err
      );

      setError(
        err.response?.data?.message ||
        "Failed to approve organizer application."
      );
    } finally {
      setActionLoading(false);
    }
  };

  /*
   * Reject an organizer application.
   *
   * The reason comes from RejectModal.
   */
  const handleReject = async (id, reason) => {
    try {
      setActionLoading(true);
      setError("");

      await rejectOrganizerApplication(id, reason);

      setRejectTarget(null);
      setReviewApp(null);

      // Remove the rejected application from the current list.
      setApps((prev) =>
        prev.filter((app) => app.id !== id)
      );

      setTotal((prev) => Math.max(0, prev - 1));
    } catch (err) {
      console.error(
        "Failed to reject organizer application:",
        err
      );

      setError(
        err.response?.data?.message ||
        "Failed to reject organizer application."
      );
    } finally {
      setActionLoading(false);
    }
  };

  /*
   * Change status tab and reset pagination.
   */
  const handleTabChange = (tab) => {
    setActiveTab(tab);
    setPage(1);
  };

  /*
   * Search the applications currently loaded on the page.
   *
   * Backend search is not implemented yet, so this remains
   * client-side for the current page.
   *
   * Applicant name comes from users.full_name through
   * the backend's `applicant_name` field.
   */
  const filtered = apps.filter((app) => {
    const query = search.trim().toLowerCase();

    if (!query) {
      return true;
    }

    return (
      app.id?.toLowerCase().includes(query) ||
      app.business_name?.toLowerCase().includes(query) ||
      app.applicant_name?.toLowerCase().includes(query) ||
      app.business_type?.toLowerCase().includes(query) ||
      app.phone?.toLowerCase().includes(query)
    );
  });

  /*
   * Backend pagination.
   */
  const totalPages = Math.max(
    1,
    Math.ceil(total / PER_PAGE)
  );

  /*
   * Since the current list API returns the total number
   * for the selected filter, use that value for All.
   *
   * Status-specific counts will represent the loaded
   * response until the backend provides aggregate counts.
   */
  const counts = {
    all: activeTab === "all" ? total : apps.length,

    pending:
      activeTab === "pending"
        ? total
        : apps.filter(
          (app) => app.status === "PENDING"
        ).length,

    approved:
      activeTab === "approved"
        ? total
        : apps.filter(
          (app) => app.status === "APPROVED"
        ).length,

    rejected:
      activeTab === "rejected"
        ? total
        : apps.filter(
          (app) => app.status === "REJECTED"
        ).length,
  };

  const tabs = [
    {
      key: "all",
      label: "All",
      count: counts.all,
    },
    {
      key: "pending",
      label: "Pending",
      count: counts.pending,
    },
    {
      key: "approved",
      label: "Approved",
      count: counts.approved,
    },
    {
      key: "rejected",
      label: "Rejected",
      count: counts.rejected,
    },
  ];

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-white w-full">
      {/* Page Header */}
      <div className="px-6 py-4 border-b border-[#dce2f7] flex items-center justify-between shrink-0">
        <div>
          <h1 className="text-[#141b2b] text-xl font-bold tracking-tight">
            Organizer Applications
          </h1>

          <p className="text-[#6d7a77] text-xs mt-0.5">
            Review and take action on incoming organizer onboarding requests.
          </p>
        </div>

        <button
          type="button"
          onClick={() =>
            alert("Exporting applications data...")
          }
          className="bg-[#f9f9ff] border border-[#bcc9c6] text-[#141b2b] text-sm font-semibold px-4 py-2 rounded-lg flex items-center gap-2 hover:bg-[#eef0fb] transition-colors cursor-pointer"
        >
          <svg
            width="12"
            height="12"
            viewBox="0 0 12 12"
            fill="none"
          >
            <path
              d="M6 0L10.5 4.5H7.5V9H4.5V4.5H1.5L6 0ZM0 10.5H12V12H0V10.5Z"
              fill="currentColor"
            />
          </svg>

          Export Applications
        </button>
      </div>

      {/* Main Content Area */}
      <div className="flex-1 overflow-y-auto p-6 space-y-5">
        {/* API Error */}
        {error && (
          <div className="flex items-center justify-between rounded-lg border border-[#ffdad6] bg-[#fff5f4] px-4 py-3">
            <span className="text-[#93000a] text-sm">
              {error}
            </span>

            <button
              type="button"
              onClick={() => setError("")}
              className="text-[#93000a] text-xs font-semibold cursor-pointer"
            >
              Dismiss
            </button>
          </div>
        )}

        {/* Metric Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {[
            {
              label: "Pending",
              count: counts.pending,
              color: "bg-[#fef3c7]",
              iconColor: "#92400e",
              icon: (
                <svg
                  width="14"
                  height="16"
                  viewBox="0 0 14 16"
                  fill="none"
                >
                  <path
                    d="M13 5.5H9.5V1.5C9.5 0.675 8.825 0 8 0H6C5.175 0 4.5 0.675 4.5 1.5V5.5H1C0.175 5.5 0 6.175 0 7L0 14.5C0 15.325 0.675 16 1.5 16H12.5C13.325 16 14 15.325 14 14.5V7C14 6.175 13.325 5.5 13 5.5Z"
                    fill="#92400e"
                  />
                </svg>
              ),
            },

            {
              label: "Approved",
              count: counts.approved,
              color: "bg-[#dcfce7]",
              iconColor: "#166534",
              icon: (
                <svg
                  width="15"
                  height="15"
                  viewBox="0 0 15 15"
                  fill="none"
                >
                  <path
                    d="M5.5 10.5L1.5 6.5L2.91 5.09L5.5 7.67L12.09 1.08L13.5 2.5L5.5 10.5Z"
                    fill="#166534"
                  />
                </svg>
              ),
            },

            {
              label: "Rejected",
              count: counts.rejected,
              color: "bg-[#ffdad6]",
              iconColor: "#93000a",
              icon: (
                <svg
                  width="15"
                  height="15"
                  viewBox="0 0 15 15"
                  fill="none"
                >
                  <path
                    d="M13.5 2.91L12.09 1.5L7.5 6.09L2.91 1.5L1.5 2.91L6.09 7.5L1.5 12.09L2.91 13.5L7.5 8.91L12.09 13.5L13.5 12.09L8.91 7.5L13.5 2.91Z"
                    fill="#93000a"
                  />
                </svg>
              ),
            },

            {
              label: "Total Applications",
              count: counts.all,
              color: "bg-[#e1e8fd]",
              iconColor: "#141b2b",
              icon: (
                <svg
                  width="13"
                  height="15"
                  viewBox="0 0 13 15"
                  fill="none"
                >
                  <path
                    d="M11.5 0H1.5C0.675 0 0 0.675 0 1.5V13.5C0 14.325 0.675 15 1.5 15H11.5C12.325 15 13 14.325 13 13.5V1.5C13 0.675 12.325 0 11.5 0ZM5 11.25L2 8.25L3.06 7.19L5 9.12L9.44 4.69L10.5 5.75L5 11.25Z"
                    fill="#141b2b"
                  />
                </svg>
              ),
            },
          ].map((card, i) => (
            <div
              key={card.label}
              className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]"
            >
              <div className="flex items-center justify-between mb-2">
                <span className="text-[#3d4947] text-sm font-semibold">
                  {card.label}
                </span>

                <span
                  className={`${card.color} w-8 h-8 rounded-lg flex items-center justify-center`}
                >
                  {card.icon}
                </span>
              </div>

              <div
                className={`text-2xl font-bold ${i === 0
                    ? "text-[#00685f]"
                    : "text-[#141b2b]"
                  }`}
              >
                {card.count}
              </div>
            </div>
          ))}
        </div>

        {/* Applications Table Card */}
        <div className="bg-white border border-[#bcc9c6] rounded-xl overflow-hidden shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
          {/* Table Controls */}
          <div className="bg-[#f1f3ff] border-b border-[#bcc9c6] px-4 py-3 flex flex-wrap items-center justify-between gap-3">
            {/* Status Tabs */}
            <div className="flex items-center">
              {tabs.map(({ key, label, count }) => (
                <button
                  key={key}
                  type="button"
                  onClick={() =>
                    handleTabChange(key)
                  }
                  className={`flex items-center gap-2 px-4 py-2 text-sm font-semibold transition-colors relative cursor-pointer ${activeTab === key
                      ? "text-[#00685f] after:absolute after:bottom-0 after:left-0 after:right-0 after:h-0.5 after:bg-[#00685f]"
                      : "text-[#3d4947] hover:text-[#141b2b]"
                    }`}
                >
                  {label}

                  {count > 0 && (
                    <span
                      className={`text-[10px] font-bold px-1.5 py-0.5 rounded-full ${activeTab === key &&
                          key === "pending"
                          ? "bg-[#fef3c7] text-[#92400e]"
                          : "bg-[#dce2f7] text-[#141b2b]"
                        }`}
                    >
                      {count}
                    </span>
                  )}
                </button>
              ))}
            </div>

            {/* Search & Filters */}
            <div className="flex items-center gap-2">
              <div className="relative">
                <svg
                  className="absolute left-3 top-1/2 -translate-y-1/2"
                  width="13"
                  height="13"
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
                  placeholder="Search applications..."
                  value={search}
                  onChange={(e) => {
                    setSearch(e.target.value);
                    setPage(1);
                  }}
                  className="bg-white border border-[#bcc9c6] rounded-lg pl-9 pr-3 py-2 text-sm text-[#141b2b] placeholder:text-[#6d7a77] w-52 focus:outline-none focus:ring-1 focus:ring-[#00685f]/40"
                />
              </div>

              <button
                type="button"
                className="bg-[#f9f9ff] border border-[#bcc9c6] text-[#141b2b] text-sm font-semibold px-3 py-2 rounded-lg flex items-center gap-2 hover:bg-[#eef0fb] cursor-pointer"
              >
                <svg
                  width="13"
                  height="9"
                  viewBox="0 0 13 9"
                  fill="none"
                >
                  <path
                    d="M0 0H13V1.5H0V0ZM2 3.75H11V5.25H2V3.75ZM4.5 7.5H8.5V9H4.5V7.5Z"
                    fill="currentColor"
                  />
                </svg>

                Filters
              </button>
            </div>
          </div>

          {/* Table */}
          <div className="overflow-x-auto">
            <table className="w-full text-left">
              <thead>
                <tr className="bg-white border-b border-[rgba(188,201,198,0.5)]">
                  <th className="px-4 py-3 text-[#3d4947] text-xs font-medium uppercase tracking-wider">
                    Application ID
                  </th>

                  <th className="px-4 py-3 text-[#3d4947] text-xs font-medium uppercase tracking-wider">
                    Business / Applicant
                  </th>

                  <th className="px-4 py-3 text-[#3d4947] text-xs font-medium uppercase tracking-wider">
                    Type
                  </th>

                  <th className="px-4 py-3 text-[#3d4947] text-xs font-medium uppercase tracking-wider">
                    Submitted
                  </th>

                  <th className="px-4 py-3 text-[#3d4947] text-xs font-medium uppercase tracking-wider">
                    Status
                  </th>

                  <th className="px-4 py-3 text-right text-[#3d4947] text-xs font-medium uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>

              <tbody>
                {loading ? (
                  <tr>
                    <td
                      colSpan={6}
                      className="px-4 py-10 text-center text-[#6d7a77] text-sm"
                    >
                      Loading applications...
                    </td>
                  </tr>
                ) : filtered.length === 0 ? (
                  <tr>
                    <td
                      colSpan={6}
                      className="px-4 py-10 text-center text-[#6d7a77] text-sm"
                    >
                      No applications found matching your criteria.
                    </td>
                  </tr>
                ) : (
                  filtered.map((app, i) => (
                    <tr
                      key={app.id}
                      className={`border-t border-[rgba(188,201,198,0.3)] ${i % 2 === 0
                          ? "bg-white"
                          : "bg-[#fafbff]"
                        } hover:bg-[#f1f3ff] transition-colors`}
                    >
                      <td className="px-4 py-4 font-mono text-[#141b2b] text-xs font-semibold">
                        {app.id}
                      </td>

                      <td className="px-4 py-4">
                        {/* Business name comes from organizer_applications.business_name */}
                        <div className="text-[#141b2b] text-sm font-medium">
                          {app.business_name || "-"}
                        </div>

                        {/* Applicant name comes from users.full_name */}
                        <div className="text-[#6d7a77] text-xs">
                          {app.applicant_name || "-"}
                        </div>
                      </td>

                      <td className="px-4 py-4 text-[#3d4947] text-sm">
                        {app.business_type || "-"}
                      </td>

                      <td className="px-4 py-4 text-[#3d4947] text-sm">
                        {app.created_at
                          ? new Date(
                            app.created_at
                          ).toLocaleDateString()
                          : "-"}
                      </td>

                      <td className="px-4 py-4">
                        <span
                          className={`px-2.5 py-1 rounded-full text-[10px] font-bold ${statusColors[
                            app.status
                            ] ||
                            "bg-gray-100 text-gray-600"
                            }`}
                        >
                          {app.status}
                        </span>
                      </td>

                      <td className="px-4 py-4 text-right">
                        <button
                          type="button"
                          onClick={() =>
                            handleReview(app)
                          }
                          disabled={
                            detailsLoading ||
                            actionLoading
                          }
                          className={`text-sm cursor-pointer transition-all disabled:opacity-50 disabled:cursor-not-allowed ${app.status === "PENDING"
                              ? "text-[#00685f] font-semibold hover:underline"
                              : "text-[#3d4947] font-medium hover:text-[#141b2b]"
                            }`}
                        >
                          {app.status === "PENDING"
                            ? "Review"
                            : "View"}
                        </button>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          {/* Pagination Footer */}
          <div className="flex items-center justify-between px-4 py-3 border-t border-[rgba(188,201,198,0.3)] bg-white">
            <span className="text-[#3d4947] text-sm">
              Showing{" "}
              {total === 0
                ? 0
                : (page - 1) * PER_PAGE + 1}{" "}
              to{" "}
              {Math.min(
                page * PER_PAGE,
                total
              )}{" "}
              of {total}{" "}
              {activeTab === "pending"
                ? "pending "
                : ""}
              applications
            </span>

            <div className="flex gap-2">
              <button
                type="button"
                onClick={() =>
                  setPage((p) =>
                    Math.max(1, p - 1)
                  )
                }
                disabled={
                  page === 1 || loading
                }
                className="bg-[#f9f9ff] border border-[#bcc9c6] text-[#3d4947] text-sm px-3 py-1 rounded disabled:opacity-40 hover:bg-[#eef0fb] transition-colors cursor-pointer disabled:cursor-not-allowed"
              >
                Previous
              </button>

              <button
                type="button"
                onClick={() =>
                  setPage((p) =>
                    Math.min(
                      totalPages,
                      p + 1
                    )
                  )
                }
                disabled={
                  page === totalPages ||
                  loading
                }
                className="bg-[#f9f9ff] border border-[#bcc9c6] text-[#3d4947] text-sm px-3 py-1 rounded disabled:opacity-40 hover:bg-[#eef0fb] transition-colors cursor-pointer disabled:cursor-not-allowed"
              >
                Next
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Review Drawer Panel */}
      {reviewApp && (
        <ReviewPanel
          app={reviewApp}
          loading={detailsLoading}
          actionLoading={actionLoading}
          onClose={() =>
            setReviewApp(null)
          }
          onReject={() =>
            setRejectTarget(reviewApp)
          }
          onApprove={() =>
            handleApprove(reviewApp.id)
          }
        />
      )}

      {/* Reject Confirmation Modal */}
      {rejectTarget && (
        <RejectModal
          appId={rejectTarget.id}
          businessName={
            rejectTarget.business_name
          }
          actionLoading={actionLoading}
          onConfirm={(reason) =>
            handleReject(
              rejectTarget.id,
              reason
            )
          }
          onCancel={() =>
            setRejectTarget(null)
          }
        />
      )}
    </div>
  );
}