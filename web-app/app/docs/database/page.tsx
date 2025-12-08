import CodeBlock from '@/components/CodeBlock';
import { CODE_EXAMPLES } from '@/lib/constants';

export default function DatabasePage() {
    return (
        <div className="space-y-8">
            <div>
                <h1 className="text-4xl font-bold mb-4 text-white">Document Database API</h1>
                <p className="text-xl text-gray-400">
                    MongoDB-like document store with Prisma-like query builder
                </p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 not-prose">
                <div className="bg-[#13131a] p-4 rounded-lg border border-purple-500/20">
                    <div className="text-2xl font-bold text-purple-400 mb-1">76K/s</div>
                    <div className="text-sm text-gray-400">Insert Throughput</div>
                </div>
                <div className="bg-[#13131a] p-4 rounded-lg border border-cyan-500/20">
                    <div className="text-2xl font-bold text-cyan-400 mb-1">13μs</div>
                    <div className="text-sm text-gray-400">Average Latency</div>
                </div>
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">Quick Example</h2>
                <CodeBlock code={CODE_EXAMPLES.database} language="go" />
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">API Methods</h2>

                <div className="space-y-6">
                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold mb-2 text-purple-400">Insert(collection string, doc map[string]interface{ }) (string, error)</h3>
                        <p className="text-gray-300 mb-3">Inserts a JSON document and returns its generated ID (UUID).</p>
                        <CodeBlock
                            code={`id, err := client.DB.Insert("users", map[string]interface{}{\n    "name":  "John Doe",\n    "email": "john@example.com",\n    "age":   30,\n})`}
                            language="go"
                        />
                    </div>

                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold mb-2 text-cyan-400">Query(collection string)</h3>
                        <p className="text-gray-300 mb-3">Starts a query builder chain with Prisma-like fluent API.</p>
                        <div className="space-y-3">
                            <div>
                                <h4 className="text-sm font-semibold text-gray-400 mb-2">Methods:</h4>
                                <ul className="text-sm text-gray-300 space-y-1 ml-4">
                                    <li>• <code className="text-purple-400">Where(field, operator, value)</code> - Add filter condition</li>
                                    <li>• <code className="text-purple-400">OrderBy(field, direction)</code> - Sort results</li>
                                    <li>• <code className="text-purple-400">Skip(n)</code> - Skip first n results</li>
                                    <li>• <code className="text-purple-400">Take(n)</code> - Limit results to n</li>
                                    <li>• <code className="text-purple-400">Exec()</code> - Execute the query</li>
                                </ul>
                            </div>
                            <CodeBlock
                                code={`users, err := client.DB.Query("users").\n    Where("age", flin.Gte, 18).\n    Where("status", flin.Eq, "active").\n    OrderBy("created_at", flin.Desc).\n    Skip(0).\n    Take(10).\n    Exec()`}
                                language="go"
                            />
                        </div>
                    </div>

                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold mb-2 text-purple-400">Update(collection string)</h3>
                        <p className="text-gray-300 mb-3">Starts an update builder chain.</p>
                        <div className="space-y-3">
                            <div>
                                <h4 className="text-sm font-semibold text-gray-400 mb-2">Methods:</h4>
                                <ul className="text-sm text-gray-300 space-y-1 ml-4">
                                    <li>• <code className="text-cyan-400">Where(...)</code> - Select documents to update</li>
                                    <li>• <code className="text-cyan-400">Set(field, value)</code> - Set field values</li>
                                    <li>• <code className="text-cyan-400">Exec()</code> - Execute the update</li>
                                </ul>
                            </div>
                            <CodeBlock
                                code={`err := client.DB.Update("users").\n    Where("id", flin.Eq, "user-123").\n    Set("active", false).\n    Set("updated_at", time.Now()).\n    Exec()`}
                                language="go"
                            />
                        </div>
                    </div>

                    <div className="glass p-6 rounded-xl">
                        <h3 className="text-xl font-semibold mb-2 text-cyan-400">Delete(collection string)</h3>
                        <p className="text-gray-300 mb-3">Starts a delete builder chain.</p>
                        <CodeBlock
                            code={`err := client.DB.Delete("users").\n    Where("active", flin.Eq, false).\n    Exec()`}
                            language="go"
                        />
                    </div>
                </div>
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">Query Operators</h2>
                <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                        <thead>
                            <tr className="border-b border-white/10">
                                <th className="text-left py-2 px-4 text-cyan-400">Operator</th>
                                <th className="text-left py-2 px-4 text-cyan-400">Description</th>
                                <th className="text-left py-2 px-4 text-cyan-400">Example</th>
                            </tr>
                        </thead>
                        <tbody className="text-gray-300">
                            <tr className="border-b border-white/5">
                                <td className="py-2 px-4 font-mono text-purple-400">flin.Eq</td>
                                <td className="py-2 px-4">Equal to</td>
                                <td className="py-2 px-4 font-mono text-sm">Where("age", flin.Eq, 30)</td>
                            </tr>
                            <tr className="border-b border-white/5">
                                <td className="py-2 px-4 font-mono text-purple-400">flin.Ne</td>
                                <td className="py-2 px-4">Not equal to</td>
                                <td className="py-2 px-4 font-mono text-sm">Where("status", flin.Ne, "deleted")</td>
                            </tr>
                            <tr className="border-b border-white/5">
                                <td className="py-2 px-4 font-mono text-purple-400">flin.Gt</td>
                                <td className="py-2 px-4">Greater than</td>
                                <td className="py-2 px-4 font-mono text-sm">Where("age", flin.Gt, 18)</td>
                            </tr>
                            <tr className="border-b border-white/5">
                                <td className="py-2 px-4 font-mono text-purple-400">flin.Gte</td>
                                <td className="py-2 px-4">Greater than or equal</td>
                                <td className="py-2 px-4 font-mono text-sm">Where("age", flin.Gte, 18)</td>
                            </tr>
                            <tr className="border-b border-white/5">
                                <td className="py-2 px-4 font-mono text-purple-400">flin.Lt</td>
                                <td className="py-2 px-4">Less than</td>
                                <td className="py-2 px-4 font-mono text-sm">Where("price", flin.Lt, 100)</td>
                            </tr>
                            <tr className="border-b border-white/5">
                                <td className="py-2 px-4 font-mono text-purple-400">flin.Lte</td>
                                <td className="py-2 px-4">Less than or equal</td>
                                <td className="py-2 px-4 font-mono text-sm">Where("price", flin.Lte, 100)</td>
                            </tr>
                            <tr className="border-b border-white/5">
                                <td className="py-2 px-4 font-mono text-purple-400">flin.In</td>
                                <td className="py-2 px-4">In array</td>
                                <td className="py-2 px-4 font-mono text-sm">{`Where("role", flin.In, []string{"admin", "mod"})`}</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>

            <div>
                <h2 className="text-2xl font-bold mb-4">Features</h2>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="bg-[#13131a] p-4 rounded-lg border border-purple-500/20">
                        <h3 className="font-semibold text-purple-400 mb-2">📊 Secondary Indexes</h3>
                        <p className="text-sm text-gray-400">Fast queries with automatic indexing</p>
                    </div>
                    <div className="bg-[#13131a] p-4 rounded-lg border border-cyan-500/20">
                        <h3 className="font-semibold text-cyan-400 mb-2">🔄 ACID Transactions</h3>
                        <p className="text-sm text-gray-400">Guaranteed consistency via BadgerDB</p>
                    </div>
                    <div className="bg-[#13131a] p-4 rounded-lg border border-purple-500/20">
                        <h3 className="font-semibold text-purple-400 mb-2">📝 Flexible Schema</h3>
                        <p className="text-sm text-gray-400">JSON documents with dynamic fields</p>
                    </div>
                    <div className="bg-[#13131a] p-4 rounded-lg border border-cyan-500/20">
                        <h3 className="font-semibold text-cyan-400 mb-2">⚡ High Performance</h3>
                        <p className="text-sm text-gray-400">76K inserts/sec, 13μs latency</p>
                    </div>
                </div>
            </div>

            <div className="bg-purple-500/10 border border-purple-500/20 rounded-lg p-6">
                <h3 className="text-lg font-semibold mb-2">💡 Best Practices</h3>
                <ul className="space-y-2 text-gray-300 text-sm">
                    <li>• Use meaningful collection names to organize your data</li>
                    <li>• Create indexes on frequently queried fields for better performance</li>
                    <li>• Use the fluent query builder for complex queries</li>
                    <li>• Combine Where() clauses for AND conditions</li>
                    <li>• Use Skip() and Take() for pagination</li>
                </ul>
            </div>
        </div>
    );
}
