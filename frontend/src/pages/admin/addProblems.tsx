import Head from 'next/head';
import { useState, FormEvent } from 'react';
import { useRouter } from 'next/router';
import Link from 'next/link';

export default function NewProblemPage() {
    const router = useRouter();
    const [formData, setFormData] = useState({
        problem_id: '',
        title: '',
        difficulty: 'Medium', // Default value
        statement: '',
        constraints_text: '',
        time_limit_ms: 2000, // Default: 2 seconds
        memory_limit_mb: 256, // Default: 256 MB
        tags: '',
    });
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
        const { name, value } = e.target;
        setFormData(prev => ({
            ...prev,
            [name]: name === 'time_limit_ms' || name === 'memory_limit_mb'
                ? parseInt(value, 10)
                : value
        }));
    };

    const handleSubmit = async (e: FormEvent) => {
        e.preventDefault();
        setIsSubmitting(true);
        setError(null);

        try {
            // Convert tags string to array
            const tagsArray = formData.tags.split(',')
                .map(tag => tag.trim())
                .filter(tag => tag !== '');

            const response = await fetch('http://localhost:8080/admin/problems', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    ...formData,
                    tags: tagsArray,
                }),
                credentials: 'include', // Important to include cookies for admin auth
            });

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.message || 'Failed to create problem');
            }

            const data = await response.json();
            router.push(`/problems/${data.id}`); // Redirect to the new problem page
        } catch (err) {
            setError(err instanceof Error ? err.message : 'An unknown error occurred');
            console.error('Error creating problem:', err);
        } finally {
            setIsSubmitting(false);
        }
    };

    return (
        <>
            <Head>
                <title>Create New Problem - Admin</title>
            </Head>
            <div className="min-h-screen bg-gray-50 py-8 px-4 sm:px-6 lg:px-8">
                <div className="max-w-4xl mx-auto">
                    <div className="mb-6 flex justify-between items-center">
                        <h1 className="text-3xl font-bold text-gray-900">Create New Problem</h1>
                        <Link href="/problems" legacyBehavior>
                            <a className="text-indigo-600 hover:text-indigo-800 font-medium">
                                Back to Problems
                            </a>
                        </Link>
                    </div>

                    {error && (
                        <div className="mb-6 bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
                            <p>{error}</p>
                        </div>
                    )}

                    <form onSubmit={handleSubmit} className="bg-white shadow-md rounded-lg p-6 mb-8">
                        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
                            <div>
                                <label htmlFor="problem_id" className="block text-sm font-medium text-gray-700">
                                    Problem ID
                                </label>
                                <input
                                    type="text"
                                    id="problem_id"
                                    name="problem_id"
                                    value={formData.problem_id}
                                    onChange={handleChange}
                                    required
                                    className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
                                    placeholder="e.g., OJ-123"
                                />
                            </div>

                            <div>
                                <label htmlFor="title" className="block text-sm font-medium text-gray-700">
                                    Title
                                </label>
                                <input
                                    type="text"
                                    id="title"
                                    name="title"
                                    value={formData.title}
                                    onChange={handleChange}
                                    required
                                    className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
                                    placeholder="Problem title"
                                />
                            </div>

                            <div>
                                <label htmlFor="difficulty" className="block text-sm font-medium text-gray-700">
                                    Difficulty
                                </label>
                                <select
                                    id="difficulty"
                                    name="difficulty"
                                    value={formData.difficulty}
                                    onChange={handleChange}
                                    required
                                    className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
                                >
                                    <option value="Easy">Easy</option>
                                    <option value="Medium">Medium</option>
                                    <option value="Hard">Hard</option>
                                </select>
                            </div>

                            <div>
                                <label htmlFor="tags" className="block text-sm font-medium text-gray-700">
                                    Tags (comma separated)
                                </label>
                                <input
                                    type="text"
                                    id="tags"
                                    name="tags"
                                    value={formData.tags}
                                    onChange={handleChange}
                                    className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
                                    placeholder="e.g., Arrays, Sorting, Dynamic Programming"
                                />
                            </div>

                            <div>
                                <label htmlFor="time_limit_ms" className="block text-sm font-medium text-gray-700">
                                    Time Limit (ms)
                                </label>
                                <input
                                    type="number"
                                    id="time_limit_ms"
                                    name="time_limit_ms"
                                    value={formData.time_limit_ms}
                                    onChange={handleChange}
                                    required
                                    min="100"
                                    className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
                                />
                            </div>

                            <div>
                                <label htmlFor="memory_limit_mb" className="block text-sm font-medium text-gray-700">
                                    Memory Limit (MB)
                                </label>
                                <input
                                    type="number"
                                    id="memory_limit_mb"
                                    name="memory_limit_mb"
                                    value={formData.memory_limit_mb}
                                    onChange={handleChange}
                                    required
                                    min="16"
                                    className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
                                />
                            </div>
                        </div>

                        <div className="mt-6">
                            <label htmlFor="statement" className="block text-sm font-medium text-gray-700">
                                Problem Statement
                            </label>
                            <textarea
                                id="statement"
                                name="statement"
                                value={formData.statement}
                                onChange={handleChange}
                                required
                                rows={10}
                                className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
                                placeholder="Provide a detailed description of the problem..."
                            />
                        </div>

                        <div className="mt-6">
                            <label htmlFor="constraints_text" className="block text-sm font-medium text-gray-700">
                                Constraints
                            </label>
                            <textarea
                                id="constraints_text"
                                name="constraints_text"
                                value={formData.constraints_text}
                                onChange={handleChange}
                                rows={5}
                                className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
                                placeholder="e.g., 1 <= nums.length <= 10^5"
                            />
                        </div>

                        <div className="mt-6">
                            <button
                                type="submit"
                                disabled={isSubmitting}
                                className={`w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 ${isSubmitting ? 'opacity-70 cursor-not-allowed' : ''
                                    }`}
                            >
                                {isSubmitting ? 'Creating...' : 'Create Problem'}
                            </button>
                        </div>
                    </form>
                </div>
            </div>
        </>
    );
} 