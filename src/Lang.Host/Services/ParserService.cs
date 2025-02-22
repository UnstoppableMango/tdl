using Grpc.Core;

namespace UnMango.Tdl.Lang.Host.Services;

public class ParserService : Tdl.Parser.ParserBase
{
	public override Task<ParseResponse> Parse(ParseRequest request, ServerCallContext context) {
		var res = new ParseResponse {
			Result = Parser.Parse(request.Data),
		};

		return Task.FromResult(res);
	}
}
