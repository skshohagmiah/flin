"use client";

import Link from "next/link";
import { ThemeSwitcher } from "./theme-switcher";

export function Topbar() {
  return (
    <header className="sticky top-0 z-20 border-b border-border/50 bg-gradient-to-b from-background/95 to-background/80 backdrop-blur-xl supports-[backdrop-filter]:bg-background/80">
      <div className="flex items-center justify-between px-6 py-3.5 gap-4">
        {/* Left Section - Status */}
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2 px-3 py-1.5 bg-muted/40 rounded-lg border border-border/50 hover:bg-muted/60 transition-colors">
            <span className="text-xs font-medium text-muted-foreground">Environment</span>
            <span className="inline-block w-1.5 h-1.5 rounded-full bg-blue-500"></span>
            <span className="text-xs font-semibold text-foreground tracking-wide">ADMIN</span>
          </div>
          <div className="hidden sm:flex items-center gap-2 text-xs text-muted-foreground">
            <span className="font-medium">Flin Data Platform</span>
          </div>
        </div>

        {/* Right Section - Actions */}
        <div className="flex items-center gap-3 ml-auto">
          {/* Connection Status */}
          <div className="flex items-center gap-2 px-3 py-1.5 bg-emerald-500/10 rounded-lg border border-emerald-500/30 hover:border-emerald-500/50 transition-colors">
            <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
            <span className="text-xs font-medium text-emerald-700 dark:text-emerald-400">Connected</span>
          </div>

          {/* Divider */}
          <div className="h-4 w-px bg-border/30"></div>

          {/* Theme Switcher */}
          <ThemeSwitcher />

          {/* User Menu / More Options */}
          <button className="flex items-center justify-center h-8 w-8 rounded-lg hover:bg-muted/60 transition-colors text-muted-foreground hover:text-foreground group">
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
            </svg>
          </button>
        </div>
      </div>
    </header>
  );
}
