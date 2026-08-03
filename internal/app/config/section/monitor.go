package section

type Monitor struct {
	LogLevel    string `default:"debug"`
	Environment string `default:"development"`
}
