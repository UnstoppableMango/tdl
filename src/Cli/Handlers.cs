using System.Text;
using System.Text.RegularExpressions;
using CliWrap;
using CliWrap.EventStream;

namespace UnMango.Tdl.Cli;

internal static partial class Handlers
{
	[GeneratedRegex("Content root path: .*")]
	private static partial Regex StartedPattern();

	[GeneratedRegex("Application is shutting down...")]
	private static partial Regex StoppedPattern();

	public static async Task<int> From(
		string source,
		string? target,
		IEnumerable<FileInfo> files,
		CancellationToken cancellationToken) {
		using var graceful = new CancellationTokenSource();
		using var forceful = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
		// forceful.CancelAfter(TimeSpan.FromMinutes(1));
		forceful.CancelAfter(TimeSpan.FromSeconds(10));

		var broker = Broker.Dev.Start("http://127.0.0.1:6969")
			.WithStandardOutputPipe(PipeTarget.ToDelegate(Console.WriteLine))
			.Observe(Encoding.Default, Encoding.Default, forceful.Token, graceful.Token);

		Console.WriteLine("Starting...");
		await broker.First(IsStarted, cancellationToken);
		Console.WriteLine("Started");
		Console.WriteLine("Cancelling...");
		forceful.CancelAfter(TimeSpan.FromSeconds(30));
		await graceful.CancelAsync();
		Console.WriteLine("Cancelled");
		Console.WriteLine("Stopping...");
		await broker.First(IsStopped, cancellationToken);
		Console.WriteLine("Stopped");

		return 0;
	}

	public static Task<int> Gen(string target, string? source, IEnumerable<FileInfo> files) {
		return Task.FromResult(0);
	}

	public static Task<int> To(string target, string? source, IEnumerable<FileInfo> files) {
		return Task.FromResult(0);
	}

	private static bool IsStarted(CommandEvent e) {
		return e is StandardOutputCommandEvent o && StartedPattern().IsMatch(o.Text);
	}

	private static bool IsStopped(CommandEvent e) {
		return e is StandardOutputCommandEvent o && StoppedPattern().IsMatch(o.Text);
	}
}
