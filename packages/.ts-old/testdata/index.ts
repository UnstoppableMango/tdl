import fs from 'node:fs'
import path from 'node:path';

export type Test = [name: string, source: string, target: string];

function testFromDir(dirent: fs.Dirent): Test {
	let source: string = '', target: string = '';
	const dir = path.join(dirent.path, dirent.name);

	fs.readdirSync(dir).forEach(fileName => {
		const file = path.join(dir, fileName);
		switch (path.basename(file, path.extname(file))) {
			case 'source':
				source = fs.readFileSync(file, 'utf-8');
				break;
			case 'target':
				target = fs.readFileSync(file, 'utf-8');
				break;
		}
	});

	if (source.length <= 0 || target.length <= 0) {
		throw new Error(`Invalid testdata in ${dir}`);
	}

	return [dirent.name, source, target];
}

export const tests = fs.readdirSync(import.meta.dir, { withFileTypes: true })
	.filter(x => x.isDirectory())
	.map(testFromDir);
