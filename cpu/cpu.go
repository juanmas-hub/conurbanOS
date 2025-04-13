package main

import (
	"log"
	"log/slog"
	"net/http"
	"strconv"

	utils_cpu "github.com/sisoputnfrba/tp-golang/cpu/utils"
	globals "github.com/sisoputnfrba/tp-golang/globals/cpu"
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

	// Cliente (manda mensaje a kernel y memoria)
	mensaje := "Mensaje desde CPU"
	utils_cpu.EnviarMensaje(globals.CpuConfig.Ip_kernel, globals.CpuConfig.Port_kernel, mensaje)
	utils_cpu.EnviarMensaje(globals.CpuConfig.Ip_memory, globals.CpuConfig.Port_memory, mensaje)

	// Servidor
	// Cuando se ejecuta CPU, hay que mandar a kernel su puerto e IP para que kernel se pueda conectar (no esta hecho)
	mux := http.NewServeMux()

	mux.HandleFunc("/mensajeDeKernel", utils_cpu.RecibirMensajeDeKernel)

	puerto := globals.CpuConfig.Port_cpu
	err := http.ListenAndServe(":"+strconv.Itoa(int(puerto)), mux)
	if err != nil {
		panic(err)
	}
}
