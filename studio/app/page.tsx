import Link from "next/link";

export default function Home() {
  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="space-y-2">
        <h1 className="text-4xl font-bold tracking-tight text-foreground">
          Flin Studio
        </h1>
        <p className="text-base text-foreground/75 max-w-2xl">
          Cluster-wide admin dashboard. Inspect and manage KV store, queues, streams,
          pub/sub, and database layers from one unified interface.
        </p>
      </div>

      {/* Stats Grid */}
      <div className="grid gap-4 md:grid-cols-3">
        <Link href="/kv" className="group">
          <section className="rounded-lg border bg-card/50 backdrop-blur-sm p-6 shadow-sm hover:shadow-md hover:border-accent/50 transition-all">
            <div className="text-xs font-semibold uppercase tracking-widest text-foreground/60">
              KV Store
            </div>
            <div className="mt-4 flex items-baseline gap-2">
              <span className="text-3xl font-bold text-foreground">—</span>
              <span className="text-sm text-foreground/70">keys</span>
            </div>
            <p className="mt-3 text-sm text-foreground/70">
              Live key browser and value inspector once connected to a Flin node.
            </p>
            <div className="mt-4 inline-flex items-center gap-1 text-xs font-medium text-accent">
              Browse Keys
              <svg className="w-3 h-3 group-hover:translate-x-1 transition-transform" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
              </svg>
            </div>
          </section>
        </Link>

        <Link href="/queues" className="group">
          <section className="rounded-lg border bg-card/50 backdrop-blur-sm p-6 shadow-sm hover:shadow-md hover:border-accent/50 transition-all">
            <div className="text-xs font-semibold uppercase tracking-widest text-foreground/60">
              Queues
            </div>
            <div className="mt-4 flex items-baseline gap-2">
              <span className="text-3xl font-bold text-foreground">—</span>
              <span className="text-sm text-foreground/70">queues</span>
            </div>
            <p className="mt-3 text-sm text-foreground/70">
              Inspect queue depth, throughput, and recent messages in real-time.
            </p>
            <div className="mt-4 inline-flex items-center gap-1 text-xs font-medium text-accent">
              View Queues
              <svg className="w-3 h-3 group-hover:translate-x-1 transition-transform" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
              </svg>
            </div>
          </section>
        </Link>

        <section className="rounded-lg border bg-card/50 backdrop-blur-sm p-6 shadow-sm opacity-60">
          <div className="text-xs font-semibold uppercase tracking-widest text-foreground/60">
            Streams & Pub/Sub
          </div>
          <div className="mt-4 flex items-baseline gap-2">
            <span className="text-xs font-semibold text-foreground/70">planned</span>
          </div>
          <p className="mt-3 text-sm text-foreground/70">
            UI is ready; wire up once Flin exposes streams and pub/sub APIs.
          </p>
          <div className="mt-4 inline-flex items-center gap-1 text-xs font-medium text-muted-foreground">
            Coming Soon
            <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>
        </section>
      </div>

      {/* Analytics Grid */}
      <div className="grid gap-4 md:grid-cols-2">
        <section className="rounded-lg border bg-card/50 backdrop-blur-sm p-6 shadow-sm">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-sm font-semibold text-foreground">Cluster Activity</h2>
              <p className="mt-1 text-xs text-foreground/70">
                Requests per second across KV and queue operations.
              </p>
            </div>
            <span className="rounded-full border bg-muted px-2 py-1 text-[10px] font-semibold uppercase text-muted-foreground">
              pending wiring
            </span>
          </div>
          <div className="mt-6 h-40 rounded-md border border-dashed bg-muted/30 flex items-center justify-center">
            <div className="text-center">
              <svg className="w-8 h-8 text-muted-foreground/40 mx-auto mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M7 12a1 1 0 110-2 1 1 0 010 2zM15 12a1 1 0 110-2 1 1 0 010 2z" />
              </svg>
              <p className="text-xs text-muted-foreground">Chart placeholder</p>
            </div>
          </div>
        </section>

        <section className="rounded-lg border bg-card/50 backdrop-blur-sm p-6 shadow-sm">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-sm font-semibold text-foreground">Recent Admin Actions</h2>
              <p className="mt-1 text-xs text-muted-foreground">
                Once auth is added, show who changed what in the cluster.
              </p>
            </div>
          </div>
          <ul className="mt-4 space-y-3">
            <li className="text-xs text-muted-foreground bg-muted/30 rounded px-3 py-2 border border-dashed">
              No actions yet. Connect to a Flin cluster to start managing data.
            </li>
          </ul>
        </section>
      </div>

      {/* Quick Stats */}
      <div className="rounded-lg border bg-card/50 backdrop-blur-sm p-6 shadow-sm">
        <h3 className="text-sm font-semibold text-foreground mb-4">Status Overview</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="text-center">
            <div className="text-2xl font-bold text-foreground">—</div>
            <div className="text-xs text-muted-foreground mt-1">Total Keys</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-foreground">—</div>
            <div className="text-xs text-muted-foreground mt-1">Queued Items</div>
          </div>
          <div className="text-center">
            <div className="inline-flex items-center gap-1">
              <span className="h-2 w-2 rounded-full bg-emerald-500" />
              <span className="text-sm font-semibold text-foreground">Healthy</span>
            </div>
            <div className="text-xs text-muted-foreground mt-1">Cluster Status</div>
          </div>
          <div className="text-center">
            <span className="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-semibold bg-accent/10 text-accent">
              Connected
            </span>
            <div className="text-xs text-muted-foreground mt-1">Connection</div>
          </div>
        </div>
      </div>
    </div>
  );
}
