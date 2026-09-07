export default function AdminPlaceholderPage({ title = "Section" }) {
  return (
    <div className="flex-1 flex flex-col items-center justify-center p-8 text-[#6d7a77] gap-3 bg-white w-full">
      <div className="w-16 h-16 rounded-2xl bg-[#f1f3ff] border border-[#dce2f7] flex items-center justify-center text-[#5c6bc0]">
        <svg width="32" height="32" viewBox="0 0 24 24" fill="none">
          <rect
            x="3"
            y="3"
            width="18"
            height="18"
            rx="4"
            stroke="currentColor"
            strokeWidth="1.5"
          />
          <path
            d="M12 8V12M12 16V16.01"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
          />
        </svg>
      </div>
      <h2 className="text-xl font-bold text-[#141b2b] tracking-tight">{title}</h2>
      <p className="text-sm text-[#3d4947] max-w-sm text-center">
        This section is currently under construction and will be available in an upcoming release.
      </p>
    </div>
  );
}
