{
  lib,
  stdenvNoCC,
  nodejs,
  jq,
  tdl-portable,
  version,
}:
let
  # tdl-portable's GOOS/GOARCH to npm's os/cpu, which name the platform packages.
  cpu = {
    amd64 = "x64";
    arm64 = "arm64";
  };
  platforms = map (
    target:
    let
      parts = lib.splitString "/" target;
      os = builtins.elemAt parts 0;
      arch = builtins.elemAt parts 1;
    in
    {
      dir = "${os}-${arch}";
      inherit os;
      cpu = cpu.${arch};
      name = "@unmango/tdl-${os}-${cpu.${arch}}";
    }
  ) tdl-portable.targets;
in
# The five npm packages, packed: @unmango/tdl, the shim, and one package per
# platform carrying tdl-portable's binaries. The release publishes these
# tarballs, and checks.npm-tdl installs them, so what is tested is what ships.
stdenvNoCC.mkDerivation {
  pname = "tdl-npm";
  inherit version;

  src = lib.fileset.toSource {
    root = ../npm/tdl;
    fileset = ../npm/tdl;
  };

  nativeBuildInputs = [
    nodejs
    jq
  ];

  passthru = { inherit platforms; };

  buildPhase = ''
    runHook preBuild
    export HOME=$TMPDIR npm_config_cache=$TMPDIR/npm-cache

    have=$(jq -r .version package.json)
    [ "$have" = "$version" ] || {
      echo "npm/tdl/package.json is version $have, but the flake is $version" >&2
      exit 1
    }

    mkdir -p $NIX_BUILD_TOP/packages/tdl
    cp -r . $NIX_BUILD_TOP/packages/tdl
    chmod -R u+w $NIX_BUILD_TOP/packages/tdl

    ${lib.concatMapStrings (p: ''
      dir=$NIX_BUILD_TOP/packages/${p.dir}
      mkdir -p $dir/bin
      cp ${tdl-portable}/${p.dir}/* $dir/bin/
      chmod -R u+w $dir
      jq -n --arg version "$version" \
        --slurpfile base package.json \
        '{
          name: "${p.name}",
          version: $version,
          description: "The tdl and tdl-gen-typescript binaries for ${p.os} ${p.cpu}, installed by @unmango/tdl",
          license: $base[0].license,
          repository: $base[0].repository,
          homepage: $base[0].homepage,
          os: ["${p.os}"],
          cpu: ["${p.cpu}"],
          files: ["bin"],
          preferUnplugged: true
        }' > $dir/package.json
      jq --arg version "$version" \
        '.optionalDependencies["${p.name}"] = $version' \
        $NIX_BUILD_TOP/packages/tdl/package.json > package.json.tmp
      mv package.json.tmp $NIX_BUILD_TOP/packages/tdl/package.json
    '') platforms}

    mkdir -p $NIX_BUILD_TOP/tarballs
    for dir in $NIX_BUILD_TOP/packages/*; do
      npm pack --offline --pack-destination $NIX_BUILD_TOP/tarballs "$dir"
    done
    runHook postBuild
  '';

  installPhase = ''
    runHook preInstall
    mkdir -p $out
    cp $NIX_BUILD_TOP/tarballs/*.tgz $out/
    runHook postInstall
  '';

  # Darwin binaries are signed by the Go linker; leave every byte as built.
  dontFixup = true;

  meta = {
    description = "The @unmango/tdl npm packages, packed";
    homepage = "https://github.com/UnstoppableMango/tdl";
    license = lib.licenses.mit;
  };
}
