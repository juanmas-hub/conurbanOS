package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	globals "github.com/sisoputnfrba/tp-golang/globals/io"
	utils_io "github.com/sisoputnfrba/tp-golang/io/utils"
	utils_logger "github.com/sisoputnfrba/tp-golang/utils/loggers"
)

func main() {

	// Configuración
	utils_logger.ConfigurarLogger("io.log")
	globals.IoConfig = utils_io.IniciarConfiguracion("config.json")
	if globals.IoConfig == nil {
		log.Fatal("No se pudo iniciar el config")
	}

	slog.SetLogLoggerLevel(utils_logger.Log_level_from_string(globals.IoConfig.LogLevel))

	// Cliente (mando mensaje a kernel)
	mensaje := "Mensaje desde IO"
	utils_io.EnviarMensajeAKernel(globals.IoConfig.IpKernel, globals.IoConfig.PortKernel, mensaje)

	// Handshake al kernel
	if len(os.Args) != 2 {
		log.Fatal("No se paso como argumento el nombre de IO") //por ej:  go run . nombreIO
	}
	nombreIO := os.Args[1]

	utils_io.HandshakeAKernel(
		globals.IoConfig.IpKernel,
		globals.IoConfig.PortKernel,
		nombreIO,
		"127.0.0.1", // Esta es la IP que hay que mandarle a kernel? No se - tomytsa
		globals.IoConfig.PortIO,
	)

	// Ahora hay que recibir la petición del Kernel para que el modulo hago un usleep (no esta hecho)

	// Servidor
	mux := http.NewServeMux()

	mux.HandleFunc("/mensajeDeKernel", utils_io.RecibirMensajeDeKernel)

	puerto := globals.IoConfig.PortIO
	err := http.ListenAndServe(":"+strconv.Itoa(int(puerto)), mux)
	if err != nil {
		panic(err)
	}

}
