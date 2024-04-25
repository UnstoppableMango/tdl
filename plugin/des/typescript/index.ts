import * as http from 'node:http';
import { connectNodeAdapter } from "@connectrpc/connect-node";
import { ConnectRouter, HandlerContext } from "@connectrpc/connect";
import { Greeter, HelloReply, HelloRequest } from "@unmango/tdl-es";

const routes = (router: ConnectRouter) => {
	router.service(Greeter, {
		async sayHello(req: HelloRequest, context: HandlerContext) {
			return new HelloReply({});
		},
	});
}

http.createServer(
	connectNodeAdapter({ routes }),
).listen(8080);
