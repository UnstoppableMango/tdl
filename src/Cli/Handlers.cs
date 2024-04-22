using CliWrap;

namespace UnMango.Tdl.Cli;

internal static class Handlers
{
	public static async Task<int> From(
		string source,
		string? target,
		IEnumerable<FileInfo> files,
		CancellationToken cancellationToken) {
		var cts = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
		var task = Broker.Dev.Start("http://127.0.0.1:6969")
			.WithStandardOutputPipe(PipeTarget.ToDelegate(Console.WriteLine))
			.ExecuteAsync(cancellationToken, cts.Token);
		Console.WriteLine("Delaying...");
		await Task.Delay(5000, cancellationToken);
		Console.WriteLine("Cancelling...");
		await cts.CancelAsync();
		Console.WriteLine("Cancelled. Waiting...");
		await task;
		await Task.Delay(5000, cancellationToken);
		return 0;
	}

	public static Task<int> Gen(string target, string? source, IEnumerable<FileInfo> files) {
		return Task.FromResult(0);
	}

	public static Task<int> To(string target, string? source, IEnumerable<FileInfo> files) {
		return Task.FromResult(0);
	}
}
