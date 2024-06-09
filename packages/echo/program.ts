import { Command, Option } from '@commander-js/extra-typings';
import * as uml from '@unmango/uml';
import * as cmd from './command';
import { name, version } from './package.json';

const mimeTypeOption = new Option('--type <TYPE>', 'The media type of the input.')
	.choices(uml.SUPPORTED_MIME_TYPES);

export const from = (program: Command): Command => {
	program.command('from')
		.description('Write back the protobuf data.')
		.addOption(mimeTypeOption)
		.action((_) => cmd.from());

	return program;
};

export const gen = (program: Command): Command => {
	program.command('gen')
		.description('Also, write back the protobuf data.')
		.addOption(mimeTypeOption)
		.action((_) => cmd.gen());

	return program;
};

export const program = (): Command =>
	new Command()
		.name(name)
		.description('Plugin to write back what you give it.')
		.version(version)
		.helpOption();
