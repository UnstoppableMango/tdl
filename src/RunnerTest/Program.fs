open System.CommandLine
open System.CommandLine.Builder
open System.CommandLine.Parsing
open System.IO
open FsCheck
open FsCheck.FSharp
open UnMango.Tdl
open UnMango.Tdl.Testing

let validatePath (arg: Argument<FileInfo>) (result: SymbolResult) : unit =
  match result.GetValueForArgument(arg) with
  | null -> result.ErrorMessage <- "Path was not provided"
  | bin when not bin.Exists -> result.ErrorMessage <- $"File does not exist: {bin.FullName}"
  | _ -> ()

let run (bin: FileInfo) =
  let from = CliRunner.from bin.FullName
  let gen = CliRunner.gen bin.FullName
  let spec = ArbMap.defaults |> Tdl.merge |> ArbMap.arbitrary<Spec>
  (gen, from) |> RunnerTest.roundTrip |> Prop.forAll spec |> Check.Quick

[<EntryPoint>]
let main args =
  let pathArg = Argument<FileInfo>("path", "Path to the binary to test")
  pathArg.AddValidator(validatePath pathArg)

  let root = RootCommand("Tests runner binaries")
  root.AddArgument(pathArg)
  root.SetHandler(run, pathArg)

  CommandLineBuilder(root).UseDefaults().Build().Parse(args).Invoke()
