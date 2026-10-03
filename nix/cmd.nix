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

  meta = {
    description = "Type Description Language";
    homepage = "https://github.com/UnstoppableMango/tdl";
    mainProgram = "tdl";
    license = lib.licenses.gpl3;
    maintainers = with lib.maintainers; [ UnstoppableMango ];
  };
}
