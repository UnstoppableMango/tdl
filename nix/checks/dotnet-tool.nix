{
  runCommand,
  dotnet-sdk,
  tdl,
  tdl-gen-csharp-tool,
}:
# Installs Tdl.Gen.CSharp from the package tdl-gen-csharp-tool packed, with no
# feed but that folder, as `dotnet tool install` would from nuget.org, and
# runs tdl gen through the tool's shim on PATH. Holds what it writes, and
# every diagnostic, to what the AOT plugin tdl ships writes for the smoke
# model.
runCommand "tdl-dotnet-tool-check"
  {
    nativeBuildInputs = [
      dotnet-sdk
      tdl
    ];
  }
  ''
    export HOME=$TMPDIR NUGET_PACKAGES=$TMPDIR/nuget
    # The shim is an apphost, which finds the runtime through DOTNET_ROOT,
    # as it would beside a dotnet installed anywhere but /usr/share/dotnet.
    export DOTNET_ROOT=${dotnet-sdk}/share/dotnet

    cat > nuget.config <<EOF
    <?xml version="1.0" encoding="utf-8"?>
    <configuration>
      <packageSources>
        <clear />
        <add key="local" value="${tdl-gen-csharp-tool}" />
      </packageSources>
    </configuration>
    EOF
    dotnet tool install Tdl.Gen.CSharp --tool-path tools \
      --configfile nuget.config --version ${tdl-gen-csharp-tool.version}

    cp ${../../testdata/gen/smoke/source.tdl} source.tdl
    tdl gen --target csharp -o aot source.tdl 2> aot.log || { cat aot.log >&2; exit 1; }

    export PATH=$PWD/tools:$PATH
    [ "$(command -v tdl-gen-csharp)" = "$PWD/tools/tdl-gen-csharp" ] || {
      echo "tdl-gen-csharp resolves to $(command -v tdl-gen-csharp), not the tool" >&2
      exit 1
    }
    tdl gen --target csharp -o tool source.tdl 2> tool.log || { cat tool.log >&2; exit 1; }

    find tool -name '*.cs' | grep -q . || { echo "the tool wrote no C#" >&2; exit 1; }
    diff -r aot tool
    diff aot.log tool.log
    touch $out
  ''
