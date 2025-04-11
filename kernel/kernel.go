package main

import (
	"log"
	"log/slog"
	"net/http"
	"strconv"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	utils_kernel "github.com/sisoputnfrba/tp-golang/utils/kernel"
	utils_logger "github.com/sisoputnfrba/tp-golang/utils/loggers"
)

func main() {
	utils_logger.ConfigurarLogger("kernel.log")

	globals.KernelConfig = utils_kernel.IniciarConfiguracion("config.json")
	if globals.KernelConfig == nil {
		log.Fatal("No se pudo iniciar el config")
	}

	slog.SetLogLoggerLevel(utils_logger.Log_level_from_string(globals.KernelConfig.Log_level))

	slog.Info(globals.KernelConfig.Ip_memory)

	// Servidor (recibir mensaje de CPU)
	// Multiplexor de servidor HTTP
	// Examina la URL de cada solicitud HTTP y la dirige al controlador correspondiente
	mux := http.NewServeMux()

	// Maneja funciones segun URL de la solicitud
	mux.HandleFunc("/mensaje", utils_kernel.RecibirMensaje)

	// Inicia un servidor que escuche en el puerto del config
	puerto := globals.KernelConfig.Port_memory
	err := http.ListenAndServe(":"+strconv.Itoa(int(puerto)), mux)
	if err != nil {
		panic(err)
	}

	// Cliente (mandar mensaje a memoria)
	mensaje := "Hola desde kernel (cliente)"
	utils_kernel.EnviarMensaje(globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory, mensaje)

}
