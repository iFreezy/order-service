package section

type (
	Repository struct {
		Postgres RepositoryPostgres
	}

	RepositoryPostgres struct {
		Address  string `required:"true"`
		Username string `required:"true"`
		Password string `required:"true"`
		Name     string `required:"true"`
	}
)
