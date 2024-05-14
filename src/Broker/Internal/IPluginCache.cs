using UnMango.Tdl.Abstractions;

namespace UnMango.Tdl.Broker.Internal;

public interface IPluginCache
{
	ValueTask<IConverter> GetConverter(string name);

	ValueTask<IGenerator> GetGenerator(string name);

	ValueTask<IRunner> GetRunner(string name);
}
