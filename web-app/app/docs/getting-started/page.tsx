import CodeBlock from '@/components/CodeBlock';
import { CODE_EXAMPLES } from '@/lib/constants';

export default function GettingStartedPage() {
    return (
        <div className="space-y-8">
            <div>
                <h1 className="text-4xl font-bold mb-4 text-white">Getting Started</h1>
                <p className="text-xl text-gray-400">
                    Get Flin up and running in minutes with Docker or local installation
                </p>
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">🐳 Docker Installation (Recommended)</h2>
                <p className="text-gray-300 mb-4">
                    The fastest way to get started with Flin is using Docker. Choose between a single node or a 3-node cluster.
                </p>

                <h3 className="text-xl font-semibold mb-3">Single Node</h3>
                <CodeBlock code={CODE_EXAMPLES.docker.split('\n\n')[0]} language="bash" />

                <h3 className="text-xl font-semibold mb-3 mt-6">3-Node Cluster</h3>
                <CodeBlock code={CODE_EXAMPLES.docker.split('\n\n')[1]} language="bash" />

                <div className="bg-cyan-500/10 border border-cyan-500/20 rounded-lg p-4 mt-4">
                    <p className="text-sm text-cyan-300">
                        <strong>💡 Tip:</strong> Both scripts automatically start the node(s), run performance benchmarks,
                        and leave the cluster running for testing.
                    </p>
                </div>
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">💻 Local Installation</h2>
                <p className="text-gray-300 mb-4">
                    For development or custom deployments, you can build and run Flin locally.
                </p>
                <CodeBlock code={CODE_EXAMPLES.local} language="bash" />
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">📦 Client Usage</h2>
                <p className="text-gray-300 mb-4">
                    Once your Flin server is running, connect using the unified Go client:
                </p>
                <CodeBlock code={CODE_EXAMPLES.client} language="go" />
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">⚙️ Configuration Options</h2>
                <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                        <thead>
                            <tr className="border-b border-white/10">
                                <th className="text-left py-2 px-4 text-cyan-400">Flag</th>
                                <th className="text-left py-2 px-4 text-cyan-400">Default</th>
                                <th className="text-left py-2 px-4 text-cyan-400">Description</th>
                            </tr>
                        </thead>
                        <tbody className="text-gray-300">
                            <tr className="border-b border-white/5">
                                <td className="py-2 px-4 font-mono text-purple-400">-node-id</td>
                                <td className="py-2 px-4">(required)</td>
                                <td className="py-2 px-4">Unique node identifier</td>
                            </tr>
                            <tr className="border-b border-white/5">
                                <td className="py-2 px-4 font-mono text-purple-400">-http</td>
                                <td className="py-2 px-4">:8080</td>
                                <td className="py-2 px-4">HTTP API address</td>
                            </tr>
                            <tr className="border-b border-white/5">
                                <td className="py-2 px-4 font-mono text-purple-400">-raft</td>
                                <td className="py-2 px-4">:9080</td>
                                <td className="py-2 px-4">Raft consensus address</td>
                            </tr>
                            <tr className="border-b border-white/5">
                                <td className="py-2 px-4 font-mono text-purple-400">-port</td>
                                <td className="py-2 px-4">:7380</td>
                                <td className="py-2 px-4">Unified server port</td>
                            </tr>
                            <tr className="border-b border-white/5">
                                <td className="py-2 px-4 font-mono text-purple-400">-data</td>
                                <td className="py-2 px-4">./data</td>
                                <td className="py-2 px-4">Data directory</td>
                            </tr>
                            <tr className="border-b border-white/5">
                                <td className="py-2 px-4 font-mono text-purple-400">-workers</td>
                                <td className="py-2 px-4">64</td>
                                <td className="py-2 px-4">Worker pool size</td>
                            </tr>
                            <tr className="border-b border-white/5">
                                <td className="py-2 px-4 font-mono text-purple-400">-memory</td>
                                <td className="py-2 px-4">false</td>
                                <td className="py-2 px-4">Use in-memory storage</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">✅ Verify Installation</h2>
                <p className="text-gray-300 mb-4">
                    Test your Flin installation by running the performance benchmarks:
                </p>
                <CodeBlock
                    code={`cd benchmarks\n\n# KV Store benchmark\n./kv-throughput.sh\n\n# Queue benchmark\n./queue-throughput.sh\n\n# Stream benchmark\n./stream-throughput.sh\n\n# Document DB benchmark\n./db-throughput.sh`}
                    language="bash"
                />
            </div>

            <div className="bg-purple-500/10 border border-purple-500/20 rounded-lg p-6">
                <h3 className="text-lg font-semibold mb-2">🎉 You're all set!</h3>
                <p className="text-gray-300 mb-4">
                    Flin is now running and ready to use. Explore the API documentation to learn more:
                </p>
                <div className="flex flex-wrap gap-2">
                    <a href="/docs/kv-store" className="btn btn-secondary text-sm">KV Store API</a>
                    <a href="/docs/queue" className="btn btn-secondary text-sm">Queue API</a>
                    <a href="/docs/stream" className="btn btn-secondary text-sm">Stream API</a>
                    <a href="/docs/database" className="btn btn-secondary text-sm">Database API</a>
                </div>
            </div>
        </div>
    );
}
