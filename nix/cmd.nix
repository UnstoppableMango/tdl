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

  subPackages = [ "cmd/tdl" ];
}
