// Holds the TypeScript generated from models/ to DefinitelyTyped's
// declarations for the same trees. An assertion that fails is a type error,
// so `tsc --noEmit` is the whole check.
//
// Two directions are checked. Every tree DefinitelyTyped accepts must be one
// the generated types accept, whole trees included. Going back, each
// generated node must be one DefinitelyTyped accepts, apart from what the
// models knowingly loosen: children, which are any node rather than the
// subset the tree allows, and hast's property values, narrowed to strings.
import type * as hast from "hast";
import type * as mdast from "mdast";
import type * as unist from "unist";
import type * as genHast from "../../../models/hast/ts/hast.js";
import type * as genMdast from "../../../models/mdast/ts/mdast.js";
import type * as genUnist from "../../../models/unist/ts/unist.js";

type Assert<T extends true> = T;
type Extends<A, B> = [A] extends [B] ? true : false;
type Same<A, B> = [Extends<A, B>, Extends<B, A>] extends [true, true] ? true : false;

// A node without the fields a model loosens.
type Strict<N, Loose extends PropertyKey> = Omit<N, Loose>;

export type Unist = [
	Assert<Same<genUnist.Point, unist.Point>>,
	Assert<Same<genUnist.Position, unist.Position>>,
	Assert<Same<genUnist.Data, unist.Data>>,
	Assert<Same<genUnist.Node, Omit<unist.Node, "type">>>,
];

// One entry per mdast node: the generated node beside DefinitelyTyped's.
interface MdastNodes {
	root: [genMdast.Root, mdast.Root];
	blockquote: [genMdast.Blockquote, mdast.Blockquote];
	break: [genMdast.Break, mdast.Break];
	code: [genMdast.Code, mdast.Code];
	definition: [genMdast.Definition, mdast.Definition];
	delete: [genMdast.Delete, mdast.Delete];
	emphasis: [genMdast.Emphasis, mdast.Emphasis];
	footnoteDefinition: [genMdast.FootnoteDefinition, mdast.FootnoteDefinition];
	footnoteReference: [genMdast.FootnoteReference, mdast.FootnoteReference];
	heading: [genMdast.Heading, mdast.Heading];
	html: [genMdast.Html, mdast.Html];
	image: [genMdast.Image, mdast.Image];
	imageReference: [genMdast.ImageReference, mdast.ImageReference];
	inlineCode: [genMdast.InlineCode, mdast.InlineCode];
	link: [genMdast.Link, mdast.Link];
	linkReference: [genMdast.LinkReference, mdast.LinkReference];
	list: [genMdast.List, mdast.List];
	listItem: [genMdast.ListItem, mdast.ListItem];
	paragraph: [genMdast.Paragraph, mdast.Paragraph];
	strong: [genMdast.Strong, mdast.Strong];
	table: [genMdast.Table, mdast.Table];
	tableCell: [genMdast.TableCell, mdast.TableCell];
	tableRow: [genMdast.TableRow, mdast.TableRow];
	text: [genMdast.Text, mdast.Text];
	thematicBreak: [genMdast.ThematicBreak, mdast.ThematicBreak];
	yaml: [genMdast.Yaml, mdast.Yaml];
}

// Every node DefinitelyTyped declares is modelled, and nothing else is.
export type MdastCoverage = [
	Assert<Same<keyof MdastNodes, mdast.Nodes["type"]>>,
	Assert<Same<keyof MdastNodes, genMdast.Nodes["type"]>>,
];

export type MdastForward = [
	Assert<Extends<mdast.Nodes, genMdast.Nodes>>,
	Assert<Extends<mdast.Root, genMdast.Root>>,
];

// Each key whose check failed, so a failure names the node.
type Failing<T> = { [K in keyof T]: T[K] extends true ? never : K }[keyof T];

// What each mdast node loosens: children, and a heading's depth, which the
// model constrains to 1 through 6 where DefinitelyTyped spells the literals.
type MdastLoose<K> = K extends "heading" ? "children" | "depth" : "children";

export type MdastBack = Assert<
	Same<
		Failing<{
			[K in keyof MdastNodes]: Extends<
				Strict<MdastNodes[K][0], MdastLoose<K>>,
				Strict<MdastNodes[K][1], MdastLoose<K>>
			>;
		}>,
		never
	>
>;

interface HastNodes {
	root: [genHast.Root, hast.Root];
	element: [genHast.Element, hast.Element];
	text: [genHast.Text, hast.Text];
	comment: [genHast.Comment, hast.Comment];
	doctype: [genHast.Doctype, hast.Doctype];
}

export type HastCoverage = [
	Assert<Same<keyof HastNodes, hast.Nodes["type"]>>,
	Assert<Same<keyof HastNodes, genHast.Nodes["type"]>>,
];

// A hast tree holding an element holds properties the model narrows, so the
// forward check runs on nodes with properties set aside, and the narrowing
// itself is asserted to still be needed.
type WithoutProperties<N> = N extends { type: "element" }
	? Omit<N, "properties" | "children" | "content">
	: Omit<N, "children">;

export type HastForward = [
	Assert<Extends<WithoutProperties<hast.Nodes>, WithoutProperties<genHast.Nodes>>>,
	Assert<Extends<Extends<hast.Properties, genHast.Element["properties"]>, false>>,
];

export type HastBack = Assert<
	Same<
		Failing<{
			[K in keyof HastNodes]: Extends<
				Strict<HastNodes[K][0], "children" | "content">,
				Strict<HastNodes[K][1], "children" | "content">
			>;
		}>,
		never
	>
>;
