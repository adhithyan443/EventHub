export default function StepsProgress({ current = 1 }) {
  const steps = ["Select Seats", "Checkout", "Payment", "Confirmation"];

  return (
    <nav aria-label="Progress" className="w-full max-w-[760px] overflow-x-auto py-2">
      <div className="flex items-center text-sm min-w-max">
        {steps.map((label, index) => {
          const stepNumber = index + 1;
          const isCompleted = stepNumber < current;
          const isActive = stepNumber <= current;
          const isCurrent = stepNumber === current;

          return (
            <div key={label} className="contents">
              <div className="flex items-center gap-2 whitespace-nowrap">
                <span
                  className={`grid h-8 w-8 place-items-center rounded-full border text-xs font-semibold transition-colors ${
                    isActive
                      ? "border-[#00796d] bg-[#00796d] text-white"
                      : "border-[#bcc9c6] bg-white text-[#8a8fa0]"
                  }`}
                >
                  {isCompleted ? "✓" : stepNumber}
                </span>
                <span
                  className={`text-xs sm:text-sm transition-colors ${
                    isCurrent
                      ? "font-bold text-[#00685f]"
                      : isActive
                      ? "font-semibold text-[#00685f]"
                      : "text-[#8a8fa0]"
                  }`}
                >
                  {label}
                </span>
              </div>
              {index < steps.length - 1 && (
                <span
                  className={`mx-3 sm:mx-6 h-px w-8 sm:w-16 flex-1 transition-colors ${
                    stepNumber < current ? "bg-[#00796d]" : "bg-[#bcc9c6]"
                  }`}
                  aria-hidden="true"
                />
              )}
            </div>
          );
        })}
      </div>
    </nav>
  );
}
