package main

import (
	"log"
	"log/slog"
	"net/http"
	"strconv"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	utils_kernel "github.com/sisoputnfrba/tp-golang/kernel/utils"
	utils_logger "github.com/sisoputnfrba/tp-golang/utils/loggers"
)

func main() {
	utils_logger.ConfigurarLogger("kernel.log")

	globals.KernelConfig = utils_kernel.IniciarConfiguracion("config.json")
	if globals.KernelConfig == nil {
		log.Fatal("No se pudo iniciar el config")
	}

	slog.SetLogLoggerLevel(utils_logger.Log_level_from_string(globals.KernelConfig.Log_level))

	// Cliente (mandar mensaje a memoria)
	mensaje := "Mensaje desde Kernel"
	utils_kernel.EnviarMensajeAMemoria(globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory, mensaje)

	// Servidor (recibir mensaje de CPU y IO)
	mux := http.NewServeMux()

	mux.HandleFunc("/mensajeDeCpu", utils_kernel.RecibirMensajeDeCpu)
	mux.HandleFunc("/mensajeDeIo", utils_kernel.RecibirMensajeDeIo)
	mux.HandleFunc("/handshakeIO", utils_kernel.RecibirHandshakeIO)
	mux.HandleFunc("/handshakeCPU", utils_kernel.RecibirHandshakeCPU)

	puerto := globals.KernelConfig.Port_kernel
	err := http.ListenAndServe(":"+strconv.Itoa(int(puerto)), mux)
	if err != nil {
		panic(err)
	}

}
