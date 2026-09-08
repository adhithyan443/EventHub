import { useNavigate } from "react-router-dom";
import { ORGANIZER_ROUTES } from "../../constants/eventConstants";

export default function OrganizerPlaceholderPage({ title = "Organizer Section" }) {
  const navigate = useNavigate();

  return (
    <div className="p-8 max-w-4xl mx-auto flex flex-col items-center justify-center min-h-[60vh] text-center gap-6">
      <div className="size-20 rounded-2xl bg-[#00685f]/10 border border-[#00685f]/20 flex items-center justify-center text-[#00685f] text-3xl">
        📋
      </div>

      <div className="flex flex-col gap-2 max-w-md">
        <h1 className="text-2xl font-bold text-[#141b2b]">{title}</h1>
        <p className="text-sm text-[#565e74] leading-relaxed">
          This section is part of the upcoming Organizer portal features. The event creation and dashboard workflows are fully functional.
        </p>
      </div>

      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={() => navigate(ORGANIZER_ROUTES.DASHBOARD)}
          className="border border-[#bcc9c6] bg-white hover:bg-gray-50 text-[#141b2b] font-semibold text-xs px-5 py-2.5 rounded-lg transition-colors cursor-pointer"
        >
          Return to Dashboard
        </button>

        <button
          type="button"
          onClick={() => navigate(ORGANIZER_ROUTES.CREATE_STEP_1)}
          className="bg-[#00685f] hover:bg-[#005550] text-white font-semibold text-xs px-5 py-2.5 rounded-lg shadow-sm transition-colors cursor-pointer"
        >
          Create New Event
        </button>
      </div>
    </div>
  );
}
