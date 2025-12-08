'use client';

import { motion } from 'framer-motion';
import { ARCHITECTURE_LAYERS } from '@/lib/constants';

export default function Architecture() {
    return (
        <section className="section-padding bg-gradient-to-b from-transparent to-[#13131a]/30">
            <div className="container-custom">
                <div className="text-center mb-20">
                    <motion.h2
                        initial={{ opacity: 0, y: 20 }}
                        whileInView={{ opacity: 1, y: 0 }}
                        viewport={{ once: true }}
                        className="mb-6"
                    >
                        Modular <span className="text-purple-400">Architecture</span>
                    </motion.h2>
                    <motion.p
                        initial={{ opacity: 0, y: 20 }}
                        whileInView={{ opacity: 1, y: 0 }}
                        viewport={{ once: true }}
                        transition={{ delay: 0.1 }}
                        className="text-xl text-gray-400 max-w-3xl mx-auto leading-relaxed"
                    >
                        Layered design for optimal performance and scalability
                    </motion.p>
                </div>

                {/* Architecture Layers */}
                <div className="max-w-4xl mx-auto">
                    <div className="space-y-6">
                        {ARCHITECTURE_LAYERS.map((layer, index) => (
                            <motion.div
                                key={layer.name}
                                initial={{ opacity: 0, x: -20 }}
                                whileInView={{ opacity: 1, x: 0 }}
                                viewport={{ once: true }}
                                transition={{ delay: index * 0.1 }}
                                className="glass glass-hover p-8 rounded-2xl"
                            >
                                <div className="flex items-start gap-6">
                                    <div className="flex-shrink-0">
                                        <div className={`w-16 h-16 rounded-xl flex items-center justify-center text-2xl font-bold ${layer.color === 'cyan'
                                                ? 'bg-cyan-500/20 text-cyan-400 border border-cyan-500/30'
                                                : 'bg-purple-500/20 text-purple-400 border border-purple-500/30'
                                            }`}>
                                            {index + 1}
                                        </div>
                                    </div>
                                    <div className="flex-1">
                                        <h3 className="text-2xl font-bold mb-3">{layer.name}</h3>
                                        <p className="text-gray-400 text-lg leading-relaxed mb-4">{layer.description}</p>
                                        <div className="flex flex-wrap gap-2">
                                            {layer.tech.map((tech) => (
                                                <span
                                                    key={tech}
                                                    className="px-4 py-2 bg-white/5 rounded-lg text-sm text-gray-300 border border-white/10"
                                                >
                                                    {tech}
                                                </span>
                                            ))}
                                        </div>
                                    </div>
                                </div>
                            </motion.div>
                        ))}
                    </div>
                </div>

                {/* Architecture Highlights */}
                <motion.div
                    initial={{ opacity: 0, y: 20 }}
                    whileInView={{ opacity: 1, y: 0 }}
                    viewport={{ once: true }}
                    className="mt-20 grid grid-cols-1 md:grid-cols-3 gap-8"
                >
                    <div className="text-center p-8 glass rounded-2xl">
                        <div className="text-4xl font-bold text-purple-400 mb-3">Hybrid</div>
                        <p className="text-gray-400">Memory + Disk storage for optimal performance</p>
                    </div>
                    <div className="text-center p-8 glass rounded-2xl">
                        <div className="text-4xl font-bold text-purple-400 mb-3">Unified</div>
                        <p className="text-gray-400">Single port for all operations</p>
                    </div>
                    <div className="text-center p-8 glass rounded-2xl">
                        <div className="text-4xl font-bold text-purple-400 mb-3">Raft</div>
                        <p className="text-gray-400">Distributed consensus for reliability</p>
                    </div>
                </motion.div>
            </div>
        </section>
    );
}
