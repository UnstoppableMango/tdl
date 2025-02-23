using Microsoft.AspNetCore.Server.Kestrel.Core;
using UnMango.Tdl.Lang.Host.Services;

var builder = WebApplication.CreateBuilder(args);
builder.WebHost.ConfigureKestrel(options => {
	if (args.Length > 0) {
		options.ListenUnixSocket(args[0], listen => {
			listen.Protocols = HttpProtocols.Http2;
		});
	}
});

builder.Services.AddGrpc();
builder.Services.AddGrpcReflection();

var app = builder.Build();

app.MapGrpcService<ParserService>();
app.MapGrpcReflectionService();

app.Run();
