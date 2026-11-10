// Case styles join the words of a name. The analysis ships every
// declaration's, field's, and variant's words, so every SDK cases a name
// identically.

/** A case style: joins the words of a name. */
export type Style = (words: readonly string[]) => string;

/**
 * Joins words with each one's first letter upper case and the rest as
 * written: `user` and `id` are `UserId`, and `user` and `ID` are `UserID`.
 */
export const pascal: Style = (words) => words.map(upperFirst).join("");

/** Is {@link pascal} with the first word entirely lower case. */
export const camel: Style = (words) => {
	const [first, ...rest] = words;
	if (first === undefined) {
		return "";
	}
	return first.toLowerCase() + rest.map(upperFirst).join("");
};

/** Joins lower-case words with underscores. */
export const snake: Style = (words) => words.map((w) => w.toLowerCase()).join("_");

/** Joins upper-case words with underscores. */
export const screamingSnake: Style = (words) => words.map((w) => w.toUpperCase()).join("_");

const letterOrDigit = /^[\p{L}\p{N}]$/u;
const upper = /^\p{Lu}$/u;
const lower = /^\p{Ll}$/u;
const digit = /^\p{N}$/u;

/**
 * Splits a name into the words a case style joins, as tdl splits the names
 * it ships: `userID` is `user` and `ID`, `HTTPServer` is `HTTP` and
 * `Server`, and `last4` is one word. Anything that is not a letter or a
 * digit separates words and is dropped.
 */
export function words(name: string): string[] {
	const chars = Array.from(name);
	const out: string[] = [];
	let cur = "";
	let prev: string | undefined;
	chars.forEach((c, i) => {
		if (!letterOrDigit.test(c)) {
			if (cur !== "") {
				out.push(cur);
				cur = "";
			}
			return;
		}
		if (upper.test(c) && cur !== "" && prev !== undefined) {
			const next = chars[i + 1];
			const nextLower = next !== undefined && lower.test(next);
			if (lower.test(prev) || digit.test(prev) || (upper.test(prev) && nextLower)) {
				out.push(cur);
				cur = "";
			}
		}
		cur += c;
		prev = c;
	});
	if (cur !== "") {
		out.push(cur);
	}
	return out;
}

/** The part of a dotted name after its last dot. */
export function lastSegment(name: string): string {
	return name.slice(name.lastIndexOf(".") + 1);
}

/** Turns SCREAMING_SNAKE into Pascal case: `IN_TRANSIT` is `InTransit`. */
export function fromScreaming(s: string): string {
	return s
		.split("_")
		.filter((w) => w !== "")
		.map((w) => upperFirst(w.toLowerCase()))
		.join("");
}

function upperFirst(w: string): string {
	const first = w.codePointAt(0);
	if (first === undefined) {
		return "";
	}
	const head = String.fromCodePoint(first);
	return head.toUpperCase() + w.slice(head.length);
}
