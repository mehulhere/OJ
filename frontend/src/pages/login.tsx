import Head from 'next/head';
import { useState, ChangeEvent, FormEvent } from 'react';
import Link from 'next/link'; // For a link to the registration page
// import { useRouter } from 'next/router'; // Uncomment to redirect after login
import '@/app/globals.css';

// Assumed response structures from your backend
interface LoginSuccessResponse {
    message: string;
    token: string;
    user: {
        user_id: string;
        username: string;
        email: string;
        firstname: string;
        lastname: string;
    };
}

interface ErrorResponse {
    message: string; // Assuming backend sends errors in this format
}

export default function LoginPage() {
    const [usernameOrEmail, setUsernameOrEmail] = useState<string>('');
    const [password, setPassword] = useState<string>('');

    const [error, setError] = useState<string | null>(null);
    const [isLoading, setIsLoading] = useState<boolean>(false);
    const [successMessage, setSuccessMessage] = useState<string | null>(null);

    // const router = useRouter(); // Uncomment to redirect after login

    const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        setIsLoading(true);
        setError(null);
        setSuccessMessage(null);

        if (!usernameOrEmail || !password) {
            setError('Username/Email and password are required.');
            setIsLoading(false);
            return;
        }

        const loginData = {
            email: usernameOrEmail,
            password,
        };

        try {
            const response = await fetch(`http://localhost:8080/login`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(loginData),
            });

            const responseData: LoginSuccessResponse | ErrorResponse = await response.json();

            if (!response.ok) {
                const errorMessage = (responseData as ErrorResponse)?.message || `Error: ${response.status} - ${response.statusText}`;
                setError(errorMessage);
                setIsLoading(false);
                return;
            }

            if ('token' in responseData) {
                setSuccessMessage(responseData.message || 'Login successful!');
                console.log('Login successful:', responseData);
                // TODO: Store the token securely (e.g., localStorage, HttpOnly cookie via backend, or state management)
                // localStorage.setItem('authToken', responseData.token);
                // TODO: Store user info if needed
                // localStorage.setItem('userInfo', JSON.stringify(responseData.user));

                // Clear form
                setUsernameOrEmail('');
                setPassword('');

                // TODO: Redirect to a protected page or dashboard
                // router.push('/dashboard'); // Example redirect
            } else {
                setError("Login succeeded, but the response format was unexpected.");
            }

        } catch (err) {
            console.error('Login request failed:', err);
            setError(err instanceof Error ? err.message : 'An unknown network or parsing error occurred.');
        } finally {
            setIsLoading(false);
        }
    };

    return (
        <>
            <Head>
                <title>Login - Online Judge</title>
            </Head>
            <div className="min-h-screen bg-gray-100 flex flex-col justify-center items-center py-12 sm:px-6 lg:px-8">
                <div className="sm:mx-auto sm:w-full sm:max-w-md">
                    <h1 className="mt-6 text-center text-3xl font-extrabold text-gray-900">
                        Sign in to your account
                    </h1>
                </div>

                <div className="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
                    <div className="bg-white py-8 px-4 shadow sm:rounded-lg sm:px-10">
                        {error && (
                            <p className="mb-4 rounded-md bg-red-50 p-4 text-sm font-medium text-red-700 text-center">
                                {error}
                            </p>
                        )}
                        {successMessage && (
                            <p className="mb-4 rounded-md bg-green-50 p-4 text-sm font-medium text-green-700 text-center">
                                {successMessage}
                            </p>
                        )}
                        <form className="space-y-6" onSubmit={handleSubmit}>
                            <div>
                                <label htmlFor="email" className="block text-sm font-medium text-gray-700">
                                    Username or Email
                                </label>
                                <div className="mt-1">
                                    <input
                                        id="email"
                                        name="email"
                                        type="text"
                                        autoComplete="username email"
                                        required
                                        value={usernameOrEmail}
                                        onChange={(e: ChangeEvent<HTMLInputElement>) => setUsernameOrEmail(e.target.value)}
                                        disabled={isLoading}
                                        className="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm placeholder-gray-400 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm text-gray-900 disabled:bg-gray-50 disabled:cursor-not-allowed"
                                    />
                                </div>
                            </div>

                            <div>
                                <label htmlFor="password" className="block text-sm font-medium text-gray-700">
                                    Password
                                </label>
                                <div className="mt-1">
                                    <input
                                        id="password"
                                        name="password"
                                        type="password"
                                        autoComplete="current-password"
                                        required
                                        value={password}
                                        onChange={(e: ChangeEvent<HTMLInputElement>) => setPassword(e.target.value)}
                                        disabled={isLoading}
                                        className="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm placeholder-gray-400 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm text-gray-900 disabled:bg-gray-50 disabled:cursor-not-allowed"
                                    />
                                </div>
                            </div>

                            {/* Optional: Remember me and Forgot password links */}
                            {/* <div className="flex items-center justify-between">
                                <div className="flex items-center">
                                    <input
                                        id="remember-me"
                                        name="remember-me"
                                        type="checkbox"
                                        className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
                                    />
                                    <label htmlFor="remember-me" className="ml-2 block text-sm text-gray-900">
                                        Remember me
                                    </label>
                                </div>
                                <div className="text-sm">
                                    <a href="#" className="font-medium text-indigo-600 hover:text-indigo-500">
                                        Forgot your password?
                                    </a>
                                </div>
                            </div> */}

                            <div>
                                <button
                                    type="submit"
                                    disabled={isLoading}
                                    className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:bg-gray-400 disabled:cursor-not-allowed"
                                >
                                    {isLoading ? 'Signing in...' : 'Sign in'}
                                </button>
                            </div>
                        </form>

                        <div className="mt-6">
                            <div className="relative">
                                <div className="absolute inset-0 flex items-center">
                                    <div className="w-full border-t border-gray-300" />
                                </div>
                                <div className="relative flex justify-center text-sm">
                                    <span className="px-2 bg-white text-gray-500">
                                        Or
                                    </span>
                                </div>
                            </div>

                            <div className="mt-6 text-center">
                                <p className="text-sm text-gray-600">
                                    Don't have an account?{' '}
                                    <Link href="/register" className="font-medium text-indigo-600 hover:text-indigo-500">
                                        Sign up
                                    </Link>
                                </p>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </>
    );
} 