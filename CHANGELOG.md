# Changelog

## [0.5.4](https://github.com/UnstoppableMango/tdl/compare/v0.5.3...v0.5.4) (2026-10-28)


### Features

* **gen:** write a make dependency file with --make ([#1269](https://github.com/UnstoppableMango/tdl/issues/1269)) ([89e91c5](https://github.com/UnstoppableMango/tdl/commit/89e91c5f631588a77e0d5293d0d2ee0401422c09))


### Documentation

* **context:** define output loss and read-back loss ([#1324](https://github.com/UnstoppableMango/tdl/issues/1324)) ([db5521c](https://github.com/UnstoppableMango/tdl/commit/db5521cacfc92b21cca9945ad6761e2bb23f2661))
* route new designs, plans, and decisions through the agent skills ([#1190](https://github.com/UnstoppableMango/tdl/issues/1190)) ([c0f619c](https://github.com/UnstoppableMango/tdl/commit/c0f619ce9b29813b5ec0c8c3efe81789bd3cc836))


### Code Refactoring

* **emit:** share the include-flattening warning ([#1322](https://github.com/UnstoppableMango/tdl/issues/1322)) ([4b13ed5](https://github.com/UnstoppableMango/tdl/commit/4b13ed56ae870e12301867991d889033b035d1c5))
* **gmk:** stop make through gmk.Error ([#1325](https://github.com/UnstoppableMango/tdl/issues/1325)) ([411919f](https://github.com/UnstoppableMango/tdl/commit/411919faf366f105344bf6b4f2dc95430d7ee71e))

## [0.5.3](https://github.com/UnstoppableMango/tdl/compare/v0.5.2...v0.5.3) (2026-10-25)


### Bug Fixes

* **ir:** write a directive's name argument without the dump's unresolved mark ([3ebb5a6](https://github.com/UnstoppableMango/tdl/commit/3ebb5a6d1a9f866be155a8ec09eac8445f99345c))
* **nix:** link tdl-gen-csharp on darwin, and build there in CI ([#1185](https://github.com/UnstoppableMango/tdl/issues/1185)) ([39a0d1e](https://github.com/UnstoppableMango/tdl/commit/39a0d1ec1fae7988f8988cb7a9cdf5bc265b3a99))


### Documentation

* **AGENTS.md:** 📝 add Graphify usage guidelines for code navigation ([#1182](https://github.com/UnstoppableMango/tdl/issues/1182)) ([9431677](https://github.com/UnstoppableMango/tdl/commit/94316773f8691b258f2bd73e410f08207bae3dda))


### Tests

* **mutation:** list 28 equivalent mutants ([#1171](https://github.com/UnstoppableMango/tdl/issues/1171)) ([1e416a3](https://github.com/UnstoppableMango/tdl/commit/1e416a347b6e1949aa4caff992f184d35965c21b)), closes [#1135](https://github.com/UnstoppableMango/tdl/issues/1135) [#1151](https://github.com/UnstoppableMango/tdl/issues/1151)


### Dependencies

* update dependency vscode-languageclient to v11 ([#1177](https://github.com/UnstoppableMango/tdl/issues/1177)) ([2d7b3b8](https://github.com/UnstoppableMango/tdl/commit/2d7b3b861bcf1cea1ad2ba3f7cac324b22a0f61f))
* update github.com/bufbuild/protocompile digest to b3c25bb ([#1172](https://github.com/UnstoppableMango/tdl/issues/1172)) ([73551c4](https://github.com/UnstoppableMango/tdl/commit/73551c4ec2683f2cadaa1b61025bc1a5fec001c2))
* update module github.com/unmango/go-gmk to v0.2.0 ([#1175](https://github.com/UnstoppableMango/tdl/issues/1175)) ([e723c0e](https://github.com/UnstoppableMango/tdl/commit/e723c0edd23642e50e90ef6df33eec6708a67326))
* update module github.com/vektah/gqlparser/v2 to v2.5.64 ([#1173](https://github.com/UnstoppableMango/tdl/issues/1173)) ([cdd5081](https://github.com/UnstoppableMango/tdl/commit/cdd5081be45edcfbe84bd214eb6066a9c60a91b8))

## [0.5.2](https://github.com/UnstoppableMango/tdl/compare/v0.5.1...v0.5.2) (2026-10-24)


### Features

* **cli:** tdl request writes the requests gen would send ([#1154](https://github.com/UnstoppableMango/tdl/issues/1154)) ([ee70f14](https://github.com/UnstoppableMango/tdl/commit/ee70f14218949df7f900fce58a0a99bb12d6dd87))
* **make:** GNU make functions for listing TDL files and imports ([#1165](https://github.com/UnstoppableMango/tdl/issues/1165)) ([e7aac48](https://github.com/UnstoppableMango/tdl/commit/e7aac480ea2bb493014c2cfd687e74ce535dc9a4))
* **typescript:** serve the backend as a TypeScript 7.1 content mapper ([#1152](https://github.com/UnstoppableMango/tdl/issues/1152)) ([40b9aea](https://github.com/UnstoppableMango/tdl/commit/40b9aea5e4ca86296d2c1b578957a72aeb023723))


### Documentation

* configure the agent skills' issue tracker, triage labels, and domain docs ([#1170](https://github.com/UnstoppableMango/tdl/issues/1170)) ([fc52d1b](https://github.com/UnstoppableMango/tdl/commit/fc52d1b879125d04cc1d6a4b3d3bf1438f49df86))
* **design:** a gRPC transport for plugins, chosen at spawn ([#1169](https://github.com/UnstoppableMango/tdl/issues/1169)) ([2cf4ede](https://github.com/UnstoppableMango/tdl/commit/2cf4edebbd29ad25236bbfb948002ba816420f91))

## [0.5.1](https://github.com/UnstoppableMango/tdl/compare/v0.5.0...v0.5.1) (2026-10-23)


### Features

* add feature pages for the backends not built yet ([#1130](https://github.com/UnstoppableMango/tdl/issues/1130)) ([374a189](https://github.com/UnstoppableMango/tdl/commit/374a189b30337fe389ff3b9aff4285c8418b26ae))
* compare every backend in generate and import matrices ([#1129](https://github.com/UnstoppableMango/tdl/issues/1129)) ([de9ec71](https://github.com/UnstoppableMango/tdl/commit/de9ec7108627c76e115c600399ca821a683e76fe))
* **dotnet:** publish tdl-gen-csharp with Native AOT ([#1122](https://github.com/UnstoppableMango/tdl/issues/1122)) ([944c01d](https://github.com/UnstoppableMango/tdl/commit/944c01dea3c421a73b3545c8df6657a0ac0982ac))
* measure each backend's feature support into a page per backend ([#1121](https://github.com/UnstoppableMango/tdl/issues/1121)) ([7ce23f8](https://github.com/UnstoppableMango/tdl/commit/7ce23f86be40e6b420ea5fb356162394ccbc389e))
* **nix:** read TDL in the devShell's graphify ([#1132](https://github.com/UnstoppableMango/tdl/issues/1132)) ([98f3e07](https://github.com/UnstoppableMango/tdl/commit/98f3e07362a32bd81cab91c8cd62dda9746738c5))
* **protobuf:** write options in buf format's layout ([#1159](https://github.com/UnstoppableMango/tdl/issues/1159)) ([b8a5384](https://github.com/UnstoppableMango/tdl/commit/b8a5384edf9acbb8d3fbe9fd0c06a18b00436ff1))
* **tree-sitter:** add the grammar's Python binding ([#1118](https://github.com/UnstoppableMango/tdl/issues/1118)) ([88ec1ef](https://github.com/UnstoppableMango/tdl/commit/88ec1ef9b153bf9c0566892eae4c1ad76e4b171b))
* **zod:** defaults, foreign schemas, and externs (phase 2) ([#1156](https://github.com/UnstoppableMango/tdl/issues/1156)) ([3fac77a](https://github.com/UnstoppableMango/tdl/commit/3fac77a85aa0ef858aa2fd5b6df1cdd1a74997e6))
* **zod:** the zod backend, phase 1 ([#1155](https://github.com/UnstoppableMango/tdl/issues/1155)) ([bc54130](https://github.com/UnstoppableMango/tdl/commit/bc54130653997842240045a733ace6cc974b8c58))


### Documentation

* design a C# source generator for live TDL types ([#1128](https://github.com/UnstoppableMango/tdl/issues/1128)) ([a965090](https://github.com/UnstoppableMango/tdl/commit/a96509057ad5706a7af380f5a0e2ee04a6b38c7b))
* design importing target-language files ([#1125](https://github.com/UnstoppableMango/tdl/issues/1125)) ([505f06b](https://github.com/UnstoppableMango/tdl/commit/505f06b87acd5c95e44115939a4741040816c806))
* design the CML backend ([#1153](https://github.com/UnstoppableMango/tdl/issues/1153)) ([8a32c04](https://github.com/UnstoppableMango/tdl/commit/8a32c04ea9914c2a2451ffb8201f89ca972c09c4))
* design the TypeScript content mapper ([#1142](https://github.com/UnstoppableMango/tdl/issues/1142)) ([86b57ab](https://github.com/UnstoppableMango/tdl/commit/86b57ab9e8a5540dee476dd3752747ff09133153))
* **graphify:** measure TDL's growth instead of a 4,000-node cap ([#1144](https://github.com/UnstoppableMango/tdl/issues/1144)) ([b9ef773](https://github.com/UnstoppableMango/tdl/commit/b9ef77300df562554d863e2dfbdbf6dde6771cdb))
* **mutation:** mark phase 3 done ([#1141](https://github.com/UnstoppableMango/tdl/issues/1141)) ([505f7b5](https://github.com/UnstoppableMango/tdl/commit/505f7b5a9c66de2dfeef3db41d6b7910c650db66))


### Code Refactoring

* **emit:** share name, owned and default loss checks ([#1127](https://github.com/UnstoppableMango/tdl/issues/1127)) ([a9021e9](https://github.com/UnstoppableMango/tdl/commit/a9021e9697ac11075604ee0358488001297fc9b2))
* **host:** move directive checks and loss silencing below gen ([#1145](https://github.com/UnstoppableMango/tdl/issues/1145)) ([5223f4e](https://github.com/UnstoppableMango/tdl/commit/5223f4ee73951726ae0c560c12e1f83dc202c984))


### Tests

* **analysis:** kill 10 mutation survivors ([#1140](https://github.com/UnstoppableMango/tdl/issues/1140)) ([f421c4c](https://github.com/UnstoppableMango/tdl/commit/f421c4cd6d7b3ce37fbe9eb58fc3c362f92f34f5))
* **analysis:** kill 10 mutation survivors ([#1150](https://github.com/UnstoppableMango/tdl/issues/1150)) ([e17e67a](https://github.com/UnstoppableMango/tdl/commit/e17e67a5cd492bae6209b6e03dbc6412ffad047c))
* **ast:** kill 9 mutation survivors ([#1138](https://github.com/UnstoppableMango/tdl/issues/1138)) ([835ef2c](https://github.com/UnstoppableMango/tdl/commit/835ef2ccbbe1012027f0880222945382c51ff1c8))
* **lex:** kill 1 mutation survivor ([#1147](https://github.com/UnstoppableMango/tdl/issues/1147)) ([a877b9e](https://github.com/UnstoppableMango/tdl/commit/a877b9ec5db826dc1713a7a22d63224a6a38114c))
* **lex:** kill 1 mutation survivors ([a877b9e](https://github.com/UnstoppableMango/tdl/commit/a877b9ec5db826dc1713a7a22d63224a6a38114c))
* **lex:** kill 10 mutation survivors ([#1136](https://github.com/UnstoppableMango/tdl/issues/1136)) ([4aa1764](https://github.com/UnstoppableMango/tdl/commit/4aa176412afa619e74b5c9271ade09d2cea78d52))
* **parser:** kill 4 mutation survivors ([#1148](https://github.com/UnstoppableMango/tdl/issues/1148)) ([7974448](https://github.com/UnstoppableMango/tdl/commit/7974448504ae237598eb87b85a2f821811dc47a0))
* **parser:** kill 8 mutation survivors ([#1137](https://github.com/UnstoppableMango/tdl/issues/1137)) ([eddded1](https://github.com/UnstoppableMango/tdl/commit/eddded194987ae2bc4c0a8db7428d5cdc74a60d1))
* **sema:** kill 10 mutation survivors ([#1139](https://github.com/UnstoppableMango/tdl/issues/1139)) ([bc17e86](https://github.com/UnstoppableMango/tdl/commit/bc17e86efa456fe9793353ea0e4e46c0fd807414))
* **sema:** kill 10 mutation survivors ([#1149](https://github.com/UnstoppableMango/tdl/issues/1149)) ([a786460](https://github.com/UnstoppableMango/tdl/commit/a786460fe3539caf0e6660c0edca80695daff8b2))


### Build System

* add make mutate ([#1123](https://github.com/UnstoppableMango/tdl/issues/1123)) ([97169c9](https://github.com/UnstoppableMango/tdl/commit/97169c993ad609395f65edda666a810f6f563309))

## [0.5.0](https://github.com/UnstoppableMango/tdl/compare/v0.4.15...v0.5.0) (2026-10-18)


### ⚠ BREAKING CHANGES

* serve the C# backend as a .NET plugin on Roslyn ([#1116](https://github.com/UnstoppableMango/tdl/issues/1116))

### Features

* serve the C# backend as a .NET plugin on Roslyn ([#1116](https://github.com/UnstoppableMango/tdl/issues/1116)) ([66e5666](https://github.com/UnstoppableMango/tdl/commit/66e566613c95929abab0eef7979c2d0c87a3a210))


### Documentation

* **agents:** keep design status in each plan, not AGENTS.md ([#1119](https://github.com/UnstoppableMango/tdl/issues/1119)) ([ce635a0](https://github.com/UnstoppableMango/tdl/commit/ce635a01c0815aa0b440cc2f5f9be78c9f4e0f32))
* design TDL support in graphify ([#1115](https://github.com/UnstoppableMango/tdl/issues/1115)) ([4b05146](https://github.com/UnstoppableMango/tdl/commit/4b0514679497767f51b3dd9df0205f761eae669d))


### Build System

* add gremlins to the devShell ([#1114](https://github.com/UnstoppableMango/tdl/issues/1114)) ([3d64b51](https://github.com/UnstoppableMango/tdl/commit/3d64b51ed051982fae1102c9655c77b51a9ce181))

## [0.4.15](https://github.com/UnstoppableMango/tdl/compare/v0.4.14...v0.4.15) (2026-10-18)


### Features

* **dotnet:** add the .NET plugin SDK and a debug port ([#1111](https://github.com/UnstoppableMango/tdl/issues/1111)) ([aca37b3](https://github.com/UnstoppableMango/tdl/commit/aca37b370df6985e318df7fef03c5f9fdc903838))

## [0.4.14](https://github.com/UnstoppableMango/tdl/compare/v0.4.13...v0.4.14) (2026-10-17)


### Features

* **plugin:** carry the model as TDL and accept imported source ([#1105](https://github.com/UnstoppableMango/tdl/issues/1105)) ([87890a4](https://github.com/UnstoppableMango/tdl/commit/87890a45cc7a94b59de192f7af659893e3e5f008))


### Documentation

* design mutation testing for the core packages ([#1109](https://github.com/UnstoppableMango/tdl/issues/1109)) ([5b07174](https://github.com/UnstoppableMango/tdl/commit/5b07174a7cf2c52ae120efff7634379e12668b47))

## [0.4.13](https://github.com/UnstoppableMango/tdl/compare/v0.4.12...v0.4.13) (2026-10-16)


### Features

* **haskell:** classes ([#1098](https://github.com/UnstoppableMango/tdl/issues/1098)) ([711c725](https://github.com/UnstoppableMango/tdl/commit/711c725d7597acd5017496eda3571ac877fced61))


### Dependencies

* update github.com/bufbuild/protocompile digest to b72c9bf ([#1107](https://github.com/UnstoppableMango/tdl/issues/1107)) ([36a5785](https://github.com/UnstoppableMango/tdl/commit/36a578585f6ec320afa8767c5cc2f796e62d944c))

## [0.4.12](https://github.com/UnstoppableMango/tdl/compare/v0.4.11...v0.4.12) (2026-10-16)


### Features

* **plugin:** number members and read constraints in the host ([#1099](https://github.com/UnstoppableMango/tdl/issues/1099)) ([7a5bd35](https://github.com/UnstoppableMango/tdl/commit/7a5bd35dd77758147acfcbc02266f56b055515dd))


### Dependencies

* update github.com/bufbuild/protocompile digest to 1dc6457 ([#1101](https://github.com/UnstoppableMango/tdl/issues/1101)) ([e909c78](https://github.com/UnstoppableMango/tdl/commit/e909c78317e81ce5d4b53cf0da4929075e1e3223))
* update golang.org/x/exp digest to f0251eb ([#1102](https://github.com/UnstoppableMango/tdl/issues/1102)) ([41cb3d3](https://github.com/UnstoppableMango/tdl/commit/41cb3d375dc21e4f058eebd77f976addeb6df50d))
* update module github.com/vektah/gqlparser/v2 to v2.5.63 ([#1103](https://github.com/UnstoppableMango/tdl/issues/1103)) ([7e7154c](https://github.com/UnstoppableMango/tdl/commit/7e7154c106444000b5d8dbae528b5151c3ad5103))

## [0.4.11](https://github.com/UnstoppableMango/tdl/compare/v0.4.10...v0.4.11) (2026-10-15)


### Features

* **haskell:** generics ([#1094](https://github.com/UnstoppableMango/tdl/issues/1094)) ([90fbfc0](https://github.com/UnstoppableMango/tdl/commit/90fbfc0a35aba300a22c2e20d7f76545b37f7ee1))
* **plugin:** check directive placement and values in the host ([#1093](https://github.com/UnstoppableMango/tdl/issues/1093)) ([aa98d4b](https://github.com/UnstoppableMango/tdl/commit/aa98d4b2965dab15f74cdc856a298708a62ae070))


### Documentation

* design anonymous unions ([#1092](https://github.com/UnstoppableMango/tdl/issues/1092)) ([b10c1a1](https://github.com/UnstoppableMango/tdl/commit/b10c1a13b33bbcbb10d0ad98e4eb685faad5c86a))

## [0.4.10](https://github.com/UnstoppableMango/tdl/compare/v0.4.9...v0.4.10) (2026-10-14)


### Features

* **csharp:** keys, defaults, foreign types, externs, and attributes ([#1083](https://github.com/UnstoppableMango/tdl/issues/1083)) ([3e6c323](https://github.com/UnstoppableMango/tdl/commit/3e6c323751360622006c455eede889f930c1181e))
* **csharp:** records, enums, and the type mapping ([#1082](https://github.com/UnstoppableMango/tdl/issues/1082)) ([dd2e6ae](https://github.com/UnstoppableMango/tdl/commit/dd2e6ae055e303adcc1b92d2e4097b22eaacef86))
* **plugin:** send an analysis of the model in every request ([#1087](https://github.com/UnstoppableMango/tdl/issues/1087)) ([0681f56](https://github.com/UnstoppableMango/tdl/commit/0681f569818db0652fb7c465b17d862561929c60))


### Documentation

* design a Zod backend ([#1078](https://github.com/UnstoppableMango/tdl/issues/1078)) ([9ca2b95](https://github.com/UnstoppableMango/tdl/commit/9ca2b95036f886d0ed89d7a6d6a470d5d7cf3533))
* design plugins in any language ([#1085](https://github.com/UnstoppableMango/tdl/issues/1085)) ([e0e3734](https://github.com/UnstoppableMango/tdl/commit/e0e3734f37cd8f76ee861a25d9ba799134987357))


### Code Refactoring

* **backend:** share the reverse targets' loss checks through emit ([#1081](https://github.com/UnstoppableMango/tdl/issues/1081)) ([d159969](https://github.com/UnstoppableMango/tdl/commit/d159969d12958e591328a817838c07c5146a3dc6))


### Dependencies

* update github.com/bufbuild/protocompile digest to b79bfe2 ([#1088](https://github.com/UnstoppableMango/tdl/issues/1088)) ([72af0bd](https://github.com/UnstoppableMango/tdl/commit/72af0bded2017f84661c68bb022e0c45d39dafcb))
* update golang.org/x/exp digest to 2ab7f3c ([#1089](https://github.com/UnstoppableMango/tdl/issues/1089)) ([7f74510](https://github.com/UnstoppableMango/tdl/commit/7f74510f88ed12cf2d568a63b9536889a99f823f))
* update golang.org/x/exp digest to faecfb2 ([#1076](https://github.com/UnstoppableMango/tdl/issues/1076)) ([dc18191](https://github.com/UnstoppableMango/tdl/commit/dc181915d5a9550ad66218304cbe1fd1ac7fde25))

## [0.4.9](https://github.com/UnstoppableMango/tdl/compare/v0.4.8...v0.4.9) (2026-10-12)


### Features

* **lsp:** list a node's target directives in its hover ([#1074](https://github.com/UnstoppableMango/tdl/issues/1074)) ([0e98770](https://github.com/UnstoppableMango/tdl/commit/0e98770f3e8e0e0ef1d8717659cf976f2aaa296b))

## [0.4.8](https://github.com/UnstoppableMango/tdl/compare/v0.4.7...v0.4.8) (2026-10-12)


### Features

* **haskell:** generate keys, foreign types, externs, and derive ([#1069](https://github.com/UnstoppableMango/tdl/issues/1069)) ([7f86090](https://github.com/UnstoppableMango/tdl/commit/7f860906d87b82c35e11bab472ba71b348e0c3ec))
* **smithy:** import Smithy back to TDL ([#1073](https://github.com/UnstoppableMango/tdl/issues/1073)) ([a775409](https://github.com/UnstoppableMango/tdl/commit/a775409abffeae5d36431c931c7cdcea6d674453))


### Bug Fixes

* **sema:** a parameter-typed field doesn't satisfy a concrete class field ([#1072](https://github.com/UnstoppableMango/tdl/issues/1072)) ([4800c37](https://github.com/UnstoppableMango/tdl/commit/4800c373f12ca0dcccc2a348ba917e3886426d7c)), closes [#1011](https://github.com/UnstoppableMango/tdl/issues/1011)


### Documentation

* design a Python backend ([#1068](https://github.com/UnstoppableMango/tdl/issues/1068)) ([36991e7](https://github.com/UnstoppableMango/tdl/commit/36991e704779c00c5340e420788183305c48c662))
* design a Rust backend ([#1066](https://github.com/UnstoppableMango/tdl/issues/1066)) ([cefb786](https://github.com/UnstoppableMango/tdl/commit/cefb786dbf749632479f87b68d569b6bc3094ea9))

## [0.4.7](https://github.com/UnstoppableMango/tdl/compare/v0.4.6...v0.4.7) (2026-10-11)


### Features

* **haskell:** generate records, enums, newtypes, and aliases ([#1067](https://github.com/UnstoppableMango/tdl/issues/1067)) ([b6c5cc9](https://github.com/UnstoppableMango/tdl/commit/b6c5cc9684c97e5330ca152fa16aad41f213d41e))
* **typescript:** import externs from foreign modules and dependencies ([#1065](https://github.com/UnstoppableMango/tdl/issues/1065)) ([2ce02b4](https://github.com/UnstoppableMango/tdl/commit/2ce02b458672e525a4ba0da3aa74098d739dc37d))


### Code Refactoring

* **models:** move unist, mdast, and hast to unmango/treespec ([#1062](https://github.com/UnstoppableMango/tdl/issues/1062)) ([b3bd9bd](https://github.com/UnstoppableMango/tdl/commit/b3bd9bd04dcd37ff53874b209c94f427671fd0cb))

## [0.4.6](https://github.com/UnstoppableMango/tdl/compare/v0.4.5...v0.4.6) (2026-10-10)


### Features

* give positions an end and add ast.Inspect ([#1059](https://github.com/UnstoppableMango/tdl/issues/1059)) ([f7b046a](https://github.com/UnstoppableMango/tdl/commit/f7b046a3b29cb82a35731a58bf0e336cc5eb9907))
* **go:** opt in to the JSON wire convention ([#1053](https://github.com/UnstoppableMango/tdl/issues/1053)) ([e6a01d9](https://github.com/UnstoppableMango/tdl/commit/e6a01d93e16db05eeed825da6703e1e10d34504a))
* **go:** refer to an extern through its dependency's go target block ([#1052](https://github.com/UnstoppableMango/tdl/issues/1052)) ([ad4a293](https://github.com/UnstoppableMango/tdl/commit/ad4a2937170fccf44bc07c74f530af82c13ace0f))
* **models:** describe unist, mdast, and hast in TDL ([#1028](https://github.com/UnstoppableMango/tdl/issues/1028)) ([ecaba0c](https://github.com/UnstoppableMango/tdl/commit/ecaba0cc22229a5f1d151187c2a64dc914dfaa11))
* **typescript:** import TypeScript back to TDL ([#1058](https://github.com/UnstoppableMango/tdl/issues/1058)) ([72b3f53](https://github.com/UnstoppableMango/tdl/commit/72b3f53a100991835ba574bb8bcb1f5c55c09326))


### Bug Fixes

* **sema:** let a value reach itself through a spelled-out collection ([#1054](https://github.com/UnstoppableMango/tdl/issues/1054)) ([09563e4](https://github.com/UnstoppableMango/tdl/commit/09563e47b2ff213156a6a2bca4ac3a572d59a333))
* **typescript:** name a block's closing brace Rbrace ([#1061](https://github.com/UnstoppableMango/tdl/issues/1061)) ([6f5051f](https://github.com/UnstoppableMango/tdl/commit/6f5051f9292b68a679517e56bafe05f0625c2d3e))


### Documentation

* **agents:** bring AGENTS.md up to date ([#1051](https://github.com/UnstoppableMango/tdl/issues/1051)) ([d019fec](https://github.com/UnstoppableMango/tdl/commit/d019fec276aac7750ca34fbd9ff3be9b3996fdbb))
* design transform plugins ([#1055](https://github.com/UnstoppableMango/tdl/issues/1055)) ([e276c6f](https://github.com/UnstoppableMango/tdl/commit/e276c6f5856634f68b486bcaf3168a6d6707058c))


### Continuous Integration

* skip the external tool checks in the coverage run ([#1060](https://github.com/UnstoppableMango/tdl/issues/1060)) ([5b74140](https://github.com/UnstoppableMango/tdl/commit/5b74140fe4c4d2b424920efa70b3c3b79be2f2a4))

## [0.4.5](https://github.com/UnstoppableMango/tdl/compare/v0.4.4...v0.4.5) (2026-10-10)


### Features

* **go:** import Go packages and round-trip through them ([#1048](https://github.com/UnstoppableMango/tdl/issues/1048)) ([147afca](https://github.com/UnstoppableMango/tdl/commit/147afca5222d913aa82e27021f35ef03d178857f))


### Dependencies

* update golang.org/x/exp digest to ca0d7ba ([#1049](https://github.com/UnstoppableMango/tdl/issues/1049)) ([198556f](https://github.com/UnstoppableMango/tdl/commit/198556ff8590a7a2b7914bb4cfe3cce335bd6186))

## [0.4.4](https://github.com/UnstoppableMango/tdl/compare/v0.4.3...v0.4.4) (2026-10-10)


### Features

* **graphql:** import GraphQL schemas into TDL ([#1044](https://github.com/UnstoppableMango/tdl/issues/1044)) ([bcd0d76](https://github.com/UnstoppableMango/tdl/commit/bcd0d76f8b45be6e28c79b6bc2a9e3b6d610b4ab))


### Bug Fixes

* **gen:** read the watch baseline before the first run ([#1042](https://github.com/UnstoppableMango/tdl/issues/1042)) ([2b7e4cf](https://github.com/UnstoppableMango/tdl/commit/2b7e4cf19ac93ac265fa5e758ca66f5aa213d6a0))


### Documentation

* add a VHS demo GIF to the README ([#1041](https://github.com/UnstoppableMango/tdl/issues/1041)) ([969d8c8](https://github.com/UnstoppableMango/tdl/commit/969d8c88893fd02a7cb91ff42a8753db22ace495))


### Continuous Integration

* run the race detector and coverage separately ([#1045](https://github.com/UnstoppableMango/tdl/issues/1045)) ([9670dbe](https://github.com/UnstoppableMango/tdl/commit/9670dbe479372e1f79c50475d677564e5b5fa5a7))

## [0.4.3](https://github.com/UnstoppableMango/tdl/compare/v0.4.2...v0.4.3) (2026-10-10)


### Features

* **typescript:** narrow a small integer range or oneOf to a literal union ([#1040](https://github.com/UnstoppableMango/tdl/issues/1040)) ([dfabd09](https://github.com/UnstoppableMango/tdl/commit/dfabd091c315ea0fa816ac94ff73301f06f3c8e8)), closes [#1033](https://github.com/UnstoppableMango/tdl/issues/1033)


### Bug Fixes

* **sema:** check class fields on an instance for an imported type ([#1039](https://github.com/UnstoppableMango/tdl/issues/1039)) ([3afc74b](https://github.com/UnstoppableMango/tdl/commit/3afc74bedaf2235ac8889998625658dc7f0b1424))

## [0.4.2](https://github.com/UnstoppableMango/tdl/compare/v0.4.1...v0.4.2) (2026-10-09)


### Features

* **protobuf:** import protobuf schemas into TDL ([#1018](https://github.com/UnstoppableMango/tdl/issues/1018)) ([5a7b77c](https://github.com/UnstoppableMango/tdl/commit/5a7b77c1ffa1ebef5b16da06a56998307709db77))
* **thrift:** import Thrift IDL into TDL ([#1036](https://github.com/UnstoppableMango/tdl/issues/1036)) ([221d184](https://github.com/UnstoppableMango/tdl/commit/221d184ce0abfd5d96b5fa9dc17fba6a8c34ad91))


### Bug Fixes

* easy wins from the open issues ([#1020](https://github.com/UnstoppableMango/tdl/issues/1020)) ([e93f527](https://github.com/UnstoppableMango/tdl/commit/e93f52725a18d5caa52f19c6be47a25960c33629))
* **protobuf:** import an extern from its dependency's per-declaration file ([#1038](https://github.com/UnstoppableMango/tdl/issues/1038)) ([832b99a](https://github.com/UnstoppableMango/tdl/commit/832b99a3b3150ff51c78130d0d521e4f58bf2a7c)), closes [#916](https://github.com/UnstoppableMango/tdl/issues/916)


### Code Refactoring

* **backend:** share repeated helpers through emit ([#1015](https://github.com/UnstoppableMango/tdl/issues/1015)) ([bda0293](https://github.com/UnstoppableMango/tdl/commit/bda029323dce28996b824dffdd14d2b7a04465dd))


### Dependencies

* update module github.com/vektah/gqlparser/v2 to v2.5.62 ([#1035](https://github.com/UnstoppableMango/tdl/issues/1035)) ([059d1b7](https://github.com/UnstoppableMango/tdl/commit/059d1b799c4470bab68e853b8e1b92f4a8355415))

## [0.4.1](https://github.com/UnstoppableMango/tdl/compare/v0.4.0...v0.4.1) (2026-10-08)


### Bug Fixes

* **go:** a decimal warning no longer says foreign types are unbuilt ([#936](https://github.com/UnstoppableMango/tdl/issues/936)) ([1b11e6f](https://github.com/UnstoppableMango/tdl/commit/1b11e6fbc28f71cd4a8c20bffb6a6b0c61e3e420))


### Documentation

* design a C# backend ([#1003](https://github.com/UnstoppableMango/tdl/issues/1003)) ([874584e](https://github.com/UnstoppableMango/tdl/commit/874584eb4e3f5d62161e8cafbfbf668585b51653))
* design a java backend ([#1005](https://github.com/UnstoppableMango/tdl/issues/1005)) ([0a366ac](https://github.com/UnstoppableMango/tdl/commit/0a366ac8eca709dd524bceee9be6c90662c8a185))


### Code Refactoring

* **emit:** share the wire convention, length, pattern, and class plan ([#1006](https://github.com/UnstoppableMango/tdl/issues/1006)) ([cf4ef4e](https://github.com/UnstoppableMango/tdl/commit/cf4ef4e0673e7dc9c0ef336b756db4ec8bda4fdb))
* **go:** read a oneOf argument's resolved variant ([#944](https://github.com/UnstoppableMango/tdl/issues/944)) ([cda2093](https://github.com/UnstoppableMango/tdl/commit/cda20936a92501407bf030b26fa67fb417cb2fb5))


### Dependencies

* update golang.org/x/exp digest to f45ad48 ([#1014](https://github.com/UnstoppableMango/tdl/issues/1014)) ([10f5163](https://github.com/UnstoppableMango/tdl/commit/10f5163200957d236b6e2e40e8d17aa8be7d69e2))

## [0.4.0](https://github.com/UnstoppableMango/tdl/compare/v0.3.5...v0.4.0) (2026-10-07)


### ⚠ BREAKING CHANGES

* **sema:** require a class's fields on every type satisfying it ([#1000](https://github.com/UnstoppableMango/tdl/issues/1000))

### Features

* **sema:** require a class's fields on every type satisfying it ([#1000](https://github.com/UnstoppableMango/tdl/issues/1000)) ([9051053](https://github.com/UnstoppableMango/tdl/commit/90510533cf357306bbda61d2bb45752792ad2ec3))


### Bug Fixes

* **sema:** resolve a variant name through an alias or a newtype ([#943](https://github.com/UnstoppableMango/tdl/issues/943)) ([6a0aea6](https://github.com/UnstoppableMango/tdl/commit/6a0aea6ab8060b632d90e0d0a59a448d03d8d441))


### Documentation

* design the ML backends ([#1001](https://github.com/UnstoppableMango/tdl/issues/1001)) ([75f5790](https://github.com/UnstoppableMango/tdl/commit/75f579061629f62678fe77f3aa7c97fcda980991))


### Code Refactoring

* **emit:** read a variant's name directive through VariantName ([#939](https://github.com/UnstoppableMango/tdl/issues/939)) ([45c6b4d](https://github.com/UnstoppableMango/tdl/commit/45c6b4d370dd8e705f00378f60599a9daef2bc3a))


### Tests

* **roundtrip:** a harness running every backend forward and back ([#1002](https://github.com/UnstoppableMango/tdl/issues/1002)) ([4f57f90](https://github.com/UnstoppableMango/tdl/commit/4f57f900f4da4c14efe60b3f2d5a0a75ec49b788))


### Dependencies

* update golang.org/x/exp digest to 7677206 ([#1007](https://github.com/UnstoppableMango/tdl/issues/1007)) ([dd86159](https://github.com/UnstoppableMango/tdl/commit/dd86159940f2fd1b20a355c592cf92a6f87e27ca))
* update module github.com/santhosh-tekuri/jsonschema/v6 to v6.0.3 ([#1008](https://github.com/UnstoppableMango/tdl/issues/1008)) ([024b56e](https://github.com/UnstoppableMango/tdl/commit/024b56e401a025163e4ea0e7525e54aadbcd9dda))
* update module github.com/vektah/gqlparser/v2 to v2.5.61 ([#1009](https://github.com/UnstoppableMango/tdl/issues/1009)) ([9f0f3e6](https://github.com/UnstoppableMango/tdl/commit/9f0f3e62d315f4db5b53cd2d8cbd0f9de35e6adb))

## [0.3.5](https://github.com/UnstoppableMango/tdl/compare/v0.3.4...v0.3.5) (2026-10-07)


### Features

* **import:** import mode in the plugin protocol, loss codes, and tdl import ([#998](https://github.com/UnstoppableMango/tdl/issues/998)) ([f61890d](https://github.com/UnstoppableMango/tdl/commit/f61890db5ccabe889e1b274eab4d535f004926f0))


### Bug Fixes

* **go:** honour a name directive on an enum variant ([#938](https://github.com/UnstoppableMango/tdl/issues/938)) ([d54984c](https://github.com/UnstoppableMango/tdl/commit/d54984ceca003a4231afdea302501381507bd187))


### Documentation

* design a haskell backend ([#997](https://github.com/UnstoppableMango/tdl/issues/997)) ([d91b3b0](https://github.com/UnstoppableMango/tdl/commit/d91b3b02684ad0371220faa76758b15a22324e08))


### Continuous Integration

* run build and test as one job without setup-nix ([#989](https://github.com/UnstoppableMango/tdl/issues/989)) ([44f6115](https://github.com/UnstoppableMango/tdl/commit/44f611551cafc8df4c7ef0dfa826c60137f83304))

## [0.3.4](https://github.com/UnstoppableMango/tdl/compare/v0.3.3...v0.3.4) (2026-10-06)


### Features

* **likec4:** a likec4 backend writing one element per declaration ([#984](https://github.com/UnstoppableMango/tdl/issues/984)) ([1594d9b](https://github.com/UnstoppableMango/tdl/commit/1594d9b077ebdf285be0526d01fa9e013587f656))
* **unlower:** turn a model back into TDL source ([#983](https://github.com/UnstoppableMango/tdl/issues/983)) ([3281643](https://github.com/UnstoppableMango/tdl/commit/32816436e5e4b3ffb20b4e3045b6fa18d7cbe0f1)), closes [#960](https://github.com/UnstoppableMango/tdl/issues/960)


### Continuous Integration

* raise the Codecov project target to 85% ([#987](https://github.com/UnstoppableMango/tdl/issues/987)) ([989cd5d](https://github.com/UnstoppableMango/tdl/commit/989cd5dd086136dfaec92786b31cf714424901e7))

## [0.3.3](https://github.com/UnstoppableMango/tdl/compare/v0.3.2...v0.3.3) (2026-10-06)


### Features

* **openapi:** add an OpenAPI backend for 3.1, 3.0, and 2.0 ([#973](https://github.com/UnstoppableMango/tdl/issues/973)) ([f657c99](https://github.com/UnstoppableMango/tdl/commit/f657c99e5c3feb4c127cab6b8fd28d41b884eb68))

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
