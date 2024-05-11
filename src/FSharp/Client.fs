module UnMango.Tdl.Client

open System
open System.IO
open System.Net
open System.Net.Http
open System.Net.Sockets
open System.Runtime.ExceptionServices
open System.Threading
open System.Threading.Tasks
open Grpc.Net.Client

// https://github.com/fsharp/fslang-suggestions/issues/660#issuecomment-382070639
type Exception with
  member this.ReRaise() =
    (ExceptionDispatchInfo.Capture this).Throw()
    Unchecked.defaultof<_>

// https://learn.microsoft.com/en-us/aspnet/core/grpc/interprocess-uds?view=aspnetcore-8.0
type UnixDomainSocketsConnectionFactory(endpoint: EndPoint) =
  member _.ConnectAsync(_: SocketsHttpConnectionContext, cancellationToken: CancellationToken) =
    task {
      let socket =
        new Socket(AddressFamily.Unix, SocketType.Stream, ProtocolType.Unspecified)

      try
        do! socket.ConnectAsync(endpoint, cancellationToken)
        return new NetworkStream(socket, true) :> Stream
      with ex ->
        do socket.Dispose()
        return ex.ReRaise()
    }
    |> ValueTask<Stream>

let createChannel socketPath =
  let endpoint = UnixDomainSocketEndPoint(socketPath)
  let connectionFactory = UnixDomainSocketsConnectionFactory(endpoint)

  let callback context ct =
    connectionFactory.ConnectAsync(context, ct)

  let httpHandler = new SocketsHttpHandler(ConnectCallback = callback)
  let options = GrpcChannelOptions(HttpHandler = httpHandler)
  GrpcChannel.ForAddress("http://0.0.0.0", options)
