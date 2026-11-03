{
  buildGoApplication,
  lib,
  go,
  tdl-dotnet,
  version,
}:
buildGoApplication {
  pname = "tdl";
  inherit version go;

  src = lib.cleanSource ../.;
  modules = ./gomod2nix.toml;

  subPackages = [
    "cmd/tdl"
    "cmd/tdl-gen-debug"
    "cmd/tdl-gen-go"
    "cmd/tdl-gen-graphql"
    "cmd/tdl-gen-haskell"
    "cmd/tdl-gen-jsonschema"
    "cmd/tdl-gen-likec4"
    "cmd/tdl-gen-openapi"
    "cmd/tdl-gen-protobuf"
    "cmd/tdl-gen-salesforce"
    "cmd/tdl-gen-smithy"
    "cmd/tdl-gen-thrift"
    "cmd/tdl-gen-typescript"
    "cmd/tdl-gen-zod"
  ];

  # The csharp backend is a .NET plugin, shipped beside the Go binaries.
  # The bat syntax sits where programs.tdl.bat in the home-manager module
  # reads it.
  postInstall = ''
    ln -s ${lib.getExe' tdl-dotnet "tdl-gen-csharp"} $out/bin/tdl-gen-csharp
    install -Dm644 editors/bat/tdl.sublime-syntax $out/share/bat/syntaxes/tdl.sublime-syntax
  '';

  meta = {
    description = "Type Description Language";
    homepage = "https://github.com/UnstoppableMango/tdl";
    mainProgram = "tdl";
    license = lib.licenses.mit;
  };
}
