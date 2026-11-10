{
  buildDotnetModule,
  dotnetCorePackages,
  lib,
  version,
}:
# tdl-gen-csharp as the .NET tool nuget.org carries: Tdl.Gen.CSharp, framework
# dependent, so one package serves every platform. The release publishes
# $out/Tdl.Gen.CSharp.<version>.nupkg; checks.dotnet-tool installs it.
buildDotnetModule {
  pname = "tdl-gen-csharp-tool";
  inherit version;

  src = lib.fileset.toSource {
    root = ../.;
    fileset = lib.fileset.unions [
      ../dotnet/Directory.Build.props
      ../dotnet/Tdl.Plugin
      ../dotnet/Tdl.DotNet
      ../dotnet/Tdl.Gen.CSharp
    ];
  };

  projectFile = "dotnet/Tdl.Gen.CSharp/Tdl.Gen.CSharp.csproj";
  nugetDeps = ../dotnet/deps.json;

  dotnet-sdk = dotnetCorePackages.sdk_10_0;
  dotnet-runtime = dotnetCorePackages.runtime_10_0;

  # The configure hook restores for this system's runtime, which also
  # restores the runtime-independent target the tool package builds. Packing
  # with no runtime and PublishAot off, which the csproj sets for
  # nix/dotnet.nix, gives a package that names no runtime and asks for any
  # .NET 10, not the patch AOT targets.
  buildPhase = ''
    runHook preBuild

    have=$(sed -n 's|.*<Version>\(.*\)</Version>.*|\1|p' dotnet/Tdl.Gen.CSharp/Tdl.Gen.CSharp.csproj)
    if [ "$have" != "${version}" ]; then
      echo "Tdl.Gen.CSharp.csproj is at $have, the flake at ${version}" >&2
      exit 1
    fi

    dotnet pack $dotnetProjectFiles \
      --configuration Release \
      --no-restore \
      --output pack \
      -p:PublishAot=false \
      -p:ContinuousIntegrationBuild=true

    runHook postBuild
  '';

  installPhase = ''
    runHook preInstall
    mkdir -p $out
    cp pack/Tdl.Gen.CSharp.${version}.nupkg $out/
    runHook postInstall
  '';

  dontFixup = true;

  meta = {
    description = "tdl-gen-csharp packed as a framework-dependent .NET tool";
    homepage = "https://github.com/UnstoppableMango/tdl";
    license = lib.licenses.mit;
  };
}
