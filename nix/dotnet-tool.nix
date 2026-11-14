{
  buildDotnetModule,
  dotnetCorePackages,
  lib,
  version,
}:
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

  # The csproj sets PublishAot for dotnet.nix's binary. Left on here, the
  # package would require the exact .NET patch the SDK ships (10.0.14)
  # instead of any .NET 10.
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
