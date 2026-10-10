// A file written by hand, in the shapes the generator writes.

/** An order, identified by its id. */
export interface Order {
  id: string;
  /** When it was placed. */
  placedAt: string;
  lines: OrderLine[];
  notes?: Array<string | null>;
  gift: boolean | null;
  payment: Payment;
  status: Status;
  totals: Record<string, number>;
  /** @deprecated nobody has one */
  fax?: string;
  "line-count": number;
  priority: 1 | 2 | 3;
}

export interface OrderLine {
  sku: string;
  quantity: number;
  grid: number[][];
}

/**
 * Where an order is.
 *
 * @deprecated use Order.status
 */
export type Status = "Draft" | "Placed" | "lowest";

export type Currency = "usd" | "eur-cents";

export type Sku = string;

/** How an order is paid. */
export type Payment = PaymentCard | Wire;

/** A card on file. */
export interface PaymentCard {
  type: "Card";
  last4: string;
}

export interface Wire {
  type: "Wire";
  iban: string;
  /** The bank's code. */
  bic?: string;
}
