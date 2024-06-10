import { Command, Option } from '@commander-js/extra-typings';
import * as tdl from '@unmango/tdl';
import * as cmd from './command';
import { name, version } from './package.json';

const mimeTypeOption = new Option('--type <TYPE>', 'The media type of the input.')
	.choices(tdl.SUPPORTED_MIME_TYPES);

export const from = (program: Command): Command => {
	program.command('from')
		.description('Write back the protobuf data.')
		.addOption(mimeTypeOption)
		.action(async (_) => {
			const echo = new cmd.Echo();
			const spec = await echo.from(Bun.stdin);
			await echo.gen(spec, Bun.stdout);
		});

	return program;
};

export const gen = (program: Command): Command => {
	program.command('gen')
		.description('Also, write back the protobuf data.')
		.addOption(mimeTypeOption)
		.action(async (_) => {
			const echo = new cmd.Echo();
			const spec = await echo.from(Bun.stdin);
			await echo.gen(spec, Bun.stdout);
		});

	return program;
};

export const program = (): Command =>
	new Command()
		.name(name)
		.description('Plugin to write back what you give it.')
		.version(version)
		.helpOption();
