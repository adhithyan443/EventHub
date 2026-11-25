const STATUS_CONFIG = {
  SUCCESS: {
    label: "VERIFIED",
    dotClass: "bg-[#00685f]",
    badgeClass: "bg-[#00685f]/10 text-[#00685f]",
    ariaDescription: "Ticket verified successfully",
  },
  CONFLICT: {
    label: "ALREADY USED / INVALID",
    dotClass: "bg-[#ba1a1a]",
    badgeClass: "bg-[#ba1a1a]/10 text-[#ba1a1a]",
    ariaDescription: "Ticket is invalid or already used",
  },
  BOOKING_QR: {
    label: "BOOKING QR",
    dotClass: "bg-amber-500",
    badgeClass: "bg-amber-100 text-amber-900 border border-amber-300",
    ariaDescription: "Booking pass scanned instead of individual ticket",
  },
  BAD_REQUEST: {
    label: "INVALID",
    dotClass: "bg-[#ba1a1a]",
    badgeClass: "bg-[#ba1a1a]/10 text-[#ba1a1a]",
    ariaDescription: "Invalid QR code payload",
  },
  UNAUTHORIZED: {
    label: "UNAUTHORIZED",
    dotClass: "bg-[#ba1a1a]",
    badgeClass: "bg-[#ba1a1a]/10 text-[#ba1a1a]",
    ariaDescription: "Not authorized to scan tickets for this event",
  },
  NOT_FOUND: {
    label: "NOT FOUND",
    dotClass: "bg-[#ba1a1a]",
    badgeClass: "bg-[#ba1a1a]/10 text-[#ba1a1a]",
    ariaDescription: "Event or ticket not found",
  },
};

const getStatusConfig = (status) => {
  return (
    STATUS_CONFIG[status] || {
      label: "REJECTED",
      dotClass: "bg-[#ba1a1a]",
      badgeClass: "bg-[#ba1a1a]/10 text-[#ba1a1a]",
      ariaDescription: "Scan rejected",
    }
  );
};

export default function RecentScansList({
  scans = [],
  onClear,
  className = "",
}) {
  const successCount = scans.filter((s) => s.status === "SUCCESS").length;
  const totalCount = scans.length;

  const formatTime = (dateObj) => {
    if (!dateObj) return "—";
    try {
      const d = dateObj instanceof Date ? dateObj : new Date(dateObj);
      if (Number.isNaN(d.getTime())) {
        return "—";
      }
      return d.toLocaleTimeString("en-IN", {
        hour: "numeric",
        minute: "2-digit",
        second: "2-digit",
        hour12: true,
      });
    } catch {
      return "—";
    }
  };

  const getScanKey = (scan, index) => {
    const timeKey =
      scan.timestamp instanceof Date
        ? scan.timestamp.getTime()
        : typeof scan.timestamp === "string" || typeof scan.timestamp === "number"
        ? scan.timestamp
        : index;
    return scan.id ? `${scan.id}-${timeKey}` : `scan-${index}-${timeKey}`;
  };

  return (
    <div
      className={`bg-white rounded-2xl border border-[#bcc9c6]/50 shadow-xs flex flex-col overflow-hidden ${className}`}
    >
      {/* Header with counter and clear */}
      <div className="p-4 border-b border-[#bcc9c6]/30 flex items-center justify-between gap-3">
        <div className="flex items-center gap-2 flex-wrap">
          <h4 id="recent-scans-heading" className="text-sm font-bold text-[#141b2b]">
            Session Log
          </h4>
          <div className="flex items-center gap-1.5 flex-wrap">
            <span
              className="text-xs px-2 py-0.5 rounded-full font-bold bg-[#00685f]/10 text-[#00685f]"
              aria-label={`${successCount} tickets verified`}
            >
              {successCount} Checked In
            </span>
            <span
              className="text-[11px] px-2 py-0.5 rounded-full font-semibold bg-gray-100 text-[#565e74]"
              aria-label={`${totalCount} total scans recorded this session`}
            >
              {totalCount} Total
            </span>
          </div>
        </div>

        {scans.length > 0 && onClear && (
          <button
            type="button"
            onClick={onClear}
            aria-label="Clear session scan history"
            className="text-[11px] font-semibold text-[#565e74] hover:text-[#ba1a1a] transition-colors cursor-pointer shrink-0"
          >
            Clear Log
          </button>
        )}
      </div>

      {/* Scans List */}
      <div
        role="feed"
        aria-labelledby="recent-scans-heading"
        className="divide-y divide-[#bcc9c6]/20 max-h-60 overflow-y-auto"
      >
        {scans.length === 0 ? (
          <div className="p-6 text-center text-xs text-[#565e74]">
            No tickets scanned yet this session.
          </div>
        ) : (
          scans.map((scan, index) => {
            const statusConfig = getStatusConfig(scan.status);
            const itemKey = getScanKey(scan, index);

            return (
              <article
                key={itemKey}
                aria-label={`${scan.ticketCode || "Scan attempt"}: ${statusConfig.ariaDescription}`}
                className="p-3.5 flex items-center justify-between gap-3 text-xs hover:bg-[#f9f9ff] transition-colors"
              >
                <div className="flex items-center gap-2.5 min-w-0">
                  <span
                    className={`size-2 rounded-full shrink-0 ${statusConfig.dotClass}`}
                    aria-hidden="true"
                  />
                  <div className="flex flex-col min-w-0">
                    <span className="font-mono font-bold text-[#141b2b] truncate">
                      {scan.ticketCode || "Scan Attempt"}
                    </span>
                    <span className="text-[11px] text-[#565e74] truncate">
                      {scan.message || statusConfig.label}
                    </span>
                  </div>
                </div>

                <div className="flex flex-col items-end shrink-0">
                  <span
                    className={`text-[10px] font-bold px-1.5 py-0.5 rounded uppercase tracking-wider ${statusConfig.badgeClass}`}
                  >
                    {statusConfig.label}
                  </span>
                  <time
                    dateTime={
                      scan.timestamp instanceof Date
                        ? scan.timestamp.toISOString()
                        : undefined
                    }
                    className="text-[10px] text-[#565e74] mt-0.5"
                  >
                    {formatTime(scan.timestamp)}
                  </time>
                </div>
              </article>
            );
          })
        )}
      </div>
    </div>
  );
}
