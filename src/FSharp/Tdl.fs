module UnMango.Tdl.Tdl

open System.IO
open UnMango.Tdl.Abstractions

type From = Stream -> Async<Spec>
type Gen = Spec -> Stream -> Async<unit>

module From =
  let convert i o = async { failwith "TODO" }

  let wrap (c: IConverter) : From =
    fun o ->
      async {
        let! ct = Async.CancellationToken
        return! c.FromAsync(o, ct) |> Async.AwaitTask
      }

module Gen =
  let generate i o = async { failwith "TODO" }

  let wrap (g: IGenerator) : Gen =
    fun i o ->
      async {
        let! ct = Async.CancellationToken
        return! g.GenerateAsync(i, o, ct) |> Async.AwaitTask
      }