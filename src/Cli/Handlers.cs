using Serilog;

namespace UnMango.Tdl.Cli;

internal static class Handlers
{
	public static async Task<int> From(
		string source,
		string? target,
		IEnumerable<FileInfo> files,
		CancellationToken cancellationToken) {
		Console.WriteLine("Delay to prove it works");
		await Task.Delay(TimeSpan.FromSeconds(2), cancellationToken);
		return 0;
	}

	public static async Task<int> Gen(
		string target,
		string? source,
		IEnumerable<FileInfo> files,
		CancellationToken cancellationToken) {
		var config = new Config(Env.Dev);
		Log.Verbose("Creating client channel");
		using var channel = Client.createChannel(config.Socket);
		var client = new UmlService.UmlServiceClient(channel);

		while (!File.Exists(config.Socket)) {
			Log.Debug("Waiting for socket");
		}

		try {
			Log.Debug("Sending gen request");
			using var stream = client.Gen(new GenRequest(), cancellationToken: cancellationToken);

			Log.Verbose("Streaming response");
			while (await stream.ResponseStream.MoveNext(cancellationToken)) {
				Log.Verbose("Received response");
				var message = stream.ResponseStream.Current.Data.ToStringUtf8();
				Console.WriteLine(message);
			}
		}
		catch (Exception e) {
			Log.Error(e, "Request failed");
			return 1;
		}

		return 0;
	}
}
