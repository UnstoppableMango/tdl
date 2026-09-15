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
    "cmd/tdl-gen-go"
    "cmd/tdl-gen-protobuf"
    "cmd/tdl-gen-smithy"
    "cmd/tdl-gen-thrift"
  ];
}
