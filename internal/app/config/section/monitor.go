package section

type Monitor struct {
	LogLevel    string `default:"debug" split_words:"true"`
	Environment string `default:"development"`
}
