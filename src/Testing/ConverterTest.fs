namespace UnMango.Tdl.Testing

open System.IO
open UnMango.Tdl.Abstractions
open UnMango.Tdl

type ConverterTest =
  static member Converts(from: Tdl.From) = async {
    use stream = new MemoryStream()

    let source =
      { new ISource with
          override this.Plugin = "test2test"
          override this.Input = stream }

    match! from source with
    | Ok _ -> return true
    | Error _ -> return false
  }
