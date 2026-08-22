import { useState } from "react";
import AuthLayout from "../../components/layout/AuthLayout";
import Input from "../../components/ui/Input";
import PasswordInput from "../../components/ui/PasswordInput";
import PasswordStrength from "../../components/ui/PasswordStrength";
import Checkbox from "../../components/ui/Checkbox";
import Button from "../../components/ui/Button";
import GoogleIcon from "../../components/ui/GoogleIcon";
import { Link } from "react-router-dom";

export default function RegisterPage() {
    const [form, setForm] = useState({
        fullName: "",
        email: "",
        phone: "",
        password: "",
        confirmPassword: "",
        agree: false,
    });
    const [errors, setErrors] = useState({});

    function handleChange(e) {
        const { name, value, type, checked } = e.target;
        setForm((prev) => ({ ...prev, [name]: type === "checkbox" ? checked : value }));
    }

    function validate() {
        const newErrors = {};
        if (!form.fullName.trim()) newErrors.fullName = "Full name is required";
        if (!form.email.trim()) newErrors.email = "Email is required";
        if (!form.phone.trim()) newErrors.phone = "Phone number is required";
        if (!form.password) newErrors.password = "Password is required";
        if (form.confirmPassword !== form.password) newErrors.confirmPassword = "Passwords do not match";
        if (!form.agree) newErrors.agree = "You must agree to the Terms and Privacy Policy";
        return newErrors;
    }

    function handleSubmit(e) {
        e.preventDefault();
        const newErrors = validate();
        setErrors(newErrors);
        if (Object.keys(newErrors).length === 0) {
            console.log("Would submit:", form);
        }
    }

    return (
        <AuthLayout
            title={<>Discover. Book.<br />Experience.</>}
            description="Create your account and start discovering amazing events."
        >
            <div className="bg-white rounded-lg shadow p-8 w-full max-w-md flex flex-col gap-6">
                <div>
                    <h2 className="font-display text-2xl font-semibold text-ink">Create your account</h2>
                    <p className="text-ink/60 text-sm mt-1">Join EventHub and start discovering amazing events.</p>
                </div>

                <form onSubmit={handleSubmit} className="flex flex-col gap-4">
                    <Input
                        label="Full Name"
                        name="fullName"
                        placeholder="Enter your full name"
                        value={form.fullName}
                        onChange={handleChange}
                        error={errors.fullName}
                    />
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
                        label="Phone Number"
                        name="phone"
                        type="tel"
                        placeholder="Enter your phone number"
                        leftAddon="+91"
                        value={form.phone}
                        onChange={handleChange}
                        error={errors.phone}
                    />
                    <div>
                        <PasswordInput
                            label="Password"
                            name="password"
                            placeholder="Create a password"
                            value={form.password}
                            onChange={handleChange}
                            error={errors.password}
                        />
                        <PasswordStrength password={form.password} />
                    </div>
                    <PasswordInput
                        label="Confirm Password"
                        name="confirmPassword"
                        placeholder="Re-enter your password"
                        value={form.confirmPassword}
                        onChange={handleChange}
                        error={errors.confirmPassword}
                    />
                    <Checkbox name="agree" checked={form.agree} onChange={handleChange} error={errors.agree}>
                        I agree to the <a href="/terms" className="text-primary font-medium">Terms of Service</a> and{" "}
                        <a href="/privacy" className="text-primary font-medium">Privacy Policy</a>.
                    </Checkbox>
                    <Button type="submit">Create Account</Button>
                </form>

                <div className="flex items-center gap-4">
                    <div className="flex-1 h-px bg-border" />
                    <span className="text-xs font-medium tracking-wider text-ink/60 uppercase">OR</span>
                    <div className="flex-1 h-px bg-border" />
                </div>

                <Button variant="outline" icon={<GoogleIcon />}>
                    Continue with Google
                </Button>

                <p className="text-center text-sm text-ink/60">
                    Already have an account? <Link to="/login" className="text-primary font-semibold">
                        Sign in
                    </Link>
                </p>
            </div>
        </AuthLayout>
    );
}