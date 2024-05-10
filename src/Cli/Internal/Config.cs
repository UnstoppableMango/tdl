namespace UnMango.Tdl.Cli.Internal;

// TODO: Move to real config
internal static class Config
{
	private static readonly string AppData = Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData);

	public const string ContainerTag =
#if DEBUG
		"local";
#else
		"main";
#endif

	public static readonly string SocketDir = Path.Join(AppData, "tdl");
}
