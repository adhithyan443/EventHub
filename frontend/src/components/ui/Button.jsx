export default function Button({ children, variant = "primary", icon, className = "", ...props }) {
    const base = "h-12 w-full rounded-md text-sm font-semibold transition-colors flex items-center justify-center gap-3";
    const variants = {
        primary: "bg-primary text-white hover:bg-primary/90",
        outline: "bg-white text-ink border border-border hover:bg-background",
    };

    return (
        <button className={`${base} ${variants[variant]} ${className}`} {...props}>
            {icon}
            {children}
        </button>
    );
}