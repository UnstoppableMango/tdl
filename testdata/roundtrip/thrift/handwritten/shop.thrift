# A handwritten file, with the comment styles and separators Thrift allows.
namespace * shop

// Not a doc comment.
/** An order. */
struct Order {
  2: string id, // trailing
  1: list<string> tags;
  3: optional Status status
}

/* A plain block comment. */
enum Status {
  OPEN = 1,
  SHIPPED = 2;
  # Done.
  DONE = 3
}
