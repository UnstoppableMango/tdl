# The flake-parts module a project *using* TDL imports into its own flake.
# `hm-module.nix` installs the language for a person; this one wires it into a
# repository: the CLI in a devShell, and a check per property a model should
# hold. The package is read out of `pkgs` for the same reason it is there, so
# the overlay stays the one thing a consumer adds.
_: {
  perSystem =
    {
      config,
      lib,
      pkgs,
      ...
    }:
    let
      cfg = config.tdl;

      # Every check runs from inside `src` rather than over a copied file:
      # an `include` resolves relative to the file that writes it, so a model
      # split across files needs its tree intact.
      #
      # Each is a single invocation over the whole list rather than a shell
      # loop: the CLI walks its arguments itself, reporting every file that
      # fails rather than stopping at the first, so a loop here would only
      # make the output worse.
      runIn =
        name: extraInputs: body:
        pkgs.runCommand name { nativeBuildInputs = [ cfg.package ] ++ extraInputs; } ''
          cd ${cfg.src}
          ${body}
          touch $out
        '';

      enabled = list: cfg.src != null && list != [ ];
    in
    {
      options.tdl = {
        enable = lib.mkEnableOption "tdl, the type description language, for this project";

        package = lib.mkPackageOption pkgs "tdl" { };

        src = lib.mkOption {
          type = lib.types.nullOr lib.types.path;
          default = null;
          example = lib.literalExpression "./model";
          description = ''
            The directory holding the project's TDL sources. Every check runs
            from inside it, so a model spread across files keeps resolving its
            includes. No check is defined while this is null.
          '';
        };

        files = lib.mkOption {
          type = lib.types.listOf lib.types.str;
          default = [ ];
          example = [
            "orders.tdl"
            "billing.tdl"
          ];
          description = ''
            The files to check, named relative to `src` rather than as paths.
            A path would be copied into the store on its own and stop resolving
            what it includes.
          '';
        };

        check.enable = lib.mkOption {
          type = lib.types.bool;
          default = true;
          description = "Define `checks.tdl-check`, which parses every file in `files`.";
        };

        fmt.enable = lib.mkOption {
          type = lib.types.bool;
          default = true;
          description = ''
            Define `checks.tdl-fmt`, which asserts every file in `files` is in
            canonical form.

            Comments survive formatting, but blank lines are the formatter's
            to decide. Turn this off for a project that would rather not be
            held to canonical form.
          '';
        };

        gen = {
          files = lib.mkOption {
            type = lib.types.listOf lib.types.str;
            default = [ ];
            example = [ "orders.tdl" ];
            description = ''
              The files `checks.tdl-gen` runs `tdl gen --verify` over, named
              relative to `src`. This is separate from `files` and empty by
              default because `tdl gen` fails on a file that declares no target
              block.
            '';
          };

          backends = lib.mkOption {
            type = lib.types.listOf lib.types.package;
            default = [ ];
            description = ''
              Packages put on PATH for the generation check and the devShell.
              A target naming a backend that tdl has no builtin for resolves to
              `tdl-gen-<name>` on PATH, and this is what puts that executable
              there.
            '';
          };
        };

        devShell.enable = lib.mkOption {
          type = lib.types.bool;
          default = true;
          description = ''
            Define `devShells.tdl`, holding the CLI and the backends. It is a
            shell to pull into another one with `inputsFrom`, not one to enter.
          '';
        };
      };

      config = lib.mkIf cfg.enable {
        devShells = lib.optionalAttrs cfg.devShell.enable {
          tdl = pkgs.mkShellNoCC { packages = [ cfg.package ] ++ cfg.gen.backends; };
        };

        checks =
          lib.optionalAttrs (cfg.check.enable && enabled cfg.files) {
            tdl-check = runIn "tdl-check" [ ] "tdl check ${lib.escapeShellArgs cfg.files}";
          }
          // lib.optionalAttrs (cfg.fmt.enable && enabled cfg.files) {
            tdl-fmt = runIn "tdl-fmt" [ ] "tdl fmt --check ${lib.escapeShellArgs cfg.files}";
          }
          // lib.optionalAttrs (enabled cfg.gen.files) {
            tdl-gen = runIn "tdl-gen" cfg.gen.backends "tdl gen --verify ${lib.escapeShellArgs cfg.gen.files}";
          };
      };
    };
}
