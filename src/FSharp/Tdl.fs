module UnMango.Tdl.Tdl

open System.IO
open UnMango.Tdl.Abstractions

type TdlError =
  | Message of string

type FromResult = Result<Spec, TdlError>
type GenResult = TdlError option

type From = Stream -> Async<FromResult>
type Gen = Spec -> Stream -> Async<GenResult>

module TdlError =
  let message =
    function
    | Message m -> m

type Runner = { From: From; Gen: Gen }

module From =
  let wrap (c: IConverter) : From =
    fun o -> async {
      let! ct = Async.CancellationToken
      return! c.FromAsync(o, ct) |> Async.AwaitTask
    }

module Gen =
  let wrap (g: IGenerator) : Gen =
    fun i o -> async {
      let! ct = Async.CancellationToken
      return! g.GenerateAsync(i, o, ct) |> Async.AwaitTask
    }

module Runner =
  let wrap (r: IRunner) =
    { From = r :> IConverter |> From.wrap
      Gen = r :> IGenerator |> Gen.wrap }
