export default function ProgressBar({ pct = 0, className = "", barClassName = "" }) {
  const clampedPct = Math.min(100, Math.max(0, Number(pct) || 0));

  return (
    <div className={`bg-[#e1e8fd] h-[6px] rounded-full w-full overflow-hidden ${className}`}>
      <div
        className={`bg-[#00685f] h-[6px] rounded-full transition-all duration-300 ${barClassName}`}
        style={{ width: `${clampedPct}%` }}
      />
    </div>
  );
}
