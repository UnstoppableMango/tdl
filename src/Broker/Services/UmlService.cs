using Google.Protobuf;
using Grpc.Core;

namespace UnMango.Tdl.Broker.Services;

using Tdl = UnMango.Tdl;

public class UmlService : Tdl.UmlService.UmlServiceBase
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
		var data = ByteString.CopyFromUtf8("Hello World!");
		await responseStream.WriteAsync(new GenResponse { Data = data }, context.CancellationToken);
	}
}
