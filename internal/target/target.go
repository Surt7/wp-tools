package target

type BackendConfig struct {
	Name    string
	ApiUrl  string
	GrpcUrl string
	UseTls  bool
}

var (
	AspBackend = BackendConfig{
		Name:    "ASP.NET Core",
		ApiUrl:  "https://api.worldpinball.local",
		GrpcUrl: "api.worldpinball.local:443",
		UseTls:  true,
	}
	SpringBackend = BackendConfig{
		Name:    "Spring Boot",
		ApiUrl:  "http://localhost:7085",
		GrpcUrl: "localhost:9090",
		UseTls:  false,
	}
)

var (
	TargetSpring bool
	TargetAsp    bool
)

// Current returns the active BackendConfig based on CLI flags.
func Current() BackendConfig {
	if TargetSpring {
		return SpringBackend
	}
	return AspBackend
}
