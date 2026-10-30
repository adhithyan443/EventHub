export default function OrganizerDetailDrawer({
  organizer,
  onClose,
}) {
  if (!organizer) return null;

  const joinedDate = organizer.created_at
    ? new Date(organizer.created_at).toLocaleDateString("en-IN", {
        day: "numeric",
        month: "short",
        year: "numeric",
      })
    : organizer.joined || "—";

  const displayName = organizer.business_name || organizer.name || "Organizer";
  const initial = displayName.trim()[0]?.toUpperCase() || "O";
  const website = organizer.website || "";
  const websiteHref = website
    ? website.startsWith("http://") || website.startsWith("https://")
      ? website
      : `https://${website}`
    : null;

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
            {organizer.logo_url ? (
              <img
                src={organizer.logo_url}
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
                    organizer.status === "ACTIVE"
                      ? "bg-[#dcfce7] text-[#166534]"
                      : "bg-[#ffdad6] text-[#93000a]"
                  }`}
                >
                  {organizer.status || "ACTIVE"}
                </span>
              </div>
              <p className="text-[#6d7a77] text-xs font-mono mt-0.5">
                {organizer.id} • Registered {joinedDate}
              </p>
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
          {/* Organization Information */}
          <section className="bg-white border border-[#dce2f7] rounded-xl overflow-hidden shadow-sm">
            <div className="bg-[#f1f3ff] px-4 py-3 border-b border-[#dce2f7]">
              <h3 className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider">
                Organization Profile
              </h3>
            </div>
            <div className="p-4 space-y-3.5">
              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  Business / Organization Name
                </div>
                <div className="text-[#141b2b] text-sm font-semibold">
                  {organizer.business_name || displayName}
                </div>
              </div>

              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  Business Type
                </div>
                <div className="text-[#141b2b] text-sm">
                  {organizer.business_type || "—"}
                </div>
              </div>

              {organizer.business_description && (
                <div>
                  <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                    Description
                  </div>
                  <p className="text-[#3d4947] text-xs leading-relaxed">
                    {organizer.business_description}
                  </p>
                </div>
              )}

              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  Official Email
                </div>
                <div className="text-[#00685f] text-sm font-medium">
                  {organizer.business_email || organizer.user_email || organizer.email || "—"}
                </div>
              </div>

              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  Phone Number
                </div>
                <div className="text-[#141b2b] text-sm">
                  {organizer.business_phone || organizer.user_phone || "—"}
                </div>
              </div>

              {websiteHref ? (
                <div>
                  <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                    Website
                  </div>
                  <a
                    href={websiteHref}
                    target="_blank"
                    rel="noreferrer"
                    className="text-[#00685f] text-sm font-medium hover:underline inline-flex items-center gap-1"
                  >
                    {website}
                    <svg width="11" height="11" viewBox="0 0 12 12" fill="none">
                      <path
                        d="M2.5 1.5H6V2.5H2.5V9.5H9.5V6H10.5V9.5C10.5 10.05 10.05 10.5 9.5 10.5H2.5C1.95 10.5 1.5 10.05 1.5 9.5V2.5C1.5 1.95 1.95 1.5 2.5 1.5ZM7.5 1.5V2.5H9.29L4.89 6.89L5.61 7.61L10 3.21V5H11V1.5H7.5Z"
                        fill="currentColor"
                      />
                    </svg>
                  </a>
                </div>
              ) : null}
            </div>
          </section>

          {/* Associated Account Owner */}
          <section className="bg-white border border-[#dce2f7] rounded-xl overflow-hidden shadow-sm">
            <div className="bg-[#f1f3ff] px-4 py-3 border-b border-[#dce2f7]">
              <h3 className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider">
                Associated User Account
              </h3>
            </div>
            <div className="p-4 space-y-3">
              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  Account Owner Name
                </div>
                <div className="text-[#141b2b] text-sm font-semibold">
                  {organizer.user_name || organizer.name || "—"}
                </div>
              </div>

              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  User Email
                </div>
                <div className="text-[#3d4947] text-sm">
                  {organizer.user_email || organizer.email || "—"}
                </div>
              </div>

              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  User Phone
                </div>
                <div className="text-[#3d4947] text-sm">
                  {organizer.user_phone || "—"}
                </div>
              </div>

              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  User ID
                </div>
                <div className="text-[#6d7a77] text-xs font-mono">
                  {organizer.user_id || "—"}
                </div>
              </div>
            </div>
          </section>

          {/* Compliance & Registration */}
          <section className="bg-white border border-[#dce2f7] rounded-xl overflow-hidden shadow-sm">
            <div className="bg-[#f1f3ff] px-4 py-3 border-b border-[#dce2f7]">
              <h3 className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider">
                Compliance & Verification
              </h3>
            </div>
            <div className="p-4 space-y-3">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                    PAN Card
                  </div>
                  <div className="text-[#141b2b] text-sm font-mono font-semibold">
                    {organizer.pan_number || "—"}
                  </div>
                </div>
                <div>
                  <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                    GSTIN
                  </div>
                  <div className="text-[#141b2b] text-sm font-mono font-semibold">
                    {organizer.gst_number || "—"}
                  </div>
                </div>
              </div>
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
