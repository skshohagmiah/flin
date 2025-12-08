'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { DOC_SECTIONS } from '@/lib/constants';
import { ChevronDown } from 'lucide-react';
import { useState } from 'react';

export default function Sidebar() {
    const pathname = usePathname();
    const [openSections, setOpenSections] = useState<string[]>(
        DOC_SECTIONS.map(s => s.title)
    );

    const toggleSection = (title: string) => {
        setOpenSections(prev =>
            prev.includes(title)
                ? prev.filter(t => t !== title)
                : [...prev, title]
        );
    };

    return (
        <nav className="space-y-6">
            {DOC_SECTIONS.map((section) => (
                <div key={section.title}>
                    <button
                        onClick={() => toggleSection(section.title)}
                        className="flex items-center justify-between w-full text-sm font-semibold text-gray-300 hover:text-white transition-colors mb-3"
                    >
                        {section.title}
                        <ChevronDown
                            className={`w-4 h-4 transition-transform ${openSections.includes(section.title) ? 'rotate-180' : ''
                                }`}
                        />
                    </button>

                    {openSections.includes(section.title) && (
                        <ul className="space-y-2 ml-2">
                            {section.items.map((item) => {
                                const isActive = pathname === item.href;
                                return (
                                    <li key={item.href}>
                                        <Link
                                            href={item.href}
                                            className={`block text-sm py-1.5 px-3 rounded transition-colors ${isActive
                                                ? 'bg-purple-500/10 text-purple-400 font-medium'
                                                : 'text-gray-400 hover:text-white hover:bg-white/5'
                                                }`}
                                        >
                                            {item.label}
                                        </Link>
                                    </li>
                                );
                            })}
                        </ul>
                    )}
                </div>
            ))}
        </nav>
    );
}
