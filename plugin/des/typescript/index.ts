import { program } from 'commander';

program
	.argument('<string>', '');

program.parse(Bun.argv);
