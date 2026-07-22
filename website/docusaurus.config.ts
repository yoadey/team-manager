import type { Config } from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

// Published via .github/workflows/docs-deploy.yml once GitHub Pages is
// enabled for this repository (Settings → Pages → Source: GitHub Actions).
const organizationName = 'yoadey';
const projectName = 'team-manager';

const config: Config = {
  title: 'Teamverwaltung — Hilfe',
  tagline: 'Dokumentation für Vereinsmitglieder und Admins',

  url: `https://${organizationName}.github.io`,
  baseUrl: `/${projectName}/`,

  organizationName,
  projectName,

  onBrokenLinks: 'throw',

  markdown: {
    hooks: {
      onBrokenMarkdownLinks: 'throw',
    },
  },

  i18n: {
    defaultLocale: 'de',
    locales: ['de'],
  },

  presets: [
    [
      'classic',
      {
        // Source of truth stays in ../docs/end-user (also readable directly
        // on GitHub) — this site only renders it, no content duplication.
        docs: {
          path: '../docs/end-user',
          routeBasePath: '/',
          sidebarPath: './sidebars.ts',
          editUrl: `https://github.com/${organizationName}/${projectName}/edit/main/docs/end-user/`,
        },
        blog: false,
        pages: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    navbar: {
      title: 'Teamverwaltung — Hilfe',
      items: [
        {
          href: `https://github.com/${organizationName}/${projectName}`,
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      copyright: `Teamverwaltung — Endanwender-Dokumentation`,
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
