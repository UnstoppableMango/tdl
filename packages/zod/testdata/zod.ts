import { z } from 'zod';

const schema = z.object({
	thing: z.string(),
}).describe('test');

export default schema;
