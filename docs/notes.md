# Notes to self

As I'm typing out these examples I'm realizing there is a lot of overlap between the commands I've hallucinated.
Intuitively these two commands should behave the same

```shell
um to ts
um gen ts
```

There really isn't a need to differentiate between a specification like protobuf and a language like typescript, they won't ever overlap.

In the CLI tools I can just alias `to` and `gen`.
However, I naively started implementing a number of abstractions for generators and converters that will take a bit of refactoring.
This is what I get for premature optimization and over-engineering shit.
