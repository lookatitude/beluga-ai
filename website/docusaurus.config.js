const { themes } = require('prism-react-renderer');
const lightCodeTheme = themes.github;
const darkCodeTheme = themes.dracula;

// With JSDoc @type annotations, IDEs can provide config autocompletion
/** @type {import('@docusaurus/types').DocusaurusConfig} */
module.exports = {
  title: 'Beluga AI Framework',
  tagline: 'A production-ready Go framework for building sophisticated AI and agentic applications.',
  url: 'https://lookatitude.github.io',
  baseUrl: '/',
  onBrokenLinks: 'warn',
  markdown: {
    mermaid: true,
    hooks: {
      onBrokenMarkdownLinks: 'warn',
    },
  },
  favicon: 'img/favicon.ico',
  organizationName: 'lookatitude',
  projectName: 'beluga-ai',
  
  // i18n configuration
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      '@docusaurus/preset-classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          // Read documentation from root docs/ directory (single source of truth)
          // This eliminates sync complexity between docs/ and website/docs/
          path: '../docs',
          sidebarPath: require.resolve('./sidebars.js'),
          editUrl: 'https://github.com/lookatitude/beluga-ai/tree/main/',
          // Exclude template files only
          exclude: ['**/_template.md'],
        },
        blog: {
          path: 'blog',
          showReadingTime: true,
          editUrl:
            'https://github.com/lookatitude/beluga-ai/tree/main/website/blog/',
          onUntruncatedBlogPosts: 'ignore',
        },
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      }),
    ],
  ],

  themes: ['@docusaurus/theme-mermaid'],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      navbar: {
        title: 'Beluga AI',
        logo: {
          alt: 'Beluga AI Framework Logo',
          src: 'img/beluga-logo.svg',
        },
        items: [
          {
            type: 'doc',
            docId: 'README',
            position: 'left',
            label: 'Documentation',
          },
          {
            type: 'doc',
            docId: 'quickstart',
            position: 'left',
            label: 'Get Started',
          },
          {
            type: 'doc',
            docId: 'cookbook/quick-solutions',
            position: 'left',
            label: 'Cookbook',
          },
          {
            href: 'https://github.com/lookatitude/beluga-ai',
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
                to: '/docs/quickstart',
              },
              {
                label: 'Concepts',
                to: '/docs/concepts/core',
              },
              {
                label: 'API Reference',
                to: '/docs/api-reference',
              },
              {
                label: 'Use Cases',
                to: '/docs/use-cases',
              },
            ],
          },
          {
            title: 'Resources',
            items: [
              {
                label: 'Examples',
                href: 'https://github.com/lookatitude/beluga-ai/tree/main/examples',
              },
              {
                label: 'Cookbook',
                to: '/docs/cookbook/quick-solutions',
              },
              {
                label: 'Best Practices',
                to: '/docs/best-practices',
              },
              {
                label: 'Architecture',
                to: '/docs/architecture',
              },
            ],
          },
          {
            title: 'Community',
            items: [
              {
                label: 'GitHub',
                href: 'https://github.com/lookatitude/beluga-ai',
              },
              {
                label: 'GitHub Discussions',
                href: 'https://github.com/lookatitude/beluga-ai/discussions',
              },
              {
                label: 'Issues',
                href: 'https://github.com/lookatitude/beluga-ai/issues',
              },
              {
                label: 'Contributing',
                href: 'https://github.com/lookatitude/beluga-ai/blob/main/CONTRIBUTING.md',
              },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} Beluga AI Framework. Built with Docusaurus.`,
      },
      prism: {
        theme: lightCodeTheme,
        darkTheme: darkCodeTheme,
        additionalLanguages: ['go', 'bash', 'yaml', 'json'],
      },
      colorMode: {
        defaultMode: 'light',
        disableSwitch: false,
        respectPrefersColorScheme: true,
      },
    }),
  
  // Custom fields for metadata
  customFields: {
    metadata: [
      {name: 'keywords', content: 'go, golang, ai, llm, langchain, agents, rag, vectorstore, embeddings, ai-framework'},
      {name: 'description', content: 'Beluga AI Framework - A production-ready Go framework for building sophisticated AI and agentic applications with enterprise-grade observability.'},
    ],
  },
};
