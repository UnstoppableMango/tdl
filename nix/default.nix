{
  buildGoApplication,
  lib,
  version,
}:
buildGoApplication {
  pname = "tdl";
  inherit version;

  src = lib.cleanSource ../.;
  modules = ./gomod2nix.toml;

  subPackages = [ "cmd/tdl" ];
}
