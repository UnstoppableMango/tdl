defmodule Shop.Billing.Invoice do
  @moduledoc "An invoice."

  @enforce_keys [:id]
  defstruct [:id]

  @type t :: %__MODULE__{id: String.t()}
end
