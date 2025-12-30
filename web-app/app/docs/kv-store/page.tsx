import CodeBlock from '@/components/CodeBlock';
import { CODE_EXAMPLES } from '@/lib/constants';

export default function KVStorePage() {
    return (
        <div className="space-y-8">
            <div>
                <h1 className="text-4xl font-bold mb-4 text-white">Key-Value Store API</h1>
                <p className="text-xl text-gray-400">
                    High-performance KV operations with 319K reads/sec and sub-10μs latency
                </p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 not-prose">
                <div className="bg-[#13131a] p-4 rounded-lg border border-cyan-500/20">
                    <div className="text-2xl font-bold text-cyan-400 mb-1">319K/s</div>
                    <div className="text-sm text-gray-400">Read Throughput</div>
                </div>
                <div className="bg-[#13131a] p-4 rounded-lg border border-purple-500/20">
                    <div className="text-2xl font-bold text-purple-400 mb-1">151K/s</div>
                    <div className="text-sm text-gray-400">Write Throughput</div>
                </div>
                <div className="bg-[#13131a] p-4 rounded-lg border border-cyan-500/20">
                    <div className="text-2xl font-bold text-cyan-400 mb-1">792K/s</div>
                    <div className="text-sm text-gray-400">Batch Operations</div>
                </div>
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">Quick Example</h2>
                <CodeBlock code={CODE_EXAMPLES.kvStore} language="go" />
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">API Methods</h2>

                <div className="space-y-6">
                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold mb-2 text-cyan-400">Set(key string, value []byte) error</h3>
                        <p className="text-gray-300 mb-3">Stores a value for a given key. Overwrites existing values.</p>
                        <CodeBlock code={`err := client.KV.Set("user:101", []byte("Alice"))`} language="go" />
                    </div>

                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold mb-2 text-cyan-400">Get(key string) ([]byte, error)</h3>
                        <p className="text-gray-300 mb-3">Retrieves the value for a key. Returns error if key does not exist.</p>
                        <CodeBlock code={`val, err := client.KV.Get("user:101")`} language="go" />
                    </div>

                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold mb-2 text-cyan-400">Delete(key string) error</h3>
                        <p className="text-gray-300 mb-3">Removes a key and its value.</p>
                        <CodeBlock code={`err := client.KV.Delete("user:101")`} language="go" />
                    </div>

                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold mb-2 text-cyan-400">Exists(key string) (bool, error)</h3>
                        <p className="text-gray-300 mb-3">Checks if a key exists.</p>
                        <CodeBlock code={`exists, err := client.KV.Exists("user:101")`} language="go" />
                    </div>

                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold mb-2 text-purple-400">Incr(key string) (int64, error)</h3>
                        <p className="text-gray-300 mb-3">Atomically increments a counter. Creates the key if it doesn't exist.</p>
                        <CodeBlock code={`newVal, err := client.KV.Incr("visits:page:home")`} language="go" />
                    </div>

                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold mb-2 text-purple-400">Decr(key string) (int64, error)</h3>
                        <p className="text-gray-300 mb-3">Atomically decrements a counter.</p>
                        <CodeBlock code={`newVal, err := client.KV.Decr("stock:item:123")`} language="go" />
                    </div>
                </div>
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">Batch Operations</h2>
                <p className="text-gray-300 mb-4">
                    Batch operations provide atomic multi-key operations with significantly higher throughput (792K ops/sec).
                </p>

                <div className="space-y-6">
                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold mb-2 text-cyan-400">MSet(keys []string, values [][]byte) error</h3>
                        <p className="text-gray-300 mb-3">Set multiple keys atomically.</p>
                        <CodeBlock
                            code={`client.KV.MSet([]string{"k1", "k2"}, [][]byte{\n    []byte("v1"),\n    []byte("v2"),\n})`}
                            language="go"
                        />
                    </div>

                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold mb-2 text-cyan-400">MGet(keys []string) ([][]byte, error)</h3>
                        <p className="text-gray-300 mb-3">Get multiple keys at once.</p>
                        <CodeBlock code={`values, err := client.KV.MGet([]string{"k1", "k2"})`} language="go" />
                    </div>

                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold mb-2 text-cyan-400">MDelete(keys []string) error</h3>
                        <p className="text-gray-300 mb-3">Delete multiple keys at once.</p>
                        <CodeBlock code={`err := client.KV.MDelete([]string{"k1", "k2", "k3"})`} language="go" />
                    </div>
                </div>
            </div>

            <div className="bg-cyan-500/10 border border-cyan-500/20 rounded-lg p-6">
                <h3 className="text-lg font-semibold mb-2">💡 Performance Tips</h3>
                <ul className="space-y-2 text-gray-300 text-sm">
                    <li>• Use batch operations (MSet/MGet/MDelete) for multiple keys to achieve 7.9x speedup</li>
                    <li>• Enable memory mode (-memory flag) for caching use cases to maximize performance</li>
                    <li>• Use connection pooling (configure MinConnections and MaxConnections)</li>
                    <li>• Keep keys and values reasonably sized for optimal throughput</li>
                </ul>
            </div>
        </div>
    );
}
