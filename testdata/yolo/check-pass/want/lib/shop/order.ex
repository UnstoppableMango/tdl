defmodule Shop.Order do
  @moduledoc "An order."

  @enforce_keys [:id]
  defstruct [:id]

  @type t :: %__MODULE__{id: String.t()}
end
