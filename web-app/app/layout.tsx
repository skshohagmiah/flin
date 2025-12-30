import type { Metadata } from "next";
import { Inter, JetBrains_Mono } from "next/font/google";
import "./globals.css";
import Navbar from "@/components/Navbar";

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
});

const jetbrainsMono = JetBrains_Mono({
  variable: "--font-jetbrains-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Flin - High-Performance Distributed Data Platform",
  description: "Blazing-fast KV Store, Message Queue, Stream Processing, and Document Database unified in a single distributed system. 3x faster than Redis.",
  keywords: ["distributed systems", "database", "key-value store", "message queue", "stream processing", "document database", "redis alternative"],
  authors: [{ name: "Flin Team" }],
  openGraph: {
    title: "Flin - High-Performance Distributed Data Platform",
    description: "Blazing-fast KV Store, Message Queue, Stream Processing, and Document Database unified in a single distributed system.",
    type: "website",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body
        className={`${inter.variable} ${jetbrainsMono.variable} antialiased`}
      >
        <Navbar />
        {children}
      </body>
    </html>
  );
}
