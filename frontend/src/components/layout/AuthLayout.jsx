import defaultHero from "../../assets/login-hero.png";

export default function AuthLayout({ children, title, description, heroImage = defaultHero }) {
  return (
    <div className="min-h-screen flex">
      <div className="hidden lg:flex flex-1 bg-primary relative overflow-hidden items-center p-12">
        <img
          src={heroImage}
          alt=""
          className="absolute inset-0 w-full h-full object-cover object-right mix-blend-overlay opacity-30"
        />
        <div className="absolute inset-0 bg-gradient-to-t from-primary/90 to-primary/40" />
        <div className="relative flex flex-col gap-6 max-w-md">
          <div className="flex items-center gap-2">
            <span className="font-display text-2xl font-bold text-white">EventHub</span>
          </div>
          <h1 className="font-display text-5xl font-bold text-white leading-tight">
            {title}
          </h1>
          <p className="text-white/90 text-lg">{description}</p>
        </div>
      </div>

      <div className="flex-1 flex items-center justify-center bg-background p-8">
        {children}
      </div>
    </div>
  );
}