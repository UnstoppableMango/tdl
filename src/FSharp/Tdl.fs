module UnMango.Tdl.Tdl

open System.IO
open UnMango.Tdl.Abstractions

type TdlError = Message of string

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

      try
        let! res = c.FromAsync(o, ct) |> Async.AwaitTask
        return Ok res
      with ex ->
        return (Message ex.Message |> Error)
    }

module Gen =
  open System

  let wrap (g: IGenerator) : Gen =
    fun i o -> async {
      let! ct = Async.CancellationToken

      try
        do! g.GenerateAsync(i, o, ct) |> Async.AwaitTask
        return None
      with ex ->
        return (Message ex.Message |> Some)
    }

module Runner =
  let wrap (r: IRunner) =
    { From = r :> IConverter |> From.wrap
      Gen = r :> IGenerator |> Gen.wrap }
