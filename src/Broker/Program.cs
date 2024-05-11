using Microsoft.AspNetCore.Server.Kestrel.Core;
using UnMango.Tdl.Broker.Services;

var builder = WebApplication.CreateBuilder(args);

var socket = builder.Environment.IsDevelopment()
	? Path.Combine(Path.GetTempPath(), "broker.sock")
	: "/var/run/tdl/broker.sock";

// Would be nice to get rid of the "Overriding address(es) warning"
builder.WebHost.ConfigureKestrel(kestrel => {
	kestrel.ListenUnixSocket(socket, listen => {
		listen.Protocols = HttpProtocols.Http2;
	});
});

builder.Services.AddGrpc();

var app = builder.Build();

app.MapGrpcService<UmlService>();
app.MapGet("/", () => "Communication with gRPC endpoints must be made through a gRPC client.");

app.Run();
