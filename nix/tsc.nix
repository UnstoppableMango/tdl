# The TypeScript compiler of the 7.1 beta, the first with content mappers,
# for checks.go-test, where the content mapper's test runs it (#1187).
# nixpkgs' typescript is 7.0; drop this for it once it reaches 7.1.
#
# The registry's platform package holds tsc, a static Go binary, and the
# lib files beside it, where it finds them.
{
  fetchurl,
  lib,
  runCommand,
  stdenv,
}:
let
  version = "7.1.0-beta";
  platforms = {
    x86_64-linux = {
      name = "linux-x64";
      hash = "sha256-WRKGhctV9EyeZkTvXts8syFhjZVHtO6UwwOFmP/VGFU=";
    };
    aarch64-linux = {
      name = "linux-arm64";
      hash = "sha256-1Ki2R750FxeL47pfE2bNg5dq+bKrW+uHVeL5z8AO5UA=";
    };
    aarch64-darwin = {
      name = "darwin-arm64";
      hash = "sha256-7BNkApijA+5HToPZFg5i6THaXNqLbhiLLE00KGf71lA=";
    };
  };
  platform =
    platforms.${stdenv.hostPlatform.system}
      or (throw "tsc ${version}: no package for ${stdenv.hostPlatform.system}");
  src = fetchurl {
    url = "https://registry.npmjs.org/@typescript/typescript-${platform.name}/-/typescript-${platform.name}-${version}.tgz";
    inherit (platform) hash;
  };
in
runCommand "tsc-${version}"
  {
    inherit version;
    meta = {
      description = "The TypeScript compiler";
      homepage = "https://www.typescriptlang.org/";
      license = lib.licenses.asl20;
      mainProgram = "tsc";
      platforms = builtins.attrNames platforms;
    };
  }
  ''
    mkdir -p $out/lib/typescript $out/bin
    tar -xzf ${src} -C $out/lib/typescript --strip-components=1
    ln -s $out/lib/typescript/lib/tsc $out/bin/tsc
    $out/bin/tsc --version | grep -qx 'Version ${version}'
  ''
