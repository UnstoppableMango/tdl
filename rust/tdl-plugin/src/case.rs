//! Case styles join the words of a name. The analysis ships every
//! declaration's, field's, and variant's words, so every SDK cases a name
//! identically.

/// A case style: joins the words of a name.
pub type Style = fn(&[String]) -> String;

/// Joins words with each one's first letter upper case and the rest as
/// written: `user` and `id` are `UserId`, and `user` and `ID` are `UserID`.
pub fn pascal(words: &[String]) -> String {
    words.iter().map(|w| upper_first(w)).collect()
}

/// Is [`pascal`] with the first word entirely lower case.
pub fn camel(words: &[String]) -> String {
    let Some((first, rest)) = words.split_first() else {
        return String::new();
    };
    let mut s = first.to_lowercase();
    for w in rest {
        s.push_str(&upper_first(w));
    }
    s
}

/// Joins lower-case words with underscores.
pub fn snake(words: &[String]) -> String {
    join(words, str::to_lowercase)
}

/// Joins upper-case words with underscores.
pub fn screaming_snake(words: &[String]) -> String {
    join(words, str::to_uppercase)
}

fn join(words: &[String], case: fn(&str) -> String) -> String {
    words.iter().map(|w| case(w)).collect::<Vec<_>>().join("_")
}

/// Splits a name into the words a case style joins, as tdl splits the names
/// it ships: `userID` is `user` and `ID`, `HTTPServer` is `HTTP` and
/// `Server`, and `last4` is one word. Anything that is not a letter or a
/// digit separates words and is dropped.
pub fn words(name: &str) -> Vec<String> {
    let chars: Vec<char> = name.chars().collect();
    let mut words = Vec::new();
    let mut cur = String::new();
    let mut prev: Option<char> = None;
    for (i, &c) in chars.iter().enumerate() {
        if !c.is_alphanumeric() {
            flush(&mut words, &mut cur);
            continue;
        }
        if c.is_uppercase() && !cur.is_empty() {
            let next_lower = chars.get(i + 1).is_some_and(|n| n.is_lowercase());
            let split = prev.is_some_and(|p| {
                p.is_lowercase() || p.is_numeric() || (p.is_uppercase() && next_lower)
            });
            if split {
                flush(&mut words, &mut cur);
            }
        }
        cur.push(c);
        prev = Some(c);
    }
    flush(&mut words, &mut cur);
    words
}

fn flush(words: &mut Vec<String>, cur: &mut String) {
    if !cur.is_empty() {
        words.push(std::mem::take(cur));
    }
}

/// The part of a dotted name after its last dot.
pub fn last_segment(name: &str) -> &str {
    name.rsplit_once('.').map_or(name, |(_, last)| last)
}

/// Turns SCREAMING_SNAKE into Pascal case: `IN_TRANSIT` is `InTransit`.
pub fn from_screaming(s: &str) -> String {
    s.split('_')
        .filter(|w| !w.is_empty())
        .map(|w| upper_first(&w.to_lowercase()))
        .collect()
}

fn upper_first(w: &str) -> String {
    let mut chars = w.chars();
    match chars.next() {
        Some(first) => first.to_uppercase().chain(chars).collect(),
        None => String::new(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn w(words: &[&str]) -> Vec<String> {
        words.iter().map(|s| s.to_string()).collect()
    }

    #[test]
    fn styles_join_words() {
        let ws = w(&["user", "ID"]);
        assert_eq!(pascal(&ws), "UserID");
        assert_eq!(camel(&ws), "userID");
        assert_eq!(snake(&ws), "user_id");
        assert_eq!(screaming_snake(&ws), "USER_ID");
        assert_eq!(pascal(&w(&["user", "id"])), "UserId");
        assert_eq!(camel(&w(&["HTTP", "server"])), "httpServer");
        assert_eq!(camel(&[]), "");
        assert_eq!(pascal(&[]), "");
    }

    #[test]
    fn words_splits_as_tdl_does() {
        let cases: [(&str, &[&str]); 9] = [
            ("userID", &["user", "ID"]),
            ("HTTPServer", &["HTTP", "Server"]),
            ("last4", &["last4"]),
            ("last4Digits", &["last4", "Digits"]),
            ("line_items", &["line", "items"]),
            ("OrderLine", &["Order", "Line"]),
            ("ID", &["ID"]),
            ("a-b c", &["a", "b", "c"]),
            ("", &[]),
        ];
        for (name, want) in cases {
            assert_eq!(words(name), w(want), "{name}");
        }
    }

    #[test]
    fn last_segment_is_after_the_last_dot() {
        assert_eq!(last_segment("shop.billing.Invoice"), "Invoice");
        assert_eq!(last_segment("Invoice"), "Invoice");
    }

    #[test]
    fn from_screaming_is_pascal() {
        assert_eq!(from_screaming("IN_TRANSIT"), "InTransit");
        assert_eq!(from_screaming("_A__B_"), "AB");
    }
}
