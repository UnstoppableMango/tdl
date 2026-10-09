namespace * shop.v1

/**
 * A line on an order.
 */
struct LineItem {
  1: string sku
  2: i32 quantity
  5: optional double price
}

enum Status {
  DRAFT = 1
  PLACED = 2
  CANCELLED = 10
  on_hold = 11
}

struct CardPayment {
  1: string last4
}

struct PaymentCash {
}

struct PaymentBankTransfer {
  1: string iban
}

union Payment {
  1: CardPayment card
  2: PaymentCash cash
  4: PaymentBankTransfer bankTransfer
}

typedef string Sku

struct order_summary {
  1: list<LineItem> items
  2: set<Sku> skus
  3: map<string, i64> totals
  4: Status status (deprecated = "use state")
  5: Payment payment
  6: binary receipt
  7: bool paid
  8: string Note
}
