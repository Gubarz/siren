import js from '@eslint/js';
import globals from 'globals';
import svelte from 'eslint-plugin-svelte';
import tailwindcss from 'eslint-plugin-tailwindcss';
import boundaries from 'eslint-plugin-boundaries';

const CUSTOM_BUTTON_CLASS_NAMES = [
  'icon-btn',
  'pick-btn',
  'clear-btn',
  'tool-icon',
  'zoom-btn',
  'action-btn',
  'close-btn',
];
const customButtonClassGroup = CUSTOM_BUTTON_CLASS_NAMES.join('|');

const buttonClassChecks = [
  {
    regex: /(^|[^\w-])\.(btn(?:-[A-Za-z0-9_]+)?)(?![\w-])/g,
    message: 'Do not define .btn* CSS selectors. Use $components/ui/Button.svelte or IconButton.svelte.',
  },
  {
    regex: /\bclass:btn(?:-[A-Za-z0-9_]+)?(?![\w-])/g,
    message: 'Do not use class:btn* directives. Use Button variant/size props instead.',
  },
  {
    regex: /(["'`])(?:btn(?:-[A-Za-z0-9_]+)?(?=$|[\s"'`])|[^"'`\n]*\sbtn(?:-[A-Za-z0-9_]+)?(?=$|[\s"'`]))[^"'`\n]*\1/g,
    message: 'Do not use btn* class tokens. Use $components/ui/Button.svelte or IconButton.svelte.',
  },
  {
    regex: new RegExp(`(^|[^\\w-])\\.(${customButtonClassGroup})(?![\\w-])`, 'g'),
    message: 'Do not define custom button CSS selectors (icon-btn, pick-btn, clear-btn, tool-icon, zoom-btn, action-btn, close-btn). Use $components/ui/Button.svelte or IconButton.svelte.',
  },
  {
    regex: new RegExp(`(["'\`])(?:(?:${customButtonClassGroup})(?=$|[\\s"'\`])|[^"'\`\\n]*\\s(?:${customButtonClassGroup})(?=$|[\\s"'\`]))[^"'\`\\n]*\\1`, 'g'),
    message: 'Do not use custom button class tokens (icon-btn, pick-btn, clear-btn, tool-icon, zoom-btn, action-btn, close-btn). Use $components/ui/Button.svelte or IconButton.svelte.',
  },
];

// Directories where raw <button> is banned; use Button/IconButton from $components/ui instead.
const RAW_BUTTON_BANNED_DIRS = [
  '/src/lib/features/',
  '/src/lib/components/forms/',
];

// Directories where raw text-like <input> is banned; use TextInput/SearchInput from $components/ui instead.
const RAW_INPUT_BANNED_DIRS = [
  '/src/lib/features/',
  '/src/lib/components/forms/',
];

// Input types that must remain raw (there's no UI primitive to swap them for).
const RAW_INPUT_ALLOWED_TYPES = new Set([
  'checkbox', 'radio', 'file', 'hidden', 'range', 'color', 'date', 'time', 'datetime-local', 'month', 'week',
]);

// Files allowed to import Flowbite's Button directly. Everyone else must use $components/ui/Button.
const FLOWBITE_BUTTON_ALLOWLIST = [
  '/src/lib/components/ui/Button.svelte',
];

// The Checkbox primitive itself is the only file allowed to render a raw <input type="checkbox">.
const RAW_CHECKBOX_ALLOWLIST = [
  '/src/lib/components/ui/Checkbox.svelte',
];

// Files allowed to import Flowbite's Dropdown/DropdownItem directly. Everyone else must use $components/ui/Select, Menu, or MenuItem.
const FLOWBITE_DROPDOWN_ALLOWLIST = [
  '/src/lib/components/ui/Select.svelte',
  '/src/lib/components/ui/Menu.svelte',
  '/src/lib/components/ui/MenuItem.svelte',
];

const localRules = {
  rules: {
    'no-btn-class': {
      meta: {
        type: 'problem',
        docs: {
          description: 'disallow btn* and custom button classes in app source',
        },
        schema: [],
      },
      create(context) {
        const filename = context.filename ?? context.getFilename?.() ?? '';
        if (!filename.includes('/src/')) return {};

        return {
          Program(node) {
            const sourceCode = context.sourceCode ?? context.getSourceCode();
            const text = sourceCode.getText();

            for (const check of buttonClassChecks) {
              check.regex.lastIndex = 0;
              for (const match of text.matchAll(check.regex)) {
                context.report({
                  node,
                  loc: sourceCode.getLocFromIndex(match.index),
                  message: check.message,
                });
              }
            }
          },
        };
      },
    },
    'no-raw-button': {
      meta: {
        type: 'problem',
        docs: {
          description: 'disallow raw <button> elements outside UI/system/layout primitives',
        },
        schema: [],
      },
      create(context) {
        const filename = context.filename ?? context.getFilename?.() ?? '';
        if (!RAW_BUTTON_BANNED_DIRS.some((dir) => filename.includes(dir))) return {};
        if (!filename.endsWith('.svelte')) return {};

        return {
          Program(node) {
            const sourceCode = context.sourceCode ?? context.getSourceCode();
            const text = sourceCode.getText();
            const regex = /<button(?=[\s>/])/g;
            for (const match of text.matchAll(regex)) {
              context.report({
                node,
                loc: sourceCode.getLocFromIndex(match.index),
                message: 'Do not use raw <button>. Use $components/ui/Button.svelte or IconButton.svelte.',
              });
            }
          },
        };
      },
    },
    'no-raw-input': {
      meta: {
        type: 'problem',
        docs: {
          description: 'disallow raw text-like <input> in feature/form code; use $components/ui/TextInput or SearchInput',
        },
        schema: [],
      },
      create(context) {
        const filename = context.filename ?? context.getFilename?.() ?? '';
        if (!RAW_INPUT_BANNED_DIRS.some((dir) => filename.includes(dir))) return {};
        if (!filename.endsWith('.svelte')) return {};

        return {
          Program(node) {
            const sourceCode = context.sourceCode ?? context.getSourceCode();
            const text = sourceCode.getText();
            const regex = /<input\b([^>]*)>/g;
            for (const match of text.matchAll(regex)) {
              const attrs = match[1] || '';
              const typeMatch = attrs.match(/\btype\s*=\s*["']([^"']+)["']/);
              const rawType = typeMatch ? typeMatch[1].toLowerCase() : 'text';
              if (RAW_INPUT_ALLOWED_TYPES.has(rawType)) continue;
              context.report({
                node,
                loc: sourceCode.getLocFromIndex(match.index),
                message: `Do not use raw <input type="${rawType}">. Use $components/ui/TextInput.svelte or SearchInput.svelte.`,
              });
            }
          },
        };
      },
    },
    'no-raw-textarea': {
      meta: {
        type: 'problem',
        docs: {
          description: 'disallow raw <textarea> in feature/form code; use $components/ui/TextArea',
        },
        schema: [],
      },
      create(context) {
        const filename = context.filename ?? context.getFilename?.() ?? '';
        if (!RAW_INPUT_BANNED_DIRS.some((dir) => filename.includes(dir))) return {};
        if (!filename.endsWith('.svelte')) return {};

        return {
          Program(node) {
            const sourceCode = context.sourceCode ?? context.getSourceCode();
            const text = sourceCode.getText();
            const regex = /<textarea(?=[\s>/])/g;
            for (const match of text.matchAll(regex)) {
              context.report({
                node,
                loc: sourceCode.getLocFromIndex(match.index),
                message: 'Do not use raw <textarea>. Use $components/ui/TextArea.svelte.',
              });
            }
          },
        };
      },
    },
    'no-raw-checkbox': {
      meta: {
        type: 'problem',
        docs: {
          description: 'disallow raw <input type="checkbox"> outside the Checkbox primitive',
        },
        schema: [],
      },
      create(context) {
        const filename = context.filename ?? context.getFilename?.() ?? '';
        if (!filename.includes('/src/')) return {};
        if (!filename.endsWith('.svelte')) return {};
        if (RAW_CHECKBOX_ALLOWLIST.some((allowed) => filename.endsWith(allowed))) return {};

        return {
          Program(node) {
            const sourceCode = context.sourceCode ?? context.getSourceCode();
            const text = sourceCode.getText();
            const regex = /<input\b[^>]*\btype\s*=\s*["']checkbox["'][^>]*>/g;
            for (const match of text.matchAll(regex)) {
              context.report({
                node,
                loc: sourceCode.getLocFromIndex(match.index),
                message: 'Do not render <input type="checkbox"> directly. Use $components/ui/Checkbox.svelte.',
              });
            }
          },
        };
      },
    },
    'no-flowbite-button-import': {
      meta: {
        type: 'problem',
        docs: {
          description: 'disallow direct Flowbite Button imports outside the UI Button wrapper',
        },
        schema: [],
      },
      create(context) {
        const filename = context.filename ?? context.getFilename?.() ?? '';
        if (!filename.includes('/src/')) return {};
        if (FLOWBITE_BUTTON_ALLOWLIST.some((allowed) => filename.endsWith(allowed))) return {};

        return {
          ImportDeclaration(node) {
            if (node.source.value !== 'flowbite-svelte') return;
            for (const spec of node.specifiers) {
              if (spec.type !== 'ImportSpecifier') continue;
              if (spec.imported && spec.imported.name === 'Button') {
                context.report({
                  node: spec,
                  message: 'Do not import Button from flowbite-svelte. Use $components/ui/Button.svelte.',
                });
              }
            }
          },
        };
      },
    },
    'no-flowbite-dropdown-import': {
      meta: {
        type: 'problem',
        docs: {
          description: 'disallow direct Flowbite Dropdown/DropdownItem imports outside the UI Select/Menu wrappers',
        },
        schema: [],
      },
      create(context) {
        const filename = context.filename ?? context.getFilename?.() ?? '';
        if (!filename.includes('/src/')) return {};
        if (FLOWBITE_DROPDOWN_ALLOWLIST.some((allowed) => filename.endsWith(allowed))) return {};

        return {
          ImportDeclaration(node) {
            if (node.source.value !== 'flowbite-svelte') return;
            for (const spec of node.specifiers) {
              if (spec.type !== 'ImportSpecifier') continue;
              const name = spec.imported && spec.imported.name;
              if (name === 'Dropdown' || name === 'DropdownItem' || name === 'DropdownDivider') {
                context.report({
                  node: spec,
                  message: `Do not import ${name} from flowbite-svelte. Use $components/ui/Select.svelte, Menu.svelte, or MenuItem.svelte.`,
                });
              }
            }
          },
        };
      },
    },
    'no-fractional-spacing': {
      meta: {
        type: 'problem',
        docs: {
          description: 'disallow fractional or arbitrary layout/spacing classes to ensure layout rhythm consistency',
        },
        schema: [],
      },
      create(context) {
        const filename = context.filename ?? context.getFilename?.() ?? '';
        if (!filename.includes('/src/')) return {};

        return {
          Program(node) {
            const sourceCode = context.sourceCode ?? context.getSourceCode();
            const text = sourceCode.getText();
            // Match any class starting with p-, m-, gap-, or space- followed by a decimal or arbitrary [value]
            const regex = /\b(p[xytbrlse]?|m[xytbrlse]?|gap(?:-[xy])?|space-[xy])-([0-9]+\.[0-9]+|\[.*?\])\b/g;

            for (const match of text.matchAll(regex)) {
              context.report({
                node,
                loc: sourceCode.getLocFromIndex(match.index),
                message: `Do not use non-standard spacing class "${match[0]}". Use standard integer values (e.g., mt-2, p-4) to keep the layout rhythm consistent.`,
              });
            }
          },
        };
      },
    },
    'no-style-blocks': {
      meta: {
        type: 'problem',
        docs: { description: 'disallow <style> blocks in .svelte files' },
        schema: [],
      },
      create(context) {
        const filename = context.filename ?? context.getFilename?.() ?? '';
        if (!filename.endsWith('.svelte')) return {};

        return {
          Program(node) {
            const sourceCode = context.sourceCode ?? context.getSourceCode();
            const text = sourceCode.getText();
            const regex = /<style\b[^>]*>/g;

            for (const match of text.matchAll(regex)) {
              context.report({
                node,
                loc: sourceCode.getLocFromIndex(match.index),
                message: 'Do not use <style> blocks in .svelte files. Use Tailwind utility classes; put unavoidable global rules in src/styles/main.css.',
              });
            }
          },
        };
      },
    },
    'no-arbitrary-tailwind-value': {
      meta: {
        type: 'problem',
        docs: { description: 'disallow arbitrary Tailwind class-[…] values' },
        schema: [],
      },
      create(context) {
        const filename = context.filename ?? context.getFilename?.() ?? '';
        if (!filename.includes('/src/')) return {};

        return {
          Program(node) {
            const sourceCode = context.sourceCode ?? context.getSourceCode();
            const text = sourceCode.getText();
            // Match any Tailwind utility with an arbitrary [...] value.
            // Supports nested parens so shadow-[0_0_6px_var(--color-x)] is caught.
            const regex = /\b[a-z][a-z0-9-]*(?::[a-z0-9-]+)?-\[(?:[^[\]]|\[[^\]]*\])+\](?!:)/g;

            for (const match of text.matchAll(regex)) {
              const cls = match[0];
              // Allow font-[inherit] — the [inherit] form is Tailwind's normal
              // way to spell "use the parent's font-family", not an escape hatch.
              if (cls === 'font-[inherit]') continue;
              // Allow the canonical CSS-var shorthand: class-(--var).
              // Suggest using it when the arbitrary value is just var(--...).
              const bareVar = cls.match(/^([a-z][a-z0-9-]*(?::[a-z0-9-]+)?)-\[var\((--[^)]+)\)\]$/);
              const suggestion = bareVar
                ? ` Use "${bareVar[1]}-(${bareVar[2]})" instead.`
                : ' Use a theme token or add the color/size to the theme so the utility can be used without brackets.';
              context.report({
                node,
                loc: sourceCode.getLocFromIndex(match.index),
                message: `Do not use arbitrary Tailwind value "${cls}".${suggestion}`,
              });
            }
          },
        };
      },
    },
  },
};

export default [
  {
    ignores: ['dist/**', 'node_modules/**', 'wailsjs/**'],
  },
  js.configs.recommended,
  ...svelte.configs.recommended,
  tailwindcss.configs.recommended,
  {
    // The tailwindcss recommended config enables its rules on **/*.js
    // globally (including test/ and config files). Settings must be
    // global too, or those files fall back to the plugin's default
    // cssConfigPath (src/style.css) and crash with ENOENT.
    settings: {
      tailwindcss: {
        cssConfigPath: 'src/styles/main.css',
      },
    },
  },
  {
    files: ['src/**/*.{js,svelte}', '*.js'],
    plugins: {
      local: localRules,
      tailwindcss,
      boundaries,
    },
    languageOptions: {
      ecmaVersion: 'latest',
      sourceType: 'module',
      globals: {
        ...globals.browser,
        ...globals.node,
      },
    },
    rules: {
      'no-empty': ['error', { allowEmptyCatch: true }],
      'local/no-btn-class': 'error',
      'local/no-raw-button': 'error',
      'local/no-raw-checkbox': 'error',
      'local/no-raw-input': 'error',
      'local/no-raw-textarea': 'error',
      'local/no-flowbite-button-import': 'error',
      'local/no-flowbite-dropdown-import': 'error',
      'local/no-fractional-spacing': 'error',
      'local/no-style-blocks': 'error',
      'local/no-arbitrary-tailwind-value': 'error',
      'svelte/require-each-key': 'off',
      'svelte/prefer-svelte-reactivity': 'off',
      'svelte/prefer-writable-derived': 'off',
      'svelte/infinite-reactive-loop': 'off',
      'svelte/no-immutable-reactive-statements': 'off',
      'svelte/no-at-html-tags': 'off',
      'svelte/no-restricted-html-elements': ['error', {
        elements: ['select'],
        message: 'Use the Flowbite-backed $components/ui/Select.svelte dropdown instead of native <select>.',
      }],

      'no-restricted-imports': ['error', {
        paths: [
          { name: 'svelte/store', message: 'This app is Svelte 5 rune-based only. Do not use writable/readable/derived/get/fromStore. See src/lib/stores/README.md.' },
        ],
        patterns: [
          { group: ['lucide-svelte*'], message: 'Use <Icon name="..." /> from $components/ui/Icon.svelte' },
          { group: ['**/wailsjs/**'], message: 'Feature/store/pattern code must call the backend through src/lib/api/*.js — never import the wailsjs bindings directly.' },
        ],
      }],

      'tailwindcss/classnames-order': 'warn',
      'tailwindcss/no-contradicting-classname': 'error',
      'tailwindcss/no-custom-classname': 'warn',

      'boundaries/dependencies': ['warn', {
        default: 'disallow',
        policies: [
          {
            from: { element: { types: 'features' } },
            allow: { to: { element: { types: { anyOf: ['ui', 'patterns', 'stores', 'api', 'utils', 'features'] } } } },
          },
          {
            from: { element: { types: 'patterns' } },
            allow: { to: { element: { types: { anyOf: ['ui', 'stores', 'api', 'utils'] } } } },
          },
          {
            from: { element: { types: 'ui' } },
            allow: { to: { element: { types: 'utils' } } },
          },
          {
            from: { element: { types: 'system' } },
            allow: { to: { element: { types: { anyOf: ['ui', 'patterns', 'stores', 'api'] } } } },
          },
          {
            from: { element: { types: 'stores' } },
            allow: { to: { element: { types: { anyOf: ['api', 'utils', 'stores'] } } } },
          },
          {
            from: { element: { types: 'api' } },
            allow: { to: { element: { types: 'utils' } } },
          },
        ],
      }],
    },
    settings: {
      'boundaries/elements': [
        { type: 'ui', pattern: 'src/lib/components/ui/**/*' },
        { type: 'patterns', pattern: 'src/lib/components/patterns/**/*' },
        { type: 'system', pattern: 'src/lib/components/system/**/*' },
        { type: 'features', pattern: 'src/lib/features/**/*' },
        { type: 'stores', pattern: 'src/lib/stores/**/*' },
        { type: 'api', pattern: 'src/lib/api/**/*' },
        { type: 'utils', pattern: 'src/lib/utils/**/*' },
      ],
    },
  },
  {
    // The api/ layer is the ONE place wailsjs imports belong. Everywhere else
    // routes through it — see no-restricted-imports above.
    files: ['src/lib/api/**/*.{js,svelte}'],
    rules: {
      'no-restricted-imports': ['error', {
        patterns: [
          { group: ['lucide-svelte*'], message: 'Use <Icon name="..." /> from $components/ui/Icon.svelte' },
        ],
      }],
    },
  },
];
