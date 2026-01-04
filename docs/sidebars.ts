import type { SidebarsConfig } from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  docsSidebar: [
    'intro',

    // Tutorials - Learning-oriented
    {
      type: 'category',
      label: 'Tutorials',
      collapsed: false,
      link: {
        type: 'generated-index',
        title: 'Tutorials',
        description: 'Learn Headjack through hands-on lessons',
        slug: '/tutorials',
      },
      items: [
        'tutorials/getting-started',
        'tutorials/first-coding-task',
        // 'tutorials/parallel-agents',
        'tutorials/custom-image',
      ],
    },

    // How-to Guides - Goal-oriented
    {
      type: 'category',
      label: 'How-to Guides',
      collapsed: true,
      link: {
        type: 'generated-index',
        title: 'How-to Guides',
        description: 'Accomplish specific tasks with Headjack',
        slug: '/how-to',
      },
      items: [
        {
          type: 'category',
          label: 'Installation & Setup',
          items: [
            'how-to/install',
            'how-to/authenticate',
          ],
        },
        'how-to/manage-sessions',
        {
          type: 'category',
          label: 'Instance Management',
          items: [
            'how-to/stop-cleanup',
            'how-to/recover-from-crash',
          ],
        },
        {
          type: 'category',
          label: 'Customization',
          items: [
            'how-to/build-custom-image',
          ],
        },
        {
          type: 'category',
          label: 'Troubleshooting',
          items: [
            'how-to/troubleshoot-auth',
          ],
        },
      ],
    },

    // Reference - Information-oriented
    {
      type: 'category',
      label: 'Reference',
      collapsed: true,
      link: {
        type: 'generated-index',
        title: 'Reference',
        description: 'Technical specifications and API reference',
        slug: '/reference',
      },
      items: [
        {
          type: 'category',
          label: 'CLI Commands',
          link: {
            type: 'generated-index',
            title: 'CLI Commands',
            description: 'Complete reference for all Headjack commands',
            slug: '/reference/cli',
          },
          items: [
            'reference/cli/run',
            'reference/cli/attach',
            'reference/cli/ps',
            'reference/cli/logs',
            'reference/cli/stop',
            'reference/cli/kill',
            'reference/cli/rm',
            'reference/cli/recreate',
            'reference/cli/auth',
            'reference/cli/config',
            'reference/cli/version',
          ],
        },
        'reference/configuration',
        'reference/environment',
        'reference/storage',
        {
          type: 'category',
          label: 'Container Images',
          link: {
            type: 'generated-index',
            title: 'Container Images',
            description: 'Reference for Headjack container images',
            slug: '/reference/images',
          },
          items: [
            'reference/images/overview',
          ],
        },
      ],
    },

    // Explanation - Understanding-oriented
    {
      type: 'category',
      label: 'Concepts',
      collapsed: true,
      link: {
        type: 'generated-index',
        title: 'Concepts',
        description: 'Understand how and why Headjack works',
        slug: '/concepts',
      },
      items: [
        'explanation/architecture',
        // 'explanation/isolation-model',
        // 'explanation/cli-agents-vs-api',
        'explanation/worktree-strategy',
        'explanation/session-lifecycle',
        'explanation/authentication',
        'explanation/image-customization',
        'explanation/version-managers',
      ],
    },

    // Architecture Decision Records
    {
      type: 'category',
      label: 'Decisions (ADRs)',
      collapsed: true,
      link: {
        type: 'doc',
        id: 'decisions/index',
      },
      items: [
        'decisions/adr-001-macos-only',
        'decisions/adr-002-apple-containerization',
        'decisions/adr-003-go-language',
        'decisions/adr-004-cli-agents',
        'decisions/adr-005-no-gpg-support',
        'decisions/adr-006-oci-customization',
      ],
    },
  ],
};

export default sidebars;
