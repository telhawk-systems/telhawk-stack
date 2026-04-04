package migrations

type CreateFindingsTable_2026_04_04_180001 struct {
	Migration
}

func (m *CreateFindingsTable_2026_04_04_180001) Up() {
	m.CreateTable("findings", func(t *Table) {
		t.UUID("id").PrimaryKey().Default("gen_random_uuid()")
		t.String("fingerprint", 64).NotNull()
		t.UUID("scan_id").NotNull().ForeignKey("scans", "id")
		t.String("tool", 50).NotNull()
		t.String("signal_type", 50).NotNull()
		t.String("severity", 20).Nullable()
		t.String("category", 100).Nullable()
		t.String("route", 500).Nullable()
		t.String("file_path", 500).Nullable()
		t.Integer("line").Nullable()
		t.JSONB("data").NotNull()
		t.Timestamp("first_seen").NotNull().Default("NOW()")
		t.Timestamp("last_seen").NotNull().Default("NOW()")
		t.Timestamps()
	})

	m.AddUniqueIndex("findings", "fingerprint", "tool")
	m.AddIndex("findings", "scan_id")
	m.AddIndex("findings", "tool", "signal_type")
	m.AddIndex("findings", "severity")
	m.AddIndex("findings", "category")
}

func (m *CreateFindingsTable_2026_04_04_180001) Down() {
	m.DropTableIfExists("findings")
}
