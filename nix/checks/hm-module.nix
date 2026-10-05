{
  pkgs,
  lib,
  homeManagerConfiguration,
  runCommand,
  tdl,
  vscode-tdl,
}:
let
  configure =
    editor: disabledModules:
    homeManagerConfiguration {
      inherit pkgs;
      modules = [
        ../hm-module.nix
        {
          inherit disabledModules;
          home = {
            username = "tdl";
            homeDirectory = "/home/tdl";
            stateVersion = "24.11";
          };
          programs.${editor}.enable = true;
          programs.tdl.enable = true;
        }
      ];
    };

  hm = configure "vscode" [ ];
  codium = configure "vscodium" [ ];
  older = configure "vscodium" [ "programs/antigravity.nix" ];
in
assert lib.assertMsg (builtins.elem tdl hm.config.home.packages)
  "programs.tdl.enable did not add tdl to home.packages";
assert lib.assertMsg
  (builtins.elem vscode-tdl hm.config.programs.vscode.profiles.default.extensions)
  "programs.tdl.vscode.enable did not add vscode-tdl to the default profile";
assert lib.assertMsg (
  hm.config.programs.vscode.profiles.default.userSettings == { }
) "programs.tdl.vscode wrote a user setting, which would make home-manager own settings.json";
assert lib.assertMsg (
  hm.config.programs.tdl.vscode.editors == [ "vscode" ]
) "programs.tdl.vscode.editors did not default to the one editor that is enabled";
assert lib.assertMsg (
  codium.config.programs.tdl.vscode.editors == [ "vscodium" ]
) "programs.tdl.vscode.editors did not find VSCodium";
assert lib.assertMsg
  (builtins.elem vscode-tdl codium.config.programs.vscodium.profiles.default.extensions)
  "programs.tdl.vscode.enable did not add vscode-tdl to the VSCodium profile";
assert lib.assertMsg (
  codium.config.programs.vscode.profiles == { }
) "programs.tdl.vscode touched VS Code, which is not enabled here";
assert lib.assertMsg (builtins.all (
  a: a.assertion
) codium.config.assertions) "the module asserted something with only VSCodium enabled";
assert lib.assertMsg
  (builtins.elem vscode-tdl older.config.programs.vscodium.profiles.default.extensions)
  "programs.tdl.vscode did not evaluate against a home-manager with no antigravity module";
runCommand "tdl-hm-module" { } "touch $out"
