import CodeBlock from '@/components/CodeBlock';
import { ArrowRight, Zap, Database } from 'lucide-react';
import Link from 'next/link';

export default function DocsPage() {
    return (
        <div className="space-y-8">
            <div>
                <h1 className="text-4xl font-bold mb-4 text-white">
                    Welcome to Flin Documentation
                </h1>
                <p className="text-xl text-gray-400">
                    High-performance distributed data platform combining KV Store, Message Queue,
                    Stream Processing, and Document Database in a single unified system.
                </p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 not-prose">
                <div className="glass p-6 rounded-xl">
                    <Zap className="w-8 h-8 text-cyan-400 mb-3" />
                    <h3 className="text-lg font-bold mb-2">Quick Start</h3>
                    <p className="text-sm text-gray-400 mb-4">
                        Get up and running with Flin in minutes
                    </p>
                    <Link href="/docs/getting-started" className="text-cyan-400 hover:text-cyan-300 text-sm font-semibold inline-flex items-center gap-2">
                        Get Started <ArrowRight className="w-4 h-4" />
                    </Link>
                </div>

                <div className="glass p-6 rounded-xl">
                    <Database className="w-8 h-8 text-purple-400 mb-3" />
                    <h3 className="text-lg font-bold mb-2">API Reference</h3>
                    <p className="text-sm text-gray-400 mb-4">
                        Explore the complete API documentation
                    </p>
                    <Link href="/docs/kv-store" className="text-purple-400 hover:text-purple-300 text-sm font-semibold inline-flex items-center gap-2">
                        Browse APIs <ArrowRight className="w-4 h-4" />
                    </Link>
                </div>
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">What is Flin?</h2>
                <p className="text-gray-300 mb-4">
                    Flin is a blazing-fast, distributed data platform that unifies four powerful engines:
                </p>
                <ul className="space-y-2 text-gray-300">
                    <li className="flex items-start gap-2">
                        <span className="text-cyan-400 mt-1">🔑</span>
                        <span><strong>Key-Value Store:</strong> 319K reads/sec with sub-10μs latency</span>
                    </li>
                    <li className="flex items-start gap-2">
                        <span className="text-purple-400 mt-1">📬</span>
                        <span><strong>Message Queue:</strong> 104K push/sec with durable persistence</span>
                    </li>
                    <li className="flex items-start gap-2">
                        <span className="text-cyan-400 mt-1">🌊</span>
                        <span><strong>Stream Processing:</strong> Kafka-like pub/sub with partitions</span>
                    </li>
                    <li className="flex items-start gap-2">
                        <span className="text-purple-400 mt-1">📄</span>
                        <span><strong>Document Database:</strong> 76K inserts/sec with Prisma-like API</span>
                    </li>
                </ul>
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">Key Features</h2>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="bg-[#13131a] p-4 rounded-lg border border-cyan-500/20">
                        <h3 className="font-semibold text-cyan-400 mb-2">⚡ High Performance</h3>
                        <p className="text-sm text-gray-400">3x faster than Redis with sub-microsecond latency</p>
                    </div>
                    <div className="bg-[#13131a] p-4 rounded-lg border border-purple-500/20">
                        <h3 className="font-semibold text-purple-400 mb-2">🔄 Unified Port</h3>
                        <p className="text-sm text-gray-400">All operations on a single port (7380)</p>
                    </div>
                    <div className="bg-[#13131a] p-4 rounded-lg border border-cyan-500/20">
                        <h3 className="font-semibold text-cyan-400 mb-2">💾 Durable</h3>
                        <p className="text-sm text-gray-400">BadgerDB persistence with ACID guarantees</p>
                    </div>
                    <div className="bg-[#13131a] p-4 rounded-lg border border-purple-500/20">
                        <h3 className="font-semibold text-purple-400 mb-2">🌐 Distributed</h3>
                        <p className="text-sm text-gray-400">Raft consensus for clustering and replication</p>
                    </div>
                </div>
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">Next Steps</h2>
                <div className="space-y-3">
                    <Link href="/docs/getting-started" className="block glass glass-hover p-4 rounded-lg">
                        <div className="flex items-center justify-between">
                            <div>
                                <h3 className="font-semibold mb-1">Installation</h3>
                                <p className="text-sm text-gray-400">Install and run Flin locally or with Docker</p>
                            </div>
                            <ArrowRight className="w-5 h-5 text-cyan-400" />
                        </div>
                    </Link>
                    <Link href="/docs/kv-store" className="block glass glass-hover p-4 rounded-lg">
                        <div className="flex items-center justify-between">
                            <div>
                                <h3 className="font-semibold mb-1">Key-Value Store API</h3>
                                <p className="text-sm text-gray-400">Learn about KV operations and batch processing</p>
                            </div>
                            <ArrowRight className="w-5 h-5 text-cyan-400" />
                        </div>
                    </Link>
                    <Link href="/docs/clustering" className="block glass glass-hover p-4 rounded-lg">
                        <div className="flex items-center justify-between">
                            <div>
                                <h3 className="font-semibold mb-1">Clustering Guide</h3>
                                <p className="text-sm text-gray-400">Set up a distributed Flin cluster</p>
                            </div>
                            <ArrowRight className="w-5 h-5 text-cyan-400" />
                        </div>
                    </Link>
                </div>
            </div>
        </div>
    );
}
