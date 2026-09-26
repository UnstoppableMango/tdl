{
  buildGoApplication,
  lib,
  go,
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
    "cmd/tdl-gen-protobuf"
    "cmd/tdl-gen-salesforce"
    "cmd/tdl-gen-smithy"
    "cmd/tdl-gen-thrift"
    "cmd/tdl-gen-typescript"
  ];

  # Nine binaries are installed, so lib.getExe needs telling which one is the
  # program; nix/vscode-extension.nix reads it.
  meta.mainProgram = "tdl";
}
