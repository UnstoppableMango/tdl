module DockerTests

open System
open System.IO
open System.Text
open Docker.DotNet
open FsCheck.FSharp
open FsCheck.Xunit
open UnMango.Tdl
open Xunit

type Message =
  static member Double() =
    ArbMap.defaults
    |> ArbMap.arbitrary<string>
    |> Arb.filter (fun s -> s.Length > 0 && not <| s.Contains('\\'))

let rand = Random()

let testWorkload = Docker.Workload.create $"tdl-test-{rand.Next()}"

[<Property(Arbitrary = [| typeof<Message> |], Skip = "This is all sorts of broken")>]
let ``Can perform IO with a container`` (message: string) = async {
  use input = new MemoryStream(Encoding.UTF8.GetBytes(message), false)
  use output = new MemoryStream()
  use config = new DockerClientConfiguration()
  use client = config.CreateClient()

  let workload =
    { testWorkload "ubuntu" "latest" with
        Entrypoint = [ "cat" ] }

  do! client |> Docker.exec workload input output
  let actual = Encoding.UTF8.GetString(output.ToArray())

  Assert.True(output.Length > 0)
  Assert.Equal(message, actual)
}
