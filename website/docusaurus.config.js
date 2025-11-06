const lightCodeTheme = require('prism-react-renderer/themes/github');
const darkCodeTheme = require('prism-react-renderer/themes/dracula');

// With JSDoc @type annotations, IDEs can provide config autocompletion
/** @type {import('@docusaurus/types').DocusaurusConfig} */
module.exports = {
  title: 'Beluga-AI',
  tagline: 'A Go framework for building sophisticated AI and agentic applications.',
  url: 'https://lookatitude.github.io',
  baseUrl: '/beluga-ai/',
  onBrokenLinks: 'warn', // Changed from 'throw' to 'warn'
  onBrokenMarkdownLinks: 'warn',
  favicon: 'img/favicon.ico', // We'll need to create this or use a default
  organizationName: 'lookatitude', // Your GitHub org/user name.
  projectName: 'beluga-ai', // Your repo name.

  presets: [
    [
      '@docusaurus/preset-classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          sidebarPath: require.resolve('./sidebars.js'),
          editUrl: 'https://github.com/lookatitude/beluga-ai/tree/main/website/',
        },
        blog: {
          path: 'blog',
          showReadingTime: true,
          editUrl:
            'https://github.com/lookatitude/beluga-ai/tree/main/website/blog/',
        },
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      navbar: {
        title: 'Beluga-AI',
        logo: {
          alt: 'Beluga-AI Logo',
          src: 'img/beluga-logo.svg',
        },
        items: [
          {
            type: 'doc',
            docId: 'intro',
            position: 'left',
            label: 'Documentation',
          },
          // {to: '/examples', label: 'Examples', position: 'left'}, // We will create this later
          // {to: '/blog', label: 'Blog', position: 'left'}, // Hidden until we have proper content
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
            title: 'Docs',
            items: [
              {
                label: 'Introduction',
                to: '/docs/intro',
              },
              {
                label: 'API Reference',
                to: '/docs/api', // We will create this later
              },
            ],
          },
          {
            title: 'Community',
            items: [
              {
                label: 'GitHub Discussions',
                href: 'https://github.com/lookatitude/beluga-ai/discussions',
              },
              // Add other community links if any
            ],
          },
          {
            title: 'More',
            items: [
              // {
              //   label: 'Blog',
              //   to: '/blog',
              // }, // Hidden until we have proper content
              {
                label: 'GitHub',
                href: 'https://github.com/lookatitude/beluga-ai',
              },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} Beluga-AI Project. Built with Docusaurus.`,
      },
      prism: {
        theme: lightCodeTheme,
        darkTheme: darkCodeTheme,
        additionalLanguages: ['go'],
      },
    }),
};
