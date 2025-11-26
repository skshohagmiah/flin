"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const navItems = [
  { 
    href: "/", 
    label: "Overview", 
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 12l2-3m0 0l7-4 7 4M5 9v10a1 1 0 001 1h12a1 1 0 001-1V9m-9 4l4 2m-8-2l4-2" />
      </svg>
    ) 
  },
  { 
    href: "/kv", 
    label: "KV Store", 
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    )
  },
  { 
    href: "/queues", 
    label: "Queues", 
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
      </svg>
    ),
    badge: "soon" 
  },
  { 
    href: "/streams", 
    label: "Streams", 
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
      </svg>
    ),
    badge: "soon" 
  },
  { 
    href: "/pubsub", 
    label: "Pub/Sub", 
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    ),
    badge: "soon" 
  },
  { 
    href: "/db", 
    label: "Database", 
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7m0 0c0 2.21-3.582 4-8 4s-8-1.79-8-4m0 0c0-2.21 3.582-4 8-4s8 1.79 8 4m0 6c0 2.21-3.582 4-8 4s-8-1.79-8-4" />
      </svg>
    ),
    badge: "soon" 
  },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="hidden md:flex md:w-[260px] flex-col border-r border-border bg-gradient-to-b from-background to-background/95">
      {/* Logo Section - Minimal and Clean */}
      <div className="px-4 py-6 flex items-center gap-2.5">
        <div className="flex items-center justify-center h-8 w-8 rounded-lg bg-gradient-to-br from-blue-500 to-blue-600 text-white font-bold text-sm">
          F
        </div>
        <div className="flex flex-col">
          <span className="text-sm font-semibold text-foreground">Flin</span>
          <span className="text-xs text-muted-foreground font-light">Studio</span>
        </div>
      </div>

      {/* Navigation */}
      <nav className="flex-1 px-3 py-4 space-y-0.5 overflow-y-auto">
        <div className="text-xs font-semibold uppercase tracking-widest text-muted-foreground px-2 py-2 mb-2">
          Menu
        </div>
        {navItems.map((item) => {
          const isActive =
            pathname === item.href ||
            (item.href !== "/" && pathname.startsWith(item.href));

          return (
            <Link
              key={item.href}
              href={item.href}
              className={`flex items-center justify-between gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-all duration-200 group ${
                isActive
                  ? "bg-blue-500/15 text-blue-600 dark:text-blue-400 shadow-sm"
                  : "text-muted-foreground hover:text-foreground hover:bg-muted/50"
              }`}
            >
              <span className="flex items-center gap-3 flex-1">
                <span className={`${isActive ? "text-blue-600 dark:text-blue-400" : "text-muted-foreground group-hover:text-foreground"} transition-colors`}>
                  {item.icon}
                </span>
                <span>{item.label}</span>
              </span>
              {item.badge ? (
                <span className={`text-[10px] font-semibold uppercase tracking-wide rounded px-1.5 py-0.5 ${
                  isActive
                    ? "bg-blue-500/25 text-blue-700 dark:text-blue-300"
                    : "bg-muted text-muted-foreground"
                }`}>
                  {item.badge}
                </span>
              ) : null}
            </Link>
          );
        })}
      </nav>

      {/* Cluster Status - Sleek Card */}
      <div className="px-3 pb-4 pt-3 border-t border-border/50">
        <div className="rounded-lg bg-gradient-to-br from-emerald-500/10 to-emerald-600/5 border border-emerald-500/20 p-3 space-y-2">
          <div className="flex items-center gap-2">
            <span className="h-2 w-2 rounded-full bg-emerald-500 animate-pulse" />
            <span className="text-xs font-semibold text-foreground">Cluster Status</span>
          </div>
          <div className="text-xs text-muted-foreground space-y-1">
            <div className="flex items-center justify-between">
              <span>Node:</span>
              <span className="font-mono font-semibold text-emerald-600 dark:text-emerald-400">dev-local</span>
            </div>
            <div className="flex items-center justify-between">
              <span>Health:</span>
              <span className="font-semibold text-emerald-600 dark:text-emerald-400">Healthy</span>
            </div>
          </div>
        </div>
      </div>
    </aside>
  );
}
