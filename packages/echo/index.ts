import type { Command } from '@commander-js/extra-typings';
import { from,  gen, program } from './program';

type ApplyCommand = (c: Command) => Command;

const app = (root: () => Command, ...commands: ApplyCommand[]) => {
	return commands.reduce((x, c) => c(x), root());
};

await app(program, from, gen).parseAsync();
