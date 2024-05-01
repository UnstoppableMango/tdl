import { Command, Option } from '@commander-js/extra-typings';
import * as uml from '@unmango/uml';
import * as cmd from './command';
import { name, version } from './package.json';

const mimeTypeOption = new Option('--type <TYPE>', 'The media type of the input.')
	.choices(uml.SUPPORTED_MIME_TYPES);

const io = (): cmd.Io => ({
	stdin: Bun.stdin,
	stdout: Bun.stdout,
});

export const gen = (program: Command): Command =>
	program.command('gen')
		.description('Generate typescript.')
		.addOption(mimeTypeOption)
		.action((opts) => cmd.gen(io(), opts.type));

export const program = (): Command =>
	new Command()
		.name(name)
		.description('Plugin to convert UML to typescript.')
		.version(version)
		.helpOption();
