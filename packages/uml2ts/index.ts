import type { Command } from '@commander-js/extra-typings';
import { gen, program } from './program';

type ApplyCommand = (c: Command) => Command;

const app = (root: () => Command, ...commands: ApplyCommand[]) => {
	return commands.reduce((x, c) => c(x), root());
};

await app(program, gen).parseAsync();
