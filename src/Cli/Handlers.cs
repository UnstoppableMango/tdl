namespace UnMango.Tdl.Cli;

internal static class Handlers
{
	public static Task<int> From(string source, string? target, IEnumerable<FileInfo> files) {
		return Task.FromResult(0);
	}

	public static Task<int> To(string target, string? source, IEnumerable<FileInfo> files) {
		return Task.FromResult(0);
	}
}
