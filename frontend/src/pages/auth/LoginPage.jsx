import { useState } from "react";
import AuthLayout from "../../components/layout/AuthLayout";
import Input from "../../components/ui/Input";
import Button from "../../components/ui/Button";
import GoogleIcon from "../../components/ui/GoogleIcon";
import { Link, useNavigate } from "react-router-dom";
import { login, startGoogleLogin } from "../../api/authApi";
import useAuthStore from "../../store/authStore";
// import apiClient from "../../api/client";

export default function LoginPage() {
    const [form, setForm] = useState({ email: "", password: "" });
    const [errors, setErrors] = useState({});

    const navigate = useNavigate();

    const setAuth = useAuthStore((state) => state.setAuth);

    function handleChange(e) {
        const { name, value } = e.target;

        setForm((prev) => ({ ...prev, [name]: value }));


        setErrors((prev) => ({
            ...prev,
            [name]: "",
            general: "",
        }));
    }

    function validate() {
        const newErrors = {};
        if (!form.email.trim()) newErrors.email = "Email is required";
        if (!form.password) newErrors.password = "Password is required";
        return newErrors;
    }

    async function handleSubmit(e) {
        //  console.log("SIGN IN CLICKED"); // DEBUG
        e.preventDefault();
        const newErrors = validate();
        setErrors(newErrors);

        if (Object.keys(newErrors).length > 0) {
            return
        }

        try {

            const response = await login(form);
            console.log("Login successful:", response);

            setAuth(
                response.user,
                response.access_token,
                response.refresh_token,
            );
            // await testProtectedAPI();

            switch (response.user.role) {
                case "ADMIN":
                    navigate("/admin");
                    break;

                case "ORGANIZER":
                    navigate("/organizer");
                    break;

                case "CUSTOMER":
                    navigate("/");
                    break;

                default:
                    setErrors({
                        general: "Invalid user role.",
                    });
            }


        } catch (error) {
            console.error("Login failed failed:", error);

            const message =
                error.response?.data?.message ||
                "Login failed. Please check your credentials and try again.";

            setErrors({
                general: message,
            });
        }
    }



    // async function testProtectedAPI() {
    //     try {
    //         const response = await apiClient.get("/protected/test");

    //         console.log("Protected API response After login", response.data);
    //     } catch (error) {
    //         console.error("Protected API failed:", error);
    //     }
    // }



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

                {errors.general && (
                    <p className="text-sm text-red-500 text-center">
                        {errors.general}
                    </p>
                )}

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

                    <div className="flex justify-end">
                        <Link
                            to="/forgot-password"
                            className="text-sm text-primary font-medium"
                        >
                            Forgot Password?
                        </Link>
                    </div>

                    <Button type="submit">Sign In</Button>
                </form>



                <div className="flex items-center gap-4">
                    <div className="flex-1 h-px bg-border" />
                    <span className="text-xs font-medium tracking-wider text-ink/60 uppercase">
                        OR
                    </span>
                    <div className="flex-1 h-px bg-border" />
                </div>

                <Button variant="outline" icon={<GoogleIcon />} onClick={startGoogleLogin}>
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