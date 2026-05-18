import type { LucideIcon } from 'lucide-react';
import { Bot, GitBranch, Palette, Vault } from 'lucide-react';

export type NavItem = {
  label: string;
  to: string;
  icon: LucideIcon;
  disabled?: boolean;
};

/** Primary dashboard navigation entries. */
export const mainNavItems: NavItem[] = [
  { label: 'Agents', to: '/agent', icon: Bot },
  { label: 'Vault', to: '/vault', icon: Vault, disabled: true },
  { label: 'Workflows', to: '/workflow', icon: GitBranch, disabled: true },
  { label: 'Design', to: '/demo', icon: Palette },
];
