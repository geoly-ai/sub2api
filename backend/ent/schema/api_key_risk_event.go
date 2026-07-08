package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// APIKeyRiskEvent records API key anomaly detections.
type APIKeyRiskEvent struct {
	ent.Schema
}

func (APIKeyRiskEvent) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "api_key_risk_events"},
	}
}

func (APIKeyRiskEvent) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.Int64("api_key_id"),
		field.String("rule_code").MaxLen(64).NotEmpty(),
		field.String("severity").MaxLen(20).NotEmpty(),
		field.Int("score").Default(0),
		field.String("status").MaxLen(20).Default("open"),
		field.JSON("evidence", map[string]any{}).
			Optional().
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Time("time_bucket").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("blocked_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("resolved_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Int64("resolved_by").
			Optional().
			Nillable(),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (APIKeyRiskEvent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("api_key_risk_events").Field("user_id").Unique().Required(),
		edge.From("api_key", APIKey.Type).Ref("risk_events").Field("api_key_id").Unique().Required(),
	}
}

func (APIKeyRiskEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("api_key_id"),
		index.Fields("rule_code"),
		index.Fields("severity"),
		index.Fields("status"),
		index.Fields("created_at"),
		index.Fields("api_key_id", "rule_code", "time_bucket").Unique(),
	}
}
