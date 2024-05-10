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
		Console.WriteLine("Delay to prove it works");
		await Task.Delay(TimeSpan.FromSeconds(2), cancellationToken);
		return 0;
	}
}
