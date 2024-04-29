import { Command } from '@commander-js/extra-typings';
import { generator } from '@unmango/2ts';
import * as tdl from '@unmango/tdl-es';
import { name, version } from './package.json';

const program = new Command()
	.name(name)
	.description('Plugin to convert UML to typescript.')
	.version(version)
	.helpOption();

program.command('gen')
	.description('Generate typescript.')
	.action(async () => {
		const bytes = new Uint8Array(0);
		const spec = tdl.Spec.fromBinary(bytes);
		const writer = new WritableStream({
			async write(chunk) {
				await Bun.write(Bun.stdout, chunk);
			},
		});
		await generator.gen(spec, writer);
	});

await program.parseAsync();
