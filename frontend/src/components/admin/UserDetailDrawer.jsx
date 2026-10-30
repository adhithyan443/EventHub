import { useState, useEffect } from "react";
import { getUserById } from "../../api/adminUserApi";
import notify from "../../utils/toast";

const roleBadgeStyles = {
  CUSTOMER: "bg-[#eef0fb] text-[#141b2b] border border-[#dce2f7]",
  ORGANIZER: "bg-[#e0f2fe] text-[#0369a1] border border-[#bae6fd]",
  ADMIN: "bg-[#f3e8ff] text-[#6b21a8] border border-[#e9d5ff]",
};

export default function UserDetailDrawer({
  user,
  onClose,
  onBlockClick,
  onUnblockClick,
}) {
  const [details, setDetails] = useState(user);
  const [loading, setLoading] = useState(false);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    let cancelled = false;

    if (user?.id) {
      setDetails(user);
      setLoading(true);
      getUserById(user.id)
        .then((res) => {
          if (!cancelled && res?.data) {
            setDetails(res.data);
          }
        })
        .catch((err) => {
          if (!cancelled) {
            console.error("Failed to fetch fresh user details:", err);
          }
        })
        .finally(() => {
          if (!cancelled) {
            setLoading(false);
          }
        });
    }

    return () => {
      cancelled = true;
    };
  }, [user]);

  if (!user) return null;

  const currentData = details || user;

  const joinedDate = currentData.created_at
    ? new Date(currentData.created_at).toLocaleDateString("en-IN", {
        day: "numeric",
        month: "short",
        year: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      })
    : "—";

  const updatedDate = currentData.updated_at
    ? new Date(currentData.updated_at).toLocaleDateString("en-IN", {
        day: "numeric",
        month: "short",
        year: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      })
    : "—";

  const displayName = currentData.full_name || currentData.name || "User";
  const initial = displayName.trim()[0]?.toUpperCase() || "U";
  const roleBadgeClass =
    roleBadgeStyles[currentData.role] || "bg-[#f1f3ff] text-[#141b2b]";

  const handleCopyId = () => {
    if (navigator.clipboard && currentData.id) {
      navigator.clipboard.writeText(currentData.id);
      setCopied(true);
      notify.success("User ID copied to clipboard");
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const isBlocked = currentData.status === "BLOCKED";
  const isAdmin = currentData.role === "ADMIN";

  return (
    <div className="fixed inset-0 z-40 flex justify-end animate-in fade-in duration-200">
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/30 backdrop-blur-[1px] transition-opacity"
        onClick={onClose}
      />

      {/* Slide Drawer */}
      <div className="relative w-full max-w-[480px] bg-white shadow-2xl flex flex-col h-full overflow-hidden z-10 animate-in slide-in-from-right duration-200">
        {/* Header */}
        <div className="px-4 sm:px-6 py-4 sm:py-5 border-b border-[#dce2f7] flex items-start justify-between bg-white shrink-0">
          <div className="flex items-center gap-3">
            {currentData.profile_image ? (
              <img
                src={currentData.profile_image}
                alt={displayName}
                className="w-12 h-12 rounded-full object-cover border border-[#bcc9c6] shrink-0"
              />
            ) : (
              <div className="w-12 h-12 rounded-full bg-[#e8eaf6] border border-[#bcc9c6] flex items-center justify-center text-base font-bold text-[#5c6bc0] shrink-0">
                {initial}
              </div>
            )}
            <div>
              <div className="flex items-center gap-2">
                <h2 className="text-[#141b2b] text-base sm:text-lg font-bold">
                  {displayName}
                </h2>
                <span
                  className={`px-2.5 py-0.5 rounded-full text-[10px] font-bold ${
                    isBlocked
                      ? "bg-[#ffdad6] text-[#93000a]"
                      : "bg-[#dcfce7] text-[#166534]"
                  }`}
                >
                  {currentData.status || "ACTIVE"}
                </span>
              </div>
              <div className="flex items-center gap-2 mt-0.5">
                <span
                  className={`px-2 py-0.2 rounded text-[10px] font-bold uppercase tracking-wide ${roleBadgeClass}`}
                >
                  {currentData.role}
                </span>
                <span className="text-[#6d7a77] text-xs font-mono">
                  Joined {currentData.created_at ? new Date(currentData.created_at).toLocaleDateString("en-IN", { month: "short", year: "numeric" }) : "—"}
                </span>
              </div>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="text-[#3d4947] hover:text-[#141b2b] p-1.5 rounded-lg hover:bg-[#f1f3ff] transition-colors cursor-pointer"
          >
            <svg width="18" height="18" viewBox="0 0 18 18" fill="none">
              <path
                d="M17 1L1 17M1 1L17 17"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
              />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-4 sm:p-6 space-y-4 sm:space-y-5 bg-[#fafbff]">
          {/* Profile Details */}
          <section className="bg-white border border-[#dce2f7] rounded-xl overflow-hidden shadow-sm">
            <div className="bg-[#f1f3ff] px-4 py-3 border-b border-[#dce2f7]">
              <h3 className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider">
                Personal Information
              </h3>
            </div>
            <div className="p-4 space-y-3.5">
              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  Full Name
                </div>
                <div className="text-[#141b2b] text-sm font-semibold">
                  {displayName}
                </div>
              </div>

              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  Email Address
                </div>
                <a
                  href={`mailto:${currentData.email}`}
                  className="text-[#00685f] text-sm font-medium hover:underline inline-flex items-center gap-1"
                >
                  {currentData.email}
                  <svg width="11" height="11" viewBox="0 0 12 12" fill="none">
                    <path
                      d="M2.5 1.5H6V2.5H2.5V9.5H9.5V6H10.5V9.5C10.5 10.05 10.05 10.5 9.5 10.5H2.5C1.95 10.5 1.5 10.05 1.5 9.5V2.5C1.5 1.95 1.95 1.5 2.5 1.5ZM7.5 1.5V2.5H9.29L4.89 6.89L5.61 7.61L10 3.21V5H11V1.5H7.5Z"
                      fill="currentColor"
                    />
                  </svg>
                </a>
              </div>

              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  Phone Number
                </div>
                <div className="text-[#141b2b] text-sm">
                  {currentData.phone || "—"}
                </div>
              </div>
            </div>
          </section>

          {/* Account & Security Metadata */}
          <section className="bg-white border border-[#dce2f7] rounded-xl overflow-hidden shadow-sm">
            <div className="bg-[#f1f3ff] px-4 py-3 border-b border-[#dce2f7]">
              <h3 className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider">
                Account & Security Metadata
              </h3>
            </div>
            <div className="p-4 space-y-3.5">
              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium flex items-center justify-between">
                  <span>User ID</span>
                  <button
                    type="button"
                    onClick={handleCopyId}
                    className="text-[#00685f] text-[11px] font-semibold hover:underline cursor-pointer flex items-center gap-1"
                  >
                    {copied ? "Copied!" : "Copy ID"}
                  </button>
                </div>
                <div className="text-[#141b2b] text-xs font-mono bg-[#f9f9ff] px-2.5 py-1.5 rounded border border-[#bcc9c6]/50 select-all">
                  {currentData.id}
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                    Platform Role
                  </div>
                  <div className="text-[#141b2b] text-sm font-semibold">
                    {currentData.role}
                  </div>
                </div>

                <div>
                  <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                    Account Status
                  </div>
                  <div className="text-[#141b2b] text-sm font-semibold">
                    {currentData.status || "ACTIVE"}
                  </div>
                </div>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1 border-t border-[rgba(188,201,198,0.3)]">
                <div>
                  <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-0.5 font-medium">
                    Registered On
                  </div>
                  <div className="text-[#3d4947] text-xs">
                    {joinedDate}
                  </div>
                </div>

                <div>
                  <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-0.5 font-medium">
                    Last Updated
                  </div>
                  <div className="text-[#3d4947] text-xs">
                    {updatedDate}
                  </div>
                </div>
              </div>
            </div>
          </section>

          {/* Account Actions / Danger Zone */}
          <section className="bg-white border border-[#ffdad6] rounded-xl overflow-hidden shadow-sm">
            <div className="bg-[#fff5f4] px-4 py-3 border-b border-[#ffdad6]">
              <h3 className="text-[#93000a] text-xs font-semibold uppercase tracking-wider">
                Account Status Management
              </h3>
            </div>
            <div className="p-4 space-y-3">
              {isAdmin ? (
                <p className="text-xs text-[#6d7a77]">
                  This account has administrator privileges and cannot be suspended from this interface.
                </p>
              ) : isBlocked ? (
                <div>
                  <p className="text-xs text-[#3d4947] mb-3 leading-relaxed">
                    This user account is currently <strong className="text-[#93000a]">suspended</strong>.
                    Reactivating will restore normal authentication, event interactions, and booking abilities.
                  </p>
                  <button
                    type="button"
                    onClick={() => onUnblockClick && onUnblockClick(currentData)}
                    className="w-full bg-[#00685f] text-white text-xs font-semibold px-4 py-2.5 rounded-lg hover:bg-[#00524b] transition-colors cursor-pointer flex items-center justify-center gap-2 shadow-sm"
                  >
                    <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                      <path
                        d="M11.6666 3.5L5.24992 9.91667L2.33325 7"
                        stroke="white"
                        strokeWidth="1.75"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                      />
                    </svg>
                    Reactivate / Unblock User
                  </button>
                </div>
              ) : (
                <div>
                  <p className="text-xs text-[#3d4947] mb-3 leading-relaxed">
                    Suspending this account will immediately revoke all active refresh tokens, invalidate their sessions, and deny access to all protected endpoints.
                  </p>
                  <button
                    type="button"
                    onClick={() => onBlockClick && onBlockClick(currentData)}
                    className="w-full bg-[#93000a] text-white text-xs font-semibold px-4 py-2.5 rounded-lg hover:bg-[#7d000a] transition-colors cursor-pointer flex items-center justify-center gap-2 shadow-sm"
                  >
                    <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                      <path
                        d="M7 0C3.13 0 0 3.13 0 7C0 10.87 3.13 14 7 14C10.87 14 14 10.87 14 7C14 3.13 10.87 0 7 0ZM8.5 10.5H5.5V6.5H8.5V10.5ZM8.5 5H5.5V2H8.5V5Z"
                        fill="white"
                      />
                    </svg>
                    Suspend / Block User
                  </button>
                </div>
              )}
            </div>
          </section>
        </div>

        {/* Footer */}
        <div className="border-t border-[#dce2f7] px-6 py-4 flex items-center justify-end bg-white shrink-0">
          <button
            type="button"
            onClick={onClose}
            className="bg-[#f9f9ff] border border-[#bcc9c6] text-[#141b2b] text-sm font-semibold px-5 py-2 rounded-lg hover:bg-[#eef0fb] transition-colors cursor-pointer"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
