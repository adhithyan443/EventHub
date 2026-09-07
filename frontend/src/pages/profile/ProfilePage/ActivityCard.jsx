export default function ActivityCard({ icon: Icon, title, description, tone = "soft", wide = false }) {
    const toneStyles = {
        primary: "bg-primary text-white",
        soft: "bg-[#e2e8f6] text-[#4f46e5]",
        muted: "bg-slate-200 text-slate-600",
    };

    return (
        <div className={`bg-[#f4f7fb] border border-[#e5edf5] rounded-2xl p-5 flex flex-col gap-2.5 transition-colors ${wide ? "col-span-2" : ""}`}>
            <div className={`h-9 w-9 rounded-full flex items-center justify-center shrink-0 ${toneStyles[tone] || toneStyles.soft}`}>
                <Icon className="w-4 h-4" />
            </div>
            <div>
                <h3 className="font-bold text-base text-slate-900 leading-tight">{title}</h3>
                <p className="text-xs text-slate-500 mt-1">{description}</p>
            </div>
        </div>
    );
}