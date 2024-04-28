// @ts-check

import eslint from '@eslint/js';
import tslint from 'typescript-eslint';

export default tslint.config(
	eslint.configs.recommended,
	...tslint.configs.recommendedTypeChecked,
	{
		ignores: [
			'.config/',
			'.idea/',
			'.make',
			'.vscode/',
			'bin/',
			'gen/',
			'obj/',
			'proto/',
		],
	},
	{
		languageOptions: {
			parserOptions: {
				project: ['tsconfig.eslint.json', 'packages/*/tsconfig.json'],
				tsconfigRootDir: import.meta.dirname,
			},
		},
		rules: {
			// My best guess is that this doesn't play nice with bun yet
			'@typescript-eslint/no-unsafe-argument': 'off',
		},
	},
	{
		files: ['eslint.config.mjs'],
		rules: {
			// Can't seem to source types for this anywhere
			'@typescript-eslint/no-unsafe-member-access': 'off',
		},
	},
	{
		files: ['**/*.js'],
		extends: [tslint.configs.disableTypeChecked],
	},
);
