package main

import (
	"log"
	"log/slog"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	utils "github.com/sisoputnfrba/tp-golang/utils/kernel"
)

func main() {
	utils.ConfigurarLogger()

	globals.KernelConfig = utils.IniciarConfiguracion("config.json")
	if globals.KernelConfig == nil {
		log.Fatal("No se pudo iniciar el config")
	}

	slog.SetLogLoggerLevel(utils.Log_level_from_string(globals.KernelConfig.Log_level))

	slog.Info(globals.KernelConfig.Ip_memory)
}
