mod common;

use tdl_plugin::buffa::MessageField;
use tdl_plugin::ir::{
    Decl, Directive, ID, Literal, Meta, Model, Position, Struct, Type, decl::Node,
};
use tdl_plugin::protocol::{Analysis, DeclView, Request, Severity, TypeView};
use tdl_plugin::{
    Error, Problem, Session, Skipped, camel, declares, fields, loss, screaming_snake, snake,
    variants,
};

fn named<'a>(s: &Session<'a>, name: &str) -> &'a Decl {
    s.model()
        .decls
        .iter()
        .find(|d| d.meta.name == name)
        .unwrap()
}

#[test]
fn refuses_a_request_without_an_analysis() {
    let mut req = common::request("shop");
    req.analysis = MessageField::default();

    let err = Session::new(&req, "Debug").err().unwrap();

    assert!(
        matches!(&err, Error::Protocol(m) if m.contains("carries no analysis")),
        "{err}"
    );
}

#[test]
fn refuses_an_analysis_of_another_model() {
    let mut req = common::request("shop");
    req.analysis.get_or_insert_default().decls.remove(0);

    assert!(Session::new(&req, "Debug").is_err());
}

#[test]
fn own_leaves_out_the_prelude() {
    let req = common::request("catalog");
    let s = Session::new(&req, "Debug").unwrap();

    let own: Vec<_> = s.own().iter().map(|d| d.meta.name.as_str()).collect();

    assert_eq!(own, ["Product", "Marker", "Availability"]);
}

#[test]
fn names_from_the_shipped_words() {
    let req = common::request("catalog");
    let s = Session::new(&req, "Debug").unwrap();
    let product = named(&s, "Product");
    let availability = named(&s, "Availability");

    assert_eq!(s.decl_name(product, snake), "product");
    assert_eq!(s.field_name(&fields(product)[0], screaming_snake), "ID");
    assert_eq!(
        s.variant_name(&variants(availability)[0], snake),
        "in_stock"
    );
    assert_eq!(s.variant_name(&variants(availability)[0], camel), "inStock");
}

#[test]
fn a_view_is_found_by_identity() {
    let req = common::request("catalog");
    let s = Session::new(&req, "Debug").unwrap();
    let product = named(&s, "Product");
    let copy = product.clone();

    assert!(s.decl_view(product).unwrap().own);
    assert!(s.decl_view(&copy).is_none());
    // The name falls back to splitting the declaration's own name.
    assert_eq!(s.decl_name(&copy, snake), "product");
    let name = &fields(product)[1];
    assert_eq!(s.source(&name.constraints[0]), "length(1..64)");
}

#[test]
fn a_name_directive_wins() {
    let mut req = common::request("catalog");
    let model = req.model.get_or_insert_default();
    let product = model
        .decls
        .iter_mut()
        .find(|d| d.meta.name == "Product")
        .unwrap();
    for (target, text) in [("debug", "Item"), ("other", "Other")] {
        product.directives.push(Directive {
            name: "name".into(),
            target: target.into(),
            args: vec![Literal {
                text: text.into(),
                ..Default::default()
            }],
            ..Default::default()
        });
    }
    let s = Session::new(&req, "Debug").unwrap();

    assert_eq!(s.decl_name(named(&s, "Product"), snake), "Item");
}

#[test]
fn reads_the_target_block() {
    let req = common::request("catalog");
    let s = Session::new(&req, "Debug").unwrap();

    assert_eq!(
        s.block("out").map(|d| d.args[0].text.as_str()),
        Some("./gen")
    );
    assert!(s.block("note").is_none());
    assert!(s.bare("roundtrip"));
    assert!(!s.bare("closed"));
}

#[test]
fn warns_with_a_position_and_a_code() {
    let req = common::request("catalog");
    let mut s = Session::new(&req, "Debug").unwrap();
    let pos = Position {
        filename: "a.tdl".into(),
        line: 1,
        ..Default::default()
    };

    s.warn(Problem::lossy(loss::COLLECTION, "lost", Some(&pos)));
    s.warn(Problem::invalid("wrong", Some(&pos)));
    s.error(None, "broken");
    s.lossy(loss::DOC, Some(&pos), "undocumented");

    let got: Vec<_> = s
        .diagnostics
        .iter()
        .map(|d| {
            (
                d.severity.as_known().unwrap(),
                d.message.as_str(),
                d.code.as_str(),
            )
        })
        .collect();
    assert_eq!(
        got,
        [
            (Severity::Warning, "lost", loss::COLLECTION),
            (Severity::Error, "wrong", ""),
            (Severity::Error, "broken", ""),
            (Severity::Warning, "undocumented", loss::DOC),
        ]
    );
    assert_eq!(s.diagnostics[0].position.as_option(), Some(&pos));
    assert!(s.diagnostics[2].position.is_unset());
    let resp = s.response(vec![]);
    assert_eq!(
        resp.diagnostics
            .iter()
            .filter(|d| d.severity == Severity::Error)
            .count(),
        2
    );
}

#[test]
fn output_loss_always_warns_and_read_back_loss_only_under_readback() {
    let req = common::request("catalog");
    let cases: [(bool, bool, &[&str]); 4] = [
        (false, false, &["output", "skipped"]),
        (true, false, &["output", "skipped", "read back"]),
        (false, true, &["output", "skipped"]),
        (true, true, &["output", "skipped"]),
    ];
    for (readback, roundtrip, want) in cases {
        let mut s = Session::new(&req, "Debug").unwrap();
        s.readback = readback;
        s.roundtrip = roundtrip;

        s.lossy(loss::CONSTRAINT, None, "output");
        s.skip(Problem::unsupported("skipped", None));
        s.lose(loss::DOC, None, "read back");

        let got: Vec<_> = s.diagnostics.iter().map(|d| d.message.as_str()).collect();
        assert_eq!(got, want, "readback {readback}, roundtrip {roundtrip}");
    }
}

#[test]
fn warns_about_unenforced_constraints() {
    let req = common::request("catalog");
    let mut s = Session::new(&req, "Debug").unwrap();

    s.warn_constraints(named(&s, "Product"), |_| false);
    s.warn_constraints(named(&s, "Marker"), |_| false);
    s.warn_constraints(named(&s, "Product"), |c| c.name == "length");

    assert_eq!(s.diagnostics.len(), 1);
    let d = &s.diagnostics[0];
    assert_eq!(
        d.message,
        "Product.name carries 1 constraint(s), and validation is not generated yet"
    );
    assert_eq!(d.code, loss::CONSTRAINT);
}

#[test]
fn says_what_a_declaration_declares() {
    let req = common::request("catalog");
    let s = Session::new(&req, "Debug").unwrap();

    assert_eq!(declares(named(&s, "Product")), Ok(true));
    assert_eq!(declares(named(&s, "string")), Ok(false));
    assert_eq!(
        declares(named(&s, "Entity")),
        Err(Problem::unsupported(
            "Entity is a class, and classes are not generated yet",
            named(&s, "Entity").meta.position.as_option()
        ))
    );
    assert!(matches!(
        declares(named(&s, "Option")),
        Err(Problem::Unsupported { .. })
    ));
}

// A request where A names B names C, and D names nothing, with the analysis
// tdl would send.
fn cascading() -> Request {
    let decl = |name: &str| Decl {
        meta: Meta {
            name: name.into(),
            ..Default::default()
        }
        .into(),
        node: Some(Node::from(Struct::default())),
        ..Default::default()
    };
    let id = |index: i32, name: &str| ID {
        index,
        name: name.into(),
        ..Default::default()
    };
    let view = |refs: Vec<ID>| DeclView {
        own: true,
        references: refs,
        ..Default::default()
    };
    Request {
        target: "x".into(),
        model: Model {
            decls: vec![decl("A"), decl("B"), decl("C"), decl("D")],
            types: vec![Type::default(), Type::default()],
            ..Default::default()
        }
        .into(),
        analysis: Analysis {
            types: vec![
                TypeView {
                    references: vec![id(1, "B")],
                    ..Default::default()
                },
                TypeView {
                    references: vec![id(2, "C")],
                    ..Default::default()
                },
            ],
            decls: vec![
                view(vec![id(1, "B")]),
                view(vec![id(2, "C")]),
                view(vec![]),
                view(vec![]),
            ],
            ..Default::default()
        }
        .into(),
        ..Default::default()
    }
}

fn names(decls: &[&Decl]) -> Vec<String> {
    decls.iter().map(|d| d.meta.name.clone()).collect()
}

#[test]
fn follows_references() {
    let req = cascading();
    let s = Session::new(&req, "X").unwrap();
    let decls = &s.model().decls;

    assert_eq!(names(&s.references(&decls[0])), ["B"]);
    assert_eq!(names(&s.references(&decls[1])), ["C"]);
    assert!(s.references(&decls[3]).is_empty());
    let types = [id(0), id(1), id(0), id(9)];
    assert_eq!(names(&s.type_references(&types)), ["B", "C"]);

    fn id(index: i32) -> ID {
        ID {
            index,
            ..Default::default()
        }
    }
}

#[test]
fn cascades_skips() {
    let req = cascading();
    let mut s = Session::new(&req, "X").unwrap();
    let decls: Vec<&Decl> = s.model().decls.iter().collect();
    let mut skipped = Skipped::new();
    skipped.insert(decls[2]);

    s.cascade(&decls, &mut skipped);

    assert_eq!(skipped.len(), 3);
    assert!(skipped.contains(decls[0]) && skipped.contains(decls[1]));
    assert!(!skipped.contains(decls[3]));
    let got: Vec<_> = s.diagnostics.iter().map(|d| d.message.as_str()).collect();
    assert_eq!(
        got,
        [
            "B names C, which is not generated",
            "A names B, which is not generated"
        ]
    );
}
