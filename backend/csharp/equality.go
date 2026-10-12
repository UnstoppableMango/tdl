package csharp

// equalitySource is TdlEquality, which a record holding a collection or
// bytes compares them through, by content and at any depth. A record's
// generated equality compares a property with EqualityComparer<T>.Default,
// which compares an immutable collection or an array by reference.
//
// A set's elements and a map's keys are never collections or bytes, so they
// are compared by the collection's own comparer.
const equalitySource = `/// <summary>Compares the collections and bytes a generated type holds by their contents.</summary>
internal static class TdlEquality
{
    public static global::System.Collections.Generic.IEqualityComparer<byte[]?> Bytes { get; } = new BytesComparer();

    public static global::System.Collections.Generic.IEqualityComparer<global::System.Collections.Immutable.IImmutableList<T>?> List<T>(global::System.Collections.Generic.IEqualityComparer<T> element) => new ListComparer<T>(element);

    public static global::System.Collections.Generic.IEqualityComparer<global::System.Collections.Immutable.IImmutableSet<T>?> Set<T>() => SetComparer<T>.Instance;

    public static global::System.Collections.Generic.IEqualityComparer<global::System.Collections.Immutable.IImmutableDictionary<TKey, TValue>?> Map<TKey, TValue>(global::System.Collections.Generic.IEqualityComparer<TValue> value)
        where TKey : notnull => new MapComparer<TKey, TValue>(value);

    private sealed class BytesComparer : global::System.Collections.Generic.IEqualityComparer<byte[]?>
    {
        public bool Equals(byte[]? x, byte[]? y) =>
            object.ReferenceEquals(x, y) || x is not null && y is not null && global::System.MemoryExtensions.SequenceEqual<byte>(x, y);

        public int GetHashCode(byte[]? obj)
        {
            if (obj is null)
            {
                return 0;
            }
            var hash = new global::System.HashCode();
            hash.AddBytes(obj);
            return hash.ToHashCode();
        }
    }

    private sealed class ListComparer<T>(global::System.Collections.Generic.IEqualityComparer<T> element) : global::System.Collections.Generic.IEqualityComparer<global::System.Collections.Immutable.IImmutableList<T>?>
    {
        public bool Equals(global::System.Collections.Immutable.IImmutableList<T>? x, global::System.Collections.Immutable.IImmutableList<T>? y)
        {
            if (object.ReferenceEquals(x, y))
            {
                return true;
            }
            if (x is null || y is null || x.Count != y.Count)
            {
                return false;
            }
            for (var i = 0; i < x.Count; i++)
            {
                if (!element.Equals(x[i], y[i]))
                {
                    return false;
                }
            }
            return true;
        }

        public int GetHashCode(global::System.Collections.Immutable.IImmutableList<T>? obj)
        {
            if (obj is null)
            {
                return 0;
            }
            var hash = new global::System.HashCode();
            foreach (var item in obj)
            {
                hash.Add(item, element);
            }
            return hash.ToHashCode();
        }
    }

    private sealed class SetComparer<T> : global::System.Collections.Generic.IEqualityComparer<global::System.Collections.Immutable.IImmutableSet<T>?>
    {
        public static readonly SetComparer<T> Instance = new();

        public bool Equals(global::System.Collections.Immutable.IImmutableSet<T>? x, global::System.Collections.Immutable.IImmutableSet<T>? y) =>
            object.ReferenceEquals(x, y) || x is not null && y is not null && x.SetEquals(y);

        public int GetHashCode(global::System.Collections.Immutable.IImmutableSet<T>? obj)
        {
            if (obj is null)
            {
                return 0;
            }
            // A set has no order, so its elements' hashes are combined by one
            // that has none either.
            var hash = 0;
            foreach (var item in obj)
            {
                hash ^= item is null ? 0 : global::System.Collections.Generic.EqualityComparer<T>.Default.GetHashCode(item);
            }
            return hash;
        }
    }

    private sealed class MapComparer<TKey, TValue>(global::System.Collections.Generic.IEqualityComparer<TValue> value) : global::System.Collections.Generic.IEqualityComparer<global::System.Collections.Immutable.IImmutableDictionary<TKey, TValue>?>
        where TKey : notnull
    {
        public bool Equals(global::System.Collections.Immutable.IImmutableDictionary<TKey, TValue>? x, global::System.Collections.Immutable.IImmutableDictionary<TKey, TValue>? y)
        {
            if (object.ReferenceEquals(x, y))
            {
                return true;
            }
            if (x is null || y is null || x.Count != y.Count)
            {
                return false;
            }
            foreach (var entry in x)
            {
                if (!y.TryGetValue(entry.Key, out var other) || !value.Equals(entry.Value, other))
                {
                    return false;
                }
            }
            return true;
        }

        public int GetHashCode(global::System.Collections.Immutable.IImmutableDictionary<TKey, TValue>? obj)
        {
            if (obj is null)
            {
                return 0;
            }
            // A map has no order, so its entries' hashes are combined by one
            // that has none either.
            var hash = 0;
            foreach (var entry in obj)
            {
                hash ^= global::System.HashCode.Combine(
                    global::System.Collections.Generic.EqualityComparer<TKey>.Default.GetHashCode(entry.Key),
                    entry.Value is null ? 0 : value.GetHashCode(entry.Value));
            }
            return hash;
        }
    }
}
`
