'use client';

import { motion } from 'framer-motion';
import { ArrowRight, Zap } from 'lucide-react';
import Link from 'next/link';
import { useState } from 'react';
import CodeBlock from './CodeBlock';

export default function Hero() {
    const [activeTab, setActiveTab] = useState<'go' | 'nodejs' | 'python'>('go');

    const codeExamples = {
        go: `import flin "github.com/skshohagmiah/flin/clients/go"

client, _ := flin.NewClient(opts)

// KV Store
client.KV.Set("user:1", []byte("data"))

// Message Queue  
client.Queue.Push("tasks", []byte("job"))

// Stream Processing
client.Stream.Publish("events", -1, "key", data)

// Document Database
client.DB.Insert("users", document)`,
        nodejs: `const { FlinClient } = require('@flin/client');

const client = new FlinClient(options);

// KV Store
await client.kv.set('user:1', 'data');

// Message Queue
await client.queue.push('tasks', 'job');

// Stream Processing
await client.stream.publish('events', 'key', data);

// Document Database
await client.db.insert('users', document);`,
        python: `from flin import FlinClient

client = FlinClient(options)

# KV Store
client.kv.set('user:1', b'data')

# Message Queue
client.queue.push('tasks', b'job')

# Stream Processing
client.stream.publish('events', 'key', data)

# Document Database
client.db.insert('users', document)`
    };

    return (
        <section className="relative min-h-screen flex items-center justify-center overflow-hidden py-32 md:py-40">
            {/* Animated Background */}
            <div className="absolute inset-0 -z-10">
                <div className="absolute inset-0 bg-gradient-to-br from-cyan-500/10 via-purple-500/10 to-transparent" />
                <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-cyan-500/20 rounded-full blur-3xl animate-float" />
                <div className="absolute bottom-1/4 right-1/4 w-96 h-96 bg-purple-500/20 rounded-full blur-3xl animate-float" style={{ animationDelay: '2s' }} />
            </div>

            <div className="container-custom">
                <div className="max-w-5xl mx-auto text-center">
                    {/* Badge */}
                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ duration: 0.5 }}
                        className="inline-flex items-center gap-2 px-4 py-2 rounded-full glass mb-8"
                    >
                        <Zap className="w-4 h-4 text-cyan-400" />
                        <span className="text-sm text-gray-300">
                            High-Performance Distributed Data Platform
                        </span>
                    </motion.div>

                    {/* Main Heading */}
                    <motion.h1
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ duration: 0.5, delay: 0.1 }}
                        className="text-3xl sm:text-4xl md:text-5xl lg:text-6xl font-bold mb-4 md:mb-6 leading-tight px-4"
                    >
                        <span className="text-purple-400">Flin</span> - Distributed Data Platform
                    </motion.h1>

                    {/* Subtitle */}
                    <motion.p
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ duration: 0.5, delay: 0.2 }}
                        className="text-base sm:text-lg md:text-xl lg:text-2xl text-gray-400 mb-6 md:mb-8 max-w-3xl mx-auto px-4 leading-relaxed"
                    >
                        Unified platform with KV Store, Message Queue, Stream Processing, and Document Database. Built for distributed systems with fault tolerance and high availability.
                    </motion.p>

                    {/* Open Source Notice */}
                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ duration: 0.5, delay: 0.25 }}
                        className="mb-12 max-w-2xl mx-auto"
                    >
                        <div className="glass px-6 py-3 rounded-xl border-l-4 border-purple-500">
                            <p className="text-sm text-gray-300">
                                🚀 Open source and actively developed. Contributions welcome!
                            </p>
                        </div>
                    </motion.div>

                    {/* CTA Buttons */}
                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ duration: 0.5, delay: 0.3 }}
                        className="flex flex-col sm:flex-row items-stretch sm:items-center justify-center gap-3 md:gap-4 mb-12 md:mb-16 px-4 max-w-md sm:max-w-none mx-auto"
                    >
                        <Link href="/docs/getting-started" className="btn btn-primary text-base md:text-lg px-6 md:px-8 py-3 md:py-4 w-full sm:w-auto">
                            Get Started
                            <ArrowRight className="w-4 h-4 md:w-5 md:h-5" />
                        </Link>
                        <Link href="/docs" className="btn btn-secondary text-base md:text-lg px-6 md:px-8 py-3 md:py-4 w-full sm:w-auto">
                            View Documentation
                        </Link>
                    </motion.div>

                    {/* Code Examples with Tabs */}
                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ duration: 0.5, delay: 0.4 }}
                        className="max-w-3xl mx-auto"
                    >
                        <div className="glass rounded-xl overflow-hidden">
                            {/* Tabs */}
                            <div className="flex border-b border-white/10">
                                <button
                                    onClick={() => setActiveTab('go')}
                                    className={`flex-1 px-6 py-3 text-sm font-medium transition-colors ${activeTab === 'go'
                                        ? 'bg-purple-500/20 text-purple-400 border-b-2 border-purple-500'
                                        : 'text-gray-400 hover:text-gray-300'
                                        }`}
                                >
                                    Go
                                </button>
                                <button
                                    onClick={() => setActiveTab('nodejs')}
                                    className={`flex-1 px-6 py-3 text-sm font-medium transition-colors ${activeTab === 'nodejs'
                                        ? 'bg-purple-500/20 text-purple-400 border-b-2 border-purple-500'
                                        : 'text-gray-400 hover:text-gray-300'
                                        }`}
                                >
                                    Node.js
                                </button>
                                <button
                                    onClick={() => setActiveTab('python')}
                                    className={`flex-1 px-6 py-3 text-sm font-medium transition-colors ${activeTab === 'python'
                                        ? 'bg-purple-500/20 text-purple-400 border-b-2 border-purple-500'
                                        : 'text-gray-400 hover:text-gray-300'
                                        }`}
                                >
                                    Python
                                </button>
                            </div>

                            {/* Code Content */}
                            <div className="text-left">
                                {activeTab === 'go' && (
                                    <CodeBlock code={codeExamples.go} language="go" />
                                )}
                                {activeTab === 'nodejs' && (
                                    <CodeBlock code={codeExamples.nodejs} language="javascript" />
                                )}
                                {activeTab === 'python' && (
                                    <CodeBlock code={codeExamples.python} language="python" />
                                )}
                            </div>
                        </div>
                    </motion.div>
                </div>
            </div>
        </section>
    );
}
