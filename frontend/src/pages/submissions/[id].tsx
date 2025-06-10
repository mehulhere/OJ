import Head from 'next/head';
import { useState, useEffect } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/router';
import '@/app/globals.css';

// Define submission detail type
interface SubmissionDetail {
    id: string;
    user_id: string;
    username: string;
    problem_id: string;
    problem_title: string;
    language: string;
    code: string;
    status: string;
    execution_time_ms: number;
    memory_used_kb: number;
    submitted_at: string;
    test_cases_passed: number;
    test_cases_total: number;
    test_case_status: string;
}

export default function SubmissionDetailPage() {
    const router = useRouter();
    const { id } = router.query;

    const [submission, setSubmission] = useState<SubmissionDetail | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [isLoggedIn, setIsLoggedIn] = useState(false);
    const [isAdmin, setIsAdmin] = useState(false);

    useEffect(() => {
        // Check if user is logged in
        const checkLoginStatus = async () => {
            try {
                const response = await fetch('http://localhost:8080/api/auth-status', {
                    method: 'GET',
                    credentials: 'include',
                    headers: {
                        'Accept': 'application/json',
                        'Content-Type': 'application/json'
                    },
                });

                if (response.ok) {
                    const data = await response.json();
                    setIsLoggedIn(data.isLoggedIn);
                    setIsAdmin(data.user?.isAdmin || false);
                } else {
                    setIsLoggedIn(false);
                    setIsAdmin(false);
                }
            } catch (err) {
                console.error("Could not fetch auth status:", err);
                setIsLoggedIn(false);
                setIsAdmin(false);
            }
        };

        checkLoginStatus();
    }, []);

    useEffect(() => {
        let isPolling = true;

        const fetchAndPoll = async () => {
            if (!isPolling) return;

            try {
                const response = await fetch(`http://localhost:8080/submissions/${id}`, {
                    method: 'GET',
                    credentials: 'include',
                    headers: {
                        'Accept': 'application/json'
                    }
                });

                if (!response.ok) {
                    if (response.status === 403) {
                        setError('You do not have permission to view this submission');
                    } else if (response.status === 404) {
                        setError('Submission not found');
                    } else {
                        throw new Error(`Error: ${response.status}`);
                    }
                    setLoading(false);
                    return;
                }

                const data = await response.json();
                setSubmission(data);
                setLoading(false); // Set loading to false as soon as we have data

                if (data.status === 'PENDING' || data.status === 'IN_PROGRESS') {
                    setTimeout(fetchAndPoll, 3000); // Poll every 3 seconds
                }

            } catch (err) {
                setError('Failed to fetch submission details');
                console.error('Error fetching submission details:', err);
                setLoading(false);
            }
        };

        if (id) {
            fetchAndPoll();
        }

        return () => {
            isPolling = false; // Cleanup to stop polling when component unmounts
        };
    }, [id]);

    const formatDate = (dateString: string) => {
        const date = new Date(dateString);
        return date.toLocaleString();
    };

    const getStatusClass = (status: string) => {
        switch (status) {
            case 'ACCEPTED':
                return 'bg-green-100 text-green-800';
            case 'WRONG_ANSWER':
                return 'bg-red-100 text-red-800';
            case 'TIME_LIMIT_EXCEEDED':
                return 'bg-yellow-100 text-yellow-800';
            case 'MEMORY_LIMIT_EXCEEDED':
                return 'bg-yellow-100 text-yellow-800';
            case 'RUNTIME_ERROR':
                return 'bg-orange-100 text-orange-800';
            case 'COMPILATION_ERROR':
                return 'bg-orange-100 text-orange-800';
            case 'PENDING':
                return 'bg-blue-100 text-blue-800';
            default:
                return 'bg-gray-100 text-gray-800';
        }
    };

    const getStatusIcon = (status: string) => {
        switch (status) {
            case 'ACCEPTED':
                return '✅';
            case 'WRONG_ANSWER':
                return '❌';
            case 'TIME_LIMIT_EXCEEDED':
                return '⏱️';
            case 'MEMORY_LIMIT_EXCEEDED':
                return '📊';
            case 'RUNTIME_ERROR':
                return '💥';
            case 'COMPILATION_ERROR':
                return '🔧';
            case 'PENDING':
                return '⏳';
            default:
                return '❓';
        }
    };

    const getStatusDescription = (status: string) => {
        switch (status) {
            case 'ACCEPTED':
                return 'Your solution passed all test cases!';
            case 'WRONG_ANSWER':
                return 'Your solution produced incorrect output for one or more test cases.';
            case 'TIME_LIMIT_EXCEEDED':
                return 'Your solution took too long to execute.';
            case 'MEMORY_LIMIT_EXCEEDED':
                return 'Your solution used too much memory.';
            case 'RUNTIME_ERROR':
                return 'Your solution encountered an error during execution.';
            case 'COMPILATION_ERROR':
                return 'Your code failed to compile or had syntax errors.';
            case 'PENDING':
                return 'Your submission is being processed...';
            default:
                return 'Unknown status.';
        }
    };

    const getLanguageFormatted = (language: string) => {
        switch (language.toLowerCase()) {
            case 'python':
                return 'Python';
            case 'javascript':
                return 'JavaScript';
            case 'cpp':
                return 'C++';
            case 'java':
                return 'Java';
            default:
                return language;
        }
    };

    const getLanguageHighlight = (language: string) => {
        switch (language.toLowerCase()) {
            case 'python':
                return 'python';
            case 'javascript':
                return 'javascript';
            case 'cpp':
                return 'cpp';
            case 'java':
                return 'java';
            default:
                return 'plaintext';
        }
    };

    return (
        <>
            <Head>
                <title>Submission Details | OJ - Online Judge</title>
                <meta name="description" content="View submission details" />
            </Head>

            {/* Header/Navigation */}
            <header className="bg-white shadow-md">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4 flex justify-between items-center">
                    <div className="flex items-center">
                        <h1 className="text-2xl font-bold text-indigo-600">OJ</h1>
                        <nav className="ml-10 flex space-x-8">
                            <Link href="/" className="text-gray-500 hover:text-indigo-600 font-medium">
                                Home
                            </Link>
                            <Link href="/problems" className="text-gray-500 hover:text-indigo-600 font-medium">
                                Problems
                            </Link>
                            {isLoggedIn && (
                                <Link href="/submissions" className="text-gray-900 hover:text-indigo-600 font-medium">
                                    Submissions
                                </Link>
                            )}
                            {isAdmin && (
                                <Link href="/admin/problems/create" className="text-gray-500 hover:text-indigo-600 font-medium">
                                    Add Problem
                                </Link>
                            )}
                        </nav>
                    </div>
                    <div className="flex items-center space-x-4">
                        {isLoggedIn ? (
                            <button
                                className="bg-gray-200 hover:bg-gray-300 text-gray-800 font-semibold py-2 px-4 rounded"
                                onClick={() => {
                                    document.cookie = "authToken=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;";
                                    setIsLoggedIn(false);
                                    router.push('/');
                                }}
                            >
                                Logout
                            </button>
                        ) : (
                            <>
                                <Link href="/login" className="text-indigo-600 hover:text-indigo-800 font-medium">
                                    Sign In
                                </Link>
                                <Link href="/register" className="bg-indigo-600 hover:bg-indigo-700 text-white font-semibold py-2 px-4 rounded">
                                    Sign Up
                                </Link>
                            </>
                        )}
                    </div>
                </div>
            </header>

            <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
                <div className="flex items-center mb-6">
                    <Link href="/submissions" className="text-indigo-600 hover:text-indigo-800 mr-4">
                        ← Back to Submissions
                    </Link>
                    <h1 className="text-3xl font-bold text-white">Submission Details</h1>
                </div>

                {loading ? (
                    <div className="text-center py-10">
                        <div className="inline-block animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-indigo-600"></div>
                        <p className="mt-2 text-gray-500">Loading submission details...</p>
                    </div>
                ) : error ? (
                    <div className="bg-red-50 border-l-4 border-red-400 p-4">
                        <p className="text-red-700">{error}</p>
                    </div>
                ) : submission ? (
                    <div className="bg-white rounded-lg shadow-md overflow-hidden">
                        {/* Submission Info */}
                        <div className="p-6 border-b border-gray-200">
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                                <div>
                                    <h2 className="text-xl font-semibold text-gray-900 mb-4">Submission Information</h2>
                                    <dl className="grid grid-cols-1 gap-x-4 gap-y-4 sm:grid-cols-2">
                                        <div className="sm:col-span-1">
                                            <dt className="text-sm font-medium text-gray-500">Problem</dt>
                                            <dd className="mt-1 text-sm text-gray-900">
                                                <Link href={`/problems/${submission.problem_id}`} className="text-indigo-600 hover:text-indigo-900">
                                                    {submission.problem_title || submission.problem_id}
                                                </Link>
                                            </dd>
                                        </div>
                                        <div className="sm:col-span-1">
                                            <dt className="text-sm font-medium text-gray-500">User</dt>
                                            <dd className="mt-1 text-sm text-gray-900">{submission.username}</dd>
                                        </div>
                                        <div className="sm:col-span-1">
                                            <dt className="text-sm font-medium text-gray-500">Submitted At</dt>
                                            <dd className="mt-1 text-sm text-gray-900">{formatDate(submission.submitted_at)}</dd>
                                        </div>
                                        <div className="sm:col-span-1">
                                            <dt className="text-sm font-medium text-gray-500">Language</dt>
                                            <dd className="mt-1 text-sm text-gray-900">{getLanguageFormatted(submission.language)}</dd>
                                        </div>
                                        <div className="sm:col-span-1">
                                            <dt className="text-sm font-medium text-gray-500">Status</dt>
                                            <dd className="mt-1">
                                                <span className={`px-3 py-1 inline-flex items-center text-sm leading-5 font-semibold rounded-full ${getStatusClass(submission.status)}`}>
                                                    {getStatusIcon(submission.status)} {submission.status.replace(/_/g, ' ')}
                                                </span>
                                                <p className="mt-1 text-xs text-gray-500">{getStatusDescription(submission.status)}</p>
                                            </dd>
                                        </div>
                                        <div className="sm:col-span-1">
                                            <dt className="text-sm font-medium text-gray-500">Execution Time</dt>
                                            <dd className="mt-1 text-sm text-gray-900">
                                                {submission.execution_time_ms > 0 ? `${submission.execution_time_ms} ms` : '-'}
                                            </dd>
                                        </div>
                                        <div className="sm:col-span-1">
                                            <dt className="text-sm font-medium text-gray-500">Memory Used</dt>
                                            <dd className="mt-1 text-sm text-gray-900">
                                                {submission.memory_used_kb > 0 ? `${submission.memory_used_kb} KB` : '-'}
                                            </dd>
                                        </div>
                                        <div className="sm:col-span-1">
                                            <dt className="text-sm font-medium text-gray-500">Test Cases</dt>
                                            <dd className="mt-1 text-sm text-gray-900">
                                                {submission.test_cases_passed}/{submission.test_cases_total} passed
                                            </dd>
                                        </div>
                                    </dl>
                                </div>
                            </div>
                        </div>

                        {/* Code Section */}
                        <div className="p-6 border-b border-gray-200">
                            <h2 className="text-xl font-semibold text-gray-900 mb-4">Submitted Code</h2>
                            <div className="bg-gray-50 rounded-md overflow-x-auto">
                                <pre className="p-4 text-sm text-gray-800 font-mono whitespace-pre">
                                    {submission.code}
                                </pre>
                            </div>
                        </div>

                        {/* Test Case Results */}
                        {submission.test_case_status && (
                            <div className="p-6">
                                <h2 className="text-xl font-semibold text-gray-900 mb-4">Test Case Results</h2>
                                <div className="bg-gray-50 rounded-md overflow-x-auto">
                                    <pre className="p-4 text-sm text-gray-800 font-mono whitespace-pre">
                                        {submission.test_case_status}
                                    </pre>
                                </div>
                            </div>
                        )}
                    </div>
                ) : (
                    <div className="text-center py-10 text-gray-500">
                        Submission not found or you don't have permission to view it.
                    </div>
                )}
            </main>
        </>
    );
} 