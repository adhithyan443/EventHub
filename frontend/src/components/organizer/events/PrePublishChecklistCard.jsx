import { CheckCircleIcon } from "./OrganizerEventsIcons";

export default function PrePublishChecklistCard({
  hasBasicInfo = true,
  hasVenueOrUrl = true,
  hasSchedule = true,
  hasTickets = true,
  hasCancellation = true,
  visibility = "PUBLIC",
}) {
  const checklistItems = [
    { label: "Basic details & category provided", valid: hasBasicInfo },
    { label: "Venue / stream location configured", valid: hasVenueOrUrl },
    { label: "Event schedule verified", valid: hasSchedule },
    { label: "Ticket tiers / seating inventory ready", valid: hasTickets },
    { label: "Cancellation policy configured", valid: hasCancellation },
  ];

  return (
    <div className="bg-white border border-[#bcc9c6] rounded-xl p-5 sm:p-6 shadow-2xs flex flex-col gap-4">
      <h3 className="text-base font-bold text-[#141b2b] pb-2 border-b border-[#bcc9c6]/40">
        Pre-Publish Checklist
      </h3>

      <div className="flex flex-col gap-2.5">
        {checklistItems.map((item, idx) => (
          <div key={idx} className="flex items-center gap-2.5 text-xs sm:text-sm">
            <CheckCircleIcon
              className={`size-4 shrink-0 ${
                item.valid ? "text-[#00685f]" : "text-gray-300"
              }`}
            />
            <span
              className={
                item.valid
                  ? "text-[#141b2b] font-medium"
                  : "text-[#565e74] line-through opacity-70"
              }
            >
              {item.label}
            </span>
          </div>
        ))}
      </div>

      <div className="rounded-lg bg-[#00685f]/5 border border-[#00685f]/20 p-3 mt-1">
        <p className="text-xs text-[#00685f] leading-relaxed">
          Your event will be immediately visible as <strong>{visibility}</strong> once published live. Saving as draft will keep it private until you choose to publish.
        </p>
      </div>
    </div>
  );
}
