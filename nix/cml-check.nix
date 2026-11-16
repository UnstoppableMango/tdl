# cml-check, the harness backend/contextmapper's tests validate CML with:
# backend/contextmapper/testdata/cmlcheck compiled against the jars
# context-mapper-cli ships, which hold context-mapper-dsl, rather than the
# Maven build its README describes.
{
  context-mapper-cli,
  jdk,
  makeWrapper,
  stdenvNoCC,
}:
let
  jars = "${context-mapper-cli}/share/context-mapper-cli/lib";
in
stdenvNoCC.mkDerivation {
  pname = "cml-check";
  inherit (context-mapper-cli) version;

  src = ../backend/contextmapper/testdata/cmlcheck/src;

  nativeBuildInputs = [
    jdk
    makeWrapper
  ];

  buildPhase = ''
    runHook preBuild
    javac -cp '${jars}/*' -d classes main/java/check/Main.java
    runHook postBuild
  '';

  installPhase = ''
    runHook preInstall
    mkdir -p $out/share/cml-check
    cp -r classes $out/share/cml-check/
    makeWrapper ${jdk}/bin/java $out/bin/cml-check \
      --add-flags "-cp '$out/share/cml-check/classes:${jars}/*' check.Main"
    runHook postInstall
  '';

  # A valid file passes, and one naming an undefined aggregate root fails.
  doInstallCheck = true;
  installCheckPhase = ''
    runHook preInstallCheck
    echo 'BoundedContext Check { Aggregate A { Entity A { aggregateRoot String name } } }' > good.cml
    $out/bin/cml-check good.cml
    echo 'BoundedContext Check { Aggregate' > bad.cml
    ! $out/bin/cml-check bad.cml
    runHook postInstallCheck
  '';

  meta.mainProgram = "cml-check";
}
