import { useNavigate } from "react-router-dom";
import useEventCreationStore from "../../store/eventCreationStore";
import { getEventCreationSteps } from "../../constants/eventConstants";

export default function EventCreationStepper({ currentStep = 1 }) {
  const navigate = useNavigate();
  const locationType = useEventCreationStore(
    (state) => state.locationType || state.eventType
  );
  const ticketMode = useEventCreationStore((state) => state.ticketMode);

  const steps = getEventCreationSteps(locationType, ticketMode);

  return (
    <div className="bg-white rounded-xl border border-[#bcc9c6]/50 shadow-sm p-3 w-full mb-6">
      <ol className="flex items-center justify-between w-full relative">
        {steps.map((item, index) => {
          const isCompleted = item.step < currentStep;
          const isActive = item.step === currentStep;
          const isClickable = isCompleted;

          return (
            <li
              key={item.step}
              className="flex items-center flex-1 last:flex-none relative"
            >
              <div
                onClick={() => {
                  if (isClickable) navigate(item.path);
                }}
                className={`flex items-center gap-2.5 px-3 py-1.5 rounded-lg transition-all ${
                  isClickable
                    ? "cursor-pointer hover:bg-gray-100/70"
                    : "cursor-default"
                }`}
              >
                {/* Step Circle */}
                <div
                  className={`size-7 rounded-full flex items-center justify-center text-xs font-semibold shrink-0 transition-all ${
                    isCompleted
                      ? "bg-[#00685f] text-white"
                      : isActive
                      ? "bg-[#00685f]/15 border-2 border-[#00685f] text-[#00685f] font-bold"
                      : "border border-[#bcc9c6] text-[#565e74] bg-white"
                  }`}
                >
                  {isCompleted ? (
                    <svg
                      className="size-3.5"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth="3"
                        d="M5 13l4 4L19 7"
                      />
                    </svg>
                  ) : (
                    item.step
                  )}
                </div>

                {/* Step Label */}
                <span
                  className={`text-[13px] whitespace-nowrap ${
                    isActive
                      ? "font-bold text-[#00685f]"
                      : isCompleted
                      ? "font-semibold text-[#141b2b]"
                      : "font-medium text-[#565e74]"
                  }`}
                >
                  {item.label}
                </span>
              </div>

              {/* Connecting Divider between steps */}
              {index < steps.length - 1 && (
                <div className="flex-1 h-[2px] mx-2 bg-[#bcc9c6]/40 relative overflow-hidden">
                  <div
                    className="h-full bg-[#00685f] transition-all duration-300"
                    style={{
                      width: isCompleted ? "100%" : "0%",
                    }}
                  />
                </div>
              )}
            </li>
          );
        })}
      </ol>
    </div>
  );
}
