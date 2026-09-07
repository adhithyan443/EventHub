import { BellIcon, LockIcon, HelpIcon, LogoutIcon, ChevronRightIcon } from "../../../components/layout/icons";

const settingsRows = [
    { label: "Notifications", icon: BellIcon },
    { label: "Privacy & Security", icon: LockIcon },
    { label: "Help & Support", icon: HelpIcon },
];

export default function AccountSettingsCard({ onLogout }) {
    return (
        <div className="bg-white rounded-2xl border border-slate-200/80 p-6 flex flex-col">
            <h2 className="font-display text-xl font-bold text-slate-900 mb-2">Account Settings</h2>

            <div className="flex flex-col divide-y divide-slate-100">
                {settingsRows.map(({ label, icon: Icon }) => (
                    <button
                        key={label}
                        className="flex items-center justify-between py-3.5 text-sm text-slate-700 hover:text-primary transition-colors text-left"
                    >
                        <span className="flex items-center gap-3 text-slate-700">
                            <span className="text-slate-500"><Icon /></span>
                            {label}
                        </span>
                        <ChevronRightIcon className="text-slate-400 w-2.5 h-2.5" />
                    </button>
                ))}

                <button
                    onClick={onLogout}
                    className="flex items-center gap-3 py-3.5 text-sm font-semibold text-red-600 hover:text-red-700 transition-colors text-left"
                >
                    <LogoutIcon /> Logout
                </button>
            </div>
        </div>
    );
}