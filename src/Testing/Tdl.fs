module UnMango.Tdl.Testing.Tdl

open FsCheck

let merge: IArbMap -> IArbMap =
  Any.merge
  >> Constructor.merge
  >> Field.merge
  >> Function.merge
  >> GenericParameter.merge
  >> Modifier.merge
  >> Spec.merge
  >> Type.merge
