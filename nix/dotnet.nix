{
  buildDotnetModule,
  dotnetCorePackages,
  lib,
  version,
}:
buildDotnetModule {
  pname = "tdl-dotnet-debug";
  inherit version;

  # The tests replay testdata/plugin and read reverse.md's loss codes.
  src = lib.fileset.toSource {
    root = ../.;
    fileset = lib.fileset.unions [
      ../dotnet
      ../testdata/plugin
      ../docs/design/reverse.md
    ];
  };

  projectFile = "dotnet/Tdl.Plugin.Debug/Tdl.Plugin.Debug.csproj";
  testProjectFile = "dotnet/Tdl.Plugin.Tests/Tdl.Plugin.Tests.csproj";
  # Every package the solution restores but those the SDK carries: the
  # runtime packs and the trimming tasks.
  nugetDeps = ../dotnet/deps.json;

  dotnet-sdk = dotnetCorePackages.sdk_10_0;
  dotnet-runtime = dotnetCorePackages.runtime_10_0;

  doCheck = true;
  executables = [ "tdl-gen-dotnet_debug" ];

  meta = {
    description = "The TDL debug backend, written on the .NET plugin SDK";
    homepage = "https://github.com/UnstoppableMango/tdl";
    mainProgram = "tdl-gen-dotnet_debug";
    license = lib.licenses.mit;
  };
}
