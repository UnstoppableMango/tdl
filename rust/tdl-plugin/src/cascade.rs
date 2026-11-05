use std::collections::HashSet;

use crate::Session;
use crate::ir::{Decl, ID, decl::Node};
use crate::problem::Problem;

/// The declarations a backend does not generate, by their identity in the
/// request.
#[derive(Debug, Default)]
pub struct Skipped(HashSet<*const Decl>);

impl Skipped {
    /// An empty set.
    pub fn new() -> Self {
        Skipped::default()
    }

    /// Adds a declaration, and says whether it was not already there.
    pub fn insert(&mut self, d: &Decl) -> bool {
        self.0.insert(d)
    }

    /// Whether a declaration is in the set.
    pub fn contains(&self, d: &Decl) -> bool {
        self.0.contains(&(d as *const Decl))
    }

    /// How many declarations are in the set.
    pub fn len(&self) -> usize {
        self.0.len()
    }

    /// Whether the set is empty.
    pub fn is_empty(&self) -> bool {
        self.0.is_empty()
    }
}

impl<'a> Session<'a> {
    /// The declarations `d` names through its fields, its variants' fields,
    /// a newtype's base, and their type arguments, with aliases expanded. A
    /// declaration naming itself is included.
    pub fn references(&self, d: &Decl) -> Vec<&'a Decl> {
        self.decl_view(d)
            .map(|v| self.decls_named(&v.references))
            .unwrap_or_default()
    }

    /// The declarations the type references name, walked as
    /// [`references`](Session::references) walks a declaration's, each once.
    pub fn type_references(&self, types: &[ID]) -> Vec<&'a Decl> {
        let mut seen = HashSet::new();
        let refs: Vec<ID> = types
            .iter()
            .filter_map(|t| self.type_view(t))
            .flat_map(|v| &v.references)
            .filter(|r| seen.insert(r.index))
            .cloned()
            .collect();
        self.decls_named(&refs)
    }

    fn decls_named(&self, ids: &[ID]) -> Vec<&'a Decl> {
        ids.iter().filter_map(|id| self.decl_at(id)).collect()
    }

    /// Extends `skipped`, transitively, with every declaration in `decls`
    /// that names a skipped one, and warns about each it adds, so the output
    /// never refers to something it does not declare.
    pub fn cascade(&mut self, decls: &[&'a Decl], skipped: &mut Skipped) {
        let mut changed = true;
        while changed {
            changed = false;
            for &d in decls {
                // An alias emits nothing; references already expands it.
                if skipped.contains(d) || matches!(d.node, Some(Node::Alias(_))) {
                    continue;
                }
                let Some(named) = self.references(d).into_iter().find(|r| skipped.contains(r))
                else {
                    continue;
                };
                skipped.insert(d);
                changed = true;
                self.warn(Problem::unsupported(
                    format!(
                        "{} names {}, which is not generated",
                        d.meta.name, named.meta.name
                    ),
                    d.meta.position.as_option(),
                ));
            }
        }
    }
}
