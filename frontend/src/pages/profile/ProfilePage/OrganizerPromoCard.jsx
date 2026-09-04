import { useNavigate } from "react-router-dom";
import { MegaphoneIcon } from "../../../components/layout/icons";
import useOrganizerStore from "../../../store/organizerStore";

export default function OrganizerPromoCard() {
    const navigate = useNavigate();
    const applicationSubmitted = useOrganizerStore((state) => state.applicationSubmitted);

    if (applicationSubmitted) {
        return (
            <div className="rounded-2xl bg-gradient-to-r from-[#e7f5f2] via-[#eef8f4] to-[#f9faf6] border border-[#d1eae2] p-6 flex flex-col gap-3.5">
                <span className="inline-flex w-fit items-center gap-1.5 rounded-full bg-primary/10 border border-primary/20 px-2.5 py-1 text-xs font-bold text-primary">
                    <span className="w-1.5 h-1.5 rounded-full bg-primary animate-pulse" />
                    Under Review
                </span>
                <h3 className="font-display text-xl font-bold text-slate-900">Application Under Review</h3>
                <p className="text-sm text-slate-600 leading-relaxed max-w-xl">
                    Your organizer application has been submitted and is currently being reviewed by our team.
                </p>
                <button
                    type="button"
                    onClick={() => navigate("/become-organizer/status")}
                    className="w-fit h-10 px-6 rounded-lg bg-[#007066] hover:bg-[#005c54] text-white text-sm font-semibold transition-colors cursor-pointer shadow-sm"
                >
                    View Status
                </button>
            </div>
        );
    }

    return (
        <div className="rounded-2xl bg-gradient-to-r from-[#e7f5f2] via-[#eef8f4] to-[#f9faf6] border border-[#d1eae2] p-6 flex flex-col gap-3.5">
            <span className="inline-flex w-fit items-center gap-1.5 rounded-full bg-[#ede9fe]/80 border border-[#ddd6fe]/60 px-2.5 py-1 text-xs font-semibold text-[#4338ca]">
                <MegaphoneIcon className="w-3.5 h-3.5" />
                Host Events
            </span>
            <h3 className="font-display text-xl font-bold text-slate-900">Become an Organizer</h3>
            <p className="text-sm text-slate-600 leading-relaxed max-w-xl">
                Reach thousands of attendees. Apply to host and manage your own events on EventHub.
            </p>
            <button
                type="button"
                onClick={() => navigate("/become-organizer")}
                className="w-fit h-10 px-6 rounded-lg bg-[#007066] hover:bg-[#005c54] text-white text-sm font-semibold transition-colors cursor-pointer shadow-sm"
            >
                Apply Now
            </button>
        </div>
    );
}