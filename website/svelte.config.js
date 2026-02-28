import adapter from '@sveltejs/adapter-static';
import { mdsvex } from 'mdsvex';

// TODO(#67): import rehypeSlug from './src/lib/plugins/rehype-slug.js';
// TODO(#67): import rehypeRewriteLinks from './src/lib/plugins/rehype-rewrite-links.js';

const config = {
  extensions: ['.svelte', '.svx', '.md'],
  preprocess: [
    mdsvex({
      extensions: ['.svx', '.md']
      // TODO(#67): add rehypePlugins: [rehypeSlug, rehypeRewriteLinks]
    })
  ],
  kit: {
    adapter: adapter({
      pages: 'build',
      assets: 'build',
      fallback: '404.html',
      precompress: false,
      strict: true
    }),
    paths: {
      base: process.env.BASE_PATH || ''
    },
    prerender: {
      handleHttpError: 'warn',
      handleMissingId: 'warn'
    }
  }
};

export default config;
