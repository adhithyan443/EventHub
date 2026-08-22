import heroImage from "../../assets/login-hero.png";

export default function AuthLayout({ children }) {
    return (
        <div className="min-h-screen flex">
            {/* Left: Promotional panel */}
            <div className="hidden lg:flex flex-1 bg-primary relative overflow-hidden items-center p-12">
                <img
                    src={heroImage}
                    alt=""
                    className="absolute inset-0 w-full h-full object-cover object-right mix-blend-overlay opacity-50"
                />
                <div className="absolute inset-0 bg-gradient-to-t from-primary/90 to-primary/40" />
                <div className="relative flex flex-col gap-6 max-w-md">
                    <div className="flex items-center gap-2">
                        <span className="font-display text-2xl font-bold text-white">
                            EventHub
                        </span>
                    </div>
                    <h1 className="font-display text-5xl font-bold text-white leading-tight">
                        Discover.
                        <br />
                        Book.
                        <br />
                        Experience.
                    </h1>
                    <p className="text-white/90 text-lg">
                        Find amazing events, book your tickets, and create unforgettable
                        experiences with EventHub.
                    </p>
                </div>
            </div>

            {/* Right: Card area (Login/Register form goes here) */}
            <div className="flex-1 flex items-center justify-center bg-background p-8">
                {children}
            </div>
        </div>
    );
}