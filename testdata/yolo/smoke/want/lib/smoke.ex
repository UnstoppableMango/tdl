defmodule Smoke.Email do
  @moduledoc "An address to write to."
  @type t :: String.t()
end

defmodule Smoke.WorkEmail do
  @type t :: Smoke.Email.t()
end

defmodule Smoke.Sku do
  @type t :: String.t()
end

defmodule Smoke.Skus do
  @type t :: [Smoke.Sku.t()]
end

defmodule Smoke.Order do
  @moduledoc "An order, identified by its id."

  @enforce_keys [:id, :created_at, :updated_at, :items, :tags, :ship_on, :took, :receipt, :paid, :contact, :skus, :payment, :status, :fax]
  defstruct [:id, :created_at, :updated_at, :items, :tags, :note, :ship_on, :took, :receipt, :paid, :contact, :skus, :payment, :status, :fax]

  @type t :: %__MODULE__{
          id: String.t(),
          created_at: DateTime.t(),
          updated_at: DateTime.t(),
          items: [Smoke.LineItem.t()],
          tags: MapSet.t(String.t()),
          note: String.t() | nil,
          ship_on: Date.t() | nil,
          took: integer(),
          receipt: binary(),
          paid: boolean(),
          contact: Smoke.WorkEmail.t(),
          skus: Smoke.Skus.t(),
          payment: Smoke.Payment.t(),
          status: Smoke.Status.t(),
          fax: String.t()
        }
end

defmodule Smoke.LineItem do
  @enforce_keys [:sku, :quantity]
  defstruct [:sku, :quantity]

  @type t :: %__MODULE__{sku: Smoke.Sku.t(), quantity: pos_integer()}
end

defmodule Smoke.Status do
  @type t :: :draft | :placed | :provisional
end

defmodule Smoke.Payment do
  defmodule Card do
    @enforce_keys [:last4]
    defstruct [:last4]
    @type t :: %__MODULE__{last4: String.t()}
  end

  defmodule Cash do
    defstruct []
    @type t :: %__MODULE__{}
  end

  @type t :: Card.t() | Cash.t()
end

defmodule Smoke.Totals do
  @enforce_keys [:by_currency]
  defstruct [:by_currency]
  @type t :: %__MODULE__{by_currency: %{optional(String.t()) => Decimal.t()}}
end

defmodule Smoke.Counts do
  @enforce_keys [:by_status]
  defstruct [:by_status]
  @type t :: %__MODULE__{by_status: %{optional(Smoke.Status.t()) => integer()}}
end

defmodule Smoke.Sparse do
  @enforce_keys [:values]
  defstruct [:values]
  @type t :: %__MODULE__{values: [integer() | nil]}
end

defmodule Smoke.Parcel do
  @moduledoc "Constraints a language with literal and tuple types states as a type."
  @enforce_keys [:carrier, :priority, :dims, :stage]
  defstruct [:carrier, :priority, :dims, :stage]

  @type t :: %__MODULE__{
          carrier: String.t(),
          priority: 1..3,
          dims: {float(), float(), float()},
          stage: :draft | :placed
        }
end

defmodule Smoke.LegacyOrder do
  @moduledoc deprecated: "use Order"
  @enforce_keys [:id]
  defstruct [:id]
  @type t :: %__MODULE__{id: String.t()}
end
