using System.CommandLine;
using UnMango.Tdl.Cli.Internal;

namespace UnMango.Tdl.Cli.Broker;

internal static class BrokerCommands
{
	public static Command Start() {
		var command = new Command("start", "Start the broker");
		command.SetHandler(Handlers.Start, Binder.Service<IBroker>(), TokenBinder.Value);
		return command;
	}

	public static Command Status() {
		var command = new Command("status", "Check the status of the broker");
		command.SetHandler(Handlers.Status, Binder.Service<IBroker>(), ConsoleBinder.Value, TokenBinder.Value);
		return command;
	}

	public static Command Stop() {
		var command = new Command("stop", "Stop the broker");
		command.SetHandler(Handlers.Stop, Binder.Service<IBroker>(), TokenBinder.Value);
		return command;
	}

	public static Command Upgrade() {
		var latestOption = new Option<bool>("--latest", "Upgrade to the latest version");
		latestOption.AddAlias("-l");

		var versionOption = new Option<string>("--version", "The version to upgrade to");
		versionOption.AddAlias("-v");

		var command = new Command("upgrade", "Upgrade the broker") {
			latestOption,
			versionOption,
		};

		command.SetHandler(
			Handlers.Upgrade,
			Binder.Service<IBroker>(),
			latestOption,
			versionOption,
			TokenBinder.Value);

		return command;
	}
}
