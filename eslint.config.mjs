// @ts-check

import eslint from "@eslint/js";
import tslint from "typescript-eslint";

export default tslint.config(
	eslint.configs.recommended,
	...tslint.configs.recommendedTypeChecked,
	{
		ignores: [
			".config/",
			".idea/",
			".make",
			".vscode/",
			"bin/",
			"gen/",
			"obj/",
			"proto/",
		],
		languageOptions: {
			parserOptions: {
				project: ["tsconfig.eslint.json", "plugin/gen/*/tsconfig.json"],
				tsconfigRootDir: import.meta.dirname,
			},
		},
	},
);
