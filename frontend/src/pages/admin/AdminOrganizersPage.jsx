import { useState, useEffect } from "react";
import OrganizerDetailDrawer from "../../components/admin/OrganizerDetailDrawer";
import { getOrganizers } from "../../api/adminOrganizerApi";
import notify from "../../utils/toast";

const PER_PAGE = 10;

export default function AdminOrganizersPage() {
  const [orgs, setOrgs] = useState([]);
  const [total, setTotal] = useState(0);
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [activeDrawerOrg, setActiveDrawerOrg] = useState(null);

  // Debounce search input by 300ms
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  // Fetch organizers when page or debounced search changes
  useEffect(() => {
    let cancelled = false;

    const fetchOrganizers = async () => {
      try {
        setLoading(true);
        setError("");

        const res = await getOrganizers({
          page,
          limit: PER_PAGE,
          search: debouncedSearch,
        });

        if (cancelled) return;

        if (res?.data) {
          setOrgs(res.data.organizers || []);
          setTotal(res.data.total || 0);
        }
      } catch (err) {
        if (cancelled) return;
        console.error("Failed to fetch organizers:", err);
        const msg = err.response?.data?.message || err.message || "Failed to load organizers.";
        setError(msg);
        setOrgs([]);
        setTotal(0);
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    fetchOrganizers();

    return () => {
      cancelled = true;
    };
  }, [page, debouncedSearch]);

  const totalPages = Math.max(1, Math.ceil(total / PER_PAGE));

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-white w-full">
      {/* Header */}
      <div className="px-4 sm:px-6 py-4 border-b border-[#dce2f7] flex flex-col sm:flex-row sm:items-center justify-between gap-3 shrink-0">
        <div>
          <h1 className="text-[#141b2b] text-xl font-bold tracking-tight">
            Organizers
          </h1>
          <p className="text-[#3d4947] text-xs sm:text-sm">
            Manage approved platform organizers and view organization details.
          </p>
        </div>
        <button
          type="button"
          onClick={() => notify.info("Exporting organizers list...")}
          className="bg-[#f9f9ff] border border-[#bcc9c6] text-[#141b2b] text-sm font-semibold px-4 py-2 rounded-lg flex items-center gap-2 hover:bg-[#eef0fb] transition-colors cursor-pointer self-start sm:self-auto shrink-0"
        >
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
            <path
              d="M6 0L10.5 4.5H7.5V9H4.5V4.5H1.5L6 0ZM0 10.5H12V12H0V10.5Z"
              fill="currentColor"
            />
          </svg>
          Export Organizers
        </button>
      </div>

      {/* Main Content Area */}
      <div className="flex-1 overflow-y-auto p-4 sm:p-6 space-y-5">
        {/* Error Alert */}
        {error && (
          <div className="flex items-center justify-between rounded-lg border border-[#ffdad6] bg-[#fff5f4] px-4 py-3">
            <span className="text-[#93000a] text-sm font-medium">{error}</span>
            <button
              type="button"
              onClick={() => setError("")}
              className="text-[#93000a] text-xs font-semibold cursor-pointer hover:underline"
            >
              Dismiss
            </button>
          </div>
        )}

        {/* Aggregate Metric Cards (Accurate backend totals) */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 max-w-lg">
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
            <div className="text-[#3d4947] text-sm font-medium mb-1">
              Total Approved Organizers
            </div>
            <div className="text-[#141b2b] text-2xl font-bold tracking-tight">
              {total}
            </div>
            <div className="text-xs mt-1 font-medium text-[#00685f]">
              Platform-wide verified accounts
            </div>
          </div>

          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
            <div className="text-[#3d4947] text-sm font-medium mb-1">
              Active Status
            </div>
            <div className="text-[#141b2b] text-2xl font-bold tracking-tight">
              {total}
            </div>
            <div className="text-xs mt-1 font-medium text-[#00685f]">
              Authorized to host events
            </div>
          </div>
        </div>

        {/* Organizers Table Card */}
        <div className="bg-white border border-[#bcc9c6] rounded-xl overflow-hidden shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
          {/* Table Search & Filter Bar */}
          <div className="px-4 py-3 border-b border-[#dce2f7] flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 bg-[#f1f3ff]">
            <div className="relative flex-1 max-w-full sm:max-w-sm">
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
                placeholder="Search by business, name, or email..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full bg-white border border-[#bcc9c6] rounded-lg pl-9 pr-3 py-1.5 text-sm text-[#141b2b] placeholder:text-[#6d7a77] focus:outline-none focus:ring-1 focus:ring-[#00685f]/40"
              />
            </div>
            <span className="text-xs text-[#6d7a77] font-medium">
              {total} {total === 1 ? "organizer" : "organizers"} found
            </span>
          </div>

          {/* Table */}
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="bg-[#f9f9ff] text-[#6d7a77] uppercase tracking-wide border-b border-[#bcc9c6]">
                <tr>
                  <th className="px-4 py-3 font-semibold">Organizer / Business</th>
                  <th className="px-4 py-3 font-semibold">Account Owner</th>
                  <th className="px-4 py-3 font-semibold">Phone</th>
                  <th className="px-4 py-3 font-semibold">Business Type</th>
                  <th className="px-4 py-3 font-semibold">Status</th>
                  <th className="px-4 py-3 font-semibold">Registered</th>
                  <th className="px-4 py-3 font-semibold text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[rgba(188,201,198,0.25)]">
                {loading ? (
                  <tr>
                    <td colSpan={7} className="px-4 py-10 text-center text-[#6d7a77] text-sm">
                      <div className="flex items-center justify-center gap-2">
                        <div className="w-5 h-5 border-2 border-[#00685f]/30 border-t-[#00685f] rounded-full animate-spin" />
                        <span>Loading organizers...</span>
                      </div>
                    </td>
                  </tr>
                ) : orgs.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="px-4 py-10 text-center text-[#6d7a77] text-sm">
                      No organizers found matching your search.
                    </td>
                  </tr>
                ) : (
                  orgs.map((org, i) => {
                    const displayName = org.business_name || org.user_name || "Organizer";
                    const initial = displayName.trim()[0]?.toUpperCase() || "O";
                    const joinedFormatted = org.created_at
                      ? new Date(org.created_at).toLocaleDateString("en-IN", {
                          day: "numeric",
                          month: "short",
                          year: "numeric",
                        })
                      : "—";

                    return (
                      <tr
                        key={org.id}
                        className={`hover:bg-[#f1f3ff] transition-colors ${
                          i % 2 === 0 ? "bg-white" : "bg-[#fafbff]"
                        }`}
                      >
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2.5">
                            {org.logo_url ? (
                              <img
                                src={org.logo_url}
                                alt={displayName}
                                className="w-8 h-8 rounded-full object-cover border border-[#bcc9c6] shrink-0"
                              />
                            ) : (
                              <div className="w-8 h-8 rounded-full bg-[#e8eaf6] border border-[#bcc9c6] flex items-center justify-center text-xs font-bold text-[#5c6bc0] shrink-0">
                                {initial}
                              </div>
                            )}
                            <div>
                              <div className="font-semibold text-[#141b2b] text-sm">
                                {displayName}
                              </div>
                              <div className="text-[10px] text-[#6d7a77] font-mono">
                                {org.id}
                              </div>
                            </div>
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          <div className="font-medium text-[#141b2b]">
                            {org.user_name || "—"}
                          </div>
                          <div className="text-[#6d7a77] text-[11px]">
                            {org.user_email}
                          </div>
                        </td>
                        <td className="px-4 py-3 text-[#3d4947]">
                          {org.business_phone || org.user_phone || "—"}
                        </td>
                        <td className="px-4 py-3 text-[#141b2b]">
                          {org.business_type || "—"}
                        </td>
                        <td className="px-4 py-3">
                          <span
                            className={`px-2 py-0.5 rounded-full text-[10px] font-bold ${
                              org.status === "ACTIVE"
                                ? "bg-[#dcfce7] text-[#166534]"
                                : "bg-[#ffdad6] text-[#93000a]"
                            }`}
                          >
                            {org.status || "ACTIVE"}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-[#6d7a77]">
                          {joinedFormatted}
                        </td>
                        <td className="px-4 py-3 text-right">
                          <button
                            type="button"
                            onClick={() => setActiveDrawerOrg(org)}
                            className="text-xs text-[#00685f] font-semibold hover:underline cursor-pointer bg-[#f1f3ff] hover:bg-[#e1e8fd] px-2.5 py-1 rounded transition-colors"
                          >
                            View Details
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
          <div className="flex flex-col sm:flex-row items-center justify-between px-4 py-3 border-t border-[rgba(188,201,198,0.3)] bg-white gap-3">
            <span className="text-[#3d4947] text-xs text-center sm:text-left">
              Showing {total === 0 ? 0 : (page - 1) * PER_PAGE + 1} to{" "}
              {Math.min(page * PER_PAGE, total)} of {total} organizers
            </span>
            <div className="flex items-center gap-1">
              <button
                type="button"
                disabled={page <= 1}
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                className="w-8 h-8 rounded bg-[#f9f9ff] border border-[#bcc9c6] text-[#3d4947] hover:bg-[#eef0fb] disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer flex items-center justify-center text-xs"
              >
                ‹
              </button>

              {Array.from({ length: totalPages }, (_, i) => i + 1)
                .filter((p) => p === 1 || p === totalPages || Math.abs(p - page) <= 1)
                .map((p, idx, arr) => {
                  const prev = arr[idx - 1];
                  const hasGap = prev && p - prev > 1;

                  return (
                    <div key={p} className="flex items-center gap-1">
                      {hasGap && <span className="text-xs text-[#6d7a77]">…</span>}
                      <button
                        type="button"
                        onClick={() => setPage(p)}
                        className={`w-8 h-8 rounded text-xs font-medium transition-colors cursor-pointer ${
                          page === p
                            ? "bg-[#00685f] text-white"
                            : "bg-[#f9f9ff] border border-[#bcc9c6] text-[#3d4947] hover:bg-[#eef0fb]"
                        }`}
                      >
                        {p}
                      </button>
                    </div>
                  );
                })}

              <button
                type="button"
                disabled={page >= totalPages}
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                className="w-8 h-8 rounded bg-[#f9f9ff] border border-[#bcc9c6] text-[#3d4947] hover:bg-[#eef0fb] disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer flex items-center justify-center text-xs"
              >
                ›
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Organizer Profile Detail Drawer */}
      {activeDrawerOrg && (
        <OrganizerDetailDrawer
          organizer={activeDrawerOrg}
          onClose={() => setActiveDrawerOrg(null)}
        />
      )}
    </div>
  );
}
