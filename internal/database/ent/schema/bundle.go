package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Bundle struct {
	ent.Schema
}

func (Bundle) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").Unique(),
		field.Time("upload_datetime"),
		field.String("download_code").Unique(),
		field.Int("download_count"),
		field.Bool("legal"),
		field.String("min_gen"),
		field.String("max_gen"),
	}
}

func (Bundle) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("pokemons", Pokemon.Type),
	}
}
