export default function ApproveModal({
  appId,
  businessName,
  actionLoading = false,
  onConfirm,
  onCancel,
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/40 backdrop-blur-[1px] transition-opacity"
        onClick={!actionLoading ? onCancel : undefined}
      />

      {/* Modal Card */}
      <div className="relative bg-white rounded-2xl shadow-2xl w-full max-w-[500px] overflow-hidden animate-in fade-in zoom-in-95 duration-150">
        <div className="p-4 sm:p-6">
          {/* Header */}
          <div className="flex items-start gap-4 mb-5">
            <div className="bg-[#ccfbf1] text-[#00685f] w-10 h-10 rounded-full flex items-center justify-center shrink-0 mt-0.5">
              <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
                <path
                  d="M16.6666 5L7.49992 14.1667L3.33325 10"
                  stroke="#00685f"
                  strokeWidth="2.2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
            </div>
            <div>
              <h2 className="text-[#141b2b] text-xl font-bold leading-7">
                Approve Application?
              </h2>
              <p className="text-[#3d4947] text-sm mt-1 leading-5">
                You are about to approve the organizer application from{" "}
                <span className="font-semibold text-[#141b2b]">{businessName}</span>{" "}
                (<span className="font-mono text-xs">{appId}</span>).
              </p>
            </div>
          </div>

          {/* Impact Warning */}
          <div className="bg-[#f0fdfa] border border-[#99f6e4] rounded-xl p-3.5 mb-5 text-xs text-[#134e4a] space-y-1.5">
            <div className="font-semibold text-[#0f766e] flex items-center gap-1.5">
              <svg width="15" height="15" viewBox="0 0 20 20" fill="currentColor">
                <path
                  fillRule="evenodd"
                  d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                  clipRule="evenodd"
                />
              </svg>
              Consequences of Approval
            </div>
            <ul className="list-disc list-inside space-y-1 text-[#115e59] pl-1">
              <li>Promotes the applicant user to an active <strong className="text-[#134e4a]">Organizer</strong> account.</li>
              <li>Grants immediate access to event creation, ticket sales, and payouts.</li>
              <li>Transfers submitted business and bank information to permanent records.</li>
            </ul>
          </div>

          {/* Actions */}
          <div className="flex flex-col-reverse sm:flex-row items-center justify-end gap-2 sm:gap-3">
            <button
              type="button"
              onClick={onCancel}
              disabled={actionLoading}
              className="w-full sm:w-auto border border-[#bcc9c6] bg-white text-[#141b2b] text-sm font-medium px-5 py-2.5 sm:py-2 rounded-lg hover:bg-[#f9f9ff] transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={onConfirm}
              disabled={actionLoading}
              className="w-full sm:w-auto bg-[#00685f] text-white text-sm font-semibold px-5 py-2.5 sm:py-2 rounded-lg hover:bg-[#005a52] transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2 cursor-pointer shadow-sm"
            >
              {actionLoading ? (
                <>
                  <svg className="animate-spin h-4 w-4 text-white" viewBox="0 0 24 24" fill="none">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                  </svg>
                  Approving...
                </>
              ) : (
                <>
                  <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                    <path
                      d="M11.6666 3.5L5.24992 9.91667L2.33325 7"
                      stroke="white"
                      strokeWidth="2"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    />
                  </svg>
                  Confirm Approval
                </>
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
