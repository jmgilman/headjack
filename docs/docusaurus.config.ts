import { themes as prismThemes } from 'prism-react-renderer';
import type { Config } from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'Headjack',
  tagline: 'Spawn isolated LLM coding agents in container environments',
  favicon: 'img/favicon.ico',

  future: {
    v4: true,
  },

  // Production URL - will be updated when Cloudflare Pages is configured
  url: 'https://headjack.gilman.io',
  baseUrl: '/',

  organizationName: 'gilmanlab',
  projectName: 'headjack',

  onBrokenLinks: 'throw',

  markdown: {
    hooks: {
      onBrokenMarkdownLinks: 'warn',
    },
  },

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          routeBasePath: '/', // Docs-only mode: serve docs at root
          editUrl: 'https://github.com/gilmanlab/headjack/tree/master/docs/',
        },
        blog: false, // Disable blog
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    colorMode: {
      defaultMode: 'dark',
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: 'Headjack',
      logo: {
        alt: 'Headjack Logo',
        src: 'img/logo.png',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'docsSidebar',
          position: 'left',
          label: 'Documentation',
        },
        {
          href: 'https://github.com/gilmanlab/headjack',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Documentation',
          items: [
            {
              label: 'Getting Started',
              to: '/tutorials/getting-started',
            },
            {
              label: 'CLI Reference',
              to: '/reference/cli/run',
            },
          ],
        },
        {
          title: 'More',
          items: [
            {
              label: 'GitHub',
              href: 'https://github.com/gilmanlab/headjack',
            },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Headjack. Built with Docusaurus.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['bash', 'yaml', 'json', 'go', 'docker'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
