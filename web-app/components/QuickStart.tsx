'use client';

import { motion } from 'framer-motion';
import { useState } from 'react';
import CodeBlock from './CodeBlock';
import { CODE_EXAMPLES } from '@/lib/constants';

export default function QuickStart() {
    const [activeTab, setActiveTab] = useState<'docker' | 'local' | 'client'>('docker');

    const tabs = [
        { id: 'docker' as const, label: '🐳 Docker', code: CODE_EXAMPLES.docker, language: 'bash' },
        { id: 'local' as const, label: '💻 Local', code: CODE_EXAMPLES.local, language: 'bash' },
        { id: 'client' as const, label: '📦 Client', code: CODE_EXAMPLES.client, language: 'go' },
    ];

    return (
        <section className="section-padding">
            <div className="container-custom">
                <div className="text-center mb-16">
                    <motion.h2
                        initial={{ opacity: 0, y: 20 }}
                        whileInView={{ opacity: 1, y: 0 }}
                        viewport={{ once: true }}
                        className="text-4xl md:text-5xl font-bold mb-4"
                    >
                        Get Started in <span className="gradient-text">Seconds</span>
                    </motion.h2>
                    <motion.p
                        initial={{ opacity: 0, y: 20 }}
                        whileInView={{ opacity: 1, y: 0 }}
                        viewport={{ once: true }}
                        transition={{ delay: 0.1 }}
                        className="text-xl text-gray-400 max-w-2xl mx-auto"
                    >
                        Choose your preferred deployment method
                    </motion.p>
                </div>

                <motion.div
                    initial={{ opacity: 0, y: 20 }}
                    whileInView={{ opacity: 1, y: 0 }}
                    viewport={{ once: true }}
                    className="max-w-4xl mx-auto"
                >
                    {/* Tabs */}
                    <div className="flex flex-wrap justify-center gap-3 md:gap-4 mb-8">
                        {tabs.map((tab) => (
                            <button
                                key={tab.id}
                                onClick={() => setActiveTab(tab.id)}
                                className={`px-4 md:px-6 py-2 md:py-3 rounded-lg text-sm md:text-base font-semibold transition-all ${activeTab === tab.id
                                    ? 'bg-purple-500 text-white'
                                    : 'glass text-gray-400 hover:text-white'
                                    }`}
                            >
                                {tab.label}
                            </button>
                        ))}
                    </div>

                    {/* Code Block */}
                    <div className="animate-scale-in">
                        {tabs.map((tab) => (
                            tab.id === activeTab && (
                                <CodeBlock
                                    key={tab.id}
                                    code={tab.code}
                                    language={tab.language}
                                />
                            )
                        ))}
                    </div>

                    {/* Additional Info */}
                    <div className="mt-8 grid grid-cols-1 md:grid-cols-3 gap-4">
                        <div className="glass p-4 rounded-lg text-center">
                            <div className="text-2xl mb-2">⚡</div>
                            <div className="text-sm text-gray-400">Ready in seconds</div>
                        </div>
                        <div className="glass p-4 rounded-lg text-center">
                            <div className="text-2xl mb-2">🔧</div>
                            <div className="text-sm text-gray-400">Zero configuration</div>
                        </div>
                        <div className="glass p-4 rounded-lg text-center">
                            <div className="text-2xl mb-2">📚</div>
                            <div className="text-sm text-gray-400">Full documentation</div>
                        </div>
                    </div>
                </motion.div>
            </div>
        </section>
    );
}
