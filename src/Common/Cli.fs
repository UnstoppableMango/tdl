module UnMango.Tdl.Cli

open CliWrap
open CliWrap.EventStream

let wait waiter (command: Command) =
  command.Observe()
