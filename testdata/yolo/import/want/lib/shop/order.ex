defmodule Shop.Order do
  @moduledoc "An order."

  @enforce_keys [:invoice, :total, :rate]
  defstruct [:invoice, :total, :rate]

  @type t :: %__MODULE__{
          invoice: Shop.Billing.Invoice.t(),
          total: Money.Money.t(),
          rate: MyApp.Tax.Rate.t()
        }
end
