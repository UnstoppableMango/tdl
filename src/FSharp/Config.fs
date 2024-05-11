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

  let debugEnabled = envSet "TDL_DEBUG"
  let verboseEnabled = envSet "TDL_VERBOSE"

  let containerTag =
    function
    | Dev -> "local"
    | Release -> "main"

  let socketDir: Env -> string =
    function
    | _ -> Path.Combine(appdata, "tdl")

  let socket e =
    Path.Combine(socketDir e, "broker.sock")

type Config(env) =
  member _.ContainerRepo = Config.containerRepo env
  member _.ContainerTag = Config.containerTag env
  member _.SocketDir = Config.socketDir env
  member _.Socket = Config.socket env

  static member DebugEnabled() = Config.debugEnabled
  static member VerboseEnabled() = Config.verboseEnabled

  static member Uid() = Tools.uid |> Async.StartAsTask
  static member Gid() = Tools.gid |> Async.StartAsTask
