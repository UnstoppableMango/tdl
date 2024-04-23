namespace UnMango.Tdl

open System
open System.Runtime.CompilerServices
open System.Runtime.InteropServices
open System.Text
open System.Threading
open System.Threading.Tasks
open CliWrap
open CliWrap.EventStream

module Observable =
  let first f o =
    async {
      let tcs = TaskCompletionSource()
      let! ct = Async.CancellationToken
      use _ = ct.Register(fun () -> tcs.SetCanceled())

      let fx x =
        if f x then
          do tcs.SetResult()

      use s = Observable.subscribe fx o
      use _ = ct.Register(fun () -> s.Dispose())

      return! tcs.Task |> Async.AwaitTask
    }

type Observable =
  [<Extension>]
  static member inline First
    (observable, predicate: Predicate<'a>, [<Optional; DefaultParameterValue(CancellationToken())>] cancellationToken) =
    observable
    |> Observable.first predicate.Invoke
    |> fun t -> Async.StartAsTask(t, cancellationToken = cancellationToken)

type CommandExtensions =
  [<Extension>]
  static member inline Observe(command: Command, forceful: CancellationToken, graceful: CancellationToken) =
    command.Observe(Encoding.Default, Encoding.Default, forceful, graceful)
