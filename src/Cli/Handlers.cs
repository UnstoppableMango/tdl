using Serilog;

namespace UnMango.Tdl.Cli;

internal static class Handlers
{
	private const int MaxRetries = 10;

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
		Log.Verbose("Creating client channel");
		using var channel = GrpcClient.createChannel(Config.Socket);
		var client = new UmlService.UmlServiceClient(channel);

		var retryCount = 0;
		while (!File.Exists(Config.Socket)) {
			Log.Debug("Waiting for socket: Attempt {RetryCount}", retryCount);
			if (retryCount++ >= MaxRetries)
				throw new Exception("Failed waiting for socket");
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
