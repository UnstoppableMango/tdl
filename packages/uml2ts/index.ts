import type { Command } from '@commander-js/extra-typings';
import { gen, program } from './program';

type ApplyCommand = (c: Command) => Command;

const app = (p: Command, ...commands: ApplyCommand[]) => {
	return commands.reduce(x => x, p);
};

await app(program(), gen).parseAsync();
