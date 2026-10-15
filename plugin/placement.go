package plugin

// Short names for the placements a [DirectiveSpec] lists.
const (
	OnBlock   = Placement_PLACEMENT_BLOCK
	OnDecl    = Placement_PLACEMENT_DECL
	OnField   = Placement_PLACEMENT_FIELD
	OnVariant = Placement_PLACEMENT_VARIANT
	OnExtern  = Placement_PLACEMENT_EXTERN
)

// On lists placements for a [DirectiveSpec].
func On(ps ...Placement) []Placement { return ps }

// OnMembers is where a name for a declaration, a field, or a variant goes.
func OnMembers() []Placement { return On(OnDecl, OnField, OnVariant) }
