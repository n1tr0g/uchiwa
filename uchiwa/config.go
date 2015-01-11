package uchiwa

import (
	"encoding/json"
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"

	"github.com/palourde/logger"
	"strings"
)

// Config struct contains []SensuConfig and UchiwaConfig structs
type Config struct {
	Sensu  []SensuConfig
	Uchiwa GlobalConfig
}

// SensuConfig struct contains conf about a Sensu API
type SensuConfig struct {
	Name     string
	Host     string
	Port     int
	Ssl      bool
	Insecure bool
	URL      string
	User     string
	Path     string
	Pass     string
	Timeout  int
}

// GlobalConfig struct contains conf about Uchiwa
type GlobalConfig struct {
	Host     string
	Port     int
	Refresh  int
	Auth     string
	Authfile string
	Users    map[string]string
	Pass     string
	User     string
}


func (c *Config) initSensu() {
	for i, api := range c.Sensu {
		prot := "http"
		if api.Name == "" {
			logger.Warningf("Sensu API %s has no name property. Generating random one...", api.URL)
			c.Sensu[i].Name = fmt.Sprintf("sensu-%v", rand.Intn(100))
		}
		if api.Host == "" {
			logger.Fatalf("Sensu API %q Host is missing", api.Name)
		}
		if api.Timeout == 0 {
			c.Sensu[i].Timeout = 10
		} else if api.Timeout >= 1000 { // backward compatibility with < 0.3.0 version
			c.Sensu[i].Timeout = api.Timeout / 1000
		}
		if api.Port == 0 {
			c.Sensu[i].Port = 4567
		}
		if api.Ssl {
			prot += "s"
		}
		c.Sensu[i].URL = fmt.Sprintf("%s://%s:%d%s", prot, api.Host, c.Sensu[i].Port, api.Path)
	}
}

func (c *Config) initGlobal() {
	if c.Uchiwa.Host == "" {
		c.Uchiwa.Host = "0.0.0.0"
	}
	if c.Uchiwa.Port == 0 {
		c.Uchiwa.Port = 3000
	}
	if c.Uchiwa.Refresh == 0 {
		c.Uchiwa.Refresh = 10
	} else if c.Uchiwa.Refresh >= 1000 { // backward compatibility with < 0.3.0 version
		c.Uchiwa.Refresh = c.Uchiwa.Refresh / 1000
	}
	// backward compatibility 0.4.0
	if c.Uchiwa.Auth == "" {
		if c.Uchiwa.User == "" || c.Uchiwa.Pass == "" {
			c.Uchiwa.Auth = "none"
		} else {
			c.Uchiwa.Auth = "simple"
		}
	}
	switch strings.ToLower(c.Uchiwa.Auth) {
		case "simple":
			if c.Uchiwa.User == "" || c.Uchiwa.Pass == "" {
				logger.Fatalf("For auth=Simple you need to define user and pass in the config.json")
			}
		case "htpasswd":
			if _, err := os.Stat(c.Uchiwa.Authfile); os.IsNotExist(err) {
				logger.Fatalf("Htpasswd %q file is missing", c.Uchiwa.Authfile)
			}
			users, err := loadHtpasswdFile(c.Uchiwa.Authfile)
			if err != nil {
				logger.Fatalf("Can't load users passwd file: %s.", err)
			}
			c.Uchiwa.Users = users
		case "none":
		default:
			logger.Fatalf("Unknown auth type %q", c.Uchiwa.Auth)
	}
}

func loadHtpasswdFile(path string) (map[string]string, error) {
	r, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	csv_reader := csv.NewReader(r)
	csv_reader.Comma = ':'
	csv_reader.Comment = '#'
	csv_reader.TrimLeadingSpace = true

	records, err := csv_reader.ReadAll()
	if err != nil {
		return nil, err
	}
	h := make(map[string]string)
	for _, record := range records {
		h[record[0]] = record[1]
	}
	return h, nil
}

func buildPublicConfig(c *Config) {
	p := new(Config)
	p.Uchiwa = c.Uchiwa
	p.Uchiwa.User = "*****"
	p.Uchiwa.Pass = "*****"
	p.Sensu = make([]SensuConfig, len(c.Sensu))
	for i := range c.Sensu {
		p.Sensu[i] = c.Sensu[i]
		p.Sensu[i].User = "*****"
		p.Sensu[i].Pass = "*****"
	}
	PublicConfig = p
}

// LoadConfig function loads a specified configuration file and return a Config struct
func LoadConfig(path string) (*Config, error) {
	logger.Infof("Loading configuration file %s", path)
	c := new(Config)
	file, err := os.Open(path)
	if err != nil {
		if len(path) > 1 {
			return nil, fmt.Errorf("Error: could not read config file %s.", path)
		}
	}

	decoder := json.NewDecoder(file)
	err = decoder.Decode(c)
	if err != nil {
		return nil, fmt.Errorf("Error decoding file %s: %s", path, err)
	}

	c.initGlobal()
	c.initSensu()

	return c, nil
}
