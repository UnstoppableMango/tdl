// A model written by hand, in the shapes the generator writes.

$version: "2"

namespace shop

/// An order, identified by its id.
structure Order {
    @required
    id: String

    /// When it was placed.
    @required
    placedAt: Timestamp

    @required
    lines: OrderLineList

    notes: SparseStringList

    @required
    payment: Payment

    @required
    status: Status

    @required
    totals: StringBigDecimalMap

    @deprecated(message: "nobody has one")
    fax: String

    @required
    labels: Labels
}

structure OrderLine {
    @required
    sku: Sku

    @required
    quantity: Integer

    @required
    grid: DoubleListList
}

/// Where an order is.
@deprecated(message: "use Order.status")
enum Status {
    DRAFT = "Draft"
    PLACED = "Placed"
    GONE = "Lost"
}

/// A stock keeping unit.
string Sku

/// Labels an order carries.
@uniqueItems
list Labels {
    member: String
}

/// How an order is paid.
union Payment {
    /// A card on file.
    card: PaymentCard

    wire: WireTransfer

    cash: Unit
}

/// A card on file.
structure PaymentCard {
    @required
    last4: String
}

structure WireTransfer {
    @required
    iban: String

    /// The bank's code.
    bic: String
}

list OrderLineList {
    member: OrderLine
}

@sparse
list SparseStringList {
    member: String
}

list DoubleList {
    member: Double
}

list DoubleListList {
    member: DoubleList
}

map StringBigDecimalMap {
    key: String
    value: BigDecimal
}
