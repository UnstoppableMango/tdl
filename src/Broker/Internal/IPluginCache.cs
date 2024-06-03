using UnMango.Tdl.Abstractions;

namespace UnMango.Tdl.Broker.Internal;

public interface IPluginCache
{
	ValueTask Add(string name, IRunner runner);

	ValueTask<IRunner> Get(string name);

	ValueTask<IReadOnlyDictionary<string, IRunner>> List();
}
