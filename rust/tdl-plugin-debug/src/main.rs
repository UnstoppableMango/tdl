use std::process::ExitCode;

fn main() -> ExitCode {
    tdl_plugin::run(&tdl_plugin_debug::Debug)
}
