import Sidebar from '@/components/Sidebar';
import Footer from '@/components/Footer';

export default function DocsLayout({
    children,
}: {
    children: React.ReactNode;
}) {
    return (
        <div className="min-h-screen pt-16">
            <div className="container-custom py-8 md:py-12">
                <div className="flex flex-col md:flex-row gap-8 md:gap-12">
                    {/* Sidebar - Hidden on mobile, shown on desktop */}
                    <aside className="w-full md:w-64 md:flex-shrink-0">
                        <div className="md:sticky md:top-24">
                            <Sidebar />
                        </div>
                    </aside>

                    {/* Main Content */}
                    <main className="flex-1 min-w-0 w-full">
                        {children}
                    </main>
                </div>
            </div>
            <Footer />
        </div>
    );
}
