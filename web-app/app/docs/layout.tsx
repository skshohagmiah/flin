'use client';

import Sidebar from '@/components/Sidebar';
import Footer from '@/components/Footer';
import { useState } from 'react';
import { Menu, X } from 'lucide-react';

export default function DocsLayout({
    children,
}: {
    children: React.ReactNode;
}) {
    const [isSidebarOpen, setIsSidebarOpen] = useState(false);

    return (
        <div className="min-h-screen pt-16">
            {/* Mobile Sidebar Toggle */}
            <button
                onClick={() => setIsSidebarOpen(!isSidebarOpen)}
                className="md:hidden fixed bottom-6 right-6 z-50 w-14 h-14 bg-purple-500 rounded-full flex items-center justify-center shadow-lg hover:bg-purple-600 transition-colors"
                aria-label="Toggle sidebar"
            >
                {isSidebarOpen ? <X className="w-6 h-6 text-white" /> : <Menu className="w-6 h-6 text-white" />}
            </button>

            {/* Mobile Sidebar Overlay */}
            {isSidebarOpen && (
                <div
                    className="md:hidden fixed inset-0 bg-black/50 z-40 pt-16"
                    onClick={() => setIsSidebarOpen(false)}
                >
                    <div
                        className="w-80 max-w-[85vw] h-full bg-[#13131a] p-6 overflow-y-auto"
                        onClick={(e) => e.stopPropagation()}
                    >
                        <Sidebar onLinkClick={() => setIsSidebarOpen(false)} />
                    </div>
                </div>
            )}

            <div className="container-custom py-6 md:py-8 lg:py-12">
                <div className="flex flex-col md:flex-row gap-6 md:gap-8 lg:gap-12">
                    {/* Desktop Sidebar */}
                    <aside className="hidden md:block w-64 flex-shrink-0">
                        <div className="sticky top-24">
                            <Sidebar />
                        </div>
                    </aside>

                    {/* Main Content */}
                    <main className="flex-1 min-w-0 w-full px-4 md:px-0">
                        {children}
                    </main>
                </div>
            </div>
            <Footer />
        </div>
    );
}
