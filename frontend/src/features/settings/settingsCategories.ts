import type { ComponentType } from 'react';
import type { AppContextValue } from '@/context/AppContext';
import { ProfilePanel } from './components/ProfilePanel';
import { AppearancePanel } from './components/AppearancePanel';
import { NotificationsPanel } from './components/NotificationsPanel';
import { PrivacyPanel } from './components/PrivacyPanel';
import { LegalPanel } from './components/LegalPanel';

export interface SettingsPanelProps {
  app: AppContextValue;
}

export type SettingsCategoryKey = 'profile' | 'appearance' | 'notifications' | 'privacy' | 'legal';

export interface SettingsCategory {
  key: SettingsCategoryKey;
  labelKey: string;
  icon: string;
  Component: ComponentType<SettingsPanelProps>;
}

/**
 * Single source of truth for Settings' sidebar/list categories. Adding a new
 * *setting* to an existing area (e.g. another notification toggle) -> add it
 * to that category's panel component. Adding a new *kind* of setting that
 * doesn't fit any existing panel -> add one entry here (key, label, icon,
 * panel component) instead of growing an existing panel indefinitely or
 * reviving a flat list.
 */
export const SETTINGS_CATEGORIES: SettingsCategory[] = [
  { key: 'profile', labelKey: 'settings.category.profile', icon: 'person', Component: ProfilePanel },
  { key: 'appearance', labelKey: 'settings.category.appearance', icon: 'palette', Component: AppearancePanel },
  {
    key: 'notifications',
    labelKey: 'settings.category.notifications',
    icon: 'notifications',
    Component: NotificationsPanel,
  },
  { key: 'privacy', labelKey: 'settings.category.privacy', icon: 'privacy_tip', Component: PrivacyPanel },
  { key: 'legal', labelKey: 'settings.category.legal', icon: 'gavel', Component: LegalPanel },
];
