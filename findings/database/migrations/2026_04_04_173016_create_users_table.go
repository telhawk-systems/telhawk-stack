package migrations

type CreateUsersTable_2026_04_04_173016 struct {
	Migration
}

func (m *CreateUsersTable_2026_04_04_173016) Up() {
	m.CreateTable("users", func(t *Table) {
		t.UUID("id").PrimaryKey().Default("gen_random_uuid()")
		t.String("name").NotNull()
		t.String("email").NotNull().Unique()
		t.String("password").NotNull()
		t.Timestamps()
	})
}

func (m *CreateUsersTable_2026_04_04_173016) Down() {
	m.DropTableIfExists("users")
}
