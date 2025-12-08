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
            <div className="flex items-center justify-between px-4 py-2 border-b border-white/10 bg-[#1e1e1e]">
                <span className="text-xs text-gray-400 font-mono uppercase tracking-wide">
                    {language}
                </span>
                <button
                    onClick={handleCopy}
                    className="flex items-center gap-2 text-xs text-gray-400 hover:text-white transition-colors"
                    aria-label="Copy code"
                >
                    {copied ? (
                        <>
                            <Check className="w-4 h-4" />
                            Copied!
                        </>
                    ) : (
                        <>
                            <Copy className="w-4 h-4" />
                            Copy
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
                    padding: '1.5rem',
                    background: '#1e1e1e',
                    fontSize: '0.875rem',
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
