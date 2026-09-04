import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { MegaphoneIcon } from "../../../components/layout/icons";
import useOrganizerStore from "../../../store/organizerStore";
import useAuthStore from "../../../store/authStore";

export default function OrganizerPromoCard() {
    const navigate = useNavigate();
    const user = useAuthStore((state) => state.user);

    const applicationSubmitted = useOrganizerStore((state) => state.applicationSubmitted);
    const applicationStatus = useOrganizerStore((state) => state.applicationStatus);
    const fetchApplication = useOrganizerStore((state) => state.fetchApplication);

    // Fetch latest application status on mount
    useEffect(() => {
        fetchApplication();
    }, [fetchApplication]);

    const isOrganizer = user?.role === "ORGANIZER" || applicationStatus === "APPROVED";
    const isPending = applicationStatus === "PENDING" || (!applicationStatus && applicationSubmitted);
    const isRejected = applicationStatus === "REJECTED";

    // 1. Approved Organizer state
    if (isOrganizer) {
        return (
            <div className="rounded-2xl bg-gradient-to-r from-emerald-50 via-teal-50/40 to-emerald-50 border border-emerald-200/90 p-6 flex flex-col gap-3.5">
                <span className="inline-flex w-fit items-center gap-1.5 rounded-full bg-emerald-100/80 border border-emerald-200 px-2.5 py-1 text-xs font-bold text-emerald-800">
                    <span className="w-1.5 h-1.5 rounded-full bg-emerald-500" />
                    Active Organizer
                </span>
                <h3 className="font-display text-xl font-bold text-slate-900">EventHub Organizer</h3>
                <p className="text-sm text-slate-600 leading-relaxed max-w-xl">
                    You are an approved event organizer on EventHub. You can host, publish, and manage your events.
                </p>
            </div>
        );
    }

    // 2. Pending / Under Review state
    if (isPending) {
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

    // 3. Rejected / Revisions Requested state
    if (isRejected) {
        return (
            <div className="rounded-2xl bg-gradient-to-r from-amber-50 via-orange-50/40 to-amber-50 border border-amber-200/90 p-6 flex flex-col gap-3.5">
                <span className="inline-flex w-fit items-center gap-1.5 rounded-full bg-amber-100/80 border border-amber-200 px-2.5 py-1 text-xs font-bold text-amber-800">
                    <span className="w-1.5 h-1.5 rounded-full bg-amber-500" />
                    Revisions Requested
                </span>
                <h3 className="font-display text-xl font-bold text-slate-900">Application Needs Revision</h3>
                <p className="text-sm text-slate-600 leading-relaxed max-w-xl">
                    Your organizer application was rejected. Please review the feedback, update your information, and resubmit.
                </p>
                <button
                    type="button"
                    onClick={() => navigate("/become-organizer")}
                    className="w-fit h-10 px-6 rounded-lg bg-primary hover:bg-primary/90 text-white text-sm font-semibold transition-colors cursor-pointer shadow-sm"
                >
                    Edit & Resubmit
                </button>
            </div>
        );
    }

    // 4. Default / No Application state
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