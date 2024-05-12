using System.Runtime.InteropServices;
using Octokit;

namespace UnMango.Tdl.Broker.Services;

internal sealed class PluginService(IGitHubClient gitHub, ILogger<PluginService> logger) : BackgroundService
{
	protected override async Task ExecuteAsync(CancellationToken stoppingToken) {
		var latest = await gitHub.Repository.Release.GetLatest("UnstoppableMango", "tdl");
		if (latest is null) {
			logger.LogError("Unable to find latest plugin release");
			return;
		}

		if (!TryGetArchitecture(out var targetArch)) {
			logger.LogError("Failed to matching arch {Arch}", RuntimeInformation.OSArchitecture);
			return;
		}

		if (!TryGetOsAndExt(out var targetOs, out var archiveExt)) {
			logger.LogError("Failed to find matching OS {Os}", RuntimeInformation.OSDescription);
		}

		var assetName = $"tdl_{targetOs}_{targetArch}.{archiveExt}";
		var asset = latest.Assets.FirstOrDefault(a => a.Name == assetName);
		if (asset is null) {
			logger.LogError("Unable to find plugin asset");
			return;
		}

		var assetUrl = asset.BrowserDownloadUrl;
		throw new NotImplementedException();
	}

	private static bool TryGetArchitecture(out string arch) {
		arch = RuntimeInformation.OSArchitecture switch {
			Architecture.Arm64 => "arm64",
			Architecture.X86 => "i386",
			Architecture.X64 => "x86_64",
			_ => string.Empty,
		};

		return string.IsNullOrWhiteSpace(arch);
	}

	private static bool TryGetOsAndExt(out string os, out string ext) {
		if (RuntimeInformation.IsOSPlatform(OSPlatform.Linux)) {
			os = "Linux";
			ext = "tar.gz";
			return true;
		}

		if (RuntimeInformation.IsOSPlatform(OSPlatform.OSX)) {
			os = "Darwin";
			ext = "tar.gz";
			return true;
		}

		if (RuntimeInformation.IsOSPlatform(OSPlatform.Windows)) {
			os = "Windows";
			ext = "zip";
			return true;
		}

		os = string.Empty;
		ext = string.Empty;
		return false;
	}
}
