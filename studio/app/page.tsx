import Link from "next/link";
import {
  Database,
  MessageSquare,
  Radio,
  Server,
  Activity,
  ArrowRight,
  HardDrive,
  Cpu
} from "lucide-react";

export default function Home() {
  return (
    <div className="space-y-8">
      {/* Hero Section */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div className="space-y-1">
          <h1 className="text-3xl font-bold tracking-tight text-foreground">
            Cluster Overview
          </h1>
          <p className="text-muted-foreground">
            Monitor and manage your Flin cluster resources in real-time.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <span className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 dark:text-emerald-400 text-sm font-medium">
            <span className="relative flex h-2 w-2">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
            </span>
            System Healthy
          </span>
          <span className="px-3 py-1.5 rounded-full bg-primary/10 border border-primary/20 text-primary text-sm font-medium">
            v0.1.0-beta
          </span>
        </div>
      </div>

      {/* Metrics Grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <div className="p-6 rounded-xl bg-card border border-border shadow-sm">
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium text-muted-foreground">Total Keys</p>
            <Database className="h-4 w-4 text-muted-foreground" />
          </div>
          <div className="mt-2 flex items-baseline gap-2">
            <span className="text-2xl font-bold text-foreground">12,403</span>
            <span className="text-xs text-emerald-500 font-medium">+12%</span>
          </div>
        </div>
        <div className="p-6 rounded-xl bg-card border border-border shadow-sm">
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium text-muted-foreground">Active Queues</p>
            <Server className="h-4 w-4 text-muted-foreground" />
          </div>
          <div className="mt-2 flex items-baseline gap-2">
            <span className="text-2xl font-bold text-foreground">8</span>
            <span className="text-xs text-muted-foreground">Idle</span>
          </div>
        </div>
        <div className="p-6 rounded-xl bg-card border border-border shadow-sm">
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium text-muted-foreground">Throughput</p>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </div>
          <div className="mt-2 flex items-baseline gap-2">
            <span className="text-2xl font-bold text-foreground">2.4k</span>
            <span className="text-xs text-muted-foreground">ops/sec</span>
          </div>
        </div>
        <div className="p-6 rounded-xl bg-card border border-border shadow-sm">
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium text-muted-foreground">Nodes</p>
            <Cpu className="h-4 w-4 text-muted-foreground" />
          </div>
          <div className="mt-2 flex items-baseline gap-2">
            <span className="text-2xl font-bold text-foreground">3</span>
            <span className="text-xs text-emerald-500 font-medium">All Online</span>
          </div>
        </div>
      </div>

      {/* Feature Cards */}
      <div className="grid gap-6 md:grid-cols-2">
        <div className="space-y-6">
          <h2 className="text-lg font-semibold text-foreground">Quick Actions</h2>
          <div className="grid gap-4">
            <Link href="/kv" className="group relative overflow-hidden rounded-xl bg-card border border-border p-6 hover:border-primary/50 transition-all duration-300 hover:shadow-lg hover:shadow-primary/5">
              <div className="flex items-start justify-between">
                <div className="space-y-2">
                  <div className="p-2 w-fit rounded-lg bg-primary/10 text-primary">
                    <Database className="h-6 w-6" />
                  </div>
                  <h3 className="font-semibold text-foreground">KV Store</h3>
                  <p className="text-sm text-muted-foreground max-w-[280px]">
                    Browse, edit, and manage key-value pairs with real-time updates.
                  </p>
                </div>
                <ArrowRight className="h-5 w-5 text-muted-foreground group-hover:text-primary group-hover:translate-x-1 transition-all" />
              </div>
            </Link>

            <Link href="/queues" className="group relative overflow-hidden rounded-xl bg-card border border-border p-6 hover:border-primary/50 transition-all duration-300 hover:shadow-lg hover:shadow-primary/5">
              <div className="flex items-start justify-between">
                <div className="space-y-2">
                  <div className="p-2 w-fit rounded-lg bg-orange-500/10 text-orange-500">
                    <Server className="h-6 w-6" />
                  </div>
                  <h3 className="font-semibold text-foreground">Queues</h3>
                  <p className="text-sm text-muted-foreground max-w-[280px]">
                    Monitor queue depth, consumer groups, and message throughput.
                  </p>
                </div>
                <ArrowRight className="h-5 w-5 text-muted-foreground group-hover:text-primary group-hover:translate-x-1 transition-all" />
              </div>
            </Link>
          </div>
        </div>

        <div className="space-y-6">
          <h2 className="text-lg font-semibold text-foreground">System Status</h2>
          <div className="rounded-xl bg-card border border-border p-6 space-y-6">
            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Memory Usage</span>
                <span className="font-medium text-foreground">64%</span>
              </div>
              <div className="h-2 rounded-full bg-secondary overflow-hidden">
                <div className="h-full w-[64%] bg-primary rounded-full" />
              </div>
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">CPU Load</span>
                <span className="font-medium text-foreground">28%</span>
              </div>
              <div className="h-2 rounded-full bg-secondary overflow-hidden">
                <div className="h-full w-[28%] bg-emerald-500 rounded-full" />
              </div>
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Storage</span>
                <span className="font-medium text-foreground">45%</span>
              </div>
              <div className="h-2 rounded-full bg-secondary overflow-hidden">
                <div className="h-full w-[45%] bg-blue-500 rounded-full" />
              </div>
            </div>

            <div className="pt-4 border-t border-border mt-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <HardDrive className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm font-medium text-foreground">Disk I/O</span>
                </div>
                <span className="text-sm text-muted-foreground">125 MB/s</span>
              </div>
            </div>
          </div>

          <div className="rounded-xl bg-card border border-border p-6 opacity-60 relative overflow-hidden">
            <div className="absolute inset-0 bg-background/50 backdrop-blur-[1px] flex items-center justify-center z-10">
              <span className="px-3 py-1 rounded-full bg-secondary text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                Coming Soon
              </span>
            </div>
            <div className="flex items-center gap-3 mb-4">
              <div className="p-2 rounded-lg bg-blue-500/10 text-blue-500">
                <Radio className="h-5 w-5" />
              </div>
              <div>
                <h3 className="font-semibold text-foreground">Streams</h3>
                <p className="text-xs text-muted-foreground">Event streaming</p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-lg bg-pink-500/10 text-pink-500">
                <MessageSquare className="h-5 w-5" />
              </div>
              <div>
                <h3 className="font-semibold text-foreground">Pub/Sub</h3>
                <p className="text-xs text-muted-foreground">Real-time messaging</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
