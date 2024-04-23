namespace UnMango.Tdl

open System
open System.Runtime.CompilerServices
open System.Runtime.InteropServices
open System.Threading
open System.Threading.Tasks

module Observable =
  let first f (ct: CancellationToken) o =
    task {
      let tcs = TaskCompletionSource()
      use _ = ct.Register(fun () -> tcs.SetCanceled())

      let fx x =
        if f x then
          do tcs.SetResult()

      use s = Observable.subscribe fx o
      use _ = ct.Register(fun () -> s.Dispose())

      return! tcs.Task
    }

type Observable =
  [<Extension>]
  static member inline First
    (observable, predicate: Predicate<'a>, [<Optional; DefaultParameterValue(CancellationToken())>] cancellationToken) =
    observable |> Observable.first predicate.Invoke cancellationToken
