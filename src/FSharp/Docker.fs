module UnMango.Tdl.Docker

open System
open System.Threading
open Docker.DotNet
open Docker.DotNet.Models

type Progress(r) =
  interface IProgress<string> with
    member _.Report(value) = r value

  interface IProgress<JSONMessage> with
    member _.Report(value) = r value.Status

type Workload =
  { Cmd: string list
    Entrypoint: string list
    Image: string
    Name: string
    Progress: Progress
    Tag: string }

module Workload =
  let create n i t =
    { Cmd = []
      Entrypoint = []
      Image = i
      Name = n
      Progress = Progress(ignore)
      Tag = t }

  let progress r w = { w with Progress = Progress(r) }

type private Builder =
  { CancellationToken: CancellationToken
    Client: IDockerClient }

module private Builder =
  module Image =
    let create w docker =
      docker.Client.Images.CreateImageAsync(
        ImagesCreateParameters(FromImage = w.Image, Tag = w.Tag),
        null,
        w.Progress :> IProgress<JSONMessage>,
        docker.CancellationToken
      )
      |> Async.AwaitTask

  let attach id docker =
    docker.Client.Containers.AttachContainerAsync(
      id,
      false,
      ContainerAttachParameters(Stdin = true, Stdout = true, Stderr = true),
      docker.CancellationToken
    )
    |> Async.AwaitTask

  let create w docker =
    docker.Client.Containers.CreateContainerAsync(
      CreateContainerParameters(
        Image = $"{w.Image}:{w.Tag}",
        Name = w.Name,
        StdinOnce = true,
        OpenStdin = true,
        AttachStdin = true,
        AttachStdout = true,
        Cmd = ResizeArray(w.Cmd),
        Entrypoint = ResizeArray(w.Entrypoint)
      ),
      docker.CancellationToken
    )
    |> Async.AwaitTask

  let from client = async {
    let! cancellationToken = Async.CancellationToken

    return
      { CancellationToken = cancellationToken
        Client = client }
  }

  let remove id docker =
    docker.Client.Containers.RemoveContainerAsync(
      id,
      ContainerRemoveParameters(),
      docker.CancellationToken
    )
    |> Async.AwaitTask

  let start id docker =
    docker.Client.Containers.StartContainerAsync(
      id,
      ContainerStartParameters(),
      docker.CancellationToken
    )
    |> Async.AwaitTask

  let stop id docker =
    docker.Client.Containers.StopContainerAsync(
      id,
      ContainerStopParameters(WaitBeforeKillSeconds = 15u),
      docker.CancellationToken
    )
    |> Async.AwaitTask

  let cleanup id docker = async {
    do! stop id docker |> Async.Ignore
    do! remove id docker
  }

  let wait id docker =
    docker.Client.Containers.WaitContainerAsync(id, docker.CancellationToken)
    |> Async.AwaitTask

let exec w i o (docker: IDockerClient) =
  task {
    let! docker = Builder.from docker
    do! docker |> Builder.Image.create w
    let! container = docker |> Builder.create w

    use! stream = docker |> Builder.attach container.ID
    let! cancellationToken = Async.CancellationToken
    do! stream.CopyFromAsync(i, cancellationToken)
    do! stream.CopyOutputToAsync(null, o, null, cancellationToken)

    let! started = docker |> Builder.start container.ID

    if not started then
      do! docker |> Builder.remove container.ID
      failwith "failed to start container"

    do! docker |> Builder.wait container.ID |> Async.Ignore
    do! docker |> Builder.cleanup container.ID
  }
  |> Async.AwaitTask
