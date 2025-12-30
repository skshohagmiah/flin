'use client';

import { motion } from 'framer-motion';
import { PERFORMANCE_METRICS } from '@/lib/constants';
import { TrendingUp, Zap, Clock, Database, MessageSquare, Workflow } from 'lucide-react';

export default function Performance() {
    return (
        <section className="section-padding">
            <div className="container-custom">
                <div className="text-center mb-12 md:mb-20 px-4">
                    <motion.h2
                        initial={{ opacity: 0, y: 20 }}
                        whileInView={{ opacity: 1, y: 0 }}
                        viewport={{ once: true }}
                        className="mb-4 md:mb-6 text-3xl sm:text-4xl md:text-5xl font-bold"
                    >
                        Blazing Fast <span className="text-purple-400">Performance</span>
                    </motion.h2>
                    <motion.p
                        initial={{ opacity: 0, y: 20 }}
                        whileInView={{ opacity: 1, y: 0 }}
                        viewport={{ once: true }}
                        transition={{ delay: 0.1 }}
                        className="text-base sm:text-lg md:text-xl text-gray-400 max-w-3xl mx-auto leading-relaxed"
                    >
                        Real benchmarks against Redis. Flin delivers exceptional performance across all operations.
                    </motion.p>
                </div>

                {/* Performance Grid */}
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 md:gap-6 mb-8 md:mb-12">
                    {/* KV Store - Read */}
                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        whileInView={{ opacity: 1, y: 0 }}
                        viewport={{ once: true }}
                        className="glass glass-hover p-6 md:p-8 rounded-2xl"
                    >
                        <div className="flex items-center gap-3 md:gap-4 mb-4 md:mb-6">
                            <div className="w-10 h-10 md:w-12 md:h-12 rounded-xl bg-cyan-500/20 flex items-center justify-center border border-cyan-500/30">
                                <Database className="w-5 h-5 md:w-6 md:h-6 text-cyan-400" />
                            </div>
                            <div>
                                <h3 className="text-base md:text-lg font-semibold">KV Read</h3>
                                <p className="text-xs md:text-sm text-gray-400">Key-Value Store</p>
                            </div>
                        </div>
                        <div className="space-y-3">
                            <div>
                                <div className="text-3xl md:text-4xl font-bold text-cyan-400 mb-1">
                                    {PERFORMANCE_METRICS.kv.read.throughput}
                                </div>
                                <div className="text-xs md:text-sm text-gray-400">ops/sec</div>
                            </div>
                            <div className="flex items-center justify-between pt-3 border-t border-white/10">
                                <span className="text-xs md:text-sm text-gray-400">Latency: {PERFORMANCE_METRICS.kv.read.latency}</span>
                                <span className="text-xs md:text-sm font-semibold text-green-400">{PERFORMANCE_METRICS.kv.read.speedup} faster</span>
                            </div>
                        </div>
                    </motion.div>

                    {/* KV Store - Write */}
                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        whileInView={{ opacity: 1, y: 0 }}
                        viewport={{ once: true }}
                        transition={{ delay: 0.1 }}
                        className="glass glass-hover p-6 md:p-8 rounded-2xl"
                    >
                        <div className="flex items-center gap-3 md:gap-4 mb-4 md:mb-6">
                            <div className="w-10 h-10 md:w-12 md:h-12 rounded-xl bg-purple-500/20 flex items-center justify-center border border-purple-500/30">
                                <Zap className="w-5 h-5 md:w-6 md:h-6 text-purple-400" />
                            </div>
                            <div>
                                <h3 className="text-base md:text-lg font-semibold">KV Write</h3>
                                <p className="text-xs md:text-sm text-gray-400">Key-Value Store</p>
                            </div>
                        </div>
                        <div className="space-y-3">
                            <div>
                                <div className="text-3xl md:text-4xl font-bold text-purple-400 mb-1">
                                    {PERFORMANCE_METRICS.kv.write.throughput}
                                </div>
                                <div className="text-xs md:text-sm text-gray-400">ops/sec</div>
                            </div>
                            <div className="flex items-center justify-between pt-3 border-t border-white/10">
                                <span className="text-xs md:text-sm text-gray-400">Latency: {PERFORMANCE_METRICS.kv.write.latency}</span>
                                <span className="text-xs md:text-sm font-semibold text-green-400">{PERFORMANCE_METRICS.kv.write.speedup} faster</span>
                            </div>
                        </div>
                    </motion.div>

                    {/* KV Store - Batch */}
                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        whileInView={{ opacity: 1, y: 0 }}
                        viewport={{ once: true }}
                        transition={{ delay: 0.2 }}
                        className="glass glass-hover p-6 md:p-8 rounded-2xl"
                    >
                        <div className="flex items-center gap-3 md:gap-4 mb-4 md:mb-6">
                            <div className="w-10 h-10 md:w-12 md:h-12 rounded-xl bg-cyan-500/20 flex items-center justify-center border border-cyan-500/30">
                                <TrendingUp className="w-5 h-5 md:w-6 md:h-6 text-cyan-400" />
                            </div>
                            <div>
                                <h3 className="text-base md:text-lg font-semibold">KV Batch</h3>
                                <p className="text-xs md:text-sm text-gray-400">Batch Operations</p>
                            </div>
                        </div>
                        <div className="space-y-3">
                            <div>
                                <div className="text-3xl md:text-4xl font-bold text-cyan-400 mb-1">
                                    {PERFORMANCE_METRICS.kv.batch.throughput}
                                </div>
                                <div className="text-xs md:text-sm text-gray-400">ops/sec</div>
                            </div>
                            <div className="flex items-center justify-between pt-3 border-t border-white/10">
                                <span className="text-xs md:text-sm text-gray-400">Latency: {PERFORMANCE_METRICS.kv.batch.latency}</span>
                                <span className="text-xs md:text-sm font-semibold text-green-400">{PERFORMANCE_METRICS.kv.batch.speedup} faster</span>
                            </div>
                        </div>
                    </motion.div>

                    {/* Queue - Push */}
                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        whileInView={{ opacity: 1, y: 0 }}
                        viewport={{ once: true }}
                        transition={{ delay: 0.3 }}
                        className="glass glass-hover p-6 md:p-8 rounded-2xl"
                    >
                        <div className="flex items-center gap-3 md:gap-4 mb-4 md:mb-6">
                            <div className="w-10 h-10 md:w-12 md:h-12 rounded-xl bg-purple-500/20 flex items-center justify-center border border-purple-500/30">
                                <MessageSquare className="w-5 h-5 md:w-6 md:h-6 text-purple-400" />
                            </div>
                            <div>
                                <h3 className="text-base md:text-lg font-semibold">Queue Push</h3>
                                <p className="text-xs md:text-sm text-gray-400">Message Queue</p>
                            </div>
                        </div>
                        <div className="space-y-3">
                            <div>
                                <div className="text-3xl md:text-4xl font-bold text-purple-400 mb-1">
                                    {PERFORMANCE_METRICS.queue.push.throughput}
                                </div>
                                <div className="text-xs md:text-sm text-gray-400">ops/sec</div>
                            </div>
                            <div className="flex items-center justify-between pt-3 border-t border-white/10">
                                <span className="text-xs md:text-sm text-gray-400">Latency: {PERFORMANCE_METRICS.queue.push.latency}</span>
                                <span className="text-xs md:text-sm font-semibold text-green-400">{PERFORMANCE_METRICS.queue.push.speedup} faster</span>
                            </div>
                        </div>
                    </motion.div>

                    {/* Queue - Pop */}
                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        whileInView={{ opacity: 1, y: 0 }}
                        viewport={{ once: true }}
                        transition={{ delay: 0.4 }}
                        className="glass glass-hover p-6 md:p-8 rounded-2xl"
                    >
                        <div className="flex items-center gap-3 md:gap-4 mb-4 md:mb-6">
                            <div className="w-10 h-10 md:w-12 md:h-12 rounded-xl bg-cyan-500/20 flex items-center justify-center border border-cyan-500/30">
                                <Clock className="w-5 h-5 md:w-6 md:h-6 text-cyan-400" />
                            </div>
                            <div>
                                <h3 className="text-base md:text-lg font-semibold">Queue Pop</h3>
                                <p className="text-xs md:text-sm text-gray-400">Message Queue</p>
                            </div>
                        </div>
                        <div className="space-y-3">
                            <div>
                                <div className="text-3xl md:text-4xl font-bold text-cyan-400 mb-1">
                                    {PERFORMANCE_METRICS.queue.pop.throughput}
                                </div>
                                <div className="text-xs md:text-sm text-gray-400">ops/sec</div>
                            </div>
                            <div className="flex items-center justify-between pt-3 border-t border-white/10">
                                <span className="text-xs md:text-sm text-gray-400">Latency: {PERFORMANCE_METRICS.queue.pop.latency}</span>
                                <span className="text-xs md:text-sm font-semibold text-green-400">{PERFORMANCE_METRICS.queue.pop.speedup} faster</span>
                            </div>
                        </div>
                    </motion.div>

                    {/* Database - Insert */}
                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        whileInView={{ opacity: 1, y: 0 }}
                        viewport={{ once: true }}
                        transition={{ delay: 0.5 }}
                        className="glass glass-hover p-6 md:p-8 rounded-2xl"
                    >
                        <div className="flex items-center gap-3 md:gap-4 mb-4 md:mb-6">
                            <div className="w-10 h-10 md:w-12 md:h-12 rounded-xl bg-purple-500/20 flex items-center justify-center border border-purple-500/30">
                                <Workflow className="w-5 h-5 md:w-6 md:h-6 text-purple-400" />
                            </div>
                            <div>
                                <h3 className="text-base md:text-lg font-semibold">DB Insert</h3>
                                <p className="text-xs md:text-sm text-gray-400">Document Database</p>
                            </div>
                        </div>
                        <div className="space-y-3">
                            <div>
                                <div className="text-3xl md:text-4xl font-bold text-purple-400 mb-1">
                                    {PERFORMANCE_METRICS.db.insert.throughput}
                                </div>
                                <div className="text-xs md:text-sm text-gray-400">ops/sec</div>
                            </div>
                            <div className="flex items-center justify-between pt-3 border-t border-white/10">
                                <span className="text-xs md:text-sm text-gray-400">Latency: {PERFORMANCE_METRICS.db.insert.latency}</span>
                                <span className="text-xs md:text-sm font-semibold text-gray-500">Baseline</span>
                            </div>
                        </div>
                    </motion.div>
                </div>

                {/* Summary Stats */}
                <motion.div
                    initial={{ opacity: 0, y: 20 }}
                    whileInView={{ opacity: 1, y: 0 }}
                    viewport={{ once: true }}
                    className="glass p-6 md:p-8 rounded-2xl text-center"
                >
                    <p className="text-sm md:text-base lg:text-lg text-gray-300 mb-3 md:mb-4">
                        Benchmarked on <span className="text-purple-400 font-semibold">AWS EC2 c5.2xlarge</span> instances with <span className="text-purple-400 font-semibold">8 vCPUs</span> and <span className="text-purple-400 font-semibold">16GB RAM</span>
                    </p>
                    <p className="text-xs md:text-sm text-gray-400">
                        All tests performed with 256 concurrent workers. Results represent average throughput over 60-second runs.
                    </p>
                </motion.div>
            </div>
        </section>
    );
}
