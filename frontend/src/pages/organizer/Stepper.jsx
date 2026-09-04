export default function Stepper({ currentStep }) {
  const steps = [
    { num: 1, label: "Business Info" },
    { num: 2, label: "Verification" },
    { num: 3, label: "Bank Details" },
    { num: 4, label: "Review" },
  ];

  return (
    <div className="relative flex items-center justify-between w-full max-w-2xl mx-auto px-2">
      {/* Background connector line */}
      <div className="absolute top-4 left-6 right-6 h-0.5 bg-slate-200 -z-0" />

      {/* Active progress fill line */}
      <div
        className="absolute top-4 left-6 h-0.5 bg-primary transition-all duration-300 -z-0"
        style={{
          width: `calc(${((currentStep - 1) / (steps.length - 1)) * 100}% - ${
            ((currentStep - 1) / (steps.length - 1)) * 48
          }px)`,
        }}
      />

      {steps.map((step) => {
        const completed = step.num < currentStep;
        const active = step.num === currentStep;

        return (
          <div
            key={step.num}
            className="flex flex-col items-center gap-2 z-10 select-none group"
          >
            {completed ? (
              <div className="w-8 h-8 rounded-full bg-primary flex items-center justify-center shadow-sm text-white transition-transform">
                <svg
                  width="12"
                  height="10"
                  viewBox="0 0 12 10"
                  fill="none"
                  aria-hidden="true"
                >
                  <path
                    d="M1.5 5L4.5 8L10.5 2"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  />
                </svg>
              </div>
            ) : active ? (
              <div className="w-8 h-8 rounded-full bg-white border-2 border-primary flex items-center justify-center shadow-sm">
                <div className="w-2.5 h-2.5 rounded-full bg-primary animate-pulse" />
              </div>
            ) : (
              <div className="w-8 h-8 rounded-full bg-white border-2 border-slate-300 flex items-center justify-center">
                <span className="text-xs font-semibold text-slate-400">
                  {step.num}
                </span>
              </div>
            )}

            <span
              className={`text-[11px] sm:text-xs font-medium tracking-wide text-center whitespace-nowrap transition-colors ${
                active
                  ? "font-bold text-primary"
                  : completed
                  ? "font-semibold text-ink"
                  : "text-slate-400"
              }`}
            >
              {step.label}
            </span>
          </div>
        );
      })}
    </div>
  );
}
