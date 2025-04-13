package main

import (
	"log"
	"log/slog"
	"net/http"
	"strconv"

	globals "github.com/sisoputnfrba/tp-golang/globals/io"
	utils_io "github.com/sisoputnfrba/tp-golang/io/utils"
	utils_logger "github.com/sisoputnfrba/tp-golang/utils/loggers"
)

func main() {
	utils_logger.ConfigurarLogger("io.log")

	globals.IoConfig = utils_io.IniciarConfiguracion("config.json")
	if globals.IoConfig == nil {
		log.Fatal("No se pudo iniciar el config")
	}

	slog.SetLogLoggerLevel(utils_logger.Log_level_from_string(globals.IoConfig.LogLevel))

	// Cliente (mando mensaje a kernel)
	mensaje := "Mensaje desde IO"
	utils_io.EnviarMensajeAKernel(globals.IoConfig.IpKernel, globals.IoConfig.PortKernel, mensaje)

	// Servidor
	// Cuando se ejecuta IO, hay que mandar a kernel su nombre, puerto e IP para que kernel se pueda conectar (no esta hecho)
	mux := http.NewServeMux()

	mux.HandleFunc("/mensajeDeKernel", utils_io.RecibirMensajeDeKernel)

	puerto := globals.IoConfig.PortIO
	err := http.ListenAndServe(":"+strconv.Itoa(int(puerto)), mux)
	if err != nil {
		panic(err)
	}

}
