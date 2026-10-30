import { useState, useEffect } from "react";
import { getUsers, blockUser, unblockUser } from "../../api/adminUserApi";
import UserDetailDrawer from "../../components/admin/UserDetailDrawer";
import BlockUserModal from "../../components/admin/BlockUserModal";
import UnblockUserModal from "../../components/admin/UnblockUserModal";
import notify from "../../utils/toast";

const PER_PAGE = 10;

const roleBadgeStyles = {
  CUSTOMER: "bg-[#eef0fb] text-[#141b2b] border border-[#dce2f7]",
  ORGANIZER: "bg-[#e0f2fe] text-[#0369a1] border border-[#bae6fd]",
  ADMIN: "bg-[#f3e8ff] text-[#6b21a8] border border-[#e9d5ff]",
};

export default function AdminUsersPage() {
  const [users, setUsers] = useState([]);
  const [total, setTotal] = useState(0);
  const [roleCounts, setRoleCounts] = useState({
    total: 0,
    customers: 0,
    organizers: 0,
    admins: 0,
  });
  const [activeRole, setActiveRole] = useState("all");
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  // Modal / Drawer state
  const [activeDrawerUser, setActiveDrawerUser] = useState(null);
  const [activeBlockUser, setActiveBlockUser] = useState(null);
  const [activeUnblockUser, setActiveUnblockUser] = useState(null);
  const [actionLoading, setActionLoading] = useState(false);

  // Debounce search input by 300ms
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  // Fetch users when page, role, or debounced search changes
  useEffect(() => {
    let cancelled = false;

    const fetchUsers = async () => {
      try {
        setLoading(true);
        setError("");

        const roleParam = activeRole === "all" ? "" : activeRole.toUpperCase();

        const res = await getUsers({
          page,
          limit: PER_PAGE,
          search: debouncedSearch,
          role: roleParam,
        });

        if (cancelled) return;

        if (res?.data) {
          setUsers(res.data.users || []);
          setTotal(res.data.total || 0);
          if (res.data.role_counts) {
            setRoleCounts(res.data.role_counts);
          }
        }
      } catch (err) {
        if (cancelled) return;
        console.error("Failed to fetch users:", err);
        const msg = err.response?.data?.message || err.message || "Failed to load users.";
        setError(msg);
        setUsers([]);
        setTotal(0);
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    fetchUsers();

    return () => {
      cancelled = true;
    };
  }, [page, activeRole, debouncedSearch]);

  const handleRoleChange = (role) => {
    setActiveRole(role);
    setPage(1);
  };

  const handleConfirmBlock = async (userId, reason) => {
    try {
      setActionLoading(true);
      await blockUser(userId, reason);
      notify.success("User account suspended successfully");

      // Optimistically update local users state
      setUsers((prev) =>
        prev.map((item) =>
          item.id === userId ? { ...item, status: "BLOCKED" } : item
        )
      );

      if (activeDrawerUser?.id === userId) {
        setActiveDrawerUser((prev) =>
          prev ? { ...prev, status: "BLOCKED" } : null
        );
      }

      setActiveBlockUser(null);
    } catch (err) {
      console.error("Failed to block user:", err);
      const msg = err.response?.data?.message || err.message || "Failed to suspend user.";
      notify.error(msg);
    } finally {
      setActionLoading(false);
    }
  };

  const handleConfirmUnblock = async (userId) => {
    try {
      setActionLoading(true);
      await unblockUser(userId);
      notify.success("User account reactivated successfully");

      // Optimistically update local users state
      setUsers((prev) =>
        prev.map((item) =>
          item.id === userId ? { ...item, status: "ACTIVE" } : item
        )
      );

      if (activeDrawerUser?.id === userId) {
        setActiveDrawerUser((prev) =>
          prev ? { ...prev, status: "ACTIVE" } : null
        );
      }

      setActiveUnblockUser(null);
    } catch (err) {
      console.error("Failed to unblock user:", err);
      const msg = err.response?.data?.message || err.message || "Failed to reactivate user.";
      notify.error(msg);
    } finally {
      setActionLoading(false);
    }
  };

  const totalPages = Math.max(1, Math.ceil(total / PER_PAGE));

  const roleTabs = [
    { key: "all", label: "All Users", count: roleCounts.total },
    { key: "customer", label: "Customers", count: roleCounts.customers },
    { key: "organizer", label: "Organizers", count: roleCounts.organizers },
    { key: "admin", label: "Admins", count: roleCounts.admins },
  ];

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-white w-full">
      {/* Header */}
      <div className="px-4 sm:px-6 py-4 border-b border-[#dce2f7] flex flex-col sm:flex-row sm:items-center justify-between gap-3 shrink-0">
        <div>
          <h1 className="text-[#141b2b] text-xl font-bold tracking-tight">
            Users
          </h1>
          <p className="text-[#3d4947] text-xs sm:text-sm">
            Manage registered platform users and view account details.
          </p>
        </div>
        <button
          type="button"
          onClick={() => notify.info("Exporting users list...")}
          className="bg-[#f9f9ff] border border-[#bcc9c6] text-[#141b2b] text-sm font-semibold px-4 py-2 rounded-lg flex items-center gap-2 hover:bg-[#eef0fb] transition-colors cursor-pointer self-start sm:self-auto shrink-0"
        >
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
            <path
              d="M6 0L10.5 4.5H7.5V9H4.5V4.5H1.5L6 0ZM0 10.5H12V12H0V10.5Z"
              fill="currentColor"
            />
          </svg>
          Export Users
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

        {/* Aggregate Metric Cards (Platform-wide counts from backend) */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
            <div className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider mb-1">
              Total Users
            </div>
            <div className="text-[#141b2b] text-2xl font-bold tracking-tight">
              {roleCounts.total}
            </div>
            <div className="text-xs mt-1 text-[#00685f] font-medium">
              Registered platform accounts
            </div>
          </div>

          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
            <div className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider mb-1">
              Customers
            </div>
            <div className="text-[#141b2b] text-2xl font-bold tracking-tight">
              {roleCounts.customers}
            </div>
            <div className="text-xs mt-1 text-[#5c6bc0] font-medium">
              Ticket buyers & attendees
            </div>
          </div>

          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
            <div className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider mb-1">
              Organizers
            </div>
            <div className="text-[#141b2b] text-2xl font-bold tracking-tight">
              {roleCounts.organizers}
            </div>
            <div className="text-xs mt-1 text-[#00685f] font-medium">
              Approved event creators
            </div>
          </div>

          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
            <div className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider mb-1">
              Administrators
            </div>
            <div className="text-[#141b2b] text-2xl font-bold tracking-tight">
              {roleCounts.admins}
            </div>
            <div className="text-xs mt-1 text-[#6b21a8] font-medium">
              Platform ops & admins
            </div>
          </div>
        </div>

        {/* Users Table Card */}
        <div className="bg-white border border-[#bcc9c6] rounded-xl overflow-hidden shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
          {/* Table Controls (Role Tabs & Search) */}
          <div className="bg-[#f1f3ff] border-b border-[#bcc9c6] px-4 py-3 flex flex-wrap items-center justify-between gap-3">
            {/* Role Tabs */}
            <div className="flex items-center">
              {roleTabs.map(({ key, label, count }) => (
                <button
                  key={key}
                  type="button"
                  onClick={() => handleRoleChange(key)}
                  className={`flex items-center gap-2 px-3 sm:px-4 py-2 text-xs sm:text-sm font-semibold transition-colors relative cursor-pointer ${
                    activeRole === key
                      ? "text-[#00685f] after:absolute after:bottom-0 after:left-0 after:right-0 after:h-0.5 after:bg-[#00685f]"
                      : "text-[#3d4947] hover:text-[#141b2b]"
                  }`}
                >
                  {label}
                  {count > 0 && (
                    <span
                      className={`text-[10px] font-bold px-1.5 py-0.5 rounded-full ${
                        activeRole === key
                          ? "bg-[#00685f] text-white"
                          : "bg-[#dce2f7] text-[#141b2b]"
                      }`}
                    >
                      {count}
                    </span>
                  )}
                </button>
              ))}
            </div>

            {/* Search Input */}
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
                placeholder="Search name, email, phone..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="bg-white border border-[#bcc9c6] rounded-lg pl-9 pr-3 py-1.5 text-sm text-[#141b2b] placeholder:text-[#6d7a77] w-56 sm:w-64 focus:outline-none focus:ring-1 focus:ring-[#00685f]/40"
              />
            </div>
          </div>

          {/* Table */}
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="bg-[#f9f9ff] text-[#6d7a77] uppercase tracking-wide border-b border-[#bcc9c6]">
                <tr>
                  <th className="px-4 py-3 font-semibold">User</th>
                  <th className="px-4 py-3 font-semibold">Email</th>
                  <th className="px-4 py-3 font-semibold">Phone</th>
                  <th className="px-4 py-3 font-semibold">Role</th>
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
                        <span>Loading users...</span>
                      </div>
                    </td>
                  </tr>
                ) : users.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="px-4 py-10 text-center text-[#6d7a77] text-sm">
                      No users found matching your search or filters.
                    </td>
                  </tr>
                ) : (
                  users.map((u, i) => {
                    const initial = (u.full_name || "U").trim()[0]?.toUpperCase() || "U";
                    const joinedFormatted = u.created_at
                      ? new Date(u.created_at).toLocaleDateString("en-IN", {
                          day: "numeric",
                          month: "short",
                          year: "numeric",
                        })
                      : "—";

                    const roleBadgeClass =
                      roleBadgeStyles[u.role] || "bg-[#f1f3ff] text-[#141b2b]";
                    const isBlocked = u.status === "BLOCKED";
                    const isAdmin = u.role === "ADMIN";

                    return (
                      <tr
                        key={u.id}
                        className={`hover:bg-[#f1f3ff] transition-colors ${
                          i % 2 === 0 ? "bg-white" : "bg-[#fafbff]"
                        }`}
                      >
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2.5">
                            {u.profile_image ? (
                              <img
                                src={u.profile_image}
                                alt={u.full_name}
                                className="w-8 h-8 rounded-full object-cover border border-[#bcc9c6] shrink-0"
                              />
                            ) : (
                              <div className="w-8 h-8 rounded-full bg-[#e8eaf6] border border-[#bcc9c6] flex items-center justify-center text-xs font-bold text-[#5c6bc0] shrink-0">
                                {initial}
                              </div>
                            )}
                            <div>
                              <div className="font-semibold text-[#141b2b] text-sm">
                                {u.full_name || "—"}
                              </div>
                              <div className="text-[10px] text-[#6d7a77] font-mono">
                                {u.id}
                              </div>
                            </div>
                          </div>
                        </td>
                        <td className="px-4 py-3 text-[#3d4947]">
                          {u.email}
                        </td>
                        <td className="px-4 py-3 text-[#3d4947]">
                          {u.phone || "—"}
                        </td>
                        <td className="px-4 py-3">
                          <span
                            className={`px-2 py-0.5 rounded-full text-[10px] font-bold tracking-wide uppercase ${roleBadgeClass}`}
                          >
                            {u.role}
                          </span>
                        </td>
                        <td className="px-4 py-3">
                          <span
                            className={`px-2 py-0.5 rounded-full text-[10px] font-bold ${
                              isBlocked
                                ? "bg-[#ffdad6] text-[#93000a]"
                                : "bg-[#dcfce7] text-[#166534]"
                            }`}
                          >
                            {u.status || "ACTIVE"}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-[#6d7a77]">
                          {joinedFormatted}
                        </td>
                        <td className="px-4 py-3 text-right">
                          <div className="flex items-center justify-end gap-1.5">
                            <button
                              type="button"
                              onClick={() => setActiveDrawerUser(u)}
                              className="text-xs text-[#00685f] font-semibold hover:underline cursor-pointer bg-[#f1f3ff] hover:bg-[#e1e8fd] px-2.5 py-1 rounded transition-colors"
                            >
                              View Details
                            </button>
                            {!isAdmin && (
                              isBlocked ? (
                                <button
                                  type="button"
                                  onClick={() => setActiveUnblockUser(u)}
                                  className="text-xs text-[#00685f] font-semibold hover:bg-[#dcfce7]/60 border border-[#bbf7d0] px-2 py-1 rounded transition-colors cursor-pointer"
                                  title="Reactivate user account"
                                >
                                  Unblock
                                </button>
                              ) : (
                                <button
                                  type="button"
                                  onClick={() => setActiveBlockUser(u)}
                                  className="text-xs text-[#93000a] font-semibold hover:bg-[#ffdad6]/60 border border-[#ffdad6] px-2 py-1 rounded transition-colors cursor-pointer"
                                  title="Suspend user account"
                                >
                                  Block
                                </button>
                              )
                            )}
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
          <div className="flex flex-col sm:flex-row items-center justify-between px-4 py-3 border-t border-[rgba(188,201,198,0.3)] bg-white gap-3">
            <span className="text-[#3d4947] text-xs text-center sm:text-left">
              Showing {total === 0 ? 0 : (page - 1) * PER_PAGE + 1} to{" "}
              {Math.min(page * PER_PAGE, total)} of {total} users
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

      {/* User Details Slide-over Drawer */}
      {activeDrawerUser && (
        <UserDetailDrawer
          user={activeDrawerUser}
          onClose={() => setActiveDrawerUser(null)}
          onBlockClick={(u) => setActiveBlockUser(u)}
          onUnblockClick={(u) => setActiveUnblockUser(u)}
        />
      )}

      {/* Block User Confirmation Modal */}
      {activeBlockUser && (
        <BlockUserModal
          user={activeBlockUser}
          loading={actionLoading}
          onConfirm={handleConfirmBlock}
          onCancel={() => {
            if (!actionLoading) setActiveBlockUser(null);
          }}
        />
      )}

      {/* Unblock User Confirmation Modal */}
      {activeUnblockUser && (
        <UnblockUserModal
          user={activeUnblockUser}
          loading={actionLoading}
          onConfirm={handleConfirmUnblock}
          onCancel={() => {
            if (!actionLoading) setActiveUnblockUser(null);
          }}
        />
      )}
    </div>
  );
}
