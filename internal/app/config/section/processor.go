package section

type (
	Processor struct {
		WebServer ProcessorWebServer
	}

	ProcessorWebServer struct {
		ListenPort int `default:"8082"`
	}
)
