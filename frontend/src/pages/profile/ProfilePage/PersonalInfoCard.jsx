import { useState } from "react";
import Input from "../../../components/ui/Input";

export default function PersonalInfoCard({ user }) {
    const [form, setForm] = useState({
        fullName: user.fullName,
        email: user.email,
        phone: user.phone,
    });

    function handleChange(e) {
        const { name, value } = e.target;
        setForm((prev) => ({ ...prev, [name]: value }));
    }

    function handleSubmit(e) {
        e.preventDefault();
        console.log("Would save:", form);
        // TODO: wire to PUT /auth/profile
    }

    return (
        <div className="bg-white rounded-2xl border border-slate-200/80 p-6 flex flex-col gap-5">
            <h2 className="font-display text-xl font-bold text-slate-900">Personal Information</h2>
            <form onSubmit={handleSubmit} className="flex flex-col gap-4">
                <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-medium text-slate-500">Full Name</label>
                    <input
                        type="text"
                        name="fullName"
                        value={form.fullName}
                        onChange={handleChange}
                        className="w-full rounded-lg border border-slate-200 bg-white px-3.5 py-2.5 text-sm text-slate-800 focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                    />
                </div>
                <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-medium text-slate-500">Email</label>
                    <input
                        type="email"
                        name="email"
                        value={form.email}
                        onChange={handleChange}
                        className="w-full rounded-lg border border-slate-200 bg-white px-3.5 py-2.5 text-sm text-slate-800 focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                    />
                </div>
                <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-medium text-slate-500">Phone Number</label>
                    <input
                        type="text"
                        name="phone"
                        value={form.phone}
                        onChange={handleChange}
                        className="w-full rounded-lg border border-slate-200 bg-white px-3.5 py-2.5 text-sm text-slate-800 focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                    />
                </div>
                <div className="flex justify-end pt-1">
                    <button
                        type="submit"
                        className="h-9 px-5 rounded-lg border border-slate-300 bg-white text-xs font-semibold text-slate-700 hover:bg-slate-50 hover:border-slate-400 transition-colors"
                    >
                        Save Changes
                    </button>
                </div>
            </form>
        </div>
    );
}