import { Command } from '@commander-js/extra-typings';
import { name, version } from './package.json';

const program = new Command()
	.name(name)
	.description('Plugin to convert UML to typescript.')
	.version(version)
	.helpOption();

program.command('gen')
	.description('Generate typescript.')
	.action(async () => {
		if (!process.stdin.readable) {
			throw new Error('need more');
		}

		const buffer = await Bun.stdin.arrayBuffer();
		await gen(new Uint8Array(buffer), process.stdout);
	});

await program.parseAsync();
