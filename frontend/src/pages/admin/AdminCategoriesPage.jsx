import { useState, useEffect, useCallback } from "react";
import {
  getAdminCategories,
  createCategory,
  updateCategory,
  toggleCategoryStatus,
} from "../../api/adminCategoryApi";
import CreateCategoryModal from "../../components/admin/CreateCategoryModal";
import EditCategoryModal from "../../components/admin/EditCategoryModal";
import ToggleCategoryStatusModal from "../../components/admin/ToggleCategoryStatusModal";
import notify from "../../utils/toast";

const PER_PAGE = 10;

export default function AdminCategoriesPage() {
  const [categories, setCategories] = useState([]);
  const [total, setTotal] = useState(0);
  const [stats, setStats] = useState({
    total: 0,
    active: 0,
    inactive: 0,
    in_use: 0,
  });
  const [activeStatus, setActiveStatus] = useState("all");
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  // Modals state
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [editCategoryTarget, setEditCategoryTarget] = useState(null);
  const [toggleStatusTarget, setToggleStatusTarget] = useState(null);
  const [targetStatusAction, setTargetStatusAction] = useState("");
  const [actionLoading, setActionLoading] = useState(false);

  // Debounce search input by 300ms
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  // Fetch categories callback
  const fetchCategories = useCallback(async () => {
    try {
      setLoading(true);
      setError("");

      const statusParam = activeStatus === "all" ? "" : activeStatus.toUpperCase();

      const res = await getAdminCategories({
        page,
        limit: PER_PAGE,
        search: debouncedSearch,
        status: statusParam,
      });

      if (res?.data) {
        setCategories(res.data.categories || []);
        setTotal(res.data.total || 0);
        if (res.data.statistics) {
          setStats(res.data.statistics);
        }
      }
    } catch (err) {
      console.error("Failed to fetch admin categories:", err);
      const msg =
        err.response?.data?.message || err.message || "Failed to load categories.";
      setError(msg);
      setCategories([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [page, activeStatus, debouncedSearch]);

  useEffect(() => {
    fetchCategories();
  }, [fetchCategories]);

  // Status tab changes
  const handleStatusChange = (statusKey) => {
    setActiveStatus(statusKey);
    setPage(1);
  };

  // Create Category Handler
  const handleCreateCategory = async (data) => {
    try {
      setActionLoading(true);
      const res = await createCategory(data);
      notify.success(res?.message || "Category created successfully!");
      setIsCreateOpen(false);
      fetchCategories();
    } catch (err) {
      throw err;
    } finally {
      setActionLoading(false);
    }
  };

  // Update Category Handler
  const handleUpdateCategory = async (id, data) => {
    try {
      setActionLoading(true);
      const res = await updateCategory(id, data);
      notify.success(res?.message || "Category updated successfully!");
      setEditCategoryTarget(null);
      fetchCategories();
    } catch (err) {
      throw err;
    } finally {
      setActionLoading(false);
    }
  };

  // Toggle Status Handler
  const handleConfirmToggleStatus = async (id, targetStatus) => {
    try {
      setActionLoading(true);
      const res = await toggleCategoryStatus(id, targetStatus);
      const actionWord = targetStatus === "ACTIVE" ? "activated" : "deactivated";
      notify.success(res?.message || `Category ${actionWord} successfully!`);
      setToggleStatusTarget(null);
      fetchCategories();
    } catch (err) {
      console.error("Failed to toggle category status:", err);
      const msg =
        err.response?.data?.message || err.message || "Failed to update category status.";
      notify.error(msg);
    } finally {
      setActionLoading(false);
    }
  };

  const totalPages = Math.max(1, Math.ceil(total / PER_PAGE));

  const statusTabs = [
    { key: "all", label: "All Categories", count: stats.total },
    { key: "active", label: "Active", count: stats.active },
    { key: "inactive", label: "Inactive", count: stats.inactive },
  ];

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-white w-full">
      {/* Top Header */}
      <div className="px-4 sm:px-6 py-4 border-b border-[#dce2f7] flex flex-col sm:flex-row sm:items-center justify-between gap-3 shrink-0">
        <div>
          <h1 className="text-[#141b2b] text-xl font-bold tracking-tight">
            Category Management
          </h1>
          <p className="text-[#3d4947] text-xs sm:text-sm">
            Create, monitor, and manage event classifications across the platform.
          </p>
        </div>
        <button
          type="button"
          onClick={() => setIsCreateOpen(true)}
          className="bg-[#00685f] hover:bg-[#00574f] text-white text-sm font-semibold px-4 py-2.5 rounded-xl flex items-center gap-2 transition-all shadow-sm cursor-pointer self-start sm:self-auto shrink-0"
        >
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
            <path d="M7 1V13M1 7H13" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
          </svg>
          Add Category
        </button>
      </div>

      {/* Main Scrollable Content */}
      <div className="flex-1 overflow-y-auto p-4 sm:p-6 space-y-5">
        {/* Error Alert */}
        {error && (
          <div className="flex items-center justify-between rounded-xl border border-[#ffdad6] bg-[#fff5f4] px-4 py-3">
            <span className="text-[#93000a] text-sm font-medium">{error}</span>
            <button
              type="button"
              onClick={fetchCategories}
              className="text-[#93000a] text-xs font-semibold cursor-pointer underline hover:no-underline ml-4"
            >
              Retry
            </button>
          </div>
        )}

        {/* Dashboard Aggregate Metric Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {/* Total Categories */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
            <div className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider mb-1">
              Total Categories
            </div>
            <div className="text-[#141b2b] text-2xl font-bold tracking-tight">
              {stats.total}
            </div>
            <div className="text-xs mt-1 text-[#00685f] font-medium">
              Registered platform taxonomies
            </div>
          </div>

          {/* Active Categories */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
            <div className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider mb-1">
              Active Categories
            </div>
            <div className="text-[#141b2b] text-2xl font-bold tracking-tight">
              {stats.active}
            </div>
            <div className="text-xs mt-1 text-[#00685f] font-medium">
              Available for public & events
            </div>
          </div>

          {/* Inactive Categories */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
            <div className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider mb-1">
              Inactive Categories
            </div>
            <div className="text-[#141b2b] text-2xl font-bold tracking-tight">
              {stats.inactive}
            </div>
            <div className="text-xs mt-1 text-[#ba1a1a] font-medium">
              Hidden from new events
            </div>
          </div>

          {/* Categories In Use */}
          <div className="bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl p-4 shadow-[0px_1px_1.5px_rgba(0,0,0,0.1)]">
            <div className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider mb-1">
              Categories In Use
            </div>
            <div className="text-[#141b2b] text-2xl font-bold tracking-tight">
              {stats.in_use}
            </div>
            <div className="text-xs mt-1 text-[#5c6bc0] font-medium">
              Linked to published/draft events
            </div>
          </div>
        </div>

        {/* Filter and Search Bar */}
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 pt-2">
          {/* Status Tabs */}
          <div className="flex items-center gap-1.5 p-1 bg-[#f1f3ff] rounded-xl border border-[#dce2f7] overflow-x-auto">
            {statusTabs.map((tab) => {
              const isSelected = activeStatus === tab.key;
              return (
                <button
                  key={tab.key}
                  type="button"
                  onClick={() => handleStatusChange(tab.key)}
                  className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all whitespace-nowrap cursor-pointer flex items-center gap-1.5 ${
                    isSelected
                      ? "bg-white text-[#141b2b] shadow-sm font-bold"
                      : "text-[#6d7a77] hover:text-[#141b2b]"
                  }`}
                >
                  <span>{tab.label}</span>
                  <span
                    className={`text-[10px] px-1.5 py-0.2 rounded-full font-bold ${
                      isSelected
                        ? "bg-[#eef0fb] text-[#141b2b]"
                        : "bg-[#e2e7f8] text-[#6d7a77]"
                    }`}
                  >
                    {tab.count}
                  </span>
                </button>
              );
            })}
          </div>

          {/* Search Box */}
          <div className="relative min-w-[240px] sm:w-72">
            <svg
              className="absolute left-3.5 top-1/2 -translate-y-1/2 w-4 h-4 text-[#6d7a77]"
              viewBox="0 0 20 20"
              fill="none"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="2"
                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
              />
            </svg>
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search category name..."
              className="w-full bg-[#f9f9ff] border border-[#bcc9c6] rounded-xl pl-9 pr-8 py-2 text-xs sm:text-sm text-[#141b2b] placeholder-[#6d7a77] focus:outline-none focus:border-[#00685f] focus:ring-1 focus:ring-[#00685f] transition-all"
            />
            {search && (
              <button
                type="button"
                onClick={() => setSearch("")}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-[#6d7a77] hover:text-[#141b2b] text-xs p-1"
              >
                ✕
              </button>
            )}
          </div>
        </div>

        {/* Categories Table Container */}
        <div className="bg-white border border-[#dce2f7] rounded-2xl overflow-hidden shadow-[0px_1px_2px_rgba(0,0,0,0.05)]">
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-[#f9f9ff] border-b border-[#dce2f7] text-[11px] font-bold uppercase tracking-wider text-[#3d4947]">
                  <th className="py-3.5 px-4 sm:px-6">Category Name</th>
                  <th className="py-3.5 px-4 sm:px-6">Description</th>
                  <th className="py-3.5 px-4 sm:px-6 text-center">Events</th>
                  <th className="py-3.5 px-4 sm:px-6 text-center">Status</th>
                  <th className="py-3.5 px-4 sm:px-6">Created Date</th>
                  <th className="py-3.5 px-4 sm:px-6 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#dce2f7] text-sm text-[#141b2b]">
                {loading ? (
                  <tr>
                    <td colSpan={6} className="py-12 text-center text-[#6d7a77]">
                      <div className="flex flex-col items-center justify-center gap-2">
                        <svg
                          className="animate-spin h-6 w-6 text-[#00685f]"
                          viewBox="0 0 24 24"
                          fill="none"
                        >
                          <circle
                            className="opacity-25"
                            cx="12"
                            cy="12"
                            r="10"
                            stroke="currentColor"
                            strokeWidth="4"
                          />
                          <path
                            className="opacity-75"
                            fill="currentColor"
                            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                          />
                        </svg>
                        <span className="text-xs">Loading categories...</span>
                      </div>
                    </td>
                  </tr>
                ) : categories.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="py-12 text-center text-[#6d7a77]">
                      <div className="flex flex-col items-center justify-center gap-2">
                        <div className="w-12 h-12 rounded-full bg-[#f1f3ff] text-[#5c6bc0] flex items-center justify-center mb-1">
                          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                            <rect x="3" y="3" width="18" height="18" rx="4" strokeWidth="1.5" />
                            <path d="M9 12h6M12 9v6" strokeWidth="2" strokeLinecap="round" />
                          </svg>
                        </div>
                        <p className="font-semibold text-sm text-[#141b2b]">No categories found</p>
                        <p className="text-xs text-[#6d7a77] max-w-xs">
                          {debouncedSearch
                            ? `No categories match the keyword "${debouncedSearch}".`
                            : "There are currently no categories in this view."}
                        </p>
                        {!debouncedSearch && (
                          <button
                            type="button"
                            onClick={() => setIsCreateOpen(true)}
                            className="mt-2 text-xs font-semibold text-[#00685f] hover:underline cursor-pointer"
                          >
                            + Create a Category
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ) : (
                  categories.map((cat) => {
                    const isActive = cat.status === "ACTIVE";
                    const formattedDate = cat.created_at
                      ? new Date(cat.created_at).toLocaleDateString("en-US", {
                          year: "numeric",
                          month: "short",
                          day: "numeric",
                        })
                      : "—";

                    return (
                      <tr
                        key={cat.id}
                        className="hover:bg-[#f9f9ff] transition-colors group"
                      >
                        {/* Name */}
                        <td className="py-4 px-4 sm:px-6 font-semibold text-[#141b2b]">
                          <div className="flex items-center gap-2.5">
                            <div className="w-8 h-8 rounded-lg bg-[#eef0fb] text-[#141b2b] flex items-center justify-center shrink-0 font-bold text-xs uppercase">
                              {cat.name.slice(0, 2)}
                            </div>
                            <span className="font-bold">{cat.name}</span>
                          </div>
                        </td>

                        {/* Description */}
                        <td className="py-4 px-4 sm:px-6 text-xs text-[#3d4947] max-w-xs truncate">
                          {cat.description || (
                            <span className="text-[#6d7a77] italic">No description provided</span>
                          )}
                        </td>

                        {/* Associated Events */}
                        <td className="py-4 px-4 sm:px-6 text-center">
                          <span
                            className={`inline-block px-2.5 py-0.5 rounded-full text-xs font-bold ${
                              cat.event_count > 0
                                ? "bg-[#e0f2fe] text-[#0369a1]"
                                : "bg-[#f1f3ff] text-[#6d7a77]"
                            }`}
                          >
                            {cat.event_count || 0}
                          </span>
                        </td>

                        {/* Status Badge */}
                        <td className="py-4 px-4 sm:px-6 text-center">
                          <span
                            className={`inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-[11px] font-bold ${
                              isActive
                                ? "bg-[#e6f4f2] text-[#00685f] border border-[#a3ded6]"
                                : "bg-[#f4f4f4] text-[#6d7a77] border border-[#dce2f7]"
                            }`}
                          >
                            <span
                              className={`w-1.5 h-1.5 rounded-full ${
                                isActive ? "bg-[#00685f]" : "bg-[#6d7a77]"
                              }`}
                            />
                            {cat.status}
                          </span>
                        </td>

                        {/* Created Date */}
                        <td className="py-4 px-4 sm:px-6 text-xs text-[#6d7a77] whitespace-nowrap">
                          {formattedDate}
                        </td>

                        {/* Actions */}
                        <td className="py-4 px-4 sm:px-6 text-right">
                          <div className="flex items-center justify-end gap-2">
                            {/* Edit Button */}
                            <button
                              type="button"
                              onClick={() => setEditCategoryTarget(cat)}
                              title="Edit Category"
                              className="p-1.5 rounded-lg text-[#3d4947] hover:text-[#141b2b] hover:bg-[#eef0fb] transition-colors cursor-pointer"
                            >
                              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                              </svg>
                            </button>

                            {/* Activate / Deactivate Toggle Button */}
                            {isActive ? (
                              <button
                                type="button"
                                onClick={() => {
                                  setToggleStatusTarget(cat);
                                  setTargetStatusAction("INACTIVE");
                                }}
                                title="Deactivate Category"
                                className="px-2.5 py-1 text-xs font-semibold rounded-lg text-[#ba1a1a] hover:bg-[#ffdad6]/60 transition-colors cursor-pointer"
                              >
                                Deactivate
                              </button>
                            ) : (
                              <button
                                type="button"
                                onClick={() => {
                                  setToggleStatusTarget(cat);
                                  setTargetStatusAction("ACTIVE");
                                }}
                                title="Activate Category"
                                className="px-2.5 py-1 text-xs font-semibold rounded-lg text-[#00685f] hover:bg-[#e6f4f2] transition-colors cursor-pointer"
                              >
                                Activate
                              </button>
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
          {!loading && total > 0 && (
            <div className="p-4 border-t border-[#dce2f7] flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-[#3d4947] bg-[#f9f9ff]">
              <div>
                Showing{" "}
                <span className="font-semibold text-[#141b2b]">
                  {Math.min((page - 1) * PER_PAGE + 1, total)}
                </span>{" "}
                to{" "}
                <span className="font-semibold text-[#141b2b]">
                  {Math.min(page * PER_PAGE, total)}
                </span>{" "}
                of <span className="font-semibold text-[#141b2b]">{total}</span>{" "}
                categories
              </div>

              <div className="flex items-center gap-1.5">
                <button
                  type="button"
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page <= 1}
                  className="px-3 py-1.5 rounded-lg border border-[#bcc9c6] text-[#141b2b] hover:bg-[#eef0fb] disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer transition-colors"
                >
                  Previous
                </button>
                <span className="px-2 font-medium">
                  Page {page} of {totalPages}
                </span>
                <button
                  type="button"
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page >= totalPages}
                  className="px-3 py-1.5 rounded-lg border border-[#bcc9c6] text-[#141b2b] hover:bg-[#eef0fb] disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer transition-colors"
                >
                  Next
                </button>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Modals */}
      <CreateCategoryModal
        isOpen={isCreateOpen}
        onClose={() => setIsCreateOpen(false)}
        onSubmit={handleCreateCategory}
        loading={actionLoading}
      />

      <EditCategoryModal
        category={editCategoryTarget}
        isOpen={Boolean(editCategoryTarget)}
        onClose={() => setEditCategoryTarget(null)}
        onSubmit={handleUpdateCategory}
        loading={actionLoading}
      />

      <ToggleCategoryStatusModal
        category={toggleStatusTarget}
        targetStatus={targetStatusAction}
        isOpen={Boolean(toggleStatusTarget)}
        onClose={() => setToggleStatusTarget(null)}
        onConfirm={handleConfirmToggleStatus}
        loading={actionLoading}
      />
    </div>
  );
}
