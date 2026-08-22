import { useState } from "react";
import AuthLayout from "../../components/layout/AuthLayout";
import Input from "../../components/ui/Input";
import Button from "../../components/ui/Button";
import GoogleIcon from "../../components/ui/GoogleIcon";
import { Link } from "react-router-dom";

export default function LoginPage() {
    const [form, setForm] = useState({ email: "", password: "" });
    const [errors, setErrors] = useState({});

    function handleChange(e) {
        const { name, value } = e.target;
        setForm((prev) => ({ ...prev, [name]: value }));
    }

    function validate() {
        const newErrors = {};
        if (!form.email.trim()) newErrors.email = "Email is required";
        if (!form.password) newErrors.password = "Password is required";
        return newErrors;
    }

    function handleSubmit(e) {
        e.preventDefault();
        const newErrors = validate();
        setErrors(newErrors);
        if (Object.keys(newErrors).length === 0) {
            console.log("Would submit:", form);
            // Step 6 will replace this with the real API call
        }
    }

    return (
        <AuthLayout
            title={<>Discover.<br />Book.<br />Experience.</>}
            description="Find amazing events, book your tickets, and create unforgettable experiences with EventHub."
        >
            <div className="bg-white rounded-lg shadow p-8 w-full max-w-md flex flex-col gap-6">
                <div>
                    <h2 className="font-display text-2xl font-semibold text-ink">
                        Welcome Back
                    </h2>
                    <p className="text-ink/60 text-sm mt-1">
                        Sign in to continue to EventHub.
                    </p>
                </div>

                <form onSubmit={handleSubmit} className="flex flex-col gap-6">
                    <Input
                        label="Email Address"
                        name="email"
                        type="email"
                        placeholder="Enter your email"
                        value={form.email}
                        onChange={handleChange}
                        error={errors.email}
                    />
                    <Input
                        label="Password"
                        name="password"
                        type="password"
                        placeholder="Enter your password"
                        value={form.password}
                        onChange={handleChange}
                        error={errors.password}
                    />
                    <Button type="submit">Sign In</Button>
                </form>



                <div className="flex items-center gap-4">
                    <div className="flex-1 h-px bg-border" />
                    <span className="text-xs font-medium tracking-wider text-ink/60 uppercase">
                        OR
                    </span>
                    <div className="flex-1 h-px bg-border" />
                </div>

                <Button variant="outline" icon={<GoogleIcon />}>
                    Continue with Google
                </Button>

                <p className="text-center text-sm text-ink/60"></p>

                <p className="text-center text-sm text-ink/60">
                    Don't have an account?{" "}
                    <Link to="/register" className="text-primary font-semibold">
                        Create an account
                    </Link>
                </p>
            </div>
        </AuthLayout>
    );
}