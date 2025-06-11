import Head from 'next/head';
import { useRouter } from 'next/router';
import { useEffect, useState, useRef } from 'react';
import Link from 'next/link';
import type { ProblemType, ApiError } from '@/types/problem'; // Adjust path
import '@/app/globals.css';
import Editor, { Monaco } from '@monaco-editor/react';

export default function SingleProblemPage() {
    const router = useRouter();
    const { problemId } = router.query; // problemId comes from the filename [problemId].tsx

    const [problem, setProblem] = useState<ProblemType | null>(null);
    const [isLoading, setIsLoading] = useState<boolean>(true);
    const [error, setError] = useState<string | null>(null);
    const [code, setCode] = useState<string>('// Start coding here...');
    const [selectedLanguage, setSelectedLanguage] = useState<string>('python');
    const editorRef = useRef<any>(null);
    const [isLoggedIn, setIsLoggedIn] = useState<boolean>(false);
    const [currentTab, setCurrentTab] = useState<string>('description');

    // New state for code execution results
    const [output, setOutput] = useState<any>(null);
    const [isExecuting, setIsExecuting] = useState<boolean>(false);
    const [executionError, setExecutionError] = useState<string | null>(null);
    const [isSubmitting, setIsSubmitting] = useState<boolean>(false);
    const [submissionResult, setSubmissionResult] = useState<any>(null);

    // Track test case results
    const [testCaseResults, setTestCaseResults] = useState<{
        stdout: string;
        stderr: string;
        status: string;
        executionTimeMs: number;
        error?: string;
        testCase?: { input: string, expected?: string };
    }[]>([]);
    const [activeResultTab, setActiveResultTab] = useState<number>(0);

    // Test cases
    const [customTestCases, setCustomTestCases] = useState<{ input: string, expected?: string }[]>([{ input: '', expected: '' }]);
    const [activeTestCase, setActiveTestCase] = useState<number>(0);
    const [testCaseInput, setTestCaseInput] = useState<string>('');

    useEffect(() => {
        // Check login status
        const checkLoginStatus = async () => {
            try {
                const response = await fetch('http://localhost:8080/api/auth-status', {
                    method: 'GET',
                    credentials: 'include',
                });
                if (response.ok) {
                    const data = await response.json();
                    setIsLoggedIn(data.isLoggedIn);
                }
            } catch (err) {
                console.error('Failed to check login status:', err);
            }
        };

        checkLoginStatus();
    }, []);

    useEffect(() => {
        if (!problemId) {
            return;
        }

        const fetchProblem = async () => {
            setIsLoading(true);
            setError(null);
            try {
                const response = await fetch(`http://localhost:8080/problems/${problemId}`);
                if (!response.ok) {
                    let errorMessage = `Failed to fetch problem: ${response.status}`;
                    try {
                        const errorData: ApiError = await response.json();
                        errorMessage = errorData.message || errorMessage;
                    } catch (jsonError) {
                        errorMessage = response.statusText || errorMessage;
                    }
                    throw new Error(errorMessage);
                }
                const data: ProblemType = await response.json();
                setProblem(data);

                // Initialize with sample test cases if available
                if (data.sample_test_cases && data.sample_test_cases.length > 0) {
                    const initialTestCases = data.sample_test_cases.map(tc => ({
                        input: tc.input,
                        expected: tc.expected_output
                    }));
                    setCustomTestCases(initialTestCases);
                    setTestCaseInput(initialTestCases[0].input);
                }
            } catch (err) {
                setError(err instanceof Error ? err.message : 'An unknown error occurred.');
                console.error(`Fetch problem ${problemId} error:`, err);
            } finally {
                setIsLoading(false);
            }
        };

        fetchProblem();
    }, [problemId]);

    // Update test case input when active test case changes
    useEffect(() => {
        if (customTestCases[activeTestCase]) {
            setTestCaseInput(customTestCases[activeTestCase].input);
        }
    }, [activeTestCase, customTestCases]);

    function handleEditorDidMount(editor: any, monaco: Monaco) {
        editorRef.current = editor;
    }

    function handleEditorChange(value: string | undefined) {
        setCode(value || '');
    }

    const handleRunCode = async () => {
        if (isExecuting) return;

        // Restore login check since authentication is required on the backend
        if (!isLoggedIn) {
            setExecutionError("Please log in to run code.");
            return;
        }

        setIsExecuting(true);
        setOutput(null);
        setExecutionError(null);
        setTestCaseResults([]);
        setActiveResultTab(0);

        try {
            // Collect all test case inputs
            const testCaseInputs = customTestCases
                .map(tc => tc.input.trim())
                .filter(input => input.length > 0);

            if (testCaseInputs.length === 0) {
                throw new Error("No test cases to run. Please add at least one test case with input.");
            }

            console.log(`Executing ${testCaseInputs.length} test cases in a single request`);

            // Create an AbortController for timeout
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), 15000); // Longer timeout for multiple test cases

            try {
                const response = await fetch('http://localhost:8080/execute', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    credentials: 'include',
                    body: JSON.stringify({
                        language: selectedLanguage,
                        code: code,
                        testCases: testCaseInputs,
                    }),
                    signal: controller.signal
                });

                // Clear the timeout
                clearTimeout(timeoutId);

                const text = await response.text();
                const responseBody = JSON.parse(text);

                if (!response.ok) {
                    throw new Error(responseBody.message || responseBody.error || `Request failed with status ${response.status}`);
                }

                // Process results from the backend
                const results = [];

                if (responseBody.results && responseBody.results.length > 0) {
                    // Process multiple test case results
                    for (let i = 0; i < responseBody.results.length; i++) {
                        const result = responseBody.results[i];
                        const testCase = customTestCases[i] || { input: testCaseInputs[i], expected: '' };

                        // Determine if the test case passed by comparing with expected output
                        let status = result.status;
                        if (status === 'success' && testCase.expected) {
                            // Clean outputs for comparison (trim whitespace, normalize line endings)
                            const cleanedActual = result.stdout.trim().replace(/\r\n/g, '\n');
                            const cleanedExpected = testCase.expected.trim().replace(/\r\n/g, '\n');

                            // Update status based on output comparison
                            if (cleanedActual !== cleanedExpected) {
                                status = 'wrong_answer';
                            }
                        }

                        results.push({
                            stdout: result.stdout || '',
                            stderr: result.stderr || '',
                            status: status,
                            executionTimeMs: result.execution_time_ms || 0,
                            error: result.error || '',
                            testCase: testCase
                        });
                    }
                } else {
                    // Fallback for backward compatibility
                    const testCase = customTestCases[0] || { input: testCaseInputs[0], expected: '' };

                    // Determine if the test case passed by comparing with expected output
                    let status = responseBody.status;
                    if (status === 'success' && testCase.expected) {
                        // Clean outputs for comparison
                        const cleanedActual = responseBody.stdout.trim().replace(/\r\n/g, '\n');
                        const cleanedExpected = testCase.expected.trim().replace(/\r\n/g, '\n');

                        // Update status based on output comparison
                        if (cleanedActual !== cleanedExpected) {
                            status = 'wrong_answer';
                        }
                    }

                    results.push({
                        stdout: responseBody.stdout || '',
                        stderr: responseBody.stderr || '',
                        status: status,
                        executionTimeMs: responseBody.execution_time_ms || 0,
                        error: responseBody.error || '',
                        testCase: testCase
                    });
                }

                // Update state with all results
                setTestCaseResults(results);

                // If there's at least one result, set it as the current output for backward compatibility
                if (results.length > 0) {
                    setOutput({
                        stdout: results[0].stdout,
                        stderr: results[0].stderr,
                        status: results[0].status,
                        execution_time_ms: results[0].executionTimeMs,
                        error: results[0].error
                    });
                }

            } catch (error: any) {
                console.error('Fetch operation failed:', error);
                if (error.name === 'AbortError') {
                    throw new Error('Request timed out. The server took too long to respond.');
                } else {
                    throw new Error(`Network error: ${error.message}. Please check if the backend server is running.`);
                }
            }
        } catch (err) {
            console.error('Failed to execute code:', err);
            setExecutionError(err instanceof Error ? err.message : 'An unknown error occurred during execution.');
            setOutput(null);
        } finally {
            setIsExecuting(false);
        }
    };

    const handleSubmitCode = async () => {
        if (!isLoggedIn) {
            alert('Please log in to submit your solution');
            return;
        }

        if (isSubmitting) return;

        setIsSubmitting(true);
        setSubmissionResult(null);

        try {
            // Ensure we have a valid problemId
            if (!problemId) {
                throw new Error("Problem ID is missing. Cannot submit without a problem ID.");
            }

            console.log('Attempting to submit code:', {
                problem_id: problemId,
                language: selectedLanguage,
                codeLength: code.length
            });

            try {
                // Simple fetch approach
                const response = await fetch('http://localhost:8080/submit', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    credentials: 'include',
                    body: JSON.stringify({
                        problem_id: problemId,
                        language: selectedLanguage,
                        code: code,
                    }),
                });

                console.log('Response received:', {
                    status: response.status,
                    statusText: response.statusText
                });

                const responseText = await response.text();
                console.log('Raw response text:', responseText);

                if (responseText) {
                    try {
                        const responseBody = JSON.parse(responseText);
                        console.log('Parsed response body:', responseBody);

                        if (!response.ok) {
                            throw new Error(responseBody.message || `Request failed with status ${response.status}`);
                        }

                        setSubmissionResult(responseBody);

                        // Navigate to submission detail page
                        if (responseBody.submission_id) {
                            router.push(`/submissions/${responseBody.submission_id}`);
                        }
                    } catch (parseError) {
                        console.error('Error parsing response:', parseError);
                        throw new Error(`Failed to parse response: ${responseText}`);
                    }
                } else {
                    throw new Error('Server returned an empty response');
                }
            } catch (error: any) {
                console.error('Submission failed:', error);
                throw new Error(`Submission failed: ${error.message}`);
            }
        } catch (err) {
            console.error('Error in handleSubmitCode:', err);
            setSubmissionResult({
                error: err instanceof Error ? err.message : 'An unknown error occurred during submission.'
            });
        } finally {
            setIsSubmitting(false);
        }
    };

    // Add a new test case
    const handleAddTestCase = () => {
        setCustomTestCases([...customTestCases, { input: '', expected: '' }]);
        setActiveTestCase(customTestCases.length);
    };

    // Update test case input for current active test case
    const handleTestCaseInputChange = (value: string) => {
        setTestCaseInput(value);
        const updatedTestCases = [...customTestCases];
        updatedTestCases[activeTestCase] = {
            ...updatedTestCases[activeTestCase],
            input: value
        };
        setCustomTestCases(updatedTestCases);
    };

    // Update expected output for current active test case
    const handleTestCaseExpectedChange = (value: string) => {
        const updatedTestCases = [...customTestCases];
        updatedTestCases[activeTestCase] = {
            ...updatedTestCases[activeTestCase],
            expected: value
        };
        setCustomTestCases(updatedTestCases);
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
                <title>{problem.title} - Online Judge</title>
            </Head>

            {/* Navigation Bar */}
            <nav className="bg-white shadow-sm">
                <div className="max-w-screen-2xl mx-auto px-4 sm:px-6 lg:px-8">
                    <div className="flex justify-between h-16">
                        <div className="flex">
                            <div className="flex-shrink-0 flex items-center">
                                <Link href="/" legacyBehavior>
                                    <a className="text-xl font-bold text-indigo-600">OJ</a>
                                </Link>
                            </div>
                            <div className="ml-6 flex items-center space-x-4">
                                <Link href="/problems" legacyBehavior>
                                    <a className="px-3 py-2 text-sm font-medium text-gray-700 hover:text-gray-900">
                                        Problems
                                    </a>
                                </Link>
                                <Link href="/submissions" legacyBehavior>
                                    <a className="px-3 py-2 text-sm font-medium text-gray-700 hover:text-gray-900">
                                        Submissions
                                    </a>
                                </Link>
                            </div>
                        </div>
                        <div className="flex items-center">
                            {isLoggedIn ? (
                                <button
                                    className="ml-4 px-3 py-2 text-sm font-medium text-indigo-600 hover:text-indigo-800"
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
                                    <Link href="/login" legacyBehavior>
                                        <a className="px-3 py-2 text-sm font-medium text-indigo-600 hover:text-indigo-800">
                                            Sign In
                                        </a>
                                    </Link>
                                    <Link href="/register" legacyBehavior>
                                        <a className="ml-4 px-4 py-2 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 rounded-md">
                                            Sign Up
                                        </a>
                                    </Link>
                                </>
                            )}
                        </div>
                    </div>
                </div>
            </nav>

            {/* Problem Title Bar */}
            <div className="bg-white shadow-sm border-b border-gray-200">
                <div className="max-w-screen-2xl mx-auto px-4 sm:px-6 lg:px-8">
                    <div className="py-4">
                        <div className="flex items-center justify-between">
                            <h1 className="text-2xl font-bold text-gray-900">
                                {problem.problem_id ? `${problem.problem_id}. ` : ''}{problem.title}
                            </h1>
                            <div className="flex items-center">
                                <span className={`px-2.5 py-1 rounded-md text-sm font-medium ${problem.difficulty?.toLowerCase() === 'easy' ? 'bg-green-100 text-green-800' :
                                    problem.difficulty?.toLowerCase() === 'medium' ? 'bg-yellow-100 text-yellow-800' :
                                        'bg-red-100 text-red-800'
                                    }`}>
                                    {problem.difficulty || 'N/A'}
                                </span>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            {/* Main Content */}
            <div className="flex flex-col md:flex-row h-[calc(100vh-128px)] bg-white">
                {/* Left Panel: Problem Description */}
                <div className="w-full md:w-1/2 lg:w-5/12 border-r border-gray-200 flex flex-col">
                    <div className="border-b border-gray-200">
                        <div className="flex">
                            <button
                                className={`px-4 py-2 text-sm font-medium ${currentTab === 'description' ? 'text-indigo-600 border-b-2 border-indigo-600' : 'text-gray-700 hover:text-gray-900'}`}
                                onClick={() => setCurrentTab('description')}
                            >
                                Description
                            </button>
                            <button
                                className={`px-4 py-2 text-sm font-medium ${currentTab === 'submissions' ? 'text-indigo-600 border-b-2 border-indigo-600' : 'text-gray-700 hover:text-gray-900'}`}
                                onClick={() => setCurrentTab('submissions')}
                            >
                                Submissions
                            </button>
                        </div>
                    </div>

                    <div className="overflow-y-auto flex-grow">
                        {currentTab === 'description' && (
                            <div className="p-4">
                                {/* Problem Statement */}
                                <div className="mb-6">
                                    <div className="prose prose-indigo max-w-none text-gray-800"
                                        dangerouslySetInnerHTML={{ __html: problem.statement.replace(/\n/g, '<br />') }}
                                    />
                                </div>

                                {/* Constraints */}
                                <div className="mb-6">
                                    <h2 className="text-lg font-semibold text-gray-800 mb-2">Constraints</h2>
                                    <div className="prose prose-indigo max-w-none text-gray-800"
                                        dangerouslySetInnerHTML={{ __html: problem.constraints_text?.replace(/\n/g, '<br />') || 'N/A' }}
                                    />
                                    <div className="mt-3 grid grid-cols-2 gap-4">
                                        <div>
                                            <p className="text-sm font-medium text-gray-700">Time Limit</p>
                                            <p className="text-sm text-gray-900">{problem.time_limit_ms / 1000} seconds</p>
                                        </div>
                                        <div>
                                            <p className="text-sm font-medium text-gray-700">Memory Limit</p>
                                            <p className="text-sm text-gray-900">{problem.memory_limit_mb} MB</p>
                                        </div>
                                    </div>
                                </div>

                                {/* Sample Test Cases */}
                                {problem.sample_test_cases && problem.sample_test_cases.length > 0 && (
                                    <div>
                                        <h2 className="text-lg font-semibold text-gray-800 mb-3">Examples</h2>
                                        {problem.sample_test_cases.map((tc, index) => (
                                            <div key={index} className="mb-5 last:mb-0">
                                                <h3 className="text-md font-medium text-gray-700 mb-2">Example {index + 1}</h3>
                                                <div className="bg-gray-50 border border-gray-200 rounded-md p-3 mb-2">
                                                    <p className="text-xs font-medium text-gray-700 uppercase mb-1">Input:</p>
                                                    <pre className="text-sm text-gray-800 whitespace-pre-wrap">{tc.input}</pre>
                                                </div>
                                                <div className="bg-gray-50 border border-gray-200 rounded-md p-3">
                                                    <p className="text-xs font-medium text-gray-700 uppercase mb-1">Output:</p>
                                                    <pre className="text-sm text-gray-800 whitespace-pre-wrap">{tc.expected_output}</pre>
                                                </div>
                                                {tc.notes && (
                                                    <div className="mt-2 text-sm text-gray-700">
                                                        <p className="font-medium">Explanation:</p>
                                                        <p>{tc.notes}</p>
                                                    </div>
                                                )}
                                            </div>
                                        ))}
                                    </div>
                                )}
                            </div>
                        )}

                        {currentTab === 'submissions' && (
                            <div className="p-4">
                                <div>
                                    <h2 className="text-lg font-semibold text-gray-800 mb-3">Your Submissions</h2>
                                    {isLoggedIn ? (
                                        <p className="text-gray-700 text-sm">View your submissions history for this problem.</p>
                                    ) : (
                                        <div className="bg-blue-50 border border-blue-200 text-blue-700 p-3 rounded-md">
                                            <p>Please <Link href="/login" className="underline">sign in</Link> to view your submissions.</p>
                                        </div>
                                    )}
                                </div>
                            </div>
                        )}
                    </div>
                </div>

                {/* Right Panel: Code Editor */}
                <div className="w-full md:w-1/2 lg:w-7/12 flex flex-col">
                    {/* Language Selector */}
                    <div className="h-12 bg-white border-b border-gray-200 px-4 flex items-center">
                        <select
                            className="mr-4 py-1 px-2 text-sm border border-gray-300 rounded-md text-gray-700"
                            value={selectedLanguage}
                            onChange={e => setSelectedLanguage(e.target.value)}
                        >
                            <option value="python">Python</option>
                            <option value="javascript">JavaScript</option>
                            <option value="java">Java</option>
                            <option value="cpp">C++</option>
                        </select>
                    </div>

                    {/* Code Editor */}
                    <div className="flex-grow overflow-hidden border-b border-gray-200">
                        <Editor
                            height="100%"
                            defaultLanguage={selectedLanguage}
                            language={selectedLanguage}
                            value={code}
                            onChange={handleEditorChange}
                            onMount={handleEditorDidMount}
                            theme="vs-light"
                            options={{
                                minimap: { enabled: false },
                                fontSize: 14,
                                lineNumbers: 'on',
                                scrollBeyondLastLine: false,
                                automaticLayout: true,
                            }}
                        />
                    </div>

                    {/* Test Cases and Console */}
                    <div className="h-64 flex flex-col bg-white">
                        {/* Tabs */}
                        <div className="h-10 bg-white border-b border-gray-200 flex items-center px-4">
                            <div className="flex">
                                <div className="flex items-center mr-4">
                                    {customTestCases.map((_, index) => (
                                        <button
                                            key={index}
                                            className={`px-3 py-1 text-xs mr-2 rounded-full ${activeTestCase === index
                                                ? 'bg-indigo-600 text-white'
                                                : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
                                                }`}
                                            onClick={() => setActiveTestCase(index)}
                                        >
                                            Case {index + 1}
                                        </button>
                                    ))}
                                    <button
                                        className="px-3 py-1 text-xs bg-gray-200 text-gray-700 hover:bg-gray-300 rounded-full"
                                        onClick={handleAddTestCase}
                                    >
                                        +
                                    </button>
                                </div>
                            </div>
                        </div>

                        {/* Input/Output Area */}
                        <div className="flex-grow grid grid-cols-2 gap-4 p-4 overflow-auto">
                            <div className="space-y-4">
                                {/* Input */}
                                <div className="flex flex-col">
                                    <p className="text-xs font-medium text-gray-700 mb-1">Input:</p>
                                    <textarea
                                        className="flex-grow p-2 text-sm font-mono border border-gray-300 rounded-md focus:outline-none focus:ring-1 focus:ring-indigo-500 text-gray-800"
                                        value={testCaseInput}
                                        onChange={(e) => handleTestCaseInputChange(e.target.value)}
                                        placeholder="Enter input for this test case..."
                                        rows={3}
                                    />
                                </div>

                                {/* Expected Output */}
                                <div className="flex flex-col">
                                    <p className="text-xs font-medium text-gray-700 mb-1">Expected Output:</p>
                                    <textarea
                                        className="flex-grow p-2 text-sm font-mono border border-gray-300 rounded-md focus:outline-none focus:ring-1 focus:ring-indigo-500 text-gray-800"
                                        value={customTestCases[activeTestCase]?.expected || ''}
                                        onChange={(e) => handleTestCaseExpectedChange(e.target.value)}
                                        placeholder="Enter expected output for verification..."
                                        rows={3}
                                    />
                                </div>
                            </div>

                            {/* Output */}
                            <div className="flex flex-col">
                                <p className="text-xs font-medium text-gray-700 mb-1">Output:</p>
                                <div className="flex-grow p-2 text-sm font-mono border border-gray-300 rounded-md bg-gray-50 overflow-auto whitespace-pre-wrap text-gray-800">
                                    {isExecuting ? (
                                        <div className="text-gray-600">Running code against all test cases...</div>
                                    ) : executionError ? (
                                        <div className="text-red-600">{executionError}</div>
                                    ) : testCaseResults.length > 0 ? (
                                        <>
                                            {/* Test Result Header */}
                                            <div className="bg-gray-800 text-white p-2 flex items-center">
                                                <div className="flex items-center">
                                                    <span className={`w-2 h-2 rounded-full mr-2 ${testCaseResults.every(r => r.status === 'success')
                                                        ? 'bg-green-500'
                                                        : 'bg-red-500'
                                                        }`}></span>
                                                    <span className="font-medium">
                                                        {testCaseResults.every(r => r.status === 'success')
                                                            ? 'Accepted'
                                                            : 'Failed'}
                                                    </span>
                                                </div>
                                                <div className="ml-4 text-xs text-gray-300">
                                                    Runtime: {testCaseResults[activeResultTab]?.executionTimeMs || 0} ms
                                                </div>
                                            </div>

                                            {/* Test Case Tabs */}
                                            <div className="bg-gray-700 text-white px-2 pt-1 flex">
                                                {testCaseResults.map((_, index) => (
                                                    <button
                                                        key={index}
                                                        className={`px-3 py-1 text-xs mr-1 rounded-t ${activeResultTab === index
                                                            ? 'bg-gray-50 text-gray-800'
                                                            : 'bg-gray-600 text-gray-200 hover:bg-gray-500'
                                                            }`}
                                                        onClick={() => setActiveResultTab(index)}
                                                    >
                                                        <span className={`w-2 h-2 rounded-full inline-block mr-1 ${testCaseResults[index].status === 'success'
                                                            ? 'bg-green-500'
                                                            : 'bg-red-500'
                                                            }`}></span>
                                                        Case {index + 1}
                                                    </button>
                                                ))}
                                            </div>

                                            {/* Current Test Case Result */}
                                            <div className="p-2 text-sm font-mono whitespace-pre-wrap flex-grow">
                                                {testCaseResults[activeResultTab] && (
                                                    <div className="space-y-2">
                                                        {/* Output */}
                                                        {testCaseResults[activeResultTab].stdout && (
                                                            <div>
                                                                <div className="font-semibold text-xs text-gray-700 mb-1">Your Output:</div>
                                                                <div className="pl-2 border-l-2 border-green-400">
                                                                    {testCaseResults[activeResultTab].stdout}
                                                                </div>
                                                            </div>
                                                        )}

                                                        {/* Error */}
                                                        {(testCaseResults[activeResultTab].stderr || testCaseResults[activeResultTab].error) && (
                                                            <div>
                                                                <div className="font-semibold text-xs text-red-700 mb-1">Error:</div>
                                                                <div className="pl-2 border-l-2 border-red-400 text-red-600">
                                                                    {testCaseResults[activeResultTab].stderr || testCaseResults[activeResultTab].error}
                                                                </div>
                                                            </div>
                                                        )}

                                                        {/* Comparison Result */}
                                                        {testCaseResults[activeResultTab].status === 'wrong_answer' && (
                                                            <div className="mt-1 text-xs text-red-600">
                                                                Your output does not match the expected output.
                                                            </div>
                                                        )}
                                                    </div>
                                                )}
                                            </div>
                                        </>
                                    ) : submissionResult?.error ? (
                                        <div className="text-red-600">Submission error: {submissionResult.error}</div>
                                    ) : (
                                        <div className="text-gray-500">
                                            Click "Run" to execute your code against all test cases.
                                        </div>
                                    )}
                                </div>
                            </div>
                        </div>

                        {/* Authentication Banner - shown when not logged in */}
                        {!isLoggedIn && (
                            <div className="mx-4 mb-2 p-2 bg-yellow-50 border border-yellow-200 rounded text-yellow-800 text-sm">
                                <p className="font-medium">Authentication Required</p>
                                <p>Please <Link href="/login" className="text-blue-600 underline">sign in</Link> to run or submit code.</p>
                            </div>
                        )}

                        {/* Action Buttons */}
                        <div className="h-16 bg-white border-t border-gray-200 flex items-center justify-end px-4">
                            <button
                                className="px-4 py-2 mr-3 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                                onClick={handleRunCode}
                                disabled={isExecuting || !isLoggedIn}
                                title="Run your code against all test cases"
                            >
                                {isExecuting ? 'Running...' : `Run All Cases (${customTestCases.length})`}
                            </button>
                            <button
                                className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 border border-transparent rounded-md shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                                onClick={handleSubmitCode}
                                disabled={isSubmitting || !isLoggedIn}
                            >
                                {isSubmitting ? 'Submitting...' : 'Submit'}
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        </>
    );
} 