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
			return new ValueTask<IRunner>();

		throw new NotImplementedException();
	}
}
