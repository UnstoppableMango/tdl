import { gen } from '@unmango/2ts';
import { fromBinary } from '@unmango/tdl/spec';

const input = await Bun.stdin.bytes();
const spec = fromBinary(input);
const output = gen(spec);
await Bun.write(Bun.stdout, output);
