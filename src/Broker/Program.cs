using Microsoft.AspNetCore.Server.Kestrel.Core;
using Octokit;
using UnMango.Tdl.Broker.Internal;
using UnMango.Tdl.Broker.Services;

var builder = WebApplication.CreateBuilder(args);

var socket = builder.Environment.IsDevelopment()
	? Path.Combine(Directory.CreateTempSubdirectory("tdl-broker").FullName, "broker.sock")
	: "/var/run/tdl/broker.sock";

// Would be nice to get rid of the "Overriding address(es) warning"
builder.WebHost.ConfigureKestrel(kestrel => {
	kestrel.ListenUnixSocket(socket, listen => {
		listen.Protocols = HttpProtocols.Http2;
	});
});

builder.Services.AddGrpc();
builder.Services.AddHttpClient(HttpClients.GitHub);
builder.Services.AddTransient<IGitHubClient>(_ => new GitHubClient(new ProductHeaderValue(Config.GitHubProduct)));
builder.Services.AddHostedService<PluginService>();

var app = builder.Build();

app.MapGrpcService<UmlService>();
app.MapGet("/", () => "Communication with gRPC endpoints must be made through a gRPC client.");

app.Run();
