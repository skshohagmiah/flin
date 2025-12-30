import CodeBlock from '@/components/CodeBlock';
import { CODE_EXAMPLES } from '@/lib/constants';

export default function ClusteringPage() {
    return (
        <div className="space-y-8">
            <div>
                <h1 className="text-4xl font-bold mb-4 text-white">Clustering & Deployment</h1>
                <p className="text-xl text-gray-400">
                    Set up a distributed Flin cluster with Raft consensus
                </p>
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">Cluster Architecture</h2>
                <p className="text-gray-300 mb-4">
                    Flin uses <strong>Raft consensus</strong> for distributed clustering, providing:
                </p>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-lg font-semibold text-cyan-400 mb-2">👑 Leader Election</h3>
                        <p className="text-sm text-gray-300">Automatic leader election with failover</p>
                    </div>
                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-lg font-semibold text-purple-400 mb-2">📋 Log Replication</h3>
                        <p className="text-sm text-gray-300">Consistent data replication across nodes</p>
                    </div>
                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-lg font-semibold text-cyan-400 mb-2">🔀 Partition Management</h3>
                        <p className="text-sm text-gray-300">Distributed partition assignment</p>
                    </div>
                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-lg font-semibold text-purple-400 mb-2">🔄 Automatic Failover</h3>
                        <p className="text-sm text-gray-300">High availability with node recovery</p>
                    </div>
                </div>
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">Setting Up a Cluster</h2>
                <p className="text-gray-300 mb-4">
                    Create a 3-node cluster for high availability and fault tolerance:
                </p>
                <CodeBlock code={CODE_EXAMPLES.clustering} language="bash" />
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">🐳 Docker Cluster Setup</h2>
                <p className="text-gray-300 mb-4">
                    The easiest way to run a cluster is using Docker Compose:
                </p>
                <CodeBlock
                    code={`cd docker/cluster\n./run.sh\n\n# This will:\n# 1. Start 3 Flin nodes\n# 2. Configure Raft cluster\n# 3. Run benchmarks\n# 4. Leave cluster running`}
                    language="bash"
                />
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">Configuration Flags</h2>
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
                                <td className="py-2 px-4">Unique identifier for this node</td>
                            </tr>
                            <tr className="border-b border-white/5">
                                <td className="py-2 px-4 font-mono text-purple-400">-http</td>
                                <td className="py-2 px-4">:8080</td>
                                <td className="py-2 px-4">HTTP API server address</td>
                            </tr>
                            <tr className="border-b border-white/5">
                                <td className="py-2 px-4 font-mono text-purple-400">-raft</td>
                                <td className="py-2 px-4">:9080</td>
                                <td className="py-2 px-4">Raft consensus protocol address</td>
                            </tr>
                            <tr className="border-b border-white/5">
                                <td className="py-2 px-4 font-mono text-purple-400">-port</td>
                                <td className="py-2 px-4">:7380</td>
                                <td className="py-2 px-4">Unified server port (KV+Queue+Stream+DB)</td>
                            </tr>
                            <tr className="border-b border-white/5">
                                <td className="py-2 px-4 font-mono text-purple-400">-data</td>
                                <td className="py-2 px-4">./data</td>
                                <td className="py-2 px-4">Data directory for persistence</td>
                            </tr>
                            <tr className="border-b border-white/5">
                                <td className="py-2 px-4 font-mono text-purple-400">-workers</td>
                                <td className="py-2 px-4">64</td>
                                <td className="py-2 px-4">Worker pool size for request handling</td>
                            </tr>
                            <tr className="border-b border-white/5">
                                <td className="py-2 px-4 font-mono text-purple-400">-partitions</td>
                                <td className="py-2 px-4">64</td>
                                <td className="py-2 px-4">Number of partitions for data sharding</td>
                            </tr>
                            <tr className="border-b border-white/5">
                                <td className="py-2 px-4 font-mono text-purple-400">-memory</td>
                                <td className="py-2 px-4">false</td>
                                <td className="py-2 px-4">Use in-memory storage (no persistence)</td>
                            </tr>
                            <tr className="border-b border-white/5">
                                <td className="py-2 px-4 font-mono text-purple-400">-join</td>
                                <td className="py-2 px-4">(empty)</td>
                                <td className="py-2 px-4">Address of existing node to join cluster</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">Storage Modes</h2>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold text-cyan-400 mb-3">💾 Disk Mode (Default)</h3>
                        <ul className="space-y-2 text-sm text-gray-300">
                            <li>✓ Durable persistence via BadgerDB</li>
                            <li>✓ Survives server restarts</li>
                            <li>✓ Optimized for throughput</li>
                            <li>✓ ACID guarantees</li>
                        </ul>
                        <CodeBlock
                            code={`./bin/flin-server -node-id=node1 -port=:7380`}
                            language="bash"
                        />
                    </div>

                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold text-purple-400 mb-3">⚡ Memory Mode</h3>
                        <ul className="space-y-2 text-sm text-gray-300">
                            <li>✓ Fastest performance</li>
                            <li>✓ Ideal for caching</li>
                            <li>✗ Data lost on restart</li>
                            <li>✓ Use for temporary data</li>
                        </ul>
                        <CodeBlock
                            code={`./bin/flin-server -node-id=node1 -port=:7380 -memory`}
                            language="bash"
                        />
                    </div>
                </div>
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">Production Recommendations</h2>
                <div className="space-y-3">
                    <div className="bg-[#13131a] p-4 rounded-lg border border-cyan-500/20">
                        <h3 className="font-semibold text-cyan-400 mb-2">🔢 Cluster Size</h3>
                        <p className="text-sm text-gray-400">Use 3 or 5 nodes for production (odd numbers for Raft quorum)</p>
                    </div>
                    <div className="bg-[#13131a] p-4 rounded-lg border border-purple-500/20">
                        <h3 className="font-semibold text-purple-400 mb-2">💪 Worker Pool</h3>
                        <p className="text-sm text-gray-400">Set workers to 2-4x CPU cores for optimal throughput</p>
                    </div>
                    <div className="bg-[#13131a] p-4 rounded-lg border border-cyan-500/20">
                        <h3 className="font-semibold text-cyan-400 mb-2">📊 Monitoring</h3>
                        <p className="text-sm text-gray-400">Monitor HTTP API endpoints for cluster health and metrics</p>
                    </div>
                    <div className="bg-[#13131a] p-4 rounded-lg border border-purple-500/20">
                        <h3 className="font-semibold text-purple-400 mb-2">💾 Backups</h3>
                        <p className="text-sm text-gray-400">Regularly backup the data directory for disaster recovery</p>
                    </div>
                </div>
            </div>

            <div className="bg-cyan-500/10 border border-cyan-500/20 rounded-lg p-6">
                <h3 className="text-lg font-semibold mb-2">💡 Best Practices</h3>
                <ul className="space-y-2 text-gray-300 text-sm">
                    <li>• Always use odd number of nodes (3, 5, 7) for Raft consensus</li>
                    <li>• Distribute nodes across different availability zones for fault tolerance</li>
                    <li>• Use disk mode for production workloads requiring durability</li>
                    <li>• Monitor cluster health via HTTP API endpoints</li>
                    <li>• Test failover scenarios before production deployment</li>
                    <li>• Use connection pooling in clients for optimal performance</li>
                </ul>
            </div>
        </div>
    );
}
