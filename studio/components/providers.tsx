"use client";

import { ThemeProvider } from "next-themes";
import { ReactNode } from "react";
import { QueryProvider } from "./providers/query-provider";

export function Providers({ children }: { children: ReactNode }) {
  return (
    <QueryProvider>
      <ThemeProvider attribute="class" defaultTheme="system" enableSystem>
        {children}
      </ThemeProvider>
    </QueryProvider>
  );
}
