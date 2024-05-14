using UnMango.Tdl.Abstractions;

namespace UnMango.Tdl.Broker.Internal;

internal sealed class CliPluginCache : IPluginCache
{
	private readonly Dictionary<string, string> _nameMap = new();

	public void Add(string name, string path) => _nameMap.Add(name, path);

	public async ValueTask<IConverter> GetConverter(string name) {
		return await GetRunner(name);
	}

	public async ValueTask<IGenerator> GetGenerator(string name) {
		return await GetRunner(name);
	}

	public ValueTask<IRunner> GetRunner(string name) {
		if (_nameMap.TryGetValue(name, out var path))
			return new ValueTask<IRunner>(new Runner(name, path));

		throw new NotImplementedException();
	}

	private sealed class Runner(string name, string path) : IRunner
	{
		public Task<Spec> FromAsync(ISource source, CancellationToken cancellationToken = default) {
			throw new NotImplementedException();
		}

		public Task GenerateAsync(Spec spec, ITarget target, CancellationToken cancellationToken = default) {
			throw new NotImplementedException();
		}
	}
}
