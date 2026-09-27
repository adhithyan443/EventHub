export default function StatusBadge({ status, className = "" }) {
  const normalizedStatus = String(status || "").toUpperCase();

  const styles = {
    PUBLISHED: {
      container: "bg-[rgba(0,131,120,0.1)] border-[rgba(0,131,120,0.2)] text-[#00685f]",
      dot: "bg-[#00685f]",
      label: "PUBLISHED",
    },
    DRAFT: {
      container: "bg-[#dce2f7] border-[#bcc9c6] text-[#3d4947]",
      dot: "bg-[#3d4947]",
      label: "DRAFT",
    },
    ONGOING: {
      container: "bg-[#e6f4ea] border-[#a8dab5] text-[#137333]",
      dot: "bg-[#137333]",
      label: "ONGOING",
    },
    COMPLETED: {
      container: "bg-[#e8f0fe] border-[#aecbfa] text-[#1a73e8]",
      dot: "bg-[#1a73e8]",
      label: "COMPLETED",
    },
    CANCELLED: {
      container: "bg-[#fce8e6] border-[#f5b3af] text-[#c5221f]",
      dot: "bg-[#c5221f]",
      label: "CANCELLED",
    },
  };

  const current = styles[normalizedStatus] || styles.DRAFT;

  return (
    <div
      className={`inline-flex items-center gap-[6px] px-[11px] py-[4px] rounded-full border border-solid text-[13px] sm:text-[14px] font-medium leading-[20px] whitespace-nowrap shrink-0 ${current.container} ${className}`}
    >
      <span className={`size-[6px] rounded-full shrink-0 ${current.dot}`} />
      <span>{current.label}</span>
    </div>
  );
}
