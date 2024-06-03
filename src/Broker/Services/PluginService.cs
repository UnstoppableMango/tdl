using System.Formats.Tar;
using System.IO.Compression;
using System.Runtime.InteropServices;
using Octokit;
using UnMango.Tdl.Broker.Internal;

namespace UnMango.Tdl.Broker.Services;

using Config = UnMango.Tdl.Broker.Internal.Config;

internal sealed class PluginService(
	IGitHubClient gitHub,
	IHttpClientFactory httpClientFactory,
	IPluginCache pluginCache,
	ILogger<PluginService> logger) : BackgroundService
{
	private const bool Overwrite = true;

	protected override async Task ExecuteAsync(CancellationToken stoppingToken) {
		if (!TryGetArchitecture(out var targetArch)) {
			logger.LogError("Failed to matching arch {Arch}", RuntimeInformation.OSArchitecture);
			return;
		}

		if (!TryGetOsAndExt(out var targetOs, out var archiveExt)) {
			logger.LogError("Failed to find matching OS {Os}", RuntimeInformation.OSDescription);
		}

		if (!Directory.Exists(Config.PluginDir)) {
			try {
				logger.LogInformation("Creating plugin directory");
				Directory.CreateDirectory(Config.PluginDir);
			}
			catch (Exception e) {
				logger.LogError(e, "Failed to create plugin directory");
				return;
			}
		}

		var assetName = $"tdl_{targetOs}_{targetArch}.{archiveExt}";
		var assetPath = Path.Combine(Config.PluginDir, assetName);

		if (!File.Exists(assetPath)) {
			logger.LogDebug("Fetching latest github release");
			var latest = await gitHub.Repository.Release.GetLatest("UnstoppableMango", "tdl");
			if (latest is null) {
				logger.LogError("Unable to find latest plugin release");
				return;
			}

			var asset = latest.Assets.FirstOrDefault(a => a.Name == assetName);
			if (asset is null) {
				logger.LogError("Unable to find plugin asset");
				return;
			}

			logger.LogTrace("Creating {Client} HttpClient", HttpClients.GitHub);
			var http = httpClientFactory.CreateClient(HttpClients.GitHub);
			var assetUrl = asset.BrowserDownloadUrl;

			logger.LogInformation("Fetching {Asset} from {Url}", assetName, assetUrl);
			using var response = await http.GetAsync(assetUrl, stoppingToken);

			if (!response.IsSuccessStatusCode) {
				logger.LogError("Failed to fetch asset: {Reason}", response.ReasonPhrase);
				return;
			}

			logger.LogTrace("Reading response stream");
			await using var stream = await response.Content.ReadAsStreamAsync(stoppingToken);

			logger.LogDebug("Opening {File} for writing", assetPath);
			await using var file = File.OpenWrite(assetPath);

			logger.LogDebug("Writing response to file");
			await stream.CopyToAsync(file, stoppingToken);
		}

		switch (archiveExt) {
		case "tar.gz":
			logger.LogInformation("Extracting tar archive");
			await ExtractTarGzAsync(assetPath, Config.PluginDir, stoppingToken);
			break;
		case "zip":
			logger.LogInformation("Extracting zip archive");
			ZipFile.ExtractToDirectory(assetPath, Config.PluginDir, Overwrite);
			break;
		default:
			logger.LogError("Unsupported file extension {Extension}", archiveExt);
			break;
		}

		foreach (var file in Directory.EnumerateFiles(Config.PluginDir, "uml*")) {
			var name = Path.GetFileName(file);
			logger.LogInformation("Adding plugin {Name} at {Path}", name, file);
			await pluginCache.Add(name, new CliRunner(file));
		}
	}

	private static bool TryGetArchitecture(out string arch) {
		arch = RuntimeInformation.OSArchitecture switch {
			Architecture.Arm64 => "arm64",
			Architecture.X86 => "i386",
			Architecture.X64 => "x86_64",
			_ => string.Empty,
		};

		return !string.IsNullOrWhiteSpace(arch);
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

	private async Task ExtractTarGzAsync(string archivePath, string destination, CancellationToken cancellationToken) {
		if (!archivePath.EndsWith(".tar.gz")) {
			logger.LogError("Expected .tar.gz file but got {File}", archivePath);
			return;
		}

		logger.LogTrace("Opening archive stream {Archive}", archivePath);
		await using var archiveStream = File.OpenRead(archivePath);

		logger.LogTrace("Creating gzip stream");
		await using var gzipStream = new GZipStream(archiveStream, CompressionMode.Decompress);

		logger.LogDebug("Extracting tar contents");
		await TarFile.ExtractToDirectoryAsync(gzipStream, destination, Overwrite, cancellationToken);
	}
}
