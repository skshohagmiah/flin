"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";
import { Menu, X } from "lucide-react";

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
  const [isMobileOpen, setIsMobileOpen] = useState(false);

  const SidebarContent = () => (
    <>
      {/* Logo Section */}
      <div className="px-4 py-6 flex items-center gap-2.5">
        <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center font-bold text-primary-foreground">
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
              onClick={() => setIsMobileOpen(false)}
              className={`flex items-center justify-between gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-all duration-200 group ${isActive
                ? "bg-primary/10 text-primary shadow-sm"
                : "text-muted-foreground hover:text-foreground hover:bg-accent"
                }`}
            >
              <span className="flex items-center gap-3 flex-1">
                <span className={`${isActive ? "text-primary" : "text-muted-foreground group-hover:text-foreground"} transition-colors`}>
                  {item.icon}
                </span>
                <span>{item.label}</span>
              </span>
              {item.badge ? (
                <span className={`text-[10px] font-semibold uppercase tracking-wide rounded px-1.5 py-0.5 ${isActive
                  ? "bg-primary/20 text-primary"
                  : "bg-muted text-muted-foreground"
                  }`}>
                  {item.badge}
                </span>
              ) : null}
            </Link>
          );
        })}
      </nav>

      {/* Cluster Status */}
      <div className="px-3 pb-4 pt-3 border-t border-border">
        <div className="rounded-lg glass p-3 space-y-2">
          <div className="flex items-center gap-2">
            <span className="h-2 w-2 rounded-full bg-emerald-500 animate-pulse-glow" />
            <span className="text-xs font-semibold text-foreground">Cluster Status</span>
          </div>
          <div className="text-xs text-muted-foreground space-y-1">
            <div className="flex items-center justify-between">
              <span>Node:</span>
              <span className="font-mono font-semibold text-emerald-500">dev-local</span>
            </div>
            <div className="flex items-center justify-between">
              <span>Health:</span>
              <span className="font-semibold text-emerald-500">Healthy</span>
            </div>
          </div>
        </div>
      </div>
    </>
  );

  return (
    <>
      {/* Desktop Sidebar */}
      <aside className="hidden md:flex md:w-[260px] flex-col border-r border-border bg-card">
        <SidebarContent />
      </aside>

      {/* Mobile Menu Button */}
      <button
        onClick={() => setIsMobileOpen(!isMobileOpen)}
        className="md:hidden fixed bottom-6 right-6 z-50 w-14 h-14 bg-primary rounded-full flex items-center justify-center shadow-lg hover:bg-primary/90 transition-colors"
        aria-label="Toggle sidebar"
      >
        {isMobileOpen ? <X className="w-6 h-6 text-primary-foreground" /> : <Menu className="w-6 h-6 text-primary-foreground" />}
      </button>

      {/* Mobile Sidebar Overlay */}
      {isMobileOpen && (
        <div
          className="md:hidden fixed inset-0 bg-background/80 backdrop-blur-sm z-40"
          onClick={() => setIsMobileOpen(false)}
        >
          <div
            className="w-80 max-w-[85vw] h-full bg-card border-r border-border flex flex-col"
            onClick={(e) => e.stopPropagation()}
          >
            <SidebarContent />
          </div>
        </div>
      )}
    </>
  );
}
