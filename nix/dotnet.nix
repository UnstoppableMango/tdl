{
  buildDotnetModule,
  darwin,
  dotnetCorePackages,
  lib,
  stdenv,
  swift,
  version,
}:
buildDotnetModule {
  pname = "tdl-dotnet";
  inherit version;

  # The SDK's tests replay testdata/plugin and read reverse.md's loss codes;
  # the C# backend's read the requests in testdata/csharp.
  src = lib.fileset.toSource {
    root = ../.;
    fileset = lib.fileset.unions [
      ../dotnet
      ../testdata/plugin
      ../testdata/csharp
      ../docs/design/reverse.md
    ];
  };

  projectFile = [
    "dotnet/Tdl.Gen.CSharp/Tdl.Gen.CSharp.csproj"
    "dotnet/Tdl.Plugin.Debug/Tdl.Plugin.Debug.csproj"
  ];
  testProjectFile = [
    "dotnet/Tdl.Gen.CSharp.Tests/Tdl.Gen.CSharp.Tests.csproj"
    "dotnet/Tdl.Plugin.Tests/Tdl.Plugin.Tests.csproj"
  ];
  # Every package the solution restores but those the SDK carries: the
  # runtime packs, the trimming tasks, and the AOT compiler.
  nugetDeps = ../dotnet/deps.json;

  dotnet-sdk = dotnetCorePackages.sdk_10_0;
  dotnet-runtime = dotnetCorePackages.runtime_10_0;

  # Each plugin is one file carrying its runtime, so tdl runs it from PATH
  # with no .NET installed. tdl-gen-csharp sets PublishAot, so it is
  # compiled ahead of time and linked by the C compiler; the debug port is
  # a single-file bundle. Symbol files stay out of the package.
  selfContainedBuild = true;
  dotnetInstallFlags = [
    "-p:PublishSingleFile=true"
    "-p:CopyOutputSymbolsToPublishDirectory=false"
  ];
  nativeBuildInputs = [ stdenv.cc ];
  # On darwin the AOT link names -lswiftCore, -lswiftFoundation, and
  # -licucore, which the SDK alone does not resolve. These are the inputs
  # nixpkgs' own dotnet aot test links with.
  buildInputs = lib.optionals stdenv.hostPlatform.isDarwin [
    swift
    darwin.ICU
  ];

  doCheck = true;
  executables = [
    "tdl-gen-csharp"
    "tdl-gen-dotnet_debug"
  ];

  meta = {
    description = "The TDL backends written on the .NET plugin SDK: csharp, and the debug port";
    homepage = "https://github.com/UnstoppableMango/tdl";
    mainProgram = "tdl-gen-csharp";
    license = lib.licenses.mit;
  };
}
