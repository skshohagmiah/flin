import Link from 'next/link';
import { Github, Twitter, Mail } from 'lucide-react';

export default function Footer() {
    return (
        <footer className="border-t border-white/10 bg-[#0a0a0f] py-12">
            <div className="container-custom py-12">
                <div className="grid grid-cols-1 md:grid-cols-4 gap-8 mb-8">
                    {/* Brand */}
                    <div>
                        <div className="flex items-center space-x-2 mb-4">
                            <div className="w-8 h-8 rounded-lg bg-purple-500 flex items-center justify-center font-bold text-white">
                                F
                            </div>
                            <span className="text-xl font-bold text-purple-400">Flin</span>
                        </div>
                        <p className="text-sm text-gray-400 mb-4">
                            High-performance distributed data platform combining KV Store, Queue, Stream, and Database.
                        </p>
                        <div className="flex items-center gap-3">
                            <a
                                href="https://github.com/skshohagmiah/flin"
                                target="_blank"
                                rel="noopener noreferrer"
                                className="w-9 h-9 rounded-lg glass flex items-center justify-center hover:bg-cyan-500/20 transition-colors"
                                aria-label="GitHub"
                            >
                                <Github className="w-5 h-5" />
                            </a>
                            <a
                                href="https://twitter.com"
                                target="_blank"
                                rel="noopener noreferrer"
                                className="w-9 h-9 rounded-lg glass flex items-center justify-center hover:bg-cyan-500/20 transition-colors"
                                aria-label="Twitter"
                            >
                                <Twitter className="w-5 h-5" />
                            </a>
                            <a
                                href="mailto:contact@flin.dev"
                                className="w-9 h-9 rounded-lg glass flex items-center justify-center hover:bg-cyan-500/20 transition-colors"
                                aria-label="Email"
                            >
                                <Mail className="w-5 h-5" />
                            </a>
                        </div>
                    </div>

                    {/* Product */}
                    <div>
                        <h3 className="font-semibold text-white mb-4">Product</h3>
                        <ul className="space-y-2 text-sm text-gray-400">
                            <li>
                                <Link href="/docs/kv-store" className="hover:text-white transition-colors">
                                    Key-Value Store
                                </Link>
                            </li>
                            <li>
                                <Link href="/docs/queue" className="hover:text-white transition-colors">
                                    Message Queue
                                </Link>
                            </li>
                            <li>
                                <Link href="/docs/stream" className="hover:text-white transition-colors">
                                    Stream Processing
                                </Link>
                            </li>
                            <li>
                                <Link href="/docs/database" className="hover:text-white transition-colors">
                                    Document Database
                                </Link>
                            </li>
                        </ul>
                    </div>

                    {/* Resources */}
                    <div>
                        <h3 className="font-semibold text-white mb-4">Resources</h3>
                        <ul className="space-y-2 text-sm text-gray-400">
                            <li>
                                <Link href="/docs" className="hover:text-white transition-colors">
                                    Documentation
                                </Link>
                            </li>
                            <li>
                                <Link href="/docs/getting-started" className="hover:text-white transition-colors">
                                    Getting Started
                                </Link>
                            </li>
                            <li>
                                <Link href="/docs/clustering" className="hover:text-white transition-colors">
                                    Clustering Guide
                                </Link>
                            </li>
                            <li>
                                <a
                                    href="https://github.com/skshohagmiah/flin"
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="hover:text-white transition-colors"
                                >
                                    GitHub
                                </a>
                            </li>
                        </ul>
                    </div>

                    {/* Company */}
                    <div>
                        <h3 className="font-semibold text-white mb-4">Company</h3>
                        <ul className="space-y-2 text-sm text-gray-400">
                            <li>
                                <a href="#" className="hover:text-white transition-colors">
                                    About
                                </a>
                            </li>
                            <li>
                                <a href="#" className="hover:text-white transition-colors">
                                    Blog
                                </a>
                            </li>
                            <li>
                                <a href="#" className="hover:text-white transition-colors">
                                    Careers
                                </a>
                            </li>
                            <li>
                                <a href="#" className="hover:text-white transition-colors">
                                    Contact
                                </a>
                            </li>
                        </ul>
                    </div>
                </div>

                {/* Bottom */}
                <div className="pt-8 border-t border-white/10 flex flex-col md:flex-row items-center justify-between gap-4">
                    <p className="text-sm text-gray-400">
                        © {new Date().getFullYear()} Flin. Built with ❤️ by the Flin team.
                    </p>
                    <div className="flex items-center gap-6 text-sm text-gray-400">
                        <a href="#" className="hover:text-white transition-colors">
                            Privacy Policy
                        </a>
                        <a href="#" className="hover:text-white transition-colors">
                            Terms of Service
                        </a>
                        <a href="#" className="hover:text-white transition-colors">
                            MIT License
                        </a>
                    </div>
                </div>
            </div>
        </footer>
    );
}
