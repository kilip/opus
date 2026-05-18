import { X } from 'lucide-react';
import { useEffect } from 'react';
import opusLogo from '@/assets/opus.svg';
import { mainNavItems } from '@/shared/components/layout/nav-config';
import { SidebarNav } from '@/shared/components/layout/SidebarNav';
import { Button } from '@/shared/components/ui/button';
import { cn } from '@/shared/lib/utils';

type MobileNavDrawerProps = {
  open: boolean;
  onClose: () => void;
};

/**
 * Full-height slide-over navigation for viewports below the desktop sidebar breakpoint.
 */
export function MobileNavDrawer({ open, onClose }: MobileNavDrawerProps) {
  useEffect(() => {
    if (!open) return;
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', onKey);
    document.body.style.overflow = 'hidden';
    return () => {
      document.removeEventListener('keydown', onKey);
      document.body.style.overflow = '';
    };
  }, [open, onClose]);

  return (
    <>
      <button
        type="button"
        aria-label="Close navigation menu"
        className={cn(
          'fixed inset-0 z-50 bg-brand-dark/40 backdrop-blur-sm transition-opacity lg:hidden',
          open ? 'opacity-100' : 'pointer-events-none opacity-0',
        )}
        onClick={onClose}
        tabIndex={open ? 0 : -1}
      />
      <aside
        className={cn(
          'fixed inset-y-0 left-0 z-50 flex w-[min(100%,18rem)] flex-col',
          'border-r border-border bg-sidebar text-sidebar-foreground shadow-elevated',
          'transition-transform duration-300 ease-out lg:hidden',
          open ? 'translate-x-0' : '-translate-x-full',
        )}
        aria-hidden={!open}
        inert={!open ? true : undefined}
      >
        <div className="flex items-center justify-between border-b border-border px-4 py-4">
          <div className="flex items-center gap-3">
            <img src={opusLogo} alt="" className="h-8 w-8" aria-hidden />
            <span className="font-sans text-base font-semibold">Opus</span>
          </div>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            onClick={onClose}
            aria-label="Close menu"
          >
            <X className="h-5 w-5" aria-hidden />
          </Button>
        </div>
        <div className="flex-1 overflow-y-auto px-3 py-4">
          <SidebarNav items={mainNavItems} onNavigate={onClose} />
        </div>
      </aside>
    </>
  );
}
