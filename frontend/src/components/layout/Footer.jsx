import { Link } from "react-router-dom";

export default function Footer() {
  return (
    <footer className="w-full bg-[#eef2fb] border-t border-slate-200/60 mt-auto">
      <div className="max-w-7xl mx-auto px-6 py-6 flex flex-col sm:flex-row items-center justify-between gap-4 text-xs sm:text-sm text-slate-600">
        <Link to="/" className="font-display font-bold text-slate-900 text-lg">
          EventHub
        </Link>

        <nav className="flex items-center gap-6">
          <Link to="/about" className="hover:text-slate-900 underline underline-offset-4 decoration-slate-300">
            About
          </Link>
          <Link to="/support" className="hover:text-slate-900 underline underline-offset-4 decoration-slate-300">
            Support
          </Link>
          <Link to="/terms" className="hover:text-slate-900 underline underline-offset-4 decoration-slate-300">
            Terms
          </Link>
          <Link to="/privacy" className="hover:text-slate-900 underline underline-offset-4 decoration-slate-300">
            Privacy
          </Link>
        </nav>

        <p className="text-slate-500 text-xs">
          &copy; 2024 EventHub. All rights reserved.
        </p>
      </div>
    </footer>
  );
}
