import { name, version } from './package.json';
import * as net from 'node:net';
import { Command } from '@commander-js/extra-typings';
import { createPromiseClient } from '@connectrpc/connect';
import { createGrpcTransport } from '@connectrpc/connect-node';
import { UmlService } from '@unmango/tdl-es';

const program = new Command()
	.name(name)
	.description('Plugin to convert UML to typescript.')
	.version(version)
	.helpOption()
	.argument('<uml>', 'The thing to do the stuff with')
	.option('--broker <uri>', 'address of the broker');

program.parse(process.argv);
const opts = program.opts();

if (!opts.broker) {
	throw new Error('Broker URI is required');
}

const transport = createGrpcTransport({
	httpVersion: '2',
	baseUrl: opts.broker,
	nodeOptions: {
		createConnection(authority, option) {
			if (!opts.broker) {
				throw new Error('Broker URI is required');
			}

			return net.connect(opts.broker);
		},
	},
});

const client = createPromiseClient(UmlService, transport);
const result = await client.pull({ actor: name });
console.log(result.uml);
