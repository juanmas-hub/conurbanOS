package main

import (
	"log"
	"log/slog"

	globals "github.com/sisoputnfrba/tp-golang/globals/cpu"
	utils_cpu "github.com/sisoputnfrba/tp-golang/utils/cpu"
	utils_logger "github.com/sisoputnfrba/tp-golang/utils/loggers"
)

func main() {

	utils_logger.ConfigurarLogger("cpu.log")

	globals.CpuConfig = utils_cpu.IniciarConfiguracion("config.json")
	if globals.CpuConfig == nil {
		log.Fatal("No se pudo iniciar el config")
	}

	slog.SetLogLoggerLevel(utils_logger.Log_level_from_string(globals.CpuConfig.Log_level))

	slog.Info(globals.CpuConfig.Log_level)

	// Cliente (manda mensaje a kernel)
	mensaje := "Hola desde cpu (cliente)"
	utils_cpu.EnviarMensaje(globals.CpuConfig.Ip_kernel, globals.CpuConfig.Port_kernel, mensaje)
}
