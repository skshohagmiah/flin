'use client';

import { motion } from 'framer-motion';
import { FEATURES } from '@/lib/constants';

export default function Features() {
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
                        Four Powerful <span className="text-purple-400">Engines</span>
                    </motion.h2>
                    <motion.p
                        initial={{ opacity: 0, y: 20 }}
                        whileInView={{ opacity: 1, y: 0 }}
                        viewport={{ once: true }}
                        transition={{ delay: 0.1 }}
                        className="text-xl text-gray-400 max-w-3xl mx-auto leading-relaxed"
                    >
                        Everything you need in one unified platform
                    </motion.p>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
                    {FEATURES.map((feature, index) => (
                        <motion.div
                            key={feature.title}
                            initial={{ opacity: 0, y: 20 }}
                            whileInView={{ opacity: 1, y: 0 }}
                            viewport={{ once: true }}
                            transition={{ delay: index * 0.1 }}
                            className="glass glass-hover p-10 rounded-2xl group"
                        >
                            <div className="flex items-start gap-6">
                                <div className="flex-shrink-0">
                                    <div className="w-16 h-16 rounded-xl bg-purple-500/20 flex items-center justify-center text-purple-400 border border-purple-500/30 group-hover:bg-purple-500/30 transition-all">
                                        {typeof feature.icon === 'string' ? (
                                            <span className="text-3xl">{feature.icon}</span>
                                        ) : (
                                            <feature.icon className="w-8 h-8" />
                                        )}
                                    </div>
                                </div>
                                <div className="flex-1">
                                    <h3 className="text-2xl font-bold mb-4">{feature.title}</h3>
                                    <p className="text-gray-400 text-lg leading-relaxed mb-6">{feature.description}</p>
                                    <ul className="space-y-3">
                                        {feature.highlights.map((highlight) => (
                                            <li key={highlight} className="flex items-start gap-3 text-gray-300">
                                                <svg className="w-5 h-5 text-purple-400 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                                                    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                                                </svg>
                                                <span>{highlight}</span>
                                            </li>
                                        ))}
                                    </ul>
                                </div>
                            </div>
                        </motion.div>
                    ))}
                </div>
            </div>
        </section>
    );
}
