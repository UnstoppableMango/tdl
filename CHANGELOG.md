# Changelog

## [0.3.2](https://github.com/UnstoppableMango/tdl/compare/v0.3.1...v0.3.2) (2026-10-06)


### Features

* **ir:** carry the package doc comment in Model.doc ([#952](https://github.com/UnstoppableMango/tdl/issues/952)) ([6b66837](https://github.com/UnstoppableMango/tdl/commit/6b66837c5dc298e98a960b3d70650cfbd312eee9)), closes [#923](https://github.com/UnstoppableMango/tdl/issues/923)
* **parser:** a doc comment above package documents the package ([#951](https://github.com/UnstoppableMango/tdl/issues/951)) ([d1f8415](https://github.com/UnstoppableMango/tdl/commit/d1f8415ac08e6996ad322664fac0763aed4e72ef))


### Bug Fixes

* **emit:** keep a doc line's indentation ([#937](https://github.com/UnstoppableMango/tdl/issues/937)) ([2ba3420](https://github.com/UnstoppableMango/tdl/commit/2ba3420895fc1d19e547e86b5ad7662e626a6b16))
* **fmt:** keep a comment between doc comment lines in place ([#956](https://github.com/UnstoppableMango/tdl/issues/956)) ([b2a287c](https://github.com/UnstoppableMango/tdl/commit/b2a287c1459c3d4e28242d629a1ec4969d6dab26)), closes [#823](https://github.com/UnstoppableMango/tdl/issues/823)


### Documentation

* state the nixpkgs overlays.default needs ([#942](https://github.com/UnstoppableMango/tdl/issues/942)) ([9b4b572](https://github.com/UnstoppableMango/tdl/commit/9b4b5728fd6dde2eda7dd3e4ed73f25cb4cbad85))

## [0.3.1](https://github.com/UnstoppableMango/tdl/compare/v0.3.0...v0.3.1) (2026-10-06)


### Features

* **jsonschema:** add a JSON Schema backend ([#954](https://github.com/UnstoppableMango/tdl/issues/954)) ([a3d56ca](https://github.com/UnstoppableMango/tdl/commit/a3d56ca9e98f0bd837587d925112686ffac25413))

## [0.3.0](https://github.com/UnstoppableMango/tdl/compare/v0.2.16...v0.3.0) (2026-10-06)


### ⚠ BREAKING CHANGES

* **gen:** a relative out directive resolves against the directory of the .tdl file declaring the target block rather than the working directory.

### Features

* **cli:** tdl check lowers each file and reports its diagnostics ([#946](https://github.com/UnstoppableMango/tdl/issues/946)) ([0456ff3](https://github.com/UnstoppableMango/tdl/commit/0456ff3844515b6193b3f97bf6eebf8ee6f38956))
* **go:** a file directive writes the whole target into one file ([#955](https://github.com/UnstoppableMango/tdl/issues/955)) ([f9713d6](https://github.com/UnstoppableMango/tdl/commit/f9713d6221514d1385a214d051d0be45122b7f47))


### Bug Fixes

* **gen:** own the files tdl wrote rather than the whole directory ([#948](https://github.com/UnstoppableMango/tdl/issues/948)) ([907b149](https://github.com/UnstoppableMango/tdl/commit/907b1495688eb1ef6dd573c96327a18c03b5cbaa))
* **gen:** resolve out against the file declaring the target block ([#947](https://github.com/UnstoppableMango/tdl/issues/947)) ([dc766ad](https://github.com/UnstoppableMango/tdl/commit/dc766adbbd4baeefb5aa60de3912ca46b99c9a31))
* **gen:** verify a shared output directory against every file given ([#949](https://github.com/UnstoppableMango/tdl/issues/949)) ([d923a4e](https://github.com/UnstoppableMango/tdl/commit/d923a4ea211c96db023256bf2af0d87bf0878749))
* **nix:** regenerate gomod2nix.toml and have Renovate keep it current ([#958](https://github.com/UnstoppableMango/tdl/issues/958)) ([4eb5077](https://github.com/UnstoppableMango/tdl/commit/4eb5077e5511177c1d35a8d864151d9c5e27bdbf))
* **thrift:** skip a union whose variant name its union already declares ([#945](https://github.com/UnstoppableMango/tdl/issues/945)) ([86d28dd](https://github.com/UnstoppableMango/tdl/commit/86d28dd90061b7b863fc0776268fd367208b9d60))


### Documentation

* design a likec4 backend ([#976](https://github.com/UnstoppableMango/tdl/issues/976)) ([e4aa14b](https://github.com/UnstoppableMango/tdl/commit/e4aa14ba0d4dcab7093961c1ff6786157bf8e7a9))
* design target profiles ([#975](https://github.com/UnstoppableMango/tdl/issues/975)) ([79b4f1d](https://github.com/UnstoppableMango/tdl/commit/79b4f1da198568f1c501ceb2324d865eeab61b2b))


### Dependencies

* update dependency @types/vscode to ~1.140.0 ([#932](https://github.com/UnstoppableMango/tdl/issues/932)) ([5da6c8e](https://github.com/UnstoppableMango/tdl/commit/5da6c8ecba1d53ed7a5a0063e49fd3f1fabbbc2f))
* update module github.com/vektah/gqlparser/v2 to v2.5.60 ([#931](https://github.com/UnstoppableMango/tdl/issues/931)) ([71f5995](https://github.com/UnstoppableMango/tdl/commit/71f5995cf1990197a3675d93e43474b40c896e3c))

## [0.2.16](https://github.com/UnstoppableMango/tdl/compare/v0.2.15...v0.2.16) (2026-10-05)


### Documentation

* trim AGENTS.md and the review skill ([#903](https://github.com/UnstoppableMango/tdl/issues/903)) ([52fea36](https://github.com/UnstoppableMango/tdl/commit/52fea36d9e92f746b87847499aa039284065cbc6))
* trim backend code comments ([#909](https://github.com/UnstoppableMango/tdl/issues/909)) ([d3f351e](https://github.com/UnstoppableMango/tdl/commit/d3f351ed0f09f8f79905a95c0aac7f916f5c1a1f))
* trim design docs and backlog ([#906](https://github.com/UnstoppableMango/tdl/issues/906)) ([600679d](https://github.com/UnstoppableMango/tdl/commit/600679d616f6d7e426b1b12f42258a37611664c1))
* trim front-end code comments ([#907](https://github.com/UnstoppableMango/tdl/issues/907)) ([a45d871](https://github.com/UnstoppableMango/tdl/commit/a45d8719bcdb820b39162f8f6f5ebfce3013c37a))
* trim nix, proto, editor, and config comments ([#910](https://github.com/UnstoppableMango/tdl/issues/910)) ([b939a9c](https://github.com/UnstoppableMango/tdl/commit/b939a9cb6ccde6b816ac527ce80fe8ed7223deca))
* trim README and user-facing docs ([#904](https://github.com/UnstoppableMango/tdl/issues/904)) ([b835785](https://github.com/UnstoppableMango/tdl/commit/b8357853ce8d64868203c1d1813ddb4ba98b6a8a))
* trim sema, ir, plugin, cli, gen, and lsp comments ([#908](https://github.com/UnstoppableMango/tdl/issues/908)) ([922b970](https://github.com/UnstoppableMango/tdl/commit/922b970d3f93a171e87294099027a642b5589cb5))
* trim spec and grammar comments ([#905](https://github.com/UnstoppableMango/tdl/issues/905)) ([fc03faa](https://github.com/UnstoppableMango/tdl/commit/fc03faa88be5e5ee8f7bd8421b6e4c04b58eae2f))


### Code Refactoring

* **nix:** extract home-manager module check to separate file ([db6912a](https://github.com/UnstoppableMango/tdl/commit/db6912aa5a5c738ffbc92ea0d049f7ee94287bf9))

## [0.2.15](https://github.com/UnstoppableMango/tdl/compare/v0.2.14...v0.2.15) (2026-10-05)


### Features

* **nix:** install the extension into every enabled VS Code-based editor ([#847](https://github.com/UnstoppableMango/tdl/issues/847)) ([05b821e](https://github.com/UnstoppableMango/tdl/commit/05b821eba709f7e093c50cdd5ed1dc0677e8d318))

## [0.2.14](https://github.com/UnstoppableMango/tdl/compare/v0.2.13...v0.2.14) (2026-10-04)


### Features

* **protobuf:** write a block-scope option as a file option ([#917](https://github.com/UnstoppableMango/tdl/issues/917)) ([b0b35ba](https://github.com/UnstoppableMango/tdl/commit/b0b35bae0b472e3bd240731c595f189459efd8bf))


### Bug Fixes

* **protobuf:** write an inlined oneof's field doc above it ([#920](https://github.com/UnstoppableMango/tdl/issues/920)) ([5a27423](https://github.com/UnstoppableMango/tdl/commit/5a27423ac1023bbc5688bf8ea743a5b2f5d0059d))


### Tests

* **vscode:** drive the extension in a headless VSCodium ([#845](https://github.com/UnstoppableMango/tdl/issues/845)) ([30bd68f](https://github.com/UnstoppableMango/tdl/commit/30bd68f095ebe22efe20c46940eab26efd1b49a2))

## [0.2.13](https://github.com/UnstoppableMango/tdl/compare/v0.2.12...v0.2.13) (2026-10-02)


### Continuous Integration

* condense CI into build and test jobs ([#898](https://github.com/UnstoppableMango/tdl/issues/898)) ([4576edf](https://github.com/UnstoppableMango/tdl/commit/4576edf6563c4367fd2d523a827bf743479b5dca))

## [0.2.12](https://github.com/UnstoppableMango/tdl/compare/v0.2.11...v0.2.12) (2026-10-01)


### Features

* **nix:** default the extension's server path to the built tdl ([#844](https://github.com/UnstoppableMango/tdl/issues/844)) ([164598c](https://github.com/UnstoppableMango/tdl/commit/164598cfb1e0516751722026674641543391d50e))

## [0.2.11](https://github.com/UnstoppableMango/tdl/compare/v0.2.10...v0.2.11) (2026-10-01)


### Features

* **emit:** allocate unpinned members around number() pins ([#876](https://github.com/UnstoppableMango/tdl/issues/876)) ([088f200](https://github.com/UnstoppableMango/tdl/commit/088f20062e5eb779556b2e1dd656c97b5538e51d))
* **parser:** allow a reserved word as a package path segment ([#877](https://github.com/UnstoppableMango/tdl/issues/877)) ([a95aa9d](https://github.com/UnstoppableMango/tdl/commit/a95aa9d8d28e58ae85a481418accdb1ee35c4419))
* **prelude:** fixed-width numerics with backend mappings ([#880](https://github.com/UnstoppableMango/tdl/issues/880)) ([ab67831](https://github.com/UnstoppableMango/tdl/commit/ab67831e897570ec5f7a039e69219987ad17c934))
* **protobuf:** a file directive names the output file and splits a package ([#885](https://github.com/UnstoppableMango/tdl/issues/885)) ([ea2ba00](https://github.com/UnstoppableMango/tdl/commit/ea2ba0022de5ec8909602b4bcb93a4d2111a322a))
* **protobuf:** a reserved directive for retired field numbers and names ([#882](https://github.com/UnstoppableMango/tdl/issues/882)) ([4ff3b52](https://github.com/UnstoppableMango/tdl/commit/4ff3b52090f434db101c04b1d4a560327616e6c0))
* **protobuf:** a service directive that emits function-typed fields as rpcs ([#889](https://github.com/UnstoppableMango/tdl/issues/889)) ([7db99e3](https://github.com/UnstoppableMango/tdl/commit/7db99e389e40ee1a82c982d770c7c82165911b27))
* **protobuf:** emit an enum-typed field as a oneof in its message ([#879](https://github.com/UnstoppableMango/tdl/issues/879)) ([e7d74aa](https://github.com/UnstoppableMango/tdl/commit/e7d74aa893fc918ed910fddda356e9e89a9fb567))
* **protobuf:** import another tdl package's generated file ([#888](https://github.com/UnstoppableMango/tdl/issues/888)) ([db31808](https://github.com/UnstoppableMango/tdl/commit/db318086b7f24a8bd30cb72f5ce5a8530279f8c2))
* **protobuf:** map a declaration to an external proto message ([#887](https://github.com/UnstoppableMango/tdl/issues/887)) ([9a83b08](https://github.com/UnstoppableMango/tdl/commit/9a83b08b970de0366c20bd18134aba37fe0fcc2d))
* **protobuf:** option and import directives ([#883](https://github.com/UnstoppableMango/tdl/issues/883)) ([da60760](https://github.com/UnstoppableMango/tdl/commit/da60760320f4969678f01ebf441e6112c6196502))
* **sema:** let a target path name an imported declaration ([#886](https://github.com/UnstoppableMango/tdl/issues/886)) ([b3f5869](https://github.com/UnstoppableMango/tdl/commit/b3f586912aedcf206084410b6abfabec96267870))


### Bug Fixes

* **ast:** keep blank lines between top-level comment groups ([#875](https://github.com/UnstoppableMango/tdl/issues/875)) ([bd37c85](https://github.com/UnstoppableMango/tdl/commit/bd37c850d5bf9ef8e4bfa2c751770796060ed651))
* be polite to consumers of the flake ([#854](https://github.com/UnstoppableMango/tdl/issues/854)) ([07e91b6](https://github.com/UnstoppableMango/tdl/commit/07e91b6c51bdd1237e42451d543bfb36631b4dd3))
* **sema:** a _ import merges lower-case primitives and units ([#874](https://github.com/UnstoppableMango/tdl/issues/874)) ([edc675d](https://github.com/UnstoppableMango/tdl/commit/edc675dfd5411aeb316825de1d1850793afc2294))


### Dependencies

* update dependency @biomejs/biome to v2.5.15 ([#895](https://github.com/UnstoppableMango/tdl/issues/895)) ([d18a5d4](https://github.com/UnstoppableMango/tdl/commit/d18a5d4bfca39fe60b21b54d1517916aded97874))

## [0.2.10](https://github.com/UnstoppableMango/tdl/compare/v0.2.9...v0.2.10) (2026-09-30)


### Features

* **gen:** let a backend declare a directive repeatable ([#881](https://github.com/UnstoppableMango/tdl/issues/881)) ([0911042](https://github.com/UnstoppableMango/tdl/commit/0911042cbca68da8979c37e7ffa4757ab7bc28a5))
* **protobuf:** emit editions instead of proto3 ([#878](https://github.com/UnstoppableMango/tdl/issues/878)) ([2683083](https://github.com/UnstoppableMango/tdl/commit/26830832ab530dd2ffedb80f36cd0bad93558671))


### Bug Fixes

* **renovate:** reference shared presets by name ([#851](https://github.com/UnstoppableMango/tdl/issues/851)) ([4b06360](https://github.com/UnstoppableMango/tdl/commit/4b0636065f619f9f91843d86a650b2546b5d6b1b)), closes [#848](https://github.com/UnstoppableMango/tdl/issues/848)


### Documentation

* add Hercules CI badge ([#850](https://github.com/UnstoppableMango/tdl/issues/850)) ([36ae05d](https://github.com/UnstoppableMango/tdl/commit/36ae05dfa26a2caf95d438afffaa4a8620c8ff7d))


### Dependencies

* update dependency @types/node to v24 ([#860](https://github.com/UnstoppableMango/tdl/issues/860)) ([8d19155](https://github.com/UnstoppableMango/tdl/commit/8d191553bb37125a7112ef43f06404f8cef9a42b))
* update dependency @types/vscode to ~1.138.0 ([#859](https://github.com/UnstoppableMango/tdl/issues/859)) ([0b16ee0](https://github.com/UnstoppableMango/tdl/commit/0b16ee0d2fb788a9bd7a84f69de21a8ed2deb524))
* update dependency vscode-languageclient to v10.1.2 ([#855](https://github.com/UnstoppableMango/tdl/issues/855)) ([e5bf1ab](https://github.com/UnstoppableMango/tdl/commit/e5bf1ab4c870d7b1685defa3df9c76674af17c3c))
* update module github.com/vektah/gqlparser/v2 to v2.5.58 ([#856](https://github.com/UnstoppableMango/tdl/issues/856)) ([a9ac723](https://github.com/UnstoppableMango/tdl/commit/a9ac7231574c814737896f258cf9ec9fb2756fac))

## [0.2.9](https://github.com/UnstoppableMango/tdl/compare/v0.2.8...v0.2.9) (2026-09-24)


### Features

* **vscode:** start the language server ([#843](https://github.com/UnstoppableMango/tdl/issues/843)) ([17d0393](https://github.com/UnstoppableMango/tdl/commit/17d0393a4c2643c50d6eafe9b059a3654534fb97))

## [0.2.8](https://github.com/UnstoppableMango/tdl/compare/v0.2.7...v0.2.8) (2026-09-21)


### Features

* **lsp:** format documents and outline their symbols ([#839](https://github.com/UnstoppableMango/tdl/issues/839)) ([994a4ef](https://github.com/UnstoppableMango/tdl/commit/994a4ef3f26bf8542ad205112457fd2ed5d070fe))
* **lsp:** hover a name to see its declaration ([#838](https://github.com/UnstoppableMango/tdl/issues/838)) ([a247f6b](https://github.com/UnstoppableMango/tdl/commit/a247f6b74e3750dcff1dd8ffeab469cdea95de51))
* **salesforce:** generate Salesforce DX metadata and Apex ([#836](https://github.com/UnstoppableMango/tdl/issues/836)) ([b7d2b4d](https://github.com/UnstoppableMango/tdl/commit/b7d2b4d3860ad4befff48e12e6941baeb3ec0a7b))


### Documentation

* **AGENTS.md:** clarify agent settings and document design plan structure ([#832](https://github.com/UnstoppableMango/tdl/issues/832)) ([a65f4fd](https://github.com/UnstoppableMango/tdl/commit/a65f4fdc2f89556b12f10e37351c1478cc08d6ec))
* **AGENTS.md:** update make play command description and add make check command ([#830](https://github.com/UnstoppableMango/tdl/issues/830)) ([4a636ac](https://github.com/UnstoppableMango/tdl/commit/4a636ac72404ff464f28d3bdbcd1b0018f5bd8fc))
* **lsp:** plan editor clients and the next server features ([#841](https://github.com/UnstoppableMango/tdl/issues/841)) ([18f5741](https://github.com/UnstoppableMango/tdl/commit/18f5741e4d96ddc749eddec6cd7b97b49b3a18e5))


### Build System

* set up TypeScript development for editors/vscode ([#842](https://github.com/UnstoppableMango/tdl/issues/842)) ([3d9da6b](https://github.com/UnstoppableMango/tdl/commit/3d9da6b30229947f1e9c2c845914125c8b4f6986))


### Continuous Integration

* configure DeepSource ([#826](https://github.com/UnstoppableMango/tdl/issues/826)) ([c9b63ae](https://github.com/UnstoppableMango/tdl/commit/c9b63aea5a68d5d9a6f721f48d5d9326a9d33d88))
* consolidate the workflow around nix and lowercase job names ([#833](https://github.com/UnstoppableMango/tdl/issues/833)) ([edfa90c](https://github.com/UnstoppableMango/tdl/commit/edfa90c1b56fd5164c499f160a96381825d7c354))

## [0.2.7](https://github.com/UnstoppableMango/tdl/compare/v0.2.6...v0.2.7) (2026-09-19)


### Features

* **graphql:** generate GraphQL schemas ([#795](https://github.com/UnstoppableMango/tdl/issues/795)) ([7b74555](https://github.com/UnstoppableMango/tdl/commit/7b745551186caf4fd29691cf93099e073cd44bb5))
* **typescript:** generate TypeScript wire types ([#796](https://github.com/UnstoppableMango/tdl/issues/796)) ([a829475](https://github.com/UnstoppableMango/tdl/commit/a82947506eff8ec9a516411d00447a4ffe2aa4e2))

## [0.2.6](https://github.com/UnstoppableMango/tdl/compare/v0.2.5...v0.2.6) (2026-09-18)


### Features

* **smithy:** generate Smithy IDL 2.0 models ([#794](https://github.com/UnstoppableMango/tdl/issues/794)) ([1824cc6](https://github.com/UnstoppableMango/tdl/commit/1824cc6e651762b9cd9fd918a9f4a8c6ed870a60))


### Bug Fixes

* **deps:** update go.lsp.dev modules to v1 ([#825](https://github.com/UnstoppableMango/tdl/issues/825)) ([ab0ebf1](https://github.com/UnstoppableMango/tdl/commit/ab0ebf17648347eeca556327c7c53a4bbcb79a7b))

## [0.2.5](https://github.com/UnstoppableMango/tdl/compare/v0.2.4...v0.2.5) (2026-09-18)


### Features

* **thrift:** generate Thrift IDL ([#793](https://github.com/UnstoppableMango/tdl/issues/793)) ([4c989b5](https://github.com/UnstoppableMango/tdl/commit/4c989b5778fb03cfafb165709323a9a2299ed738))

## [0.2.4](https://github.com/UnstoppableMango/tdl/compare/v0.2.3...v0.2.4) (2026-09-18)


### Features

* **go:** generate where constraints as Validate methods ([#787](https://github.com/UnstoppableMango/tdl/issues/787)) ([918d354](https://github.com/UnstoppableMango/tdl/commit/918d35426e6cdb1b130075070a28ff8cb7df7e46))
* **go:** map a declaration to a foreign type ([#813](https://github.com/UnstoppableMango/tdl/issues/813)) ([fdeef4a](https://github.com/UnstoppableMango/tdl/commit/fdeef4aa6273392dbf83424cdeb3a4f42102367d))


### Bug Fixes

* **fmt:** keep a doc comment and an ordinary comment in written order ([#814](https://github.com/UnstoppableMango/tdl/issues/814)) ([5eba6c6](https://github.com/UnstoppableMango/tdl/commit/5eba6c6ce658c4b090ba099675b18d84090696d1)), closes [#753](https://github.com/UnstoppableMango/tdl/issues/753)
* **go:** escape a file name Go reads as a test or a build constraint ([#819](https://github.com/UnstoppableMango/tdl/issues/819)) ([3a96b60](https://github.com/UnstoppableMango/tdl/commit/3a96b60abb493b3913ad19e122393cf059e4ad76)), closes [#785](https://github.com/UnstoppableMango/tdl/issues/785)
* **go:** warn on a unit argument instead of dropping it ([#817](https://github.com/UnstoppableMango/tdl/issues/817)) ([3292d3b](https://github.com/UnstoppableMango/tdl/commit/3292d3be3aaa9dac1731f00b581b0f71a9408be6)), closes [#779](https://github.com/UnstoppableMango/tdl/issues/779)
* **nix:** ship tdl-gen-debug in subPackages ([#818](https://github.com/UnstoppableMango/tdl/issues/818)) ([7d2064a](https://github.com/UnstoppableMango/tdl/commit/7d2064a9cc3e20f6d5575f826dc569d720caeb33))
* **sema:** do not lower a unit that lost its name binding ([#815](https://github.com/UnstoppableMango/tdl/issues/815)) ([6246b05](https://github.com/UnstoppableMango/tdl/commit/6246b05dd8593aafd261bc5a6ddbb56afe8d1ea3)), closes [#767](https://github.com/UnstoppableMango/tdl/issues/767)
* **sema:** resolve a NAME constraint argument to its variant ([#816](https://github.com/UnstoppableMango/tdl/issues/816)) ([8e4b8ad](https://github.com/UnstoppableMango/tdl/commit/8e4b8adec7917dc943b8422ff48736b7a8d9cb9f)), closes [#784](https://github.com/UnstoppableMango/tdl/issues/784)


### Continuous Integration

* cache the Go module and build caches ([#820](https://github.com/UnstoppableMango/tdl/issues/820)) ([16e6030](https://github.com/UnstoppableMango/tdl/commit/16e603086cc09bec8386dc42588a621a9fae6efa)), closes [#806](https://github.com/UnstoppableMango/tdl/issues/806)

## [0.2.3](https://github.com/UnstoppableMango/tdl/compare/v0.2.2...v0.2.3) (2026-09-18)


### Features

* **go:** generate type parameters as Go generics and classes as interfaces ([#782](https://github.com/UnstoppableMango/tdl/issues/782)) ([9dac09b](https://github.com/UnstoppableMango/tdl/commit/9dac09b6b50b06ed2bab0564fe445d9489e0bb66))


### Bug Fixes

* **go:** skip a declaration that names a skipped one ([#781](https://github.com/UnstoppableMango/tdl/issues/781)) ([da60d72](https://github.com/UnstoppableMango/tdl/commit/da60d72e4a7708a0850b3fd4cb414f5e94926263))


### Continuous Integration

* give the Tree-sitter job a shell of its own and a reason to run ([#803](https://github.com/UnstoppableMango/tdl/issues/803)) ([26ea99c](https://github.com/UnstoppableMango/tdl/commit/26ea99c6ae97c363baa8bfc1ac6c24051450e848))
* run release-please as thecluster[bot] ([#805](https://github.com/UnstoppableMango/tdl/issues/805)) ([60bfd3f](https://github.com/UnstoppableMango/tdl/commit/60bfd3fd1b5005021993658e70766f8cc39c6bf8))

## [0.2.2](https://github.com/UnstoppableMango/tdl/compare/v0.2.1...v0.2.2) (2026-09-18)


### Features

* add LSP dependencies to go.mod for editor integration ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))
* **cli:** add LSP command to root CLI ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))
* **internal/cli/lsp.go:** add LSP command to serve Language ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))
* **lsp/unimplemented.go:** add unimplemented protocol server ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))
* **lsp:** add language server protocol implementation ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))
* **lsp:** add position conversion utilities for LSP protocol ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))
* **lsp:** add Serve function to run LSP server over connection ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))
* **lsp:** implement document store with snapshot caching ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))
* **lsp:** implement import overlay for editor buffer resolution ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))
* **lsp:** serve diagnostics and go to definition ([#799](https://github.com/UnstoppableMango/tdl/issues/799)) ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))
* **sema/class.go:** record class reference lookups for LSP support ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))
* **sema:** add support for recording name references during lowering ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))
* **sema:** clarify binding position comment to indicate it may reference declarations in imported files ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))
* **sema:** record lookups for target entry paths and type references ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))


### Bug Fixes

* **sema:** keep a field's constraints and default through include ([#786](https://github.com/UnstoppableMango/tdl/issues/786)) ([b1b9ea6](https://github.com/UnstoppableMango/tdl/commit/b1b9ea61307733c6023823908a2b73b113ef5e6e)), closes [#783](https://github.com/UnstoppableMango/tdl/issues/783)
* **sema:** use declaration position instead of import position for ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))


### Documentation

* add language server protocol design document ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))
* add lsp-plan.md with language server implementation phases ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))
* **AGENTS.md:** add lsp command to internal/cli cobra commands list ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))
* **AGENTS.md:** document internal/lsp package and refs.go reference index ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))
* **backlog.md:** move language server from backlog to completed work ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))
* **backlog.md:** update editor support section with lsp design reference ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))


### Tests

* add session_test.go with LSP session testing utilities ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))
* **lsp:** add comprehensive server protocol tests for diagnostics and definition ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))
* **sema:** add comprehensive tests for reference resolution ([7d13744](https://github.com/UnstoppableMango/tdl/commit/7d137449d5b90814a6ef097f87174e7513aa9f54))

## [0.2.1](https://github.com/UnstoppableMango/tdl/compare/v0.2.0...v0.2.1) (2026-09-18)


### Features

* **go:** generate an entity's key from the key directive ([#775](https://github.com/UnstoppableMango/tdl/issues/775)) ([b1ee649](https://github.com/UnstoppableMango/tdl/commit/b1ee649ee29ec3ecb97b675a6b6cbecc55bf2179))
* **ir:** directives on enum variants ([#791](https://github.com/UnstoppableMango/tdl/issues/791)) ([c225216](https://github.com/UnstoppableMango/tdl/commit/c225216103e026736a4fd82e2f8f94df4cd579bd))
* **Makefile:** add `check` target to run nix flake checks ([#776](https://github.com/UnstoppableMango/tdl/issues/776)) ([624e892](https://github.com/UnstoppableMango/tdl/commit/624e892b1e7f6cd8769efc69dd769db0c095b2f3))
* **protobuf:** generate proto3 schemas ([#792](https://github.com/UnstoppableMango/tdl/issues/792)) ([26098ff](https://github.com/UnstoppableMango/tdl/commit/26098ff43d0209a0ae323e389e75ba49819de9c0))
* runs-on thecluster ([#739](https://github.com/UnstoppableMango/tdl/issues/739)) ([e516e44](https://github.com/UnstoppableMango/tdl/commit/e516e44409722368f007ceb196bf1606517f14c1))


### Documentation

* **go:** give every backend warning a phase or a deferred decision ([#778](https://github.com/UnstoppableMango/tdl/issues/778)) ([59f6b9f](https://github.com/UnstoppableMango/tdl/commit/59f6b9fc18ae07c686307ec238c529086074dffc))


### Code Refactoring

* **backend:** share backend helpers in backend/internal/emit ([#790](https://github.com/UnstoppableMango/tdl/issues/790)) ([c9ec7da](https://github.com/UnstoppableMango/tdl/commit/c9ec7da5d50f0e53ce38a1cfe214d7365ffc1699))

## [0.2.0](https://github.com/UnstoppableMango/tdl/compare/v0.1.8...v0.2.0) (2026-09-13)


### ⚠ BREAKING CHANGES

* the `entity`, `value`, and `key` keywords are removed, and the IR drops Field.key (3) and Class.requires_key (6).

### Features

* identity is conformance to Entity ([#773](https://github.com/UnstoppableMango/tdl/issues/773)) ([31fdd53](https://github.com/UnstoppableMango/tdl/commit/31fdd53bd5ab5496d8d7fdd4183d1f2f68dc1556))


### Bug Fixes

* **deps:** update golang.org/x/exp digest to 85c1c22 ([#769](https://github.com/UnstoppableMango/tdl/issues/769)) ([58ccc01](https://github.com/UnstoppableMango/tdl/commit/58ccc0181146a3b0890f437f65029778271a976a))


### Documentation

* **design:** identity is conformance, and entity, value, and key leave the language ([#766](https://github.com/UnstoppableMango/tdl/issues/766)) ([9076f6d](https://github.com/UnstoppableMango/tdl/commit/9076f6ddfd52819af88daaf6d005ed68268b911e))
* record how the review bots behave in a stack ([#768](https://github.com/UnstoppableMango/tdl/issues/768)) ([19593a2](https://github.com/UnstoppableMango/tdl/commit/19593a2242d40f90cb9ade12e241a43809ca25ed))

## [0.1.8](https://github.com/UnstoppableMango/tdl/compare/v0.1.7...v0.1.8) (2026-09-08)


### Features

* add Go code generation for enums, newtypes, and declarations ([d67e927](https://github.com/UnstoppableMango/tdl/commit/d67e927d758e447a350e5f41cc5036a09480089c))
* **backend/golang:** add Go source code generator backend ([d67e927](https://github.com/UnstoppableMango/tdl/commit/d67e927d758e447a350e5f41cc5036a09480089c))
* **gen:** register Go backend as a builtin and add integration tests ([d67e927](https://github.com/UnstoppableMango/tdl/commit/d67e927d758e447a350e5f41cc5036a09480089c))
* **golang:** add comparability check for Set elements and Map keys ([d67e927](https://github.com/UnstoppableMango/tdl/commit/d67e927d758e447a350e5f41cc5036a09480089c))
* **golang:** add Go backend type system and plugin entrypoint ([d67e927](https://github.com/UnstoppableMango/tdl/commit/d67e927d758e447a350e5f41cc5036a09480089c))


### Bug Fixes

* **parser:** record a name's position before reading the name ([#761](https://github.com/UnstoppableMango/tdl/issues/761)) ([3fe0aac](https://github.com/UnstoppableMango/tdl/commit/3fe0aacb855f8705141c9f8c63ef0685ba1f069e))


### Documentation

* add Go backend design document ([d67e927](https://github.com/UnstoppableMango/tdl/commit/d67e927d758e447a350e5f41cc5036a09480089c))
* add Go backend implementation plan ([d67e927](https://github.com/UnstoppableMango/tdl/commit/d67e927d758e447a350e5f41cc5036a09480089c))
* add Go backend to AGENTS.md and README.md documentation ([#754](https://github.com/UnstoppableMango/tdl/issues/754)) ([d67e927](https://github.com/UnstoppableMango/tdl/commit/d67e927d758e447a350e5f41cc5036a09480089c))
* **go-backend:** clarify prelude filtering, Set/Map key constraints, newtype constraint handling, and directive formatting ([d67e927](https://github.com/UnstoppableMango/tdl/commit/d67e927d758e447a350e5f41cc5036a09480089c))


### Code Refactoring

* **ast:** one head per declaration ([#757](https://github.com/UnstoppableMango/tdl/issues/757)) ([c4b9904](https://github.com/UnstoppableMango/tdl/commit/c4b99047a119980a63e7856d5bd0fd5b5bced79e))
* **gen:** report directive problems as diagnostics ([#759](https://github.com/UnstoppableMango/tdl/issues/759)) ([0e0033c](https://github.com/UnstoppableMango/tdl/commit/0e0033c6faae6d596dd43366177e44aeb4e0676a))
* **sema:** read the scope instead of two side tables ([#758](https://github.com/UnstoppableMango/tdl/issues/758)) ([6109c22](https://github.com/UnstoppableMango/tdl/commit/6109c223622c144eb85b9e9fd9fd647786a04ced))
* use the standard library where it says the same thing ([#760](https://github.com/UnstoppableMango/tdl/issues/760)) ([a939e51](https://github.com/UnstoppableMango/tdl/commit/a939e5173ec03d8e7e292746d03181f67cc36329))


### Tests

* add comprehensive test suite for Go backend code generation ([d67e927](https://github.com/UnstoppableMango/tdl/commit/d67e927d758e447a350e5f41cc5036a09480089c))
* **golang:** add comprehensive test suite for Go backend code generation ([d67e927](https://github.com/UnstoppableMango/tdl/commit/d67e927d758e447a350e5f41cc5036a09480089c))
* **golang:** upgrade file assertion to full type checking and add new test cases ([d67e927](https://github.com/UnstoppableMango/tdl/commit/d67e927d758e447a350e5f41cc5036a09480089c))

## [0.1.7](https://github.com/UnstoppableMango/tdl/compare/v0.1.6...v0.1.7) (2026-09-07)


### Features

* **ast:** record ordinary comments and where each block closes ([#750](https://github.com/UnstoppableMango/tdl/issues/750)) ([7327303](https://github.com/UnstoppableMango/tdl/commit/7327303c49748e050c0db9a7fe2e6d0adc41c156))
* **cli:** add fmt --check, and keep the file mode when writing ([#747](https://github.com/UnstoppableMango/tdl/issues/747)) ([784bd02](https://github.com/UnstoppableMango/tdl/commit/784bd02b00c2f4cbce8ead9ba3b6f22b206c7a5e))
* **cli:** read standard input when handed - ([#748](https://github.com/UnstoppableMango/tdl/issues/748)) ([d83823e](https://github.com/UnstoppableMango/tdl/commit/d83823e0bbef75124dd540a0905869d440559c73))
* **cli:** take more than one file argument ([#746](https://github.com/UnstoppableMango/tdl/issues/746)) ([9f2c29c](https://github.com/UnstoppableMango/tdl/commit/9f2c29cb2734440339c179a5a52773a636fdab4f))
* **fmt:** keep ordinary comments ([#751](https://github.com/UnstoppableMango/tdl/issues/751)) ([2fa76ec](https://github.com/UnstoppableMango/tdl/commit/2fa76ec4f4444efb509a83c32da9362b5f9b4540))


### Bug Fixes

* **cli:** hold a reusable plugin open under gen --watch ([#756](https://github.com/UnstoppableMango/tdl/issues/756)) ([a8ad074](https://github.com/UnstoppableMango/tdl/commit/a8ad0746254070b959cc6b68d9289447f142f894))


### Code Refactoring

* delete code nothing reaches ([#755](https://github.com/UnstoppableMango/tdl/issues/755)) ([fa79ea4](https://github.com/UnstoppableMango/tdl/commit/fa79ea4498f4174f2e0ef5a24a47a6e8ef3c50c4))


### Tests

* **parser:** assert the corpus and prelude are stored canonically ([#745](https://github.com/UnstoppableMango/tdl/issues/745)) ([874640a](https://github.com/UnstoppableMango/tdl/commit/874640a4905bf0330fff5ed2d4e667b7890614b3))

## [0.1.6](https://github.com/UnstoppableMango/tdl/compare/v0.1.5...v0.1.6) (2026-09-05)


### Features

* **nix/default.nix:** expose flake-parts module as flakeModules.default and tdl alias with integration check ([897ab26](https://github.com/UnstoppableMango/tdl/commit/897ab2668bc12c42066f560179a4368853cc889c))
* **nix:** add flake-parts module for integrating TDL into projects ([897ab26](https://github.com/UnstoppableMango/tdl/commit/897ab2668bc12c42066f560179a4368853cc889c))
* **nix:** add home-manager module for tdl and vscode-tdl ([#743](https://github.com/UnstoppableMango/tdl/issues/743)) ([d349a56](https://github.com/UnstoppableMango/tdl/commit/d349a56a0b23120da8c3c63020c60eb0a034f5ff))
* **nix:** add home-manager module for tdl CLI and VS Code extension ([d349a56](https://github.com/UnstoppableMango/tdl/commit/d349a56a0b23120da8c3c63020c60eb0a034f5ff))


### Documentation

* **AGENTS.md:** document flake-module.nix purpose, design decisions, and checks ([#744](https://github.com/UnstoppableMango/tdl/issues/744)) ([897ab26](https://github.com/UnstoppableMango/tdl/commit/897ab2668bc12c42066f560179a4368853cc889c))
* document nix overlay and installation instructions ([#738](https://github.com/UnstoppableMango/tdl/issues/738)) ([890ffe0](https://github.com/UnstoppableMango/tdl/commit/890ffe0cc6c03cb23f3dbe664f4a9e81b72c165c))
* one sentence per line throughout AGENTS.md ([#740](https://github.com/UnstoppableMango/tdl/issues/740)) ([c05c347](https://github.com/UnstoppableMango/tdl/commit/c05c347c0e2464d9184713aa0f29ff825b8945e7))
* **README.md:** add flakeModules.default usage example and fmt caveat ([897ab26](https://github.com/UnstoppableMango/tdl/commit/897ab2668bc12c42066f560179a4368853cc889c))


### Code Refactoring

* **nix:** extract packages and overlay into separate nix modules ([890ffe0](https://github.com/UnstoppableMango/tdl/commit/890ffe0cc6c03cb23f3dbe664f4a9e81b72c165c))

## [0.1.5](https://github.com/UnstoppableMango/tdl/compare/v0.1.4...v0.1.5) (2026-09-05)


### Features

* add grammar-driven contextual keyword and pattern utilities ([24b3c96](https://github.com/UnstoppableMango/tdl/commit/24b3c96b6ea19c8e35b1f43ac2a89a8b5910bfca))
* **release-please:** add VSCode extension package.json to release-please versioning config to keep editor extension version in sync with releases ([24b3c96](https://github.com/UnstoppableMango/tdl/commit/24b3c96b6ea19c8e35b1f43ac2a89a8b5910bfca))
* **textmate:** add TextMate grammar emitter for VS Code syntax highlighting ([24b3c96](https://github.com/UnstoppableMango/tdl/commit/24b3c96b6ea19c8e35b1f43ac2a89a8b5910bfca))
* **tools/textmate:** add build tool to derive VS Code TextMate grammar from EBNF grammar definition ([24b3c96](https://github.com/UnstoppableMango/tdl/commit/24b3c96b6ea19c8e35b1f43ac2a89a8b5910bfca))
* **vscode:** add VS Code extension with TDL syntax highlighting ([24b3c96](https://github.com/UnstoppableMango/tdl/commit/24b3c96b6ea19c8e35b1f43ac2a89a8b5910bfca))
* **vscode:** syntax highlighting derived from the grammar ([#735](https://github.com/UnstoppableMango/tdl/issues/735)) ([24b3c96](https://github.com/UnstoppableMango/tdl/commit/24b3c96b6ea19c8e35b1f43ac2a89a8b5910bfca))


### Documentation

* register the tree-sitter parser with nvim-treesitter ([#737](https://github.com/UnstoppableMango/tdl/issues/737)) ([1f62a1e](https://github.com/UnstoppableMango/tdl/commit/1f62a1ef0d2c3c1279488f711e01438abf1df7be))


### Tests

* **textmate:** add comprehensive tests for TextMate grammar emission ([24b3c96](https://github.com/UnstoppableMango/tdl/commit/24b3c96b6ea19c8e35b1f43ac2a89a8b5910bfca))

## [0.1.4](https://github.com/UnstoppableMango/tdl/compare/v0.1.3...v0.1.4) (2026-09-04)


### Features

* **ir:** model units in the schema ([#726](https://github.com/UnstoppableMango/tdl/issues/726)) ([2e66aa8](https://github.com/UnstoppableMango/tdl/commit/2e66aa886b1dcfe058aeae160ba211660e71567f))
* **sema:** lower unit arguments ([#728](https://github.com/UnstoppableMango/tdl/issues/728)) ([28a956c](https://github.com/UnstoppableMango/tdl/commit/28a956c8f2dcd69c72018949bc2d761608fcac1e))
* **treesitter:** derive grammar.js from the ebnf ([#722](https://github.com/UnstoppableMango/tdl/issues/722)) ([d935268](https://github.com/UnstoppableMango/tdl/commit/d93526880581d22dcefdf1931e7eaedb95a8237c))
* **treesitter:** scan regex literals externally ([#723](https://github.com/UnstoppableMango/tdl/issues/723)) ([e0ac880](https://github.com/UnstoppableMango/tdl/commit/e0ac880c91660ea6e6eae51a165056a5425f3a5a))
* **treesitter:** wire the derivation into CI ([#725](https://github.com/UnstoppableMango/tdl/issues/725)) ([51e174a](https://github.com/UnstoppableMango/tdl/commit/51e174a6c8009442152c07aad9b56f66a0976a4d))


### Bug Fixes

* **ci:** stop release-please owning a version a generator embeds ([#734](https://github.com/UnstoppableMango/tdl/issues/734)) ([6eb998e](https://github.com/UnstoppableMango/tdl/commit/6eb998e34535e923128c6f63ee740af419e9806e))


### Documentation

* **design:** editor support ([#733](https://github.com/UnstoppableMango/tdl/issues/733)) ([270605c](https://github.com/UnstoppableMango/tdl/commit/270605c923dc679f9a8a58de3e13cc1eb1d761f5))
* **ir:** units land, and the deferral goes ([#729](https://github.com/UnstoppableMango/tdl/issues/729)) ([cde3c58](https://github.com/UnstoppableMango/tdl/commit/cde3c588c6d6a12b517666e92cebd521478060ef))
* **README.md:** update README to reflect current project status and tooling ([#720](https://github.com/UnstoppableMango/tdl/issues/720)) ([2e974cd](https://github.com/UnstoppableMango/tdl/commit/2e974cd22af9030539940bae7d02380b2f33f527))


### Continuous Integration

* keep build tools out of coverage ([#732](https://github.com/UnstoppableMango/tdl/issues/732)) ([5184ee9](https://github.com/UnstoppableMango/tdl/commit/5184ee9b9e1e4706576a38264bcf102801311933))

## [0.1.3](https://github.com/UnstoppableMango/tdl/compare/v0.1.2...v0.1.3) (2026-09-01)


### Features

* drop the union keyword ([#717](https://github.com/UnstoppableMango/tdl/issues/717)) ([75f3373](https://github.com/UnstoppableMango/tdl/commit/75f3373ff0a5c5bef5a1c74755a7cec68668cd0c))
* **ebnf:** read the grammar's annotations ([#719](https://github.com/UnstoppableMango/tdl/issues/719)) ([272a11f](https://github.com/UnstoppableMango/tdl/commit/272a11f98f96d6dd8aca5ad6f4e8087381632f2c))

## [0.1.2](https://github.com/UnstoppableMango/tdl/compare/v0.1.1...v0.1.2) (2026-09-01)


### Features

* **ebnf:** annotate the grammar ([#713](https://github.com/UnstoppableMango/tdl/issues/713)) ([5bfba57](https://github.com/UnstoppableMango/tdl/commit/5bfba57273c69b6c981270737bbb75b3a462aef4))


### Bug Fixes

* **gen:** stop racing a plugin's stderr, and run CI with -race ([#715](https://github.com/UnstoppableMango/tdl/issues/715)) ([c63257e](https://github.com/UnstoppableMango/tdl/commit/c63257e00cec53891e586cd7b228a70dfc7dbcb3))
* run gomod2nix from the tidy recipe ([#712](https://github.com/UnstoppableMango/tdl/issues/712)) ([49423f3](https://github.com/UnstoppableMango/tdl/commit/49423f3b9a002ffdaec29ba01ea5a63251295b20)), closes [#711](https://github.com/UnstoppableMango/tdl/issues/711)

## [0.1.1](https://github.com/UnstoppableMango/tdl/compare/v0.1.0...v0.1.1) (2026-09-01)


### Features

* **ebnf:** formalize the notation and lint the grammar ([#709](https://github.com/UnstoppableMango/tdl/issues/709)) ([7caa49a](https://github.com/UnstoppableMango/tdl/commit/7caa49a2c8f343018a736adc6e30e7dab44e349c))
* **lex:** describe the lexer to a program ([#703](https://github.com/UnstoppableMango/tdl/issues/703)) ([12e00f2](https://github.com/UnstoppableMango/tdl/commit/12e00f251872c7b0059f7dfc25d67e78f9f45b57))


### Bug Fixes

* **parser:** stop a field named where from being the previous field's constraints ([#707](https://github.com/UnstoppableMango/tdl/issues/707)) ([545a946](https://github.com/UnstoppableMango/tdl/commit/545a946857a87e32514740ebd62d8553a32dfe82))


### Documentation

* add a support matrix ([#702](https://github.com/UnstoppableMango/tdl/issues/702)) ([6425f95](https://github.com/UnstoppableMango/tdl/commit/6425f95c3b972cf0584f12a0272dc495b17fa26b))
* **design:** derive the tree-sitter grammar from the ebnf ([#704](https://github.com/UnstoppableMango/tdl/issues/704)) ([a90e306](https://github.com/UnstoppableMango/tdl/commit/a90e3063f55586d5561dcf49484c81dac894f904))
* state where a name may be a reserved word ([#708](https://github.com/UnstoppableMango/tdl/issues/708)) ([503d3c3](https://github.com/UnstoppableMango/tdl/commit/503d3c3269985eb9384b5afc0bf1da637eee9e96))


### Continuous Integration

* upload coverage to Codecov ([#700](https://github.com/UnstoppableMango/tdl/issues/700)) ([5fbef4e](https://github.com/UnstoppableMango/tdl/commit/5fbef4ea5cfb37a18bfc6cf46d28a2c5d384b179))

## [0.1.0](https://github.com/UnstoppableMango/tdl/compare/v0.0.34...v0.1.0) (2026-08-30)


### ⚠ BREAKING CHANGES

* rewrite the lexer and parser for the new grammar ([#677](https://github.com/UnstoppableMango/tdl/issues/677))

### Features

* **cli:** add `play` command for interactive TDL file playground ([cf8f453](https://github.com/UnstoppableMango/tdl/commit/cf8f453eb8b828a00bfc3e67ad533db622c2ff70))
* **cli:** add tokens, ast, and play commands with tests ([cf8f453](https://github.com/UnstoppableMango/tdl/commit/cf8f453eb8b828a00bfc3e67ad533db622c2ff70))
* **examples:** add example TDL files and CLI ast command ([cf8f453](https://github.com/UnstoppableMango/tdl/commit/cf8f453eb8b828a00bfc3e67ad533db622c2ff70))
* **gen:** add --verify and --clean ([#691](https://github.com/UnstoppableMango/tdl/issues/691)) ([0f811f0](https://github.com/UnstoppableMango/tdl/commit/0f811f0472dc56ca9bccc4127fe3410f84b21ed9))
* **gen:** add tdl gen, in process ([#690](https://github.com/UnstoppableMango/tdl/issues/690)) ([dfc1ec2](https://github.com/UnstoppableMango/tdl/commit/dfc1ec23dd9c7a191e9e5b7ea931440de1148022))
* **gen:** check directives against what a backend declares ([#695](https://github.com/UnstoppableMango/tdl/issues/695)) ([f93543d](https://github.com/UnstoppableMango/tdl/commit/f93543d37fb697900306b20d62004e9e466e1b16))
* **gen:** reuse plugins and regenerate on save ([#696](https://github.com/UnstoppableMango/tdl/issues/696)) ([d026b94](https://github.com/UnstoppableMango/tdl/commit/d026b946e8f08113fef3c0d9a17cea07ff18166f))
* **gen:** run backends as subprocesses ([#692](https://github.com/UnstoppableMango/tdl/issues/692)) ([427124e](https://github.com/UnstoppableMango/tdl/commit/427124ec9ef0139c1ce64d9c8492a6c69272cd5b))
* imports, classes, instances, constraints, and target resolution ([#679](https://github.com/UnstoppableMango/tdl/issues/679)) ([be2e30c](https://github.com/UnstoppableMango/tdl/commit/be2e30c4b88b6862b5dc2ffe43836018fbecb593))
* M1 lexer, parser, and the Nix build ([#675](https://github.com/UnstoppableMango/tdl/issues/675)) ([cf8f453](https://github.com/UnstoppableMango/tdl/commit/cf8f453eb8b828a00bfc3e67ad533db622c2ff70))
* **plugin:** add the backend contract and a backend that proves it ([#689](https://github.com/UnstoppableMango/tdl/issues/689)) ([27356a3](https://github.com/UnstoppableMango/tdl/commit/27356a3850c8851ebe66254f22d98682ff3e44d6))
* **plugin:** add the wire schema and the framing codec ([#688](https://github.com/UnstoppableMango/tdl/issues/688)) ([750c2b7](https://github.com/UnstoppableMango/tdl/commit/750c2b7b4d5fd2a044b26fe3f097233a5ae1a2d8))
* rewrite the lexer and parser for the new grammar ([#677](https://github.com/UnstoppableMango/tdl/issues/677)) ([c727439](https://github.com/UnstoppableMango/tdl/commit/c7274390162f7dc69c5b6aeb6e9c7968dc380836))
* the ir schema, name resolution, and the loaded prelude ([#678](https://github.com/UnstoppableMango/tdl/issues/678)) ([8a79324](https://github.com/UnstoppableMango/tdl/commit/8a79324fd5e1b3f98552cb15d1624175e2cb75f3))


### Documentation

* add a backlog ([#685](https://github.com/UnstoppableMango/tdl/issues/685)) ([638dd6a](https://github.com/UnstoppableMango/tdl/commit/638dd6ae5b9886e50ec2dcee1eae8525c0f46e8c))
* add badges and make the status section scannable ([#684](https://github.com/UnstoppableMango/tdl/issues/684)) ([ec01301](https://github.com/UnstoppableMango/tdl/commit/ec013012b87cc816b14e39eedfdecb23684045bf))
* **design:** plan the plugin protocol ([#687](https://github.com/UnstoppableMango/tdl/issues/687)) ([0764ec5](https://github.com/UnstoppableMango/tdl/commit/0764ec511369d0b2db6212796f022a3cfbce5e40))
* **design:** revise the plugin protocol against the built ir ([#686](https://github.com/UnstoppableMango/tdl/issues/686)) ([7c46d11](https://github.com/UnstoppableMango/tdl/commit/7c46d11f0a321cab41874124dafce8a19df15ec0))
* **design:** the workflow, ir, plugin protocol, and implementation plans ([#676](https://github.com/UnstoppableMango/tdl/issues/676)) ([d124337](https://github.com/UnstoppableMango/tdl/commit/d12433748095136dd836761c6e7a69bb912ffbc9))
* **plugin:** document the SDK and record protocol exchanges ([#694](https://github.com/UnstoppableMango/tdl/issues/694)) ([bc77ee5](https://github.com/UnstoppableMango/tdl/commit/bc77ee5cc0c1b4272a15b1054e927b1499980fc0))
* rewrite spec to reflect current language design ([cf8f453](https://github.com/UnstoppableMango/tdl/commit/cf8f453eb8b828a00bfc3e67ad533db622c2ff70))
* **spec.md:** add documentation for class-based path directives in target blocks ([cf8f453](https://github.com/UnstoppableMango/tdl/commit/cf8f453eb8b828a00bfc3e67ad533db622c2ff70))
* **spec.md:** expand TDL specification with new language features ([cf8f453](https://github.com/UnstoppableMango/tdl/commit/cf8f453eb8b828a00bfc3e67ad533db622c2ff70))
* **spec.md:** rewrite TDL specification to reflect updated language design ([cf8f453](https://github.com/UnstoppableMango/tdl/commit/cf8f453eb8b828a00bfc3e67ad533db622c2ff70))


### Build System

* format nearly everything with treefmt ([#682](https://github.com/UnstoppableMango/tdl/issues/682)) ([e9f3c95](https://github.com/UnstoppableMango/tdl/commit/e9f3c954b06ceb415eb617c794a1cd2e43c2f255))
* move the protos to editions 2024 and go_package to managed mode ([#697](https://github.com/UnstoppableMango/tdl/issues/697)) ([4fbbb12](https://github.com/UnstoppableMango/tdl/commit/4fbbb12926710a2e8d3bc9c8f7de24ebebf88891))


### Continuous Integration

* bump actions to their latest releases and pin every one to a SHA ([#683](https://github.com/UnstoppableMango/tdl/issues/683)) ([7c13e50](https://github.com/UnstoppableMango/tdl/commit/7c13e50691e3d398fbaeb1454a7da956a5336676))
* check protos with buf again ([#681](https://github.com/UnstoppableMango/tdl/issues/681)) ([f07bb47](https://github.com/UnstoppableMango/tdl/commit/f07bb4721019144145f28bfda12b23505ff8e4cd))
* configure release-please ([#698](https://github.com/UnstoppableMango/tdl/issues/698)) ([3bf9c5c](https://github.com/UnstoppableMango/tdl/commit/3bf9c5c94e5e950c71882304fbef4a5ed4a34f02))
