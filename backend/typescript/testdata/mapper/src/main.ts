import type { Order, Status } from "./shop.tdl";

const status: Status = "Open";

export const order: Order = {
  id: "1",
  total: { amount: "1.00", currency: "USD" },
  status,
};
