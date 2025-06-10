import Head from 'next/head';
import { useState, useEffect } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/router';
import '@/app/globals.css';

// Define problem type
interface Problem {
    id: string;
    problem_id: string;
    title: string;
    difficulty: string;
    tags: string[];
    solved?: boolean; // Optional, for logged-in users
}

export default function HomePage() {
    const router = useRouter();
    const [problems, setProblems] = useState<Problem[]>([]);
    const [filteredProblems, setFilteredProblems] = useState<Problem[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [isLoggedIn, setIsLoggedIn] = useState(false);
    const [isAdmin, setIsAdmin] = useState(false);

    // Filter states
    const [selectedDifficulty, setSelectedDifficulty] = useState<string>('all');
    const [selectedStatus, setSelectedStatus] = useState<string>('all');
    const [selectedTag, setSelectedTag] = useState<string>('all');
    const [searchQuery, setSearchQuery] = useState<string>('');

    // Available tags extracted from problems
    const [availableTags, setAvailableTags] = useState<string[]>([]);

    useEffect(() => {
        // Fetch problems from API
        const fetchProblems = async () => {
            setLoading(true);
            try {
                const response = await fetch('http://localhost:8080/problems');
                if (!response.ok) {
                    throw new Error(`Error: ${response.status}`);
                }
                const data = await response.json();
                setProblems(data);
                setFilteredProblems(data);

                // Extract unique tags
                const tags = new Set<string>();
                data.forEach((problem: Problem) => {
                    problem.tags?.forEach(tag => tags.add(tag));
                });
                setAvailableTags(Array.from(tags));
            } catch (err) {
                setError('Failed to fetch problems');
                console.error('Error fetching problems:', err);
            } finally {
                setLoading(false);
            }
        };

        // Check if user is logged in by calling our new backend endpoint
        const checkLoginStatus = async () => {
            const maxRetries = 3;
            let retries = 0;

            const attemptFetch = async () => {
                try {
                    console.log("Attempting to fetch auth status...");
                    const response = await fetch('http://localhost:8080/api/auth-status', {
                        method: 'GET',
                        credentials: 'include', // This is crucial to send the HttpOnly cookie
                        headers: {
                            'Accept': 'application/json',
                            'Content-Type': 'application/json'
                        },
                    });

                    console.log("Auth status response:", response.status);
                    if (response.ok) {
                        const data = await response.json();
                        console.log("Auth data:", data);
                        setIsLoggedIn(data.isLoggedIn);
                        setIsAdmin(data.user?.isAdmin || false);
                    } else {
                        setIsLoggedIn(false);
                        setIsAdmin(false);
                        console.error("Auth status response not OK:", response.status);
                    }
                } catch (err) {
                    console.error("Could not fetch auth status:", err);
                    if (retries < maxRetries) {
                        retries++;
                        console.log(`Retrying auth status fetch (${retries}/${maxRetries})...`);
                        setTimeout(attemptFetch, 1000); // Wait 1 second before retrying
                    } else {
                        setIsLoggedIn(false);
                        setIsAdmin(false);
                    }
                }
            };

            await attemptFetch();
        };

        fetchProblems();
        checkLoginStatus();
    }, []);

    // Apply filters whenever any filter changes
    useEffect(() => {
        let result = [...problems];

        // Filter by difficulty
        if (selectedDifficulty !== 'all') {
            result = result.filter(problem => problem.difficulty.toLowerCase() === selectedDifficulty.toLowerCase());
        }

        // Filter by status (only if logged in)
        if (isLoggedIn && selectedStatus !== 'all') {
            result = result.filter(problem => {
                if (selectedStatus === 'solved') return problem.solved;
                if (selectedStatus === 'unsolved') return !problem.solved;
                return true;
            });
        }

        // Filter by tag
        if (selectedTag !== 'all') {
            result = result.filter(problem => problem.tags?.includes(selectedTag));
        }

        // Filter by search query
        if (searchQuery) {
            const query = searchQuery.toLowerCase();
            result = result.filter(problem =>
                problem.title.toLowerCase().includes(query) ||
                problem.problem_id.toLowerCase().includes(query)
            );
        }

        setFilteredProblems(result);
    }, [selectedDifficulty, selectedStatus, selectedTag, searchQuery, problems, isLoggedIn]);

    const difficultyColor = (difficulty: string) => {
        switch (difficulty.toLowerCase()) {
            case 'easy': return 'text-green-600 bg-green-100';
            case 'medium': return 'text-yellow-600 bg-yellow-100';
            case 'hard': return 'text-red-600 bg-red-100';
            default: return 'text-gray-600 bg-gray-100';
        }
    };

    return (
        <>
            <Head>
                <title>OJ - Online Judge</title>
                <meta name="description" content="Practice coding problems and improve your skills" />
            </Head>

            {/* Header/Navigation */}
            <header className="bg-white shadow-md">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4 flex justify-between items-center">
                    <div className="flex items-center">
                        <h1 className="text-2xl font-bold text-indigo-600">OJ</h1>
                        <nav className="ml-10 flex space-x-8">
                            <Link href="/" className="text-gray-900 hover:text-indigo-600 font-medium">
                                Home
                            </Link>
                            <Link href="/problems" className="text-gray-500 hover:text-indigo-600 font-medium">
                                Problems
                            </Link>
                            {isLoggedIn && (
                                <Link href="/submissions" className="text-gray-500 hover:text-indigo-600 font-medium">
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
                {/* Hero Section */}
                <div className="bg-gradient-to-r from-indigo-500 to-blue-600 rounded-lg shadow-xl p-8 mb-10">
                    <h2 className="text-3xl font-bold text-white mb-4">Welcome to Online Judge</h2>
                    <p className="text-indigo-100 text-xl mb-6">Improve your coding skills by solving algorithmic challenges.</p>
                    <Link href="/problems" className="bg-white text-indigo-600 font-semibold py-3 px-6 rounded-md shadow hover:bg-indigo-50">
                        Start Coding Now
                    </Link>
                </div>

                {/* Problem List Section */}
                <div className="bg-white rounded-lg shadow-md p-6 mb-10">
                    <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center mb-6">
                        <h2 className="text-2xl font-bold text-gray-900 mb-4 sm:mb-0">Featured Problems</h2>
                        <Link href="/problems" className="text-indigo-600 hover:text-indigo-800 font-medium">
                            View All Problems →
                        </Link>
                    </div>

                    {/* Filter Controls */}
                    <div className="mb-6 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                        {/* Search */}
                        <div>
                            <label htmlFor="search" className="block text-sm font-medium text-gray-700 mb-1">Search</label>
                            <input
                                type="text"
                                id="search"
                                className="shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full sm:text-sm border border-gray-300 rounded-md p-2"
                                placeholder="Problem name or ID"
                                value={searchQuery}
                                onChange={(e) => setSearchQuery(e.target.value)}
                            />
                        </div>

                        {/* Difficulty Filter */}
                        <div>
                            <label htmlFor="difficulty" className="block text-sm font-medium text-gray-700 mb-1">Difficulty</label>
                            <select
                                id="difficulty"
                                className="shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full sm:text-sm border border-gray-300 rounded-md p-2"
                                value={selectedDifficulty}
                                onChange={(e) => setSelectedDifficulty(e.target.value)}
                            >
                                <option value="all">All Difficulties</option>
                                <option value="easy">Easy</option>
                                <option value="medium">Medium</option>
                                <option value="hard">Hard</option>
                            </select>
                        </div>

                        {/* Status Filter (only if logged in) */}
                        {isLoggedIn && (
                            <div>
                                <label htmlFor="status" className="block text-sm font-medium text-gray-700 mb-1">Status</label>
                                <select
                                    id="status"
                                    className="shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full sm:text-sm border border-gray-300 rounded-md p-2"
                                    value={selectedStatus}
                                    onChange={(e) => setSelectedStatus(e.target.value)}
                                >
                                    <option value="all">All Problems</option>
                                    <option value="solved">Solved</option>
                                    <option value="unsolved">Unsolved</option>
                                </select>
                            </div>
                        )}

                        {/* Tag Filter */}
                        <div>
                            <label htmlFor="tag" className="block text-sm font-medium text-gray-700 mb-1">Tag</label>
                            <select
                                id="tag"
                                className="shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full sm:text-sm border border-gray-300 rounded-md p-2"
                                value={selectedTag}
                                onChange={(e) => setSelectedTag(e.target.value)}
                            >
                                <option value="all">All Tags</option>
                                {availableTags.map(tag => (
                                    <option key={tag} value={tag}>{tag}</option>
                                ))}
                            </select>
                        </div>
                    </div>

                    {/* Problem Table */}
                    {loading ? (
                        <div className="text-center py-10">
                            <div className="inline-block animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-indigo-600"></div>
                            <p className="mt-2 text-gray-500">Loading problems...</p>
                        </div>
                    ) : error ? (
                        <div className="bg-red-50 border-l-4 border-red-400 p-4 mb-4">
                            <p className="text-red-700">{error}</p>
                        </div>
                    ) : (
                        <div className="overflow-x-auto">
                            <table className="min-w-full divide-y divide-gray-200">
                                <thead className="bg-gray-50">
                                    <tr>
                                        <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                            Status
                                        </th>
                                        <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                            #
                                        </th>
                                        <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                            Title
                                        </th>
                                        <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                            Difficulty
                                        </th>
                                        <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                            Tags
                                        </th>
                                    </tr>
                                </thead>
                                <tbody className="bg-white divide-y divide-gray-200">
                                    {filteredProblems.slice(0, 5).map((problem) => (
                                        <tr key={problem.id} className="hover:bg-gray-50">
                                            <td className="px-6 py-4 whitespace-nowrap">
                                                {isLoggedIn ? (
                                                    problem.solved ? (
                                                        <span className="text-green-500">✓</span>
                                                    ) : (
                                                        <span className="text-gray-300">○</span>
                                                    )
                                                ) : (
                                                    <span className="text-gray-300">-</span>
                                                )}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                                                {problem.problem_id}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm">
                                                <Link href={`/problems/${problem.problem_id}`} className="text-indigo-600 hover:text-indigo-900">
                                                    {problem.title}
                                                </Link>
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm">
                                                <span className={`${difficultyColor(problem.difficulty)} font-medium px-2 py-1 rounded-full`}>
                                                    {problem.difficulty}
                                                </span>
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                                <div className="flex flex-wrap gap-1">
                                                    {problem.tags?.map((tag) => (
                                                        <span key={tag} className="bg-blue-100 text-blue-800 text-xs font-semibold px-2.5 py-0.5 rounded-full border border-blue-200">
                                                            {tag}
                                                        </span>
                                                    ))}
                                                </div>
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    )}

                    {filteredProblems.length === 0 && !loading && !error && (
                        <div className="text-center py-10 text-gray-500">
                            No problems match your filters. Try adjusting your search criteria.
                        </div>
                    )}
                </div>

                {/* Stats Section */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-10">
                    <div className="bg-white rounded-lg shadow p-6 text-center">
                        <p className="text-4xl font-bold text-indigo-600">{problems.length}</p>
                        <p className="text-gray-500 mt-2">Total Problems</p>
                    </div>
                    <div className="bg-white rounded-lg shadow p-6 text-center">
                        <p className="text-4xl font-bold text-green-500">
                            {problems.filter(p => p.difficulty.toLowerCase() === 'easy').length}
                        </p>
                        <p className="text-gray-500 mt-2">Easy Problems</p>
                    </div>
                    <div className="bg-white rounded-lg shadow p-6 text-center">
                        <p className="text-4xl font-bold text-indigo-600">
                            {isLoggedIn ? problems.filter(p => p.solved).length : '-'}
                        </p>
                        <p className="text-gray-500 mt-2">Problems Solved</p>
                    </div>
                </div>
            </main>

            <footer className="bg-gray-800 text-white py-8">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                    <div className="flex flex-col md:flex-row justify-between">
                        <div className="mb-6 md:mb-0">
                            <h2 className="text-xl font-bold">OJ - Online Judge</h2>
                            <p className="text-gray-400 mt-2">Practice coding problems and improve your skills</p>
                        </div>
                        <div className="grid grid-cols-2 gap-8">
                            <div>
                                <h3 className="text-sm font-semibold text-gray-400 tracking-wider uppercase">Resources</h3>
                                <ul className="mt-4 space-y-2">
                                    <li>
                                        <Link href="/problems" className="text-gray-300 hover:text-white">
                                            Problems
                                        </Link>
                                    </li>
                                    <li>
                                        <Link href="#" className="text-gray-300 hover:text-white">
                                            Learn
                                        </Link>
                                    </li>
                                </ul>
                            </div>
                            <div>
                                <h3 className="text-sm font-semibold text-gray-400 tracking-wider uppercase">Account</h3>
                                <ul className="mt-4 space-y-2">
                                    <li>
                                        <Link href="/register" className="text-gray-300 hover:text-white">
                                            Sign Up
                                        </Link>
                                    </li>
                                    <li>
                                        <Link href="/login" className="text-gray-300 hover:text-white">
                                            Sign In
                                        </Link>
                                    </li>
                                </ul>
                            </div>
                        </div>
                    </div>
                    <div className="mt-8 border-t border-gray-700 pt-8 flex flex-col md:flex-row justify-between">
                        <p className="text-gray-400">&copy; 2025 OJ. All rights reserved.</p>
                    </div>
                </div>
            </footer>
        </>
    );
} 