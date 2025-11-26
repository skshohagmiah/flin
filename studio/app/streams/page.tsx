export default function StreamsPage() {
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="space-y-2">
        <h1 className="text-3xl font-bold tracking-tight">Streams</h1>
        <p className="text-sm text-muted-foreground">
          Flin streams are not implemented yet, but this UI is ready for when they are available.
        </p>
      </div>

      {/* Grid Layout */}
      <div className="grid gap-6 md:grid-cols-2">
        {/* Stream List Card */}
        <div className="rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm overflow-hidden flex flex-col">
          <div className="px-6 py-4 border-b border-border/50 bg-gradient-to-b from-muted/30 to-transparent">
            <h2 className="text-sm font-semibold text-foreground">Stream List</h2>
            <p className="mt-1 text-xs text-muted-foreground">
              Browse streams, consumer groups and retention policies.
            </p>
          </div>
          <div className="flex-1 flex items-center justify-center px-6 py-12">
            <div className="text-center">
              <svg className="w-12 h-12 text-muted-foreground/30 mx-auto mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
              </svg>
              <p className="text-xs text-muted-foreground">Streams table placeholder</p>
              <p className="text-[11px] text-muted-foreground/60 mt-1">Data will come from the Flin streams API</p>
            </div>
          </div>
        </div>

        {/* Stream Inspector Card */}
        <div className="rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm overflow-hidden flex flex-col">
          <div className="px-6 py-4 border-b border-border/50 bg-gradient-to-b from-muted/30 to-transparent">
            <h2 className="text-sm font-semibold text-foreground">Stream Inspector</h2>
            <p className="mt-1 text-xs text-muted-foreground">
              Inspect messages in order, seek by offset and replay windows.
            </p>
          </div>
          <div className="flex-1 flex items-center justify-center px-6 py-12">
            <div className="text-center">
              <svg className="w-12 h-12 text-muted-foreground/30 mx-auto mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
              <p className="text-xs text-muted-foreground">Message timeline placeholder</p>
              <p className="text-[11px] text-muted-foreground/60 mt-1">Ready once Flin exposes stream APIs</p>
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
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
                </svg>
              </div>
            </div>
            <div className="flex-1">
              <h3 className="text-xs font-semibold text-foreground">Consumer Groups</h3>
              <p className="mt-1 text-xs text-muted-foreground">Manage stream consumer groups and offsets</p>
            </div>
          </div>
        </div>

        <div className="rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm p-4">
          <div className="flex items-start gap-3">
            <div className="flex-shrink-0">
              <div className="flex items-center justify-center h-8 w-8 rounded-lg bg-purple-500/10 text-purple-600 dark:text-purple-400">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
            </div>
            <div className="flex-1">
              <h3 className="text-xs font-semibold text-foreground">Retention Policies</h3>
              <p className="mt-1 text-xs text-muted-foreground">Configure stream retention and cleanup</p>
            </div>
          </div>
        </div>

        <div className="rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm p-4">
          <div className="flex items-start gap-3">
            <div className="flex-shrink-0">
              <div className="flex items-center justify-center h-8 w-8 rounded-lg bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14 10l-2 1m0 0l-2-1m2 1v2.5M20 7l-2 1m2-1l-2-1m2 1v2.5M14 4l-2 1m2-1l-2-1m2 1v2.5" />
                </svg>
              </div>
            </div>
            <div className="flex-1">
              <h3 className="text-xs font-semibold text-foreground">Message Replay</h3>
              <p className="mt-1 text-xs text-muted-foreground">Replay and seek through stream messages</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
