# The zod package the zod backend's output imports, for checks.gen-zod and
# the devShell's NODE_PATH, where the backend's tests find it. It has no
# dependencies, so the registry's tarball is the whole package. 4.3 is the
# oldest release the output supports, the one adding exactOptional.
{ fetchurl, runCommand }:
let
  version = "4.3.6";
  src = fetchurl {
    url = "https://registry.npmjs.org/zod/-/zod-${version}.tgz";
    hash = "sha512-rftlrkhHZOcjDwkGlnUtZZkvaPHCsDATp4pGpuOOMDaTdDDXF91wuVDJoWoPsKX/3YPQ5fHuF3STjcYyKr+Qhg==";
  };
in
runCommand "zod-${version}" { } ''
  mkdir -p $out/lib/node_modules/zod
  tar -xzf ${src} -C $out/lib/node_modules/zod --strip-components=1
''
