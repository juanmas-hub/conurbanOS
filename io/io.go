package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

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
	globals.NombreIO = os.Args[1]

	utils_io.HandshakeAKernel(
		globals.IoConfig.IpKernel,
		globals.IoConfig.PortKernel,
		globals.NombreIO,
		globals.IoConfig.IpIO,
		globals.IoConfig.PortIO,
	)

	// Canal para recibir señales del sistema
	sigs := make(chan os.Signal, 1)

	// Notificar al canal si se recibe SIGINT o SIGTERM
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Goroutine que espera la señal
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println("Señal recibida:", sig)

		utils_io.Desconectar(globals.IoConfig.IpKernel, globals.IoConfig.PortKernel)
	}()

	// Servidor
	mux := http.NewServeMux()

	mux.HandleFunc("/mensajeDeKernel", utils_io.RecibirMensajeDeKernel)
	// Todavia no se usa
	mux.HandleFunc("/solicitudDeIo", utils_io.RecibirSolicitudDeKernel)

	puerto := globals.IoConfig.PortIO
	err := http.ListenAndServe(":"+strconv.Itoa(int(puerto)), mux)
	if err != nil {
		panic(err)
	}

}
