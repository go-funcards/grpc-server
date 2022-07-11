package grpcserver

import (
	"fmt"
	"go.uber.org/zap"
	"net"
	"os"
	"path"
	"path/filepath"
)

type Config struct {
	Type   string `yaml:"type" env:"TYPE" env-default:"port"` // port or sock
	BindIP string `yaml:"bind_ip" env:"BIND_IP" env-default:""`
	Port   uint16 `yaml:"port" env:"PORT" env-default:"0"`
}

func (cfg Config) Listener(logger *zap.Logger) net.Listener {
	if cfg.Type == "sock" {
		return sock(logger)
	}

	logger.Debug("Create and listen tcp")
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.BindIP, cfg.Port))
	if err != nil {
		logger.Fatal("Failed to create listener on tcp", zap.Error(err))
	}
	logger.Info(fmt.Sprintf("Bind application to addr: %s", listener.Addr().(*net.TCPAddr).String()))
	return listener
}

func sock(logger *zap.Logger) net.Listener {
	appDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		logger.Fatal("Unexpected error", zap.Error(err))
	}

	socketPath := path.Join(appDir, "app.sock")
	logger.Info(fmt.Sprintf("Socket path: %s", socketPath))

	logger.Debug("Create and listen unix socket")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		logger.Fatal("Failed to create listener on unix socket", zap.Error(err))
	}
	return listener
}
