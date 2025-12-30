import CodeBlock from '@/components/CodeBlock';
import { CODE_EXAMPLES } from '@/lib/constants';

export default function QueuePage() {
    return (
        <div className="space-y-8">
            <div>
                <h1 className="text-4xl font-bold mb-4 text-white">Message Queue API</h1>
                <p className="text-xl text-gray-400">
                    Durable message queue with 104K push/sec on unified port
                </p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 not-prose">
                <div className="bg-[#13131a] p-4 rounded-lg border border-purple-500/20">
                    <div className="text-2xl font-bold text-purple-400 mb-1">104K/s</div>
                    <div className="text-sm text-gray-400">Push Throughput</div>
                </div>
                <div className="bg-[#13131a] p-4 rounded-lg border border-cyan-500/20">
                    <div className="text-2xl font-bold text-cyan-400 mb-1">100K/s</div>
                    <div className="text-sm text-gray-400">Pop Throughput</div>
                </div>
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">Quick Example</h2>
                <CodeBlock code={CODE_EXAMPLES.queue} language="go" />
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">API Methods</h2>

                <div className="space-y-6">
                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold mb-2 text-purple-400">Push(queue string, item []byte) error</h3>
                        <p className="text-gray-300 mb-3">Adds an item to the end of the queue.</p>
                        <CodeBlock
                            code={`err := client.Queue.Push("email_tasks", []byte(\`{"to":"user@example.com"}\`))`}
                            language="go"
                        />
                    </div>

                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold mb-2 text-purple-400">Pop(queue string) ([]byte, error)</h3>
                        <p className="text-gray-300 mb-3">Removes and returns the item from the front of the queue.</p>
                        <CodeBlock code={`task, err := client.Queue.Pop("email_tasks")`} language="go" />
                    </div>

                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold mb-2 text-cyan-400">Peek(queue string) ([]byte, error)</h3>
                        <p className="text-gray-300 mb-3">Returns the item at the front without removing it.</p>
                        <CodeBlock code={`task, err := client.Queue.Peek("email_tasks")`} language="go" />
                    </div>

                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold mb-2 text-cyan-400">Len(queue string) (int64, error)</h3>
                        <p className="text-gray-300 mb-3">Returns the number of items in the queue.</p>
                        <CodeBlock code={`count, err := client.Queue.Len("email_tasks")`} language="go" />
                    </div>

                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold mb-2 text-cyan-400">Clear(queue string) error</h3>
                        <p className="text-gray-300 mb-3">Removes all items from the queue.</p>
                        <CodeBlock code={`err := client.Queue.Clear("email_tasks")`} language="go" />
                    </div>
                </div>
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">Use Cases</h2>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="bg-[#13131a] p-4 rounded-lg border border-purple-500/20">
                        <h3 className="font-semibold text-purple-400 mb-2">🔄 Task Processing</h3>
                        <p className="text-sm text-gray-400">Distribute background jobs across multiple workers</p>
                    </div>
                    <div className="bg-[#13131a] p-4 rounded-lg border border-cyan-500/20">
                        <h3 className="font-semibold text-cyan-400 mb-2">📧 Email Queue</h3>
                        <p className="text-sm text-gray-400">Queue emails for asynchronous delivery</p>
                    </div>
                    <div className="bg-[#13131a] p-4 rounded-lg border border-purple-500/20">
                        <h3 className="font-semibold text-purple-400 mb-2">🔔 Notifications</h3>
                        <p className="text-sm text-gray-400">Buffer and process user notifications</p>
                    </div>
                    <div className="bg-[#13131a] p-4 rounded-lg border border-cyan-500/20">
                        <h3 className="font-semibold text-cyan-400 mb-2">⚡ Event Processing</h3>
                        <p className="text-sm text-gray-400">Handle event-driven workflows</p>
                    </div>
                </div>
            </div>

            <div className="bg-purple-500/10 border border-purple-500/20 rounded-lg p-6">
                <h3 className="text-lg font-semibold mb-2">💡 Best Practices</h3>
                <ul className="space-y-2 text-gray-300 text-sm">
                    <li>• Use multiple queues to separate different types of tasks</li>
                    <li>• Implement retry logic in your workers for failed tasks</li>
                    <li>• Monitor queue length to detect processing bottlenecks</li>
                    <li>• Use Peek() to inspect items without removing them for debugging</li>
                    <li>• Queue operations run on the same port (7380) as KV operations</li>
                </ul>
            </div>
        </div>
    );
}
