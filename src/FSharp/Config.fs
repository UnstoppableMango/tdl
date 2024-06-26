namespace UnMango.Tdl

open System
open System.IO

type Env =
  | Dev
  | Release

module Config =
  let appdata = Environment.SpecialFolder.ApplicationData |> Environment.GetFolderPath

  let containerRepo =
    function
    | Dev -> "unstoppablemango"
    | Release -> "ghcr.io/unstoppablemango"

  let envSet e =
    match Environment.GetEnvironmentVariable(e) with
    | "false"
    | "0"
    | ""
    | null -> false
    | _ -> true

  let env =
#if DEBUG
    if envSet "TDL_DEV" then Dev else Release
#else
    Release
#endif

  let debugEnabled = envSet "TDL_DEBUG"
  let verboseEnabled = envSet "TDL_VERBOSE"

  let ownerLabel = "tdl.owner"

  let runningInContainer = envSet "DOTNET_RUNNING_IN_CONTAINER"

  let toolDir (env: Env) : string =
    match env, runningInContainer with
    | _, true -> "/app/plugins" // TODO: Rename
    | _ -> Directory.CreateTempSubdirectory("tdl").FullName

  let socketDir: Env -> string =
    function
    | _ -> Path.Combine(appdata, "tdl")

  let socket e =
    Path.Combine(socketDir e, "broker.sock")

type Config =
  static member Env = Config.env

  static member DebugEnabled = Config.debugEnabled
  static member VerboseEnabled = Config.verboseEnabled

  static member ContainerRepo = Config.containerRepo Config.env
  static member SocketDir = Config.socketDir Config.env
  static member Socket = Config.socket Config.env

  static member OwnerLabel = Config.ownerLabel

  static member Uid() = Tools.uid |> Async.StartAsTask
  static member Gid() = Tools.gid |> Async.StartAsTask
