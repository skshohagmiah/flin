import Hero from '@/components/Hero';
import Features from '@/components/Features';
import Performance from '@/components/Performance';
import Architecture from '@/components/Architecture';
import QuickStart from '@/components/QuickStart';
import Footer from '@/components/Footer';

export default function Home() {
  return (
    <main className="min-h-screen">
      <Hero />
      <Features />
      <Performance />
      <Architecture />
      <QuickStart />
      <Footer />
    </main>
  );
}
