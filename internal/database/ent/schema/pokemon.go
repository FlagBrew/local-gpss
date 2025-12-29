package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Pokemon struct {
	ent.Schema
}

func (Pokemon) Fields() []ent.Field {
	return []ent.Field{
		field.Time("upload_datetime"),
		field.String("download_code").Unique(),
		field.Int("download_count").Default(0),
		field.String("generation"),
		field.Bool("legal"),
		field.String("base_64"),
	}
}

func (Pokemon) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("bundles", Bundle.Type).Ref("pokemons"),
	}
}
