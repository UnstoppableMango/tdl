{
  pkgs,
  lib,
  homeManagerConfiguration,
  runCommand,
  tdl,
  vscode-tdl,
}:
let
  hm = homeManagerConfiguration {
    inherit pkgs;

    modules = [
      ../hm-module.nix
      {
        home = {
          username = "tdl";
          homeDirectory = "/home/tdl";
          stateVersion = "24.11";
        };
        programs.vscode.enable = true;
        programs.tdl.enable = true;
      }
    ];
  };
in
assert lib.assertMsg (builtins.elem tdl hm.config.home.packages)
  "programs.tdl.enable did not add tdl to home.packages";
assert lib.assertMsg
  (builtins.elem vscode-tdl hm.config.programs.vscode.profiles.default.extensions)
  "programs.tdl.vscode.enable did not add vscode-tdl to the default profile";
assert lib.assertMsg (
  hm.config.programs.vscode.profiles.default.userSettings == { }
) "programs.tdl.vscode wrote a user setting, which would make home-manager own settings.json";
runCommand "tdl-hm-module" { } "touch $out"
