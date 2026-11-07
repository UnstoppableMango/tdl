mod common;

use tdl_plugin::loss;

// The loss codes are the ones docs/architecture/reverse.md tabulates, in its
// order, as emit.LossCodes is in Go.
#[test]
fn match_the_design() {
    let doc = std::fs::read_to_string(common::root().join("docs/architecture/reverse.md")).unwrap();

    let documented: Vec<&str> = doc
        .lines()
        .filter_map(|l| l.strip_prefix("| `"))
        .filter_map(|l| l.split_once("` |"))
        .map(|(code, _)| code)
        .filter(|code| code.starts_with("lossy."))
        .collect();

    assert_eq!(documented, loss::ALL);
}
