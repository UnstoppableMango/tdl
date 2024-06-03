namespace UnMango.Tdl.Broker.Internal;

internal static class Config
{
	private static bool RunningInDocker
		=> Environment.GetEnvironmentVariable("DOTNET_RUNNING_IN_CONTAINER") == "true";

	public const string GitHubProduct = "UnstoppableMango-tdl";

	public static readonly string PluginDir = RunningInDocker
		? "/app/plugins"
		: Directory.CreateTempSubdirectory("tdl-plugins-").FullName;
}
