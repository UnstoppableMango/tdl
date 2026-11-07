{
  fetchCrate,
  lib,
  rustPlatform,
}:
# buffa's protoc plugin, which writes rust/tdl-plugin/src/generated for
# `make generate`. nixpkgs does not package it. Its version must match the
# buffa crate rust/Cargo.toml pins.
rustPlatform.buildRustPackage rec {
  pname = "protoc-gen-buffa";
  version = "0.9.2";

  src = fetchCrate {
    inherit pname version;
    hash = "sha256-BEK+Nfn3/uBO78vXCm0zEQYgAkGFRkiPwRvTOZwFc00=";
  };
  cargoHash = "sha256-tYBuO9RjeeqsiXvIrLVmd2HHC4B9o8dGqpl3GyWDjFQ=";

  meta = {
    description = "protoc plugin generating Rust code with buffa";
    homepage = "https://github.com/anthropics/buffa";
    license = lib.licenses.asl20;
    mainProgram = "protoc-gen-buffa";
  };
}
