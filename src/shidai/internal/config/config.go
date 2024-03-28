package config

type contextKey string

const ConfigContextKey contextKey = "config"

type Config struct {
	Sekaid SekaidConfig `json:"sekaid"`
}

type SekaidConfig struct {
	Home string `json:"home"`
}

/*
{
    "sekaid": {
        "home": "/path/to/sekai/home"
    }
}
*/
