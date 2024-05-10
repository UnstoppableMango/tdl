namespace UnMango.Tdl.Cli.Internal;

// TODO: Move to real config
internal static class Config
{
	private static readonly string AppData = Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData);

#if DEBUG
	public const string ContainerRepo = "unstoppablemango";
	public const string ContainerTag = "local";
#else
	public const string ContainerRepo = "ghcr.io/unstoppablemango";
	public const string ContainerTag = "main";
#endif

	public static readonly string SocketDir = Path.Join(AppData, "tdl");
}
