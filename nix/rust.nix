{
  lib,
  rustPlatform,
  version,
}:
rustPlatform.buildRustPackage {
  pname = "tdl-rust";
  inherit version;

  # The SDK's tests replay testdata/plugin and read reverse.md's loss codes.
  src = lib.fileset.toSource {
    root = ../.;
    fileset = lib.fileset.unions [
      ../rust/Cargo.toml
      ../rust/Cargo.lock
      ../rust/tdl-plugin
      ../rust/tdl-plugin-debug
      ../testdata/plugin
      ../docs/architecture/reverse.md
    ];
  };

  # The workspace sits in rust/, beside the files its tests read.
  cargoRoot = "rust";
  buildAndTestSubdir = "rust";
  cargoLock.lockFile = ../rust/Cargo.lock;

  # buildRustPackage runs the workspace's tests, which replay the recorded
  # exchanges through the debug port.
  doCheck = true;

  meta = {
    description = "The TDL backends written on the Rust plugin SDK: the debug port";
    homepage = "https://github.com/UnstoppableMango/tdl";
    mainProgram = "tdl-gen-rust_debug";
    license = lib.licenses.mit;
  };
}
