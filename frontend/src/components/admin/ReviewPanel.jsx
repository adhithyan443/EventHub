export default function ReviewPanel({
  app,
  onClose,
  onReject,
  onApprove,
}) {
  if (!app) return null;

  const submittedDate = app.created_at
    ? new Date(app.created_at).toLocaleDateString()
    : "-";

  const websiteUrl = app.website
    ? app.website.startsWith("http://") ||
      app.website.startsWith("https://")
      ? app.website
      : `https://${app.website}`
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
        <div className="px-6 py-5 border-b border-[#dce2f7] flex items-start justify-between bg-white shrink-0">
          <div>
            <div className="flex items-center gap-2 mb-1">
              <h2 className="text-[#141b2b] text-xl font-bold">
                Review Application
              </h2>

              <span
                className={`px-2.5 py-0.5 rounded-full text-[10px] font-bold ${app.status === "PENDING"
                    ? "bg-[#fef3c7] text-[#92400e]"
                    : app.status === "APPROVED"
                      ? "bg-[#dcfce7] text-[#166534]"
                      : "bg-[#ffdad6] text-[#93000a]"
                  }`}
              >
                {app.status}
              </span>
            </div>

            <p className="text-[#3d4947] text-sm font-medium">
              <span className="font-mono">{app.id}</span>{" "}
              • Submitted {submittedDate}
            </p>
          </div>

          <button
            type="button"
            onClick={onClose}
            className="text-[#3d4947] hover:text-[#141b2b] p-1.5 rounded-lg hover:bg-[#f1f3ff] transition-colors cursor-pointer"
          >
            <svg
              width="18"
              height="18"
              viewBox="0 0 18 18"
              fill="none"
            >
              <path
                d="M17 1L1 17M1 1L17 17"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
              />
            </svg>
          </button>
        </div>

        {/* Scrollable Content */}
        <div className="flex-1 overflow-y-auto p-6 space-y-5 bg-[#fafbff]">
          {/* Business Information Card */}
          <section className="bg-white border border-[#dce2f7] rounded-xl overflow-hidden shadow-sm">
            <div className="bg-[#f1f3ff] px-4 py-3 border-b border-[#dce2f7]">
              <h3 className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider">
                Business Information
              </h3>
            </div>

            <div className="p-4 space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                    Business Name
                  </div>

                  <div className="text-[#141b2b] text-sm font-semibold">
                    {app.business_name || "-"}
                  </div>
                </div>

                <div>
                  <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                    Entity Type
                  </div>

                  <div className="text-[#141b2b] text-sm font-semibold">
                    {app.business_type || "-"}
                  </div>
                </div>
              </div>

              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  Description
                </div>

                <div className="text-[#3d4947] text-sm leading-5">
                  {app.description || "-"}
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                    Website
                  </div>

                  {websiteUrl ? (
                    <a
                      href={websiteUrl}
                      target="_blank"
                      rel="noreferrer"
                      className="text-[#00685f] text-sm font-medium hover:underline break-all"
                    >
                      {app.website}
                    </a>
                  ) : (
                    <div className="text-[#3d4947] text-sm">
                      -
                    </div>
                  )}
                </div>

                <div>
                  <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                    Business Phone
                  </div>

                  <div className="text-[#141b2b] text-sm">
                    {app.phone || "-"}
                  </div>
                </div>
              </div>
            </div>
          </section>

          {/* Verification Documents Card */}
          <section className="bg-white border border-[#dce2f7] rounded-xl overflow-hidden shadow-sm">
            <div className="bg-[#f1f3ff] px-4 py-3 border-b border-[#dce2f7]">
              <h3 className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider">
                Verification Documents
              </h3>
            </div>

            <div className="p-4 space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                    PAN Number
                  </div>

                  <div className="text-[#141b2b] text-sm font-mono font-semibold">
                    {app.pan_number || "-"}
                  </div>
                </div>

                <div>
                  <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                    GSTIN
                  </div>

                  <div className="text-[#141b2b] text-sm font-mono font-semibold">
                    {app.gst_number || "-"}
                  </div>
                </div>
              </div>

              {/* Document attachment tile */}
              <div className="border border-[#dce2f7] bg-[#f9f9ff] rounded-lg flex items-center justify-between px-3.5 py-2.5">
                <div className="flex items-center gap-3">
                  <div className="w-8 h-8 bg-[#e8eaf6] rounded flex items-center justify-center shrink-0">
                    <svg
                      width="14"
                      height="16"
                      viewBox="0 0 14 16"
                      fill="none"
                    >
                      <path
                        d="M8.5 0H1.5C0.675 0 0 0.675 0 1.5V14.5C0 15.325 0.675 16 1.5 16H12.5C13.325 16 14 15.325 14 14.5V5.5L8.5 0ZM12.5 14.5H1.5V1.5H7.75V6.25H12.5V14.5Z"
                        fill="#5c6bc0"
                      />
                    </svg>
                  </div>

                  <div>
                    <div className="text-[#141b2b] text-sm font-medium">
                      {app.verification_document_url
                        ? "Verification Document"
                        : "No verification document"}
                    </div>

                    <div className="text-[#6d7a77] text-xs">
                      {app.verification_document_url
                        ? "PDF Document"
                        : "-"}
                    </div>
                  </div>
                </div>

                {app.verification_document_url && (
                  <a
                    href={
                      app.verification_document_url
                    }
                    target="_blank"
                    rel="noreferrer"
                    className="text-[#00685f] text-sm font-medium hover:underline flex items-center gap-1 cursor-pointer shrink-0"
                  >
                    View

                    <svg
                      width="10"
                      height="10"
                      viewBox="0 0 10 10"
                      fill="none"
                    >
                      <path
                        d="M1 9L9 1M9 1H4M9 1V6"
                        stroke="#00685f"
                        strokeWidth="1.2"
                        strokeLinecap="round"
                      />
                    </svg>
                  </a>
                )}
              </div>
            </div>
          </section>

          {/* Primary Contact Card */}
          <section className="bg-white border border-[#dce2f7] rounded-xl overflow-hidden shadow-sm">
            <div className="bg-[#f1f3ff] px-4 py-3 border-b border-[#dce2f7]">
              <h3 className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider">
                Primary Contact
              </h3>
            </div>

            <div className="p-4 grid grid-cols-2 gap-4">
              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  Name
                </div>

                <div className="text-[#141b2b] text-sm font-semibold">
                  {app.applicant_name || "-"}
                </div>
              </div>

              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  Role
                </div>

                <div className="text-[#141b2b] text-sm">
                  -
                </div>
              </div>

              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  Email
                </div>

                <div className="text-[#00685f] text-sm font-medium break-all">
                  -
                </div>
              </div>

              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  Phone
                </div>

                <div className="text-[#141b2b] text-sm">
                  {app.phone || "-"}
                </div>
              </div>
            </div>
          </section>

          {/* Bank Details Card */}
          <section className="bg-white border border-[#dce2f7] rounded-xl overflow-hidden shadow-sm">
            <div className="bg-[#f1f3ff] px-4 py-3 border-b border-[#dce2f7]">
              <h3 className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider">
                Bank Details
              </h3>
            </div>

            <div className="p-4 grid grid-cols-2 gap-4">
              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  Bank Name
                </div>

                <div className="text-[#141b2b] text-sm font-semibold">
                  {app.bank_name || "-"}
                </div>
              </div>

              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  Account Holder
                </div>

                <div className="text-[#141b2b] text-sm font-semibold">
                  {app.account_holder_name || "-"}
                </div>
              </div>

              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  Account Number
                </div>

                <div className="text-[#141b2b] text-sm font-mono">
                  ••••••4521
                </div>
              </div>

              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  IFSC Code
                </div>

                <div className="text-[#141b2b] text-sm font-mono font-semibold">
                  {app.ifsc_code || "-"}
                </div>
              </div>
            </div>
          </section>

          {/* Address Information Card */}
          <section className="bg-white border border-[#dce2f7] rounded-xl overflow-hidden shadow-sm">
            <div className="bg-[#f1f3ff] px-4 py-3 border-b border-[#dce2f7]">
              <h3 className="text-[#3d4947] text-xs font-semibold uppercase tracking-wider">
                Address
              </h3>
            </div>

            <div className="p-4 space-y-3">
              <div>
                <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                  Address Line
                </div>

                <div className="text-[#141b2b] text-sm">
                  {app.address_line || "-"}
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                    City
                  </div>

                  <div className="text-[#141b2b] text-sm">
                    {app.city || "-"}
                  </div>
                </div>

                <div>
                  <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                    State
                  </div>

                  <div className="text-[#141b2b] text-sm">
                    {app.state || "-"}
                  </div>
                </div>

                <div>
                  <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                    Country
                  </div>

                  <div className="text-[#141b2b] text-sm">
                    {app.country || "-"}
                  </div>
                </div>

                <div>
                  <div className="text-[#6d7a77] text-xs uppercase tracking-wide mb-1 font-medium">
                    Postal Code
                  </div>

                  <div className="text-[#141b2b] text-sm">
                    {app.postal_code || "-"}
                  </div>
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
            Back
          </button>

          <div className="flex items-center gap-3">
            {app.status === "PENDING" && (
              <>
                <button
                  type="button"
                  onClick={onReject}
                  className="border border-[#93000a] text-[#93000a] text-sm font-semibold px-4 py-2 rounded-lg hover:bg-[#ffdad6] transition-colors cursor-pointer"
                >
                  Reject
                </button>

                <button
                  type="button"
                  onClick={onApprove}
                  className="bg-[#00685f] text-white text-sm font-semibold px-4 py-2 rounded-lg hover:bg-[#005a52] transition-colors cursor-pointer shadow-sm"
                >
                  Approve Application
                </button>
              </>
            )}

            {app.status === "APPROVED" && (
              <span className="bg-[#dcfce7] text-[#166534] text-sm font-semibold px-4 py-2 rounded-lg inline-flex items-center gap-1.5">
                <svg
                  width="14"
                  height="14"
                  viewBox="0 0 15 15"
                  fill="none"
                >
                  <path
                    d="M5.5 10.5L1.5 6.5L2.91 5.09L5.5 7.67L12.09 1.08L13.5 2.5L5.5 10.5Z"
                    fill="currentColor"
                  />
                </svg>

                Approved
              </span>
            )}

            {app.status === "REJECTED" && (
              <span className="bg-[#ffdad6] text-[#93000a] text-sm font-semibold px-4 py-2 rounded-lg inline-flex items-center gap-1.5">
                <svg
                  width="14"
                  height="14"
                  viewBox="0 0 15 15"
                  fill="none"
                >
                  <path
                    d="M13.5 2.91L12.09 1.5L7.5 6.09L2.91 1.5L1.5 2.91L1.5 12.09L2.91 13.5L7.5 8.91L12.09 13.5L13.5 12.09L8.91 7.5L13.5 2.91Z"
                    fill="currentColor"
                  />
                </svg>

                Rejected
              </span>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}