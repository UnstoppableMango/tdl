# UnstoppableMango's Type Description Language

Language for generating code and mapping types between languages.

## Usage

Nothing currently works, but this is how it works in my head.

## Generating

```yaml
# myTypes.uml
# It looks like yaml now, it might not later. idk
name: MyTypes
types:
  someType:
    type: object
    fields:
      someField:
        type: string
```

```shell
$ ux gen ts myTypes.uml
Wrote 69B to ./myTypes.ts
```

```ts
// myTypes.ts
export interface SomeType {
	someField: string;
}
```

## Development

`make` kinda builds everything, or at least its supposed to.
If you run `make` or `make build` everything should build, hopefully.
If it doesn't, make sure you have the stuff listed below.

### Prerequisites

Probably put some links here to install docs.

- `buf`
- `bun`
- `docker`
- `dotnet`
- `go`
- `make`
- `dprint`

Probably good to have but not needed

- `goreleaser`
- `node`
- `nvm`

### Setup

Run `make .envrc` to configure a local `.envrc` based on `hack/example.envrc`.

### Building

Run `make generate` to perform any codegen required by the project.

Run `make build` to build everything.

🚧 Work in progress 🚧

~~Run `make docker` to build all docker images.~~

### Testing

Run `make test` to run all test suites.

### Workflow

Run `make lint` to lint everything.

Run `make clean` to remove local artifacts such as `/.make` targets.

### Repository Structure

|            Directory | Description                                                     |
| -------------------: | :-------------------------------------------------------------- |
|           `/.config` | Just `dotnet` tools at the moment                               |
|           `/.github` | GitHub configuration files                                      |
|   `/.github/actions` | GitHub actions                                                  |
| `/.github/workflows` | GitHub workflows                                                |
|             `/.idea` | JetBrains IDE configuration (I check in some of this, fight me) |
|             `/.make` | Local `make` sentinel target files                              |
|              `/.run` | JetBrains IDE run configurations                                |
|         `/.versions` | Version files for dependency pinning                            |
|           `/.vscode` | VSCode configuration                                            |
|               `/bin` | Binaries                                                        |
|               `/cli` | Old directory for Go CLI applications                           |
|               `/cmd` | Go CLI applications                                             |
|            `/docker` | Dockerfiles                                                     |
|              `/docs` | Any documentation too large for the README                      |
|              `/hack` | Any files that help with hacking on the project such as scripts |
|          `/packages` | Node-ish ecosystem packages and applications                    |
|               `/pkg` | Go packages                                                     |
|             `/proto` | Protobuf definitions                                            |
|               `/src` | .NET ecosystem libraries and applications                       |

## Design Philosophy

- Codegen everything on the path of least resistance.
- Integrate existing tools before writing new ones. i.e. `protoc`, `graphql-codegen`, etc.
- Tools can have overlapping responsibilities. i.e. Two generators can output TypeScript code.
- Developer productivity and ease of use takes priority.
- Output the least amount code to accomplish a task. i.e. Don't generate a `package.json` in a TypeScript generator.

## Architecture

The primary entrypoint is the `ux` CLI.
This tool doesn't perform any codegen on it's own and instead orchestrates codegen pipelines.
Codegen pipelines are primarily composed of generator applications.
A generator application receives a protobuf encoded specification (`Spec`) via stdin and writes its output to stdout.
The `ux` CLI can perform conformance tests on a generator application with `ux conform` to ensure the generator is compatible with `ux`.

The intent behind this design is to allow generators to be written in the language that is most convenient for performing its task.
For example, when generating TypeScript code the `typescript` package contains all of the tools required for reading, manipulating, and writing TypeScript code.
While it would be possible to generate TypeScript in i.e. Go, it is much easier to simply write the generator in a language compatible with `npm` packages.
Additionally, this design should allow for integrating existing codegen tools without needing to compile them into the source language of `ux`, which is currently Go.

### Thoughts from when I started this project

I've got a dozen conflicting ideas but the current path I'm working towards is a primary CLI `um` calling a "runner" CLI `um2something` and communicating between stdin and stdout.

The reason for the binary separation is so that the conversion/generation logic can be written as close to the ecosystem as possible (i.e. we write the typescript converter in TS/JS so we have easy programatic access to the `typescript` package).

I'm aware of gRPC being used for IPC on unix sockets so I thought it could be fun and at least semi-correct to have the two processes communicate this way.
As fun as that might be, I'm worried I might be pushing the limits of "how over-engineered does this _really_ need to be".

CLI tools should be really snappy so the overhead of setting up a gRPC server might be ridiculous. If it's not though... I might do that. It sounds really cool "the user CLI communicates with the runner CLI via gRPC on a unix domain socket".

If we want to REALLY over-engineer everything I was thinking we could have a little broker do-dad that hangs out in the background and loads up plugins that the user CLI can call to convert things. It would be fun and ridiculous.
