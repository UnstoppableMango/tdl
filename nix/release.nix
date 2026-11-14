{
  lib,
  stdenvNoCC,
  go,
  tdl-portable,
  tdl-gen-mermaid,
  version,
}:
let
  # The build platform's binary, which writes the completion scripts. They
  # name no path and read nothing from the platform, so one set serves every
  # tarball.
  tdl = "${tdl-portable}/${go.GOOS}-${go.GOARCH}/tdl";
in
# The release assets: tdl_<version>_<os>_<arch>.tar.gz per tdl-portable target,
# each holding tdl, tdl-gen-typescript, tdl-gen-mermaid, LICENSE, README.md,
# and completions/,
# and SHA256SUMS. The release job uploads and attests these files as built, so
# every byte must be reproducible: names sorted, owner and mtime fixed, gzip
# without a name or timestamp.
stdenvNoCC.mkDerivation {
  pname = "tdl-release";
  inherit version;

  src = lib.fileset.toSource {
    root = ../.;
    fileset = lib.fileset.unions [
      ../LICENSE
      ../README.md
    ];
  };

  buildPhase = ''
    runHook preBuild
    mkdir completions
    for shell in bash zsh fish; do
      ${tdl} completion $shell > completions/tdl.$shell
    done

    mkdir $NIX_BUILD_TOP/dist
    for target in ${lib.escapeShellArgs tdl-portable.targets}; do
      name=tdl_${version}_''${target/\//_}
      stage=$NIX_BUILD_TOP/stage/$name
      mkdir -p $stage
      cp ${tdl-portable}/''${target/\//-}/* ${tdl-gen-mermaid.portable}/''${target/\//-}/* \
        LICENSE README.md $stage/
      cp -r completions $stage/
      chmod -R u+w,go-w $stage
      # Members are named without ./, so `tar -xzf <tarball> tdl` finds one.
      (cd $stage && tar --create --sort=name --format=gnu \
        --owner=0 --group=0 --numeric-owner --mtime=@$SOURCE_DATE_EPOCH \
        -- *) | gzip -9n > $NIX_BUILD_TOP/dist/$name.tar.gz
    done

    (cd $NIX_BUILD_TOP/dist && sha256sum *.tar.gz > SHA256SUMS)
    runHook postBuild
  '';

  installPhase = ''
    runHook preInstall
    cp -r $NIX_BUILD_TOP/dist $out
    runHook postInstall
  '';

  # The tarballs hold darwin binaries the Go linker and bun signed; leave
  # every byte.
  dontFixup = true;

  meta = {
    description = "The tdl release assets: a tarball per platform and SHA256SUMS";
    homepage = "https://github.com/UnstoppableMango/tdl";
    license = lib.licenses.mit;
  };
}
