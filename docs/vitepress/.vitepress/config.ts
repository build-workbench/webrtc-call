import { defineConfig } from 'vitepress'

export default defineConfig({
  base: '/webrtc/',
  title: 'LessUp WebRTC 文档',
  description: 'Go 信令服务器 + 原生 JavaScript',

  // Output to docs/.site for GitHub Pages
  outDir: '../.site',
  cacheDir: '.vitepress/cache',

  locales: {
    zh: {
      label: '简体中文',
      lang: 'zh-CN',
      link: '/zh/',
      themeConfig: {
        nav: [
          { text: '技术指南', link: '/zh/guide', activeMatch: '/zh/' },
          { text: '部署', link: '/zh/deployment' },
          { text: 'API 参考', link: '/zh/api' },
          { text: 'GitHub', link: 'https://github.com/LessUp/webrtc' },
        ],
        sidebar: {
          '/zh/': [
            {
              text: '文档',
              items: [
                { text: '简介', link: '/zh/' },
                { text: '技术指南', link: '/zh/guide' },
                { text: '信令协议', link: '/zh/signaling' },
                { text: '部署', link: '/zh/deployment' },
                { text: 'API 参考', link: '/zh/api' },
                { text: 'OpenSpec', link: '/zh/specs' },
                { text: '故障排查', link: '/zh/troubleshooting' },
              ],
            },
          ],
        },
      },
    },
  },

  themeConfig: {
    outline: [2, 3],
    search: { provider: 'local' },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/LessUp/webrtc' },
    ],
  },
})
