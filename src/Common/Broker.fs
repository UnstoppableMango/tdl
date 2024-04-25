namespace UnMango.Tdl

open System
open System.IO
open System.Runtime.InteropServices
open System.Text.RegularExpressions
open System.Threading
open System.Threading.Tasks
open CliWrap.EventStream
open FSharp.Control.Reactive
open UnMango.CliWrap.FSharp

type DevelopmentMeta = { Configuration: string; Tfm: string }
type BrokerLocation = Development of DevelopmentMeta

type Broker =
  { Endpoint: string
    Location: BrokerLocation
    Bin: string }

module Broker =
  let private dir = Path.Join("..", "Broker")
  let private startedPattern = Regex("Content root path: .*")
  let private stoppedPattern = Regex("Application is shutting down")

  let private matchPattern (pattern: Regex) (e: CommandEvent) =
    match e with
    | :? StandardOutputCommandEvent as o -> pattern.IsMatch(o.Text)
    | _ -> false

  let private started e = matchPattern startedPattern e
  let private stopped e = matchPattern stoppedPattern e

  type Stop(obs, cts: CancellationTokenSource) =
    interface IDisposable with
      member this.Dispose() =
        do cts.Cancel()
        do obs |> Observable.firstIf stopped |> Observable.wait |> ignore
        do cts.Dispose()

    interface IAsyncDisposable with
      member this.DisposeAsync() =
        task {
          do! cts.CancelAsync()
          // TODO: Is there any async version like Rx.NET?
          do obs |> Observable.firstIf stopped |> Observable.wait |> ignore
          do cts.Dispose()
        }
        |> ValueTask

  let debugMeta =
    { Configuration = "Debug"
      Tfm = "net9.0" }

  let dev =
    { Endpoint = "http://127.0.0.1:6969"
      Location = Development debugMeta
      Bin = "UnMango.Tdl.Broker.dll" }

  let binDir broker =
    match broker.Location with
    | Development meta -> Path.Join(dir, "bin", meta.Configuration, meta.Tfm)

  let buildCmd =
    command "dotnet" {
      workingDirectory dir
      args [ "build" ]
      stdout (PipeTo.f Console.WriteLine)
    }

  let startCmd broker =
    match broker.Location with
    | Development _ ->
      command "dotnet" {
        workingDirectory (binDir broker)
        env [ "ASPNETCORE_URLS", broker.Endpoint ]
        args [ broker.Bin ]
        stdout (PipeTo.f Console.WriteLine)
      }

  let start broker : Async<IDisposable> =
    async {
      match broker.Location with
      | Development _ ->
        do! buildCmd |> Cli.exec |> Async.Ignore
        let! ct = Async.CancellationToken
        use forceful = CancellationTokenSource.CreateLinkedTokenSource(ct)
        let graceful = new CancellationTokenSource()

        forceful.CancelAfter(TimeSpan.FromSeconds(30L))
        let obs = startCmd broker |> _.Observe(forceful.Token, graceful.Token)
        do obs |> Observable.firstIf started |> Observable.wait |> ignore

        return new Stop(obs, graceful)
    }

type Broker with
  static member Dev = Broker.dev

  member this.Start(endpoint, [<Optional; DefaultParameterValue(CancellationToken())>] cancellationToken) =
    { this with Endpoint = endpoint }
    |> Broker.start
    |> fun t -> Async.StartAsTask(t, cancellationToken = cancellationToken)
