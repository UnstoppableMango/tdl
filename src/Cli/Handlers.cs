namespace UnMango.Tdl.Cli;

internal static class Handlers
{
	public static async Task<int> From(
		string source,
		string? target,
		IEnumerable<FileInfo> files,
		CancellationToken cancellationToken) {
		Console.WriteLine("Delay to prove it works");
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
