using UnMango.Tdl.Abstractions;

namespace UnMango.Tdl.Broker.Internal;

internal sealed class MemoryPluginCache : IPluginCache
{
	private readonly Dictionary<string, IRunner> _cache = new();

	public ValueTask Add(string name, IRunner runner) {
		_cache.Add(name, runner);
		return new ValueTask();
	}

	public ValueTask<IRunner> Get(string name) {
		var runner = _cache[name];
		return new ValueTask<IRunner>(runner);
	}

	public ValueTask<IReadOnlyDictionary<string, IRunner>> List() {
		return new ValueTask<IReadOnlyDictionary<string, IRunner>>(_cache);
	}
}
