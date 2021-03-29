package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Server Server `yaml:"server"`
}

type Server struct {
	Timeout Timeout `yaml:"timeout"`
	Host    string  `yaml:"host"`
	Port    string  `yaml:"port"`
}

type Timeout struct {
	Idle   int `yaml:"idle"`
	Server int `yaml:"server"`
	Read   int `yaml:"read"`
	Write  int `yaml:"write"`
}

func newConfig(configPath string) (*Config, error) {

	config := &Config{}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	d := yaml.NewDecoder(file)

	if err := d.Decode(&config); err != nil {
		return nil, err
	}

	return config, nil
}

func validateConfigPath(path string) error {
	s, err := os.Stat(path)
	if err != nil {
		return err
	}
	if s.IsDir() {
		return fmt.Errorf("'%s' é um diretório e não um arquivo", path)
	}
	return nil
}

func ParseFlags() (string, error) {

	var config string

	flag.StringVar(&config, "config", "./config.yml", "caminho para o arquivo de configuração")

	flag.Parse()

	if err := validateConfigPath(config); err != nil {
		return "", err
	}
	return config, nil
}

func NewRouter() *http.ServeMux {
	router := http.NewServeMux()

	router.HandleFunc("/home", func(res http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(res, "Olá, resquest: %s\n", req.URL.Path)
	})

	return router
}

func (config Config) run() {
	var runChan = make(chan os.Signal, 1)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(config.Server.Timeout.Server),
	)
	defer cancel()

	server := &http.Server{
		Addr:         config.Server.Host + ":" + config.Server.Port,
		Handler:      NewRouter(),
		ReadTimeout:  time.Duration(config.Server.Timeout.Read * int(time.Second)),
		WriteTimeout: time.Duration(config.Server.Timeout.Write * int(time.Second)),
		IdleTimeout:  time.Duration(config.Server.Timeout.Idle * int(time.Second)),
	}

	signal.Notify(runChan, os.Interrupt, syscall.SIGTSTP)

	log.Printf("Servidor rodando %s\n", server.Addr)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
			} else {
				log.Fatalf("Falha ao iniciar o servidor: %v", err)
			}
		}
	}()

	interrupt := <-runChan

	log.Printf("Servidor %v\n", interrupt)
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Servidor desligado")
	}

}

func main() {

	cPath, err := ParseFlags()
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := newConfig(cPath)
	if err != nil {
		log.Fatal(err)
	}

	cfg.run()
}
