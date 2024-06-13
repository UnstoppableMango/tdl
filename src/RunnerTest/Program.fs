open System
open System.CommandLine
open System.CommandLine.Builder
open System.CommandLine.Parsing
open System.IO
open FsCheck
open FsCheck.FSharp
open UnMango.Tdl
open UnMango.Tdl.Testing

let maxTest =
  match Environment.GetEnvironmentVariable("CI") with
  | ci when String.IsNullOrWhiteSpace(ci) -> 100
  | _ -> 500

let validatePath (arg: Argument<FileInfo>) (result: SymbolResult) : unit =
  match result.GetValueForArgument(arg) with
  | null -> result.ErrorMessage <- "<path> is required"
  | bin when not bin.Exists -> result.ErrorMessage <- $"File does not exist: {bin.FullName}"
  | _ -> ()

let run (bin: FileInfo) =
  let from = CliRunner.from bin.FullName
  let gen = CliRunner.gen bin.FullName
  let spec = ArbMap.defaults |> TdlArbs.merge |> ArbMap.arbitrary<Spec>
  let config = Config.Quick.WithMaxTest(maxTest)

  Console.WriteLine("Executing runner test suite")

  [ "Should round-trip", RunnerTest.roundTrip
    "Should generate data", RunnerTest.generateData ]
  |> List.map (fun (n, t) -> n, t (gen, from))
  |> List.map (fun (n, t) -> n, Prop.forAll spec t)
  |> List.map (fun (n, t) -> Check.One(n, config, t))
  |> ignore

[<EntryPoint>]
let main args =
  let pathArg = Argument<FileInfo>("path", "Path to the binary to test")
  pathArg.AddValidator(validatePath pathArg)

  let root = RootCommand("Tests runner binaries")
  root.AddArgument(pathArg)
  root.SetHandler(run, pathArg)

  CommandLineBuilder(root).UseDefaults().Build().Parse(args).Invoke()
