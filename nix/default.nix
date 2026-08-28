{
  buildGoApplication,
  lib,
  version,
}:
buildGoApplication {
  pname = "tdl";
  inherit version;

  pwd = ../.;
  src = lib.cleanSource ../.;
  modules = ./gomod2nix.toml;

  subPackages = [ "cmd/tdl" ];
}
