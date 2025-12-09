'use client';

import { useState } from 'react';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { Copy, Check } from 'lucide-react';

interface CodeBlockProps {
    code: string;
    language?: string;
    showLineNumbers?: boolean;
}

export default function CodeBlock({
    code,
    language = 'bash',
    showLineNumbers = false
}: CodeBlockProps) {
    const [copied, setCopied] = useState(false);

    const handleCopy = async () => {
        await navigator.clipboard.writeText(code);
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
    };

    return (
        <div className="code-block relative group">
            {/* Header */}
            <div className="flex items-center justify-between px-3 py-2 md:px-4 md:py-3 border-b border-white/10 bg-[#1e1e1e]">
                <span className="text-xs md:text-sm text-gray-400 font-mono uppercase tracking-wide">
                    {language}
                </span>
                <button
                    onClick={handleCopy}
                    className="flex items-center gap-1.5 md:gap-2 px-2 py-1 md:px-3 md:py-1.5 rounded bg-white/5 hover:bg-white/10 transition-colors text-xs md:text-sm text-gray-400 hover:text-white"
                    aria-label="Copy code"
                >
                    {copied ? (
                        <>
                            <Check className="w-3 h-3 md:w-4 md:h-4 text-green-400" />
                            <span className="text-green-400">Copied!</span>
                        </>
                    ) : (
                        <>
                            <Copy className="w-3 h-3 md:w-4 md:h-4" />
                            <span>Copy</span>
                        </>
                    )}
                </button>
            </div>

            {/* Code */}
            <SyntaxHighlighter
                language={language}
                style={vscDarkPlus}
                showLineNumbers={showLineNumbers}
                customStyle={{
                    margin: 0,
                    padding: '1rem',
                    background: '#1e1e1e',
                    fontSize: '0.75rem',
                    lineHeight: '1.7',
                }}
                codeTagProps={{
                    style: {
                        fontFamily: "'JetBrains Mono', 'Courier New', monospace",
                    },
                }}
            >
                {code}
            </SyntaxHighlighter>
        </div>
    );
}
