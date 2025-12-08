import CodeBlock from '@/components/CodeBlock';
import { CODE_EXAMPLES } from '@/lib/constants';

export default function StreamPage() {
    return (
        <div className="space-y-8">
            <div>
                <h1 className="text-4xl font-bold mb-4 text-white">Stream Processing API</h1>
                <p className="text-xl text-gray-400">
                    Kafka-like pub/sub with partitions and consumer groups
                </p>
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">Quick Example</h2>
                <CodeBlock code={CODE_EXAMPLES.stream} language="go" />
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">Core Concepts</h2>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-lg font-semibold text-cyan-400 mb-2">📊 Topics</h3>
                        <p className="text-sm text-gray-300">Named streams of messages, divided into partitions for scalability</p>
                    </div>
                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-lg font-semibold text-purple-400 mb-2">🔀 Partitions</h3>
                        <p className="text-sm text-gray-300">Parallel processing units within a topic for horizontal scaling</p>
                    </div>
                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-lg font-semibold text-cyan-400 mb-2">👥 Consumer Groups</h3>
                        <p className="text-sm text-gray-300">Multiple consumers working together to process messages</p>
                    </div>
                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-lg font-semibold text-purple-400 mb-2">📍 Offsets</h3>
                        <p className="text-sm text-gray-300">Track message position for reliable delivery and replay</p>
                    </div>
                </div>
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">API Methods</h2>

                <div className="space-y-6">
                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold mb-2 text-cyan-400">CreateTopic(topic string, partitions int, retentionMs int64) error</h3>
                        <p className="text-gray-300 mb-3">Creates a new topic with specified partitions and retention period.</p>
                        <CodeBlock
                            code={`// 4 partitions, 7 days retention\nerr := client.Stream.CreateTopic("logs", 4, 7*24*60*60*1000)`}
                            language="go"
                        />
                    </div>

                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold mb-2 text-purple-400">Publish(topic string, partition int, key string, value []byte) error</h3>
                        <p className="text-gray-300 mb-3">Publishes a message to a topic. Use partition: -1 for automatic partitioning based on key hash.</p>
                        <CodeBlock
                            code={`err := client.Stream.Publish("logs", -1, "server-1", []byte("Error: 500"))`}
                            language="go"
                        />
                    </div>

                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold mb-2 text-cyan-400">Subscribe(topic, group, consumer string) error</h3>
                        <p className="text-gray-300 mb-3">Registers a consumer as part of a consumer group.</p>
                        <CodeBlock
                            code={`err := client.Stream.Subscribe("logs", "log-processors", "worker-1")`}
                            language="go"
                        />
                    </div>

                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold mb-2 text-purple-400">Consume(topic, group, consumer string, count int) ([]StreamMessage, error)</h3>
                        <p className="text-gray-300 mb-3">Fetches a batch of messages for the consumer.</p>
                        <CodeBlock
                            code={`msgs, err := client.Stream.Consume("logs", "log-processors", "worker-1", 10)\nfor _, msg := range msgs {\n    fmt.Printf("Partition: %d, Offset: %d\\n", msg.Partition, msg.Offset)\n}`}
                            language="go"
                        />
                    </div>

                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold mb-2 text-cyan-400">Commit(topic, group string, partition int, offset uint64) error</h3>
                        <p className="text-gray-300 mb-3">Commits the processed offset for a consumer group.</p>
                        <CodeBlock
                            code={`err := client.Stream.Commit("logs", "log-processors", msg.Partition, msg.Offset+1)`}
                            language="go"
                        />
                    </div>
                </div>
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">Features</h2>
                <div className="space-y-3">
                    <div className="flex items-start gap-3 p-4 bg-[#13131a] rounded-lg border border-cyan-500/20">
                        <span className="text-cyan-400 text-xl">✓</span>
                        <div>
                            <h4 className="font-semibold text-cyan-400 mb-1">At-Least-Once Delivery</h4>
                            <p className="text-sm text-gray-400">Messages are guaranteed to be delivered at least once with proper offset management</p>
                        </div>
                    </div>
                    <div className="flex items-start gap-3 p-4 bg-[#13131a] rounded-lg border border-purple-500/20">
                        <span className="text-purple-400 text-xl">✓</span>
                        <div>
                            <h4 className="font-semibold text-purple-400 mb-1">Automatic Rebalancing</h4>
                            <p className="text-sm text-gray-400">Consumer groups automatically rebalance when consumers join or leave</p>
                        </div>
                    </div>
                    <div className="flex items-start gap-3 p-4 bg-[#13131a] rounded-lg border border-cyan-500/20">
                        <span className="text-cyan-400 text-xl">✓</span>
                        <div>
                            <h4 className="font-semibold text-cyan-400 mb-1">Retention Policies</h4>
                            <p className="text-sm text-gray-400">Automatic cleanup of old messages based on time-based retention</p>
                        </div>
                    </div>
                    <div className="flex items-start gap-3 p-4 bg-[#13131a] rounded-lg border border-purple-500/20">
                        <span className="text-purple-400 text-xl">✓</span>
                        <div>
                            <h4 className="font-semibold text-purple-400 mb-1">Message Replay</h4>
                            <p className="text-sm text-gray-400">Replay messages from any offset for debugging or reprocessing</p>
                        </div>
                    </div>
                </div>
            </div>

            <div className="bg-cyan-500/10 border border-cyan-500/20 rounded-lg p-6">
                <h3 className="text-lg font-semibold mb-2">💡 Best Practices</h3>
                <ul className="space-y-2 text-gray-300 text-sm">
                    <li>• Use meaningful keys for automatic partitioning (e.g., user ID, session ID)</li>
                    <li>• Choose partition count based on expected throughput and parallelism needs</li>
                    <li>• Always commit offsets after successfully processing messages</li>
                    <li>• Use consumer groups for parallel processing across multiple workers</li>
                    <li>• Set appropriate retention periods based on your replay requirements</li>
                </ul>
            </div>
        </div>
    );
}
