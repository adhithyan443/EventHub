import { useEffect, useState } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { getOrganizerEventById } from "../../api/organizerApi";
import useEventCreationStore from "../../store/eventCreationStore";
import {
  ORGANIZER_ROUTES,
  getEventDetailsRoute,
} from "../../constants/eventConstants";

export default function EditEventPage() {
  const { eventId } = useParams();
  const navigate = useNavigate();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    let isMounted = true;

    const loadEvent = async () => {
      try {
        setLoading(true);
        setError(null);

        const response = await getOrganizerEventById(eventId);
        const eventData = response?.data || response;

        if (!isMounted) return;

        const eventStatus = String(eventData?.Event?.Status || "").toUpperCase();

        if (eventStatus !== "DRAFT") {
          setError(
            `Only draft events can be edited. This event is currently "${eventStatus}".`
          );
          setLoading(false);
          return;
        }

        // Hydrate store for edit
        useEventCreationStore.getState().hydrateForEdit(eventData);

        // Redirect to Step 1 in edit mode
        navigate(ORGANIZER_ROUTES.CREATE_STEP_1, { replace: true });
      } catch (err) {
        if (!isMounted) return;
        console.error("Failed to load event for editing:", err);
        setError(
          err?.response?.data?.message ||
            "Unable to load event details. Please verify the event ID and try again."
        );
        setLoading(false);
      }
    };

    if (eventId) {
      loadEvent();
    } else {
      setError("Event ID is missing.");
      setLoading(false);
    }

    return () => {
      isMounted = false;
    };
  }, [eventId, navigate]);

  if (loading) {
    return (
      <div className="min-h-[60vh] flex flex-col items-center justify-center gap-4 p-6">
        <div className="size-10 border-3 border-[#00685f]/20 border-t-[#00685f] rounded-full animate-spin" />
        <div className="flex flex-col items-center gap-1">
          <p className="text-sm font-semibold text-[#141b2b]">
            Loading Event Configuration...
          </p>
          <p className="text-xs text-[#565e74]">
            Preparing edit mode and hydrating event details.
          </p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-[60vh] flex flex-col items-center justify-center p-6">
        <div className="max-w-md w-full bg-white border border-red-200 rounded-2xl p-6 shadow-sm flex flex-col items-center text-center gap-4">
          <div className="size-12 rounded-full bg-red-50 text-red-600 flex items-center justify-center text-xl font-bold">
            ⚠️
          </div>

          <div className="flex flex-col gap-1.5">
            <h2 className="text-base font-bold text-[#141b2b]">
              Cannot Edit Event
            </h2>
            <p className="text-xs text-[#565e74] leading-relaxed">{error}</p>
          </div>

          <div className="flex items-center gap-3 w-full pt-2">
            <Link
              to={ORGANIZER_ROUTES.MY_EVENTS}
              className="flex-1 py-2 px-3 rounded-lg border border-[#bcc9c6] text-xs font-semibold text-[#141b2b] hover:bg-gray-50 text-center transition-colors"
            >
              My Events
            </Link>
            {eventId && (
              <Link
                to={getEventDetailsRoute(eventId)}
                className="flex-1 py-2 px-3 rounded-lg bg-[#00685f] text-white text-xs font-semibold hover:bg-[#005a52] text-center transition-colors shadow-2xs"
              >
                Event Details
              </Link>
            )}
          </div>
        </div>
      </div>
    );
  }

  return null;
}
