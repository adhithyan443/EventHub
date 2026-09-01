import { useState } from "react";
import Input from "./Input";

function EyeIcon({ open }) {
    return open ? (
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M1 12s4-7 11-7 11 7 11 7-4 7-11 7-11-7-11-7Z" />
            <circle cx="12" cy="12" r="3" />
        </svg>
    ) : (
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M17.94 17.94A10.94 10.94 0 0 1 12 19c-7 0-11-7-11-7a21.3 21.3 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 7 11 7a21.3 21.3 0 0 1-2.16 3.19M14.12 14.12a3 3 0 1 1-4.24-4.24" />
            <path d="M1 1l22 22" />
        </svg>
    );
}

export default function PasswordInput({ label, name, value, onChange, placeholder, error }) {
    const [visible, setVisible] = useState(false);

    return (
        <Input
            label={label}
            name={name}
            type={visible ? "text" : "password"}
            value={value}
            onChange={onChange}
            placeholder={placeholder}
            error={error}
            rightElement={
                <button type="button" onClick={() => setVisible((v) => !v)} className="text-ink/50 hover:text-ink" tabIndex={-1}>
                    <EyeIcon open={visible} />
                </button>
            }
        />
    );
}