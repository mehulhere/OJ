import Head from 'next/head';
import { useState, useEffect } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/router';
import '@/app/globals.css';

// Define submission type
interface Submission {
    id: string;
    user_id: string;
    username: string;
    problem_id: string;
    problem_title: string;
    language: string;
    status: string;
    execution_time_ms: number;
    submitted_at: string;
}

// Define pagination type
interface Pagination {
    total: number;
    page: number;
    limit: number;
    total_pages: number;
}

export default function SubmissionsPage() {
    const router = useRouter();
    const [submissions, setSubmissions] = useState<Submission[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [isLoggedIn, setIsLoggedIn] = useState(false);
    const [isAdmin, setIsAdmin] = useState(false);
    const [pagination, setPagination] = useState<Pagination>({
        total: 0,
        page: 1,
        limit: 50,
        total_pages: 0
    });

    // Filter states
    const [problemId, setProblemId] = useState<string>("");
    const [statusFilter, setStatusFilter] = useState<string>("all");
    const [languageFilter, setLanguageFilter] = useState<string>("all");
    const [mySubmissionsOnly, setMySubmissionsOnly] = useState<boolean>(false);

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

                    // If user is logged in, default to showing their submissions
                    if (data.isLoggedIn) {
                        setMySubmissionsOnly(true);
                    }
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
        // Fetch submissions when filters change
        fetchSubmissions();
    }, [problemId, statusFilter, languageFilter, mySubmissionsOnly, pagination.page]);

    const fetchSubmissions = async () => {
        setLoading(true);
        try {
            // Build query parameters
            const queryParams = new URLSearchParams();

            if (problemId) {
                queryParams.append('problem_id', problemId);
            }

            if (mySubmissionsOnly) {
                queryParams.append('my_submissions', 'true');
            }

            if (pagination.page > 1) {
                queryParams.append('page', pagination.page.toString());
            }

            const response = await fetch(`http://localhost:8080/submissions?${queryParams.toString()}`, {
                method: 'GET',
                credentials: 'include',
                headers: {
                    'Accept': 'application/json'
                }
            });

            if (!response.ok) {
                throw new Error(`Error: ${response.status}`);
            }

            const data = await response.json();

            // Filter submissions by status and language if needed
            let filteredSubmissions = data.submissions;

            if (statusFilter !== 'all') {
                filteredSubmissions = filteredSubmissions.filter(
                    (sub: Submission) => sub.status === statusFilter
                );
            }

            if (languageFilter !== 'all') {
                filteredSubmissions = filteredSubmissions.filter(
                    (sub: Submission) => sub.language === languageFilter
                );
            }

            setSubmissions(filteredSubmissions);
            setPagination(data.pagination);
        } catch (err) {
            setError('Failed to fetch submissions');
            console.error('Error fetching submissions:', err);
        } finally {
            setLoading(false);
        }
    };

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

    const handlePageChange = (newPage: number) => {
        if (newPage >= 1 && newPage <= pagination.total_pages) {
            setPagination({ ...pagination, page: newPage });
        }
    };

    return (
        <>
            <Head>
                <title>Submissions | OJ - Online Judge</title>
                <meta name="description" content="View code submissions" />
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
                <div className="flex justify-between items-center mb-6">
                    <h1 className="text-3xl font-bold text-gray-900">Submissions</h1>
                </div>

                {/* Filter Controls */}
                <div className="bg-white rounded-lg shadow-md p-6 mb-6">
                    <div className="mb-6 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                        {/* Problem ID Filter */}
                        <div>
                            <label htmlFor="problem-id" className="block text-sm font-medium text-gray-700 mb-1">Problem ID</label>
                            <input
                                type="text"
                                id="problem-id"
                                className="shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full sm:text-sm border border-gray-300 rounded-md p-2"
                                placeholder="Filter by problem ID"
                                value={problemId}
                                onChange={(e) => setProblemId(e.target.value)}
                            />
                        </div>

                        {/* Status Filter */}
                        <div>
                            <label htmlFor="status" className="block text-sm font-medium text-gray-700 mb-1">Status</label>
                            <select
                                id="status"
                                className="shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full sm:text-sm border border-gray-300 rounded-md p-2"
                                value={statusFilter}
                                onChange={(e) => setStatusFilter(e.target.value)}
                            >
                                <option value="all">All Statuses</option>
                                <option value="ACCEPTED">Accepted</option>
                                <option value="WRONG_ANSWER">Wrong Answer</option>
                                <option value="TIME_LIMIT_EXCEEDED">Time Limit Exceeded</option>
                                <option value="MEMORY_LIMIT_EXCEEDED">Memory Limit Exceeded</option>
                                <option value="RUNTIME_ERROR">Runtime Error</option>
                                <option value="COMPILATION_ERROR">Compilation Error</option>
                                <option value="PENDING">Pending</option>
                            </select>
                        </div>

                        {/* Language Filter */}
                        <div>
                            <label htmlFor="language" className="block text-sm font-medium text-gray-700 mb-1">Language</label>
                            <select
                                id="language"
                                className="shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full sm:text-sm border border-gray-300 rounded-md p-2"
                                value={languageFilter}
                                onChange={(e) => setLanguageFilter(e.target.value)}
                            >
                                <option value="all">All Languages</option>
                                <option value="python">Python</option>
                                <option value="javascript">JavaScript</option>
                                <option value="cpp">C++</option>
                                <option value="java">Java</option>
                            </select>
                        </div>

                        {/* My Submissions Toggle */}
                        {isLoggedIn && (
                            <div className="flex items-end">
                                <label className="inline-flex items-center">
                                    <input
                                        type="checkbox"
                                        className="form-checkbox h-5 w-5 text-indigo-600"
                                        checked={mySubmissionsOnly}
                                        onChange={(e) => setMySubmissionsOnly(e.target.checked)}
                                    />
                                    <span className="ml-2 text-gray-700">My submissions only</span>
                                </label>
                            </div>
                        )}
                    </div>

                    <div className="flex justify-end">
                        <button
                            className="bg-indigo-600 hover:bg-indigo-700 text-white font-semibold py-2 px-4 rounded"
                            onClick={fetchSubmissions}
                        >
                            Apply Filters
                        </button>
                    </div>
                </div>

                {/* Submissions Table */}
                <div className="bg-white rounded-lg shadow-md overflow-hidden">
                    {loading ? (
                        <div className="text-center py-10">
                            <div className="inline-block animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-indigo-600"></div>
                            <p className="mt-2 text-gray-500">Loading submissions...</p>
                        </div>
                    ) : error ? (
                        <div className="bg-red-50 border-l-4 border-red-400 p-4">
                            <p className="text-red-700">{error}</p>
                        </div>
                    ) : submissions.length === 0 ? (
                        <div className="text-center py-10 text-gray-500">
                            No submissions found matching your criteria.
                        </div>
                    ) : (
                        <div className="overflow-x-auto">
                            <table className="min-w-full divide-y divide-gray-200">
                                <thead className="bg-gray-50">
                                    <tr>
                                        <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                            Submission Time
                                        </th>
                                        <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                            User
                                        </th>
                                        <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                            Problem
                                        </th>
                                        <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                            Language
                                        </th>
                                        <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                            Status
                                        </th>
                                        <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                            Time
                                        </th>
                                    </tr>
                                </thead>
                                <tbody className="bg-white divide-y divide-gray-200">
                                    {submissions.map((submission) => (
                                        <tr key={submission.id} className="hover:bg-gray-50">
                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                                <Link href={`/submissions/${submission.id}`} className="text-indigo-600 hover:text-indigo-900">
                                                    {formatDate(submission.submitted_at)}
                                                </Link>
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                                                {submission.username}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                                <Link href={`/problems/${submission.problem_id}`} className="text-indigo-600 hover:text-indigo-900">
                                                    {submission.problem_title || submission.problem_id}
                                                </Link>
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                                {submission.language}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap">
                                                <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${getStatusClass(submission.status)}`}>
                                                    {submission.status.replace(/_/g, ' ')}
                                                </span>
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                                {submission.execution_time_ms > 0 ? `${submission.execution_time_ms} ms` : '-'}
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    )}

                    {/* Pagination */}
                    {pagination.total_pages > 1 && (
                        <div className="px-6 py-4 bg-gray-50 border-t border-gray-200 flex items-center justify-between">
                            <div className="flex-1 flex justify-between sm:hidden">
                                <button
                                    onClick={() => handlePageChange(pagination.page - 1)}
                                    disabled={pagination.page === 1}
                                    className={`relative inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md ${pagination.page === 1 ? 'bg-gray-100 text-gray-400' : 'bg-white text-gray-700 hover:bg-gray-50'
                                        }`}
                                >
                                    Previous
                                </button>
                                <button
                                    onClick={() => handlePageChange(pagination.page + 1)}
                                    disabled={pagination.page === pagination.total_pages}
                                    className={`relative inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md ${pagination.page === pagination.total_pages ? 'bg-gray-100 text-gray-400' : 'bg-white text-gray-700 hover:bg-gray-50'
                                        }`}
                                >
                                    Next
                                </button>
                            </div>
                            <div className="hidden sm:flex-1 sm:flex sm:items-center sm:justify-between">
                                <div>
                                    <p className="text-sm text-gray-700">
                                        Showing <span className="font-medium">{(pagination.page - 1) * pagination.limit + 1}</span> to{' '}
                                        <span className="font-medium">
                                            {Math.min(pagination.page * pagination.limit, pagination.total)}
                                        </span>{' '}
                                        of <span className="font-medium">{pagination.total}</span> results
                                    </p>
                                </div>
                                <div>
                                    <nav className="relative z-0 inline-flex rounded-md shadow-sm -space-x-px" aria-label="Pagination">
                                        <button
                                            onClick={() => handlePageChange(1)}
                                            disabled={pagination.page === 1}
                                            className={`relative inline-flex items-center px-2 py-2 rounded-l-md border border-gray-300 bg-white text-sm font-medium ${pagination.page === 1 ? 'text-gray-300' : 'text-gray-500 hover:bg-gray-50'
                                                }`}
                                        >
                                            <span className="sr-only">First Page</span>
                                            <span>«</span>
                                        </button>
                                        <button
                                            onClick={() => handlePageChange(pagination.page - 1)}
                                            disabled={pagination.page === 1}
                                            className={`relative inline-flex items-center px-2 py-2 border border-gray-300 bg-white text-sm font-medium ${pagination.page === 1 ? 'text-gray-300' : 'text-gray-500 hover:bg-gray-50'
                                                }`}
                                        >
                                            <span className="sr-only">Previous</span>
                                            <span>‹</span>
                                        </button>

                                        {/* Page numbers */}
                                        {[...Array(Math.min(5, pagination.total_pages))].map((_, i) => {
                                            let pageNum;

                                            if (pagination.total_pages <= 5) {
                                                pageNum = i + 1;
                                            } else if (pagination.page <= 3) {
                                                pageNum = i + 1;
                                            } else if (pagination.page >= pagination.total_pages - 2) {
                                                pageNum = pagination.total_pages - 4 + i;
                                            } else {
                                                pageNum = pagination.page - 2 + i;
                                            }

                                            return (
                                                <button
                                                    key={pageNum}
                                                    onClick={() => handlePageChange(pageNum)}
                                                    className={`relative inline-flex items-center px-4 py-2 border text-sm font-medium ${pagination.page === pageNum
                                                        ? 'z-10 bg-indigo-50 border-indigo-500 text-indigo-600'
                                                        : 'bg-white border-gray-300 text-gray-500 hover:bg-gray-50'
                                                        }`}
                                                >
                                                    {pageNum}
                                                </button>
                                            );
                                        })}

                                        <button
                                            onClick={() => handlePageChange(pagination.page + 1)}
                                            disabled={pagination.page === pagination.total_pages}
                                            className={`relative inline-flex items-center px-2 py-2 border border-gray-300 bg-white text-sm font-medium ${pagination.page === pagination.total_pages ? 'text-gray-300' : 'text-gray-500 hover:bg-gray-50'
                                                }`}
                                        >
                                            <span className="sr-only">Next</span>
                                            <span>›</span>
                                        </button>
                                        <button
                                            onClick={() => handlePageChange(pagination.total_pages)}
                                            disabled={pagination.page === pagination.total_pages}
                                            className={`relative inline-flex items-center px-2 py-2 rounded-r-md border border-gray-300 bg-white text-sm font-medium ${pagination.page === pagination.total_pages ? 'text-gray-300' : 'text-gray-500 hover:bg-gray-50'
                                                }`}
                                        >
                                            <span className="sr-only">Last Page</span>
                                            <span>»</span>
                                        </button>
                                    </nav>
                                </div>
                            </div>
                        </div>
                    )}
                </div>
            </main>
        </>
    );
}
