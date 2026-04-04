package migrations

type CreateScansTable_2026_04_04_180000 struct {
	Migration
}

func (m *CreateScansTable_2026_04_04_180000) Up() {
	m.CreateTable("scans", func(t *Table) {
		t.UUID("id").PrimaryKey().Default("gen_random_uuid()")
		t.UUID("user_id").NotNull().ForeignKey("users", "id")
		t.String("tool", 50).NotNull()
		t.String("project", 255).NotNull()
		t.String("commit_hash", 64).Nullable()
		t.String("tool_version", 50).Nullable()
		t.Integer("signal_count").NotNull().Default("0")
		t.Timestamps()
	})

	m.AddIndex("scans", "tool")
	m.AddIndex("scans", "tool", "project")
}

func (m *CreateScansTable_2026_04_04_180000) Down() {
	m.DropTableIfExists("scans")
}
