export default function OrganizerDetailDrawer({
  organizer,
  onClose,
  onOpenSuspend,
}) {
  if (!organizer) return null;

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
        <div className="px-6 py-5 border-b border-[#dce2f7] flex items-start justify-between bg-white shrink-0">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 rounded-full bg-[#e8eaf6] border border-[#bcc9c6] flex items-center justify-center text-base font-bold text-[#5c6bc0]">
              {organizer.name[0]}
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h2 className="text-[#141b2b] text-lg font-bold">
                  {organizer.name}
                </h2>
                <span
                  className={`px-2.5 py-0.5 rounded-full text-[10px] font-bold ${
                    organizer.status === "ACTIVE"
                      ? "bg-[#dcfce7] text-[#166534]"
                      : "bg-[#ffdad6] text-[#93000a]"
                  }`}
                >
                  {organizer.status}
                </span>
              </div>
              <p className="text-[#6d7a77] text-xs font-mono mt-0.5">
                {organizer.id} • Joined {organizer.joined}
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
        <div className="flex-1 overflow-y-auto p-6 space-y-5 bg-[#fafbff]">
          {/* Quick Metrics */}
          <div className="grid grid-cols-2 gap-3">
            <div className="bg-white border border-[#dce2f7] rounded-xl p-4 shadow-sm">
              <div className="text-[#6d7a77] text-xs font-medium uppercase tracking-wider">
                Total Events
              </div>
              <div className="text-[#141b2b] text-2xl font-bold mt-1">
                {organizer.events}
              </div>
              <div className="text-[#00685f] text-xs mt-1 font-medium">
                4 currently active
              </div>
            </div>

            <div className="bg-white border border-[#dce2f7] rounded-xl p-4 shadow-sm">
              <div className="text-[#6d7a77] text-xs font-medium uppercase tracking-wider">
                Gross Revenue
              </div>
              <div className="text-[#141b2b] text-2xl font-bold mt-1">
                {organizer.revenue}
              </div>
              <div className="text-[#00685f] text-xs mt-1 font-medium">
                8% platform fee
              </div>
            </div>
          </div>

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
                  Organization Name
                </div>
                <div className="text-[#141b2b] text-sm font-semibold">
                  {organizer.org}
                </div>
              </div>
              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  Official Email
                </div>
                <div className="text-[#00685f] text-sm font-medium">
                  {organizer.email}
                </div>
              </div>
              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  Website
                </div>
                <a
                  href={`https://${organizer.org.toLowerCase().replace(/\s+/g, "")}.in`}
                  target="_blank"
                  rel="noreferrer"
                  className="text-[#00685f] text-sm font-medium hover:underline"
                >
                  {organizer.org.toLowerCase().replace(/\s+/g, "")}.in
                </a>
              </div>
              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  Phone Number
                </div>
                <div className="text-[#141b2b] text-sm">+91 98450 11223</div>
              </div>
            </div>
          </section>

          {/* Compliance & Bank Account */}
          <section className="bg-white border border-[#dce2f7] rounded-xl overflow-hidden shadow-sm">
            <div className="bg-[#f1f3ff] px-4 py-3 border-b border-[#dce2f7]">
              <h3 className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider">
                Compliance & Payout
              </h3>
            </div>
            <div className="p-4 space-y-3">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                    PAN Card
                  </div>
                  <div className="text-[#141b2b] text-sm font-mono font-semibold">
                    AAACT2948K
                  </div>
                </div>
                <div>
                  <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                    GSTIN
                  </div>
                  <div className="text-[#141b2b] text-sm font-mono font-semibold">
                    29AAACT2948K1Z3
                  </div>
                </div>
              </div>

              <div className="border-t border-[#f1f3ff] pt-3">
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  Verified Payout Bank
                </div>
                <div className="text-[#141b2b] text-sm font-semibold">
                  HDFC Bank • Account •••• 8831
                </div>
                <div className="text-[#6d7a77] text-xs mt-0.5">
                  IFSC: HDFC0001824 • Verified KYC
                </div>
              </div>
            </div>
          </section>
        </div>

        {/* Footer Actions */}
        <div className="border-t border-[#dce2f7] px-6 py-4 flex items-center justify-between bg-white shrink-0">
          <button
            type="button"
            onClick={onClose}
            className="text-[#3d4947] text-sm font-medium hover:text-[#141b2b] px-3 py-2 rounded-lg hover:bg-[#f1f3ff] transition-colors cursor-pointer"
          >
            Close
          </button>

          {organizer.status === "ACTIVE" ? (
            <button
              type="button"
              onClick={() => onOpenSuspend(organizer)}
              className="border border-[#ba1a1a] text-[#ba1a1a] hover:bg-[#ffdad6] text-sm font-semibold px-4 py-2 rounded-lg transition-colors cursor-pointer flex items-center gap-1.5"
            >
              <svg width="14" height="14" viewBox="0 0 20 20" fill="none">
                <path
                  d="M10 1.5L1 18H19L10 1.5Z"
                  stroke="currentColor"
                  strokeWidth="1.5"
                  fill="none"
                />
                <circle cx="10" cy="15" r="0.75" fill="currentColor" />
              </svg>
              Suspend Organizer
            </button>
          ) : (
            <span className="text-[#ba1a1a] text-xs font-semibold bg-[#ffdad6] px-3 py-1.5 rounded-lg">
              Account Suspended
            </span>
          )}
        </div>
      </div>
    </div>
  );
}
