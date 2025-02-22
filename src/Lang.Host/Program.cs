using UnMango.Tdl.Lang.Host.Services;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddGrpc();

var app = builder.Build();

app.MapGrpcService<ParserService>();

app.Run();
