namespace UnMango.Tdl

open System.Runtime.CompilerServices
open System.Text
open System.Threading
open CliWrap
open CliWrap.EventStream

type CommandExtensions =
  [<Extension>]
  static member inline Observe
    (command: Command, forceful: CancellationToken, graceful: CancellationToken)
    =
    command.Observe(Encoding.Default, Encoding.Default, forceful, graceful)
