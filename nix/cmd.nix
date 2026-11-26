{
  buildGoApplication,
  installShellFiles,
  lib,
  go,
  stdenv,
  tdl-dotnet,
  tdl-gen-mermaid,
  version,
}:
buildGoApplication {
  pname = "tdl";
  inherit version go;

  src = lib.cleanSource ../.;
  modules = ./gomod2nix.toml;

  nativeBuildInputs = [ installShellFiles ];

  subPackages = [
    "cmd/tdl"
    "cmd/tdl-gen-debug"
    "cmd/tdl-gen-go"
    "cmd/tdl-gen-graphql"
    "cmd/tdl-gen-haskell"
    "cmd/tdl-gen-jsonschema"
    "cmd/tdl-gen-likec4"
    "cmd/tdl-gen-openapi"
    "cmd/tdl-gen-protobuf"
    "cmd/tdl-gen-salesforce"
    "cmd/tdl-gen-smithy"
    "cmd/tdl-gen-thrift"
    "cmd/tdl-gen-typescript"
    "cmd/tdl-gen-yolo"
    "cmd/tdl-gen-zod"
  ];

  # The csharp backend is a .NET plugin, and the mermaid backend a Bun single
  # executable, each shipped beside the Go binaries.
  # The bat syntax sits where programs.tdl.bat in the home-manager module
  # reads it. Completion scripts come from running the built tdl, which a
  # cross build cannot do, so one goes without them.
  postInstall = ''
    ln -s ${lib.getExe' tdl-dotnet "tdl-gen-csharp"} $out/bin/tdl-gen-csharp
    ln -s ${lib.getExe tdl-gen-mermaid} $out/bin/tdl-gen-mermaid
    install -Dm644 editors/bat/tdl.sublime-syntax $out/share/bat/syntaxes/tdl.sublime-syntax
  ''
  + lib.optionalString (stdenv.buildPlatform.canExecute stdenv.hostPlatform) ''
    installShellCompletion --cmd tdl \
      --bash <($out/bin/tdl completion bash) \
      --zsh <($out/bin/tdl completion zsh) \
      --fish <($out/bin/tdl completion fish)
  '';

  meta = {
    description = "Type Description Language";
    homepage = "https://github.com/UnstoppableMango/tdl";
    mainProgram = "tdl";
    license = lib.licenses.mit;
  };
}
