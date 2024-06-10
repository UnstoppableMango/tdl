import { Command, Option } from '@commander-js/extra-typings';
import * as tdl from '@unmango/tdl';
import * as cmd from './command';
import { name, version } from './package.json';

const mimeTypeOption = new Option('--type <TYPE>', 'The media type of the input.')
	.choices(tdl.SUPPORTED_MIME_TYPES);

export const gen = (program: Command): Command =>
	program.command('gen')
		.description('Generate typescript.')
		.addOption(mimeTypeOption)
		.action((opts) => cmd.gen(opts.type));

export const program = (): Command =>
	new Command()
		.name(name)
		.description('Plugin to convert UML to typescript.')
		.version(version)
		.helpOption();
