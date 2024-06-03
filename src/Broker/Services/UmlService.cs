using Google.Protobuf;
using Grpc.Core;
using UnMango.Tdl.Broker.Internal;

namespace UnMango.Tdl.Broker.Services;

using Tdl = UnMango.Tdl;

public class UmlService(IPluginCache pluginCache) : Tdl.UmlService.UmlServiceBase
{
	public override async Task<FromResponse> From(
		IAsyncStreamReader<FromRequest> requestStream,
		ServerCallContext context) {
		await Task.Delay(1); // Get async warning to go away because I'm lazy
		return new FromResponse {
			Spec = new Spec {
				Name = "Hello World!",
			},
		};
	}

	public override async Task Gen(
		GenRequest request,
		IServerStreamWriter<GenResponse> responseStream,
		ServerCallContext context) {
		var runner = await pluginCache.Get(request.Target);

		await using var stream = new MemoryStream();
		await runner.GenerateAsync(request.Spec, stream, context.CancellationToken);
		var data = await ByteString.FromStreamAsync(stream, context.CancellationToken);

		await responseStream.WriteAsync(
			new GenResponse { Data = data },
			context.CancellationToken);
	}
}
