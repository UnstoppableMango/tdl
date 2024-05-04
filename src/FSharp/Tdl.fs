module UnMango.Tdl.Tdl

open System.IO
open UnMango.Tdl.Abstractions

type From = Stream -> Stream -> Async<unit>
type To = Stream -> Stream -> Async<unit>
type Gen = Stream -> Stream -> Async<unit>

module From =
  let convert i o = async { failwith "TODO" }

  let wrap (c: IConverterFrom) : From =
    fun i o ->
      async {
        let! ct = Async.CancellationToken
        return! c.FromAsync(i, o, ct) |> Async.AwaitTask
      }

module Gen =
  let generate i o = async { failwith "TODO" }

  let wrap (g: IGenerator) : Gen =
    fun i o ->
      async {
        let! ct = Async.CancellationToken
        return! g.GenerateAsync(i, o, ct) |> Async.AwaitTask
      }

module To =
  let convert i o = async { failwith "TODO" }

  let wrap (c: IConverterTo) : To =
    fun i o ->
      async {
        let! ct = Async.CancellationToken
        return! c.ToAsync(i, o, ct) |> Async.AwaitTask
      }
