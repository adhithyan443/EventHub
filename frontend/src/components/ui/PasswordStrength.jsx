function getScore(password) {
    if (!password) return 0;
    let score = 0;
    if (password.length >= 8) score++;
    if (/[A-Z]/.test(password)) score++;
    if (/[0-9]/.test(password)) score++;
    if (/[^A-Za-z0-9]/.test(password)) score++;
    return score;
}

const LEVELS = [
    { label: "Weak", bar: "bg-red-500", text: "text-red-500" },
    { label: "Weak", bar: "bg-red-500", text: "text-red-500" },
    { label: "Medium", bar: "bg-accent", text: "text-accent" },
    { label: "Strong", bar: "bg-primary", text: "text-primary" },
    { label: "Strong", bar: "bg-primary", text: "text-primary" },
];

export default function PasswordStrength({ password }) {
    if (!password) return null;
    const score = getScore(password);
    const level = LEVELS[score];

    return (
        <div className="flex items-center gap-3 pt-1">
            <div className="flex-1 flex gap-1">
                {[0, 1, 2, 3].map((i) => (
                    <div key={i} className={`h-1 flex-1 rounded-full ${i < score ? level.bar : "bg-border"}`} />
                ))}
            </div>
            <span className={`text-xs font-medium tracking-wide ${level.text}`}>{level.label}</span>
        </div>
    );
}