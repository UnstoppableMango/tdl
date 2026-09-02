# Changelog

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
