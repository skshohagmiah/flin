export default function DbPage() {
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="space-y-2">
        <h1 className="text-3xl font-bold tracking-tight">Database</h1>
        <p className="text-sm text-muted-foreground">
          Future Flin database / table abstractions will be managed here. The layout is ready for when those features land.
        </p>
      </div>

      {/* Grid Layout */}
      <div className="grid gap-6 md:grid-cols-2">
        {/* Schemas / Collections Card */}
        <div className="rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm overflow-hidden flex flex-col">
          <div className="px-6 py-4 border-b border-border/50 bg-gradient-to-b from-muted/30 to-transparent">
            <h2 className="text-sm font-semibold text-foreground">Schemas & Collections</h2>
            <p className="mt-1 text-xs text-muted-foreground">
              Visualize logical databases, schemas or collections mapped onto Flin primitives.
            </p>
          </div>
          <div className="flex-1 flex items-center justify-center px-6 py-12">
            <div className="text-center">
              <svg className="w-12 h-12 text-muted-foreground/30 mx-auto mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7m0 0c0 2.21-3.582 4-8 4s-8-1.79-8-4m0 0c0-2.21 3.582-4 8-4s8 1.79 8 4m0 6c0 2.21-3.582 4-8 4s-8-1.79-8-4" />
              </svg>
              <p className="text-xs text-muted-foreground">Schema list placeholder</p>
              <p className="text-[11px] text-muted-foreground/60 mt-1">Ready for database abstractions</p>
            </div>
          </div>
        </div>

        {/* Query Playground Card */}
        <div className="rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm overflow-hidden flex flex-col">
          <div className="px-6 py-4 border-b border-border/50 bg-gradient-to-b from-muted/30 to-transparent">
            <h2 className="text-sm font-semibold text-foreground">Query Playground</h2>
            <p className="mt-1 text-xs text-muted-foreground">
              Run read/write queries against Flin once a query layer exists.
            </p>
          </div>
          <div className="flex-1 flex items-center justify-center px-6 py-12">
            <div className="text-center">
              <svg className="w-12 h-12 text-muted-foreground/30 mx-auto mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M14 10l-2 1m0 0l-2-1m2 1v2.5M20 7l-2 1m2-1l-2-1m2 1v2.5M14 4l-2 1m2-1l-2-1m2 1v2.5" />
              </svg>
              <p className="text-xs text-muted-foreground">Query editor placeholder</p>
              <p className="text-[11px] text-muted-foreground/60 mt-1">Execute queries on your Flin cluster</p>
            </div>
          </div>
        </div>
      </div>

      {/* Feature Overview Cards */}
      <div className="grid gap-4 md:grid-cols-3">
        <div className="rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm p-4">
          <div className="flex items-start gap-3">
            <div className="flex-shrink-0">
              <div className="flex items-center justify-center h-8 w-8 rounded-lg bg-blue-500/10 text-blue-600 dark:text-blue-400">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
              </div>
            </div>
            <div className="flex-1">
              <h3 className="text-xs font-semibold text-foreground">Table Management</h3>
              <p className="mt-1 text-xs text-muted-foreground">Create and manage database tables</p>
            </div>
          </div>
        </div>

        <div className="rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm p-4">
          <div className="flex items-start gap-3">
            <div className="flex-shrink-0">
              <div className="flex items-center justify-center h-8 w-8 rounded-lg bg-purple-500/10 text-purple-600 dark:text-purple-400">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
              </div>
            </div>
            <div className="flex-1">
              <h3 className="text-xs font-semibold text-foreground">Query Builder</h3>
              <p className="mt-1 text-xs text-muted-foreground">Build and execute queries visually</p>
            </div>
          </div>
        </div>

        <div className="rounded-lg border border-border/50 bg-card/50 backdrop-blur-sm p-4">
          <div className="flex items-start gap-3">
            <div className="flex-shrink-0">
              <div className="flex items-center justify-center h-8 w-8 rounded-lg bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
            </div>
            <div className="flex-1">
              <h3 className="text-xs font-semibold text-foreground">Schema Browser</h3>
              <p className="mt-1 text-xs text-muted-foreground">Explore database structure and relations</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
