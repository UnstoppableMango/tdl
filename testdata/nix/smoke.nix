# What checks.gen-nix evaluates: the smoke model's types against values
# lib.evalModules must accept and refuse, and what toJSON writes.
{ lib, file }:
let
  smoke = import file { inherit lib; };
  eval =
    type: value:
    builtins.toJSON
      (lib.evalModules {
        modules = [
          {
            options.x = lib.mkOption { inherit type; };
            config.x = value;
          }
        ];
      }).config.x;
  refused = type: value: !(builtins.tryEval (eval type value)).success;
in
assert eval smoke.Counts { byStatus.Draft = 1; } == ''{"byStatus":{"Draft":1}}'';
assert
  eval smoke.Sparse {
    values = [
      1
      null
    ];
  } == ''{"values":[1,null]}'';
assert refused smoke.Counts { byStatus.Nope = 1; };
assert refused smoke.Sparse { values = [ "one" ]; };
assert refused smoke.LegacyOrder { };
assert refused smoke.LegacyOrder {
  id = "a";
  extra = 1;
};
true
