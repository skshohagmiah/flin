'use client';

import { motion } from 'framer-motion';
import { FEATURES } from '@/lib/constants';

export default function Features() {
    return (
        <section className="section-padding bg-gradient-to-b from-transparent to-[#13131a]/30">
            <div className="container-custom">
                <div className="text-center mb-12 md:mb-20 px-4">
                    <motion.h2
                        initial={{ opacity: 0, y: 20 }}
                        whileInView={{ opacity: 1, y: 0 }}
                        viewport={{ once: true }}
                        className="mb-4 md:mb-6"
                    >
                        Four Powerful <span className="text-purple-400">Engines</span>
                    </motion.h2>
                    <motion.p
                        initial={{ opacity: 0, y: 20 }}
                        whileInView={{ opacity: 1, y: 0 }}
                        viewport={{ once: true }}
                        transition={{ delay: 0.1 }}
                        className="text-base sm:text-lg md:text-xl text-gray-400 max-w-3xl mx-auto leading-relaxed"
                    >
                        Everything you need in one unified platform
                    </motion.p>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-6 md:gap-8">
                    {FEATURES.map((feature, index) => (
                        <motion.div
                            key={feature.title}
                            initial={{ opacity: 0, y: 20 }}
                            whileInView={{ opacity: 1, y: 0 }}
                            viewport={{ once: true }}
                            transition={{ delay: index * 0.1 }}
                            className="glass glass-hover p-6 md:p-10 rounded-2xl group"
                        >
                            <div className="flex items-start gap-4 md:gap-6">
                                <div className="flex-shrink-0">
                                    <div className="w-12 h-12 md:w-16 md:h-16 rounded-xl bg-purple-500/20 flex items-center justify-center text-purple-400 border border-purple-500/30 group-hover:bg-purple-500/30 transition-all">
                                        {typeof feature.icon === 'string' ? (
                                            <span className="text-2xl md:text-3xl">{feature.icon}</span>
                                        ) : (
                                            //@ts-ignore
                                            <feature.icon className="w-6 h-6 md:w-8 md:h-8" />
                                        )}
                                    </div>
                                </div>
                                <div className="flex-1">
                                    <h3 className="text-xl md:text-2xl font-bold mb-3 md:mb-4">{feature.title}</h3>
                                    <p className="text-gray-400 text-base md:text-lg leading-relaxed mb-4 md:mb-6">{feature.description}</p>
                                    <ul className="space-y-2 md:space-y-3">
                                        {feature.highlights.map((highlight) => (
                                            <li key={highlight} className="flex items-start gap-2 md:gap-3 text-gray-300 text-sm md:text-base">
                                                <svg className="w-4 h-4 md:w-5 md:h-5 text-purple-400 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
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
