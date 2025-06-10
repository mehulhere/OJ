import Head from 'next/head';
import { useState, FormEvent, useEffect } from 'react';
import { useRouter } from 'next/router';
import Link from 'next/link';
import type { ProblemType } from '@/types/problem';

export default function NewTestCasePage() {
    const router = useRouter();
    const { problemId } = router.query;
    const [problem, setProblem] = useState<ProblemType | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const [formData, setFormData] = useState({
        input: '',
        expected_output: '',
        is_sample: false,
        points: 10,
        sequence_number: 1,
        notes: '',
    });
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [submitError, setSubmitError] = useState<string | null>(null);

    useEffect(() => {
        if (!problemId) {
            return;
        }

        const fetchProblem = async () => {
            setIsLoading(true);
            setError(null);
            try {
                const response = await fetch(`http://localhost:8080/problems/${problemId}`, {
                    credentials: 'include',
                });
                if (!response.ok) {
                    throw new Error(`Failed to fetch problem: ${response.status}`);
                }
                const data = await response.json();
                setProblem(data);
            } catch (err) {
                setError(err instanceof Error ? err.message : 'An unknown error occurred');
                console.error(`Fetch problem ${problemId} error:`, err);
            } finally {
                setIsLoading(false);
            }
        };

        fetchProblem();
    }, [problemId]);

    const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
        const { name, value, type } = e.target;
        const checked = type === 'checkbox' ? (e.target as HTMLInputElement).checked : undefined;

        setFormData(prev => ({
            ...prev,
            [name]: type === 'checkbox' ? checked :
                (name === 'points' || name === 'sequence_number') ? parseInt(value, 10) : value
        }));
    };

    const handleSubmit = async (e: FormEvent) => {
        e.preventDefault();
        setIsSubmitting(true);
        setSubmitError(null);

        try {
            const response = await fetch('http://localhost:8080/admin/testcases', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    ...formData,
                    problem_db_id: problemId,
                }),
                credentials: 'include', // Important to include cookies for admin auth
            });

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.message || 'Failed to create test case');
            }

            // Redirect back to the problem page
            router.push(`/problems/${problemId}`);
        } catch (err) {
            setSubmitError(err instanceof Error ? err.message : 'An unknown error occurred');
            console.error('Error creating test case:', err);
        } finally {
            setIsSubmitting(false);
        }
    };

    if (isLoading) {
        return (
            <div className="min-h-screen bg-gray-100 flex justify-center items-center">
                <p className="text-xl text-gray-700">Loading problem details...</p>
            </div>
        );
    }

    if (error) {
        return (
            <div className="min-h-screen bg-gray-100 flex flex-col justify-center items-center p-4">
                <p className="text-xl text-red-600 bg-red-100 p-4 rounded-md mb-4">Error: {error}</p>
                <Link href="/problems" legacyBehavior>
                    <a className="px-4 py-2 bg-indigo-600 text-white rounded-md hover:bg-indigo-700">
                        Back to Problems
                    </a>
                </Link>
            </div>
        );
    }

    if (!problem) {
        return (
            <div className="min-h-screen bg-gray-100 flex flex-col justify-center items-center p-4">
                <p className="text-xl text-gray-700 mb-4">Problem not found.</p>
                <Link href="/problems" legacyBehavior>
                    <a className="px-4 py-2 bg-indigo-600 text-white rounded-md hover:bg-indigo-700">
                        Back to Problems
                    </a>
                </Link>
            </div>
        );
    }

    return (
        <>
            <Head>
                <title>Add Test Case - {problem.title}</title>
            </Head>
            <div className="min-h-screen bg-gray-50 py-8 px-4 sm:px-6 lg:px-8">
                <div className="max-w-4xl mx-auto">
                    <div className="mb-6 flex justify-between items-center">
                        <h1 className="text-3xl font-bold text-gray-900">Add Test Case</h1>
                        <Link href={`/problems/${problemId}`} legacyBehavior>
                            <a className="text-indigo-600 hover:text-indigo-800 font-medium">
                                Back to Problem
                            </a>
                        </Link>
                    </div>

                    <div className="mb-6 bg-white shadow overflow-hidden sm:rounded-lg">
                        <div className="px-4 py-5 sm:px-6">
                            <h2 className="text-lg font-medium text-gray-900">Problem: {problem.title}</h2>
                            <p className="mt-1 max-w-2xl text-sm text-gray-500">
                                Difficulty: {problem.difficulty}
                            </p>
                        </div>
                    </div>

                    {submitError && (
                        <div className="mb-6 bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
                            <p>{submitError}</p>
                        </div>
                    )}

                    <form onSubmit={handleSubmit} className="bg-white shadow-md rounded-lg p-6 mb-8">
                        <div className="mb-6">
                            <label htmlFor="input" className="block text-sm font-medium text-gray-700">
                                Input
                            </label>
                            <textarea
                                id="input"
                                name="input"
                                value={formData.input}
                                onChange={handleChange}
                                required
                                rows={5}
                                className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
                                placeholder="Input for the test case"
                            />
                        </div>

                        <div className="mb-6">
                            <label htmlFor="expected_output" className="block text-sm font-medium text-gray-700">
                                Expected Output
                            </label>
                            <textarea
                                id="expected_output"
                                name="expected_output"
                                value={formData.expected_output}
                                onChange={handleChange}
                                required
                                rows={5}
                                className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
                                placeholder="Expected output for the test case"
                            />
                        </div>

                        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
                            <div>
                                <label htmlFor="points" className="block text-sm font-medium text-gray-700">
                                    Points
                                </label>
                                <input
                                    type="number"
                                    id="points"
                                    name="points"
                                    value={formData.points}
                                    onChange={handleChange}
                                    required
                                    min="1"
                                    className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
                                />
                            </div>

                            <div>
                                <label htmlFor="sequence_number" className="block text-sm font-medium text-gray-700">
                                    Sequence Number
                                </label>
                                <input
                                    type="number"
                                    id="sequence_number"
                                    name="sequence_number"
                                    value={formData.sequence_number}
                                    onChange={handleChange}
                                    required
                                    min="1"
                                    className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
                                />
                            </div>
                        </div>

                        <div className="mt-6">
                            <label htmlFor="notes" className="block text-sm font-medium text-gray-700">
                                Notes
                            </label>
                            <textarea
                                id="notes"
                                name="notes"
                                value={formData.notes}
                                onChange={handleChange}
                                rows={3}
                                className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
                                placeholder="Optional notes about this test case"
                            />
                        </div>

                        <div className="mt-6 flex items-center">
                            <input
                                id="is_sample"
                                name="is_sample"
                                type="checkbox"
                                checked={formData.is_sample}
                                onChange={handleChange}
                                className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
                            />
                            <label htmlFor="is_sample" className="ml-2 block text-sm text-gray-900">
                                Is Sample Test Case (visible to users)
                            </label>
                        </div>

                        <div className="mt-6">
                            <button
                                type="submit"
                                disabled={isSubmitting}
                                className={`w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 ${isSubmitting ? 'opacity-70 cursor-not-allowed' : ''
                                    }`}
                            >
                                {isSubmitting ? 'Adding...' : 'Add Test Case'}
                            </button>
                        </div>
                    </form>
                </div>
            </div>
        </>
    );
} 