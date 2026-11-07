defmodule Shop.Order do
  @moduledoc "An order."

  @enforce_keys [:id, :tags]
  defstruct [:id, :tags]

  @type t :: %__MODULE__{
          id: String.t(),
          tags: MapSet.t(String.t())
        }
end
