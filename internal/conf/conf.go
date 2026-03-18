package conf

type Bootstrap struct {
	App    App    `yaml:"app"`
	Server Server `yaml:"server"`
	Log    Log    `yaml:"log"`
	Auth   Auth   `yaml:"auth"`
	Data   Data   `yaml:"data"`
}

type App struct {
	Name string `yaml:"name"`
	Env  string `yaml:"env"`
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

type Log struct {
	Level string `yaml:"level"`
}

type Auth struct {
	JWT JWTAuth `yaml:"jwt"`
}

type JWTAuth struct {
	Issuer         string `yaml:"issuer"`
	SigningKey     string `yaml:"signing_key"`
	AccessTokenTTL string `yaml:"access_token_ttl"`
}

type Data struct {
	Database Database `yaml:"database"`
}

type Database struct {
	DSN string `yaml:"dsn"`
}
