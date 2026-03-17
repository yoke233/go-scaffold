package conf

type Bootstrap struct {
	Server Server `yaml:"server"`
	Data   Data   `yaml:"data"`
}

type Server struct {
	HTTP HTTPServer `yaml:"http"`
	GRPC GRPCServer `yaml:"grpc"`
}

type HTTPServer struct {
	Addr string `yaml:"addr"`
}

type GRPCServer struct {
	Addr string `yaml:"addr"`
}

type Data struct {
	Database Database `yaml:"database"`
}

type Database struct {
	DSN string `yaml:"dsn"`
}
