export default function PubSubPage() {
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="space-y-2">
        <h1 className="text-3xl font-bold tracking-tight">Pub/Sub</h1>
        <p className="text-sm text-muted-foreground">
          Pub/Sub channels and topics will appear here once Flin exposes a pub/sub API.
        </p>
      </div>

      {/* Grid Layout */}
      <div className="grid gap-6 md:grid-cols-2">
        {/* Topics Card */}
        <div className="rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm overflow-hidden flex flex-col">
          <div className="px-6 py-4 border-b border-border/50 bg-gradient-to-b from-muted/30 to-transparent">
            <h2 className="text-sm font-semibold text-foreground">Topics</h2>
            <p className="mt-1 text-xs text-muted-foreground">
              Listing topics, subscriber counts and recent messages.
            </p>
          </div>
          <div className="flex-1 flex items-center justify-center px-6 py-12">
            <div className="text-center">
              <svg className="w-12 h-12 text-muted-foreground/30 mx-auto mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01" />
              </svg>
              <p className="text-xs text-muted-foreground">Topics table placeholder</p>
              <p className="text-[11px] text-muted-foreground/60 mt-1">Ready for Flin pub/sub API</p>
            </div>
          </div>
        </div>

        {/* Publish & Subscribe Card */}
        <div className="rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm overflow-hidden flex flex-col">
          <div className="px-6 py-4 border-b border-border/50 bg-gradient-to-b from-muted/30 to-transparent">
            <h2 className="text-sm font-semibold text-foreground">Publish & Subscribe</h2>
            <p className="mt-1 text-xs text-muted-foreground">
              Send test messages and watch live deliveries to subscribers.
            </p>
          </div>
          <div className="flex-1 flex items-center justify-center px-6 py-12">
            <div className="text-center">
              <svg className="w-12 h-12 text-muted-foreground/30 mx-auto mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
              <p className="text-xs text-muted-foreground">Live traffic placeholder</p>
              <p className="text-[11px] text-muted-foreground/60 mt-1">Monitor real-time message delivery</p>
            </div>
          </div>
        </div>
      </div>

      {/* Feature Cards */}
      <div className="grid gap-4 md:grid-cols-3">
        <div className="rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm p-4">
          <div className="flex items-start gap-3">
            <div className="flex-shrink-0">
              <div className="flex items-center justify-center h-8 w-8 rounded-lg bg-blue-500/10 text-blue-600 dark:text-blue-400">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
            </div>
            <div className="flex-1">
              <h3 className="text-xs font-semibold text-foreground">Topic Management</h3>
              <p className="mt-1 text-xs text-muted-foreground">Create and manage pub/sub topics</p>
            </div>
          </div>
        </div>

        <div className="rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm p-4">
          <div className="flex items-start gap-3">
            <div className="flex-shrink-0">
              <div className="flex items-center justify-center h-8 w-8 rounded-lg bg-purple-500/10 text-purple-600 dark:text-purple-400">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14 10l-2 1m0 0l-2-1m2 1v2.5M20 7l-2 1m2-1l-2-1m2 1v2.5M14 4l-2 1m2-1l-2-1m2 1v2.5" />
                </svg>
              </div>
            </div>
            <div className="flex-1">
              <h3 className="text-xs font-semibold text-foreground">Subscribers</h3>
              <p className="mt-1 text-xs text-muted-foreground">Monitor active subscribers and delivery</p>
            </div>
          </div>
        </div>

        <div className="rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm p-4">
          <div className="flex items-start gap-3">
            <div className="flex-shrink-0">
              <div className="flex items-center justify-center h-8 w-8 rounded-lg bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                </svg>
              </div>
            </div>
            <div className="flex-1">
              <h3 className="text-xs font-semibold text-foreground">Message Stats</h3>
              <p className="mt-1 text-xs text-muted-foreground">Track publish/delivery metrics</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
