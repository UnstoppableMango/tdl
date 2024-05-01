import { Command } from '@commander-js/extra-typings';
import { generator } from '@unmango/2ts';
import { read } from '@unmango/uml';
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
			throw new Error('stdin was not readable');
		}

		const buffer = await Bun.stdin.arrayBuffer();
		const spec = read(new Uint8Array(buffer));
		const ts = await generator.gen(spec);
		process.stdout.write(ts, 'utf-8');
	});

await program.parseAsync();
