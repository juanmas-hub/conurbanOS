package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/io"
	utils_io "github.com/sisoputnfrba/tp-golang/io/utils"
	utils_logger "github.com/sisoputnfrba/tp-golang/utils/loggers"
)

func main() {

	// Configuración
	utils_logger.ConfigurarLogger("io.log")
	globals.IoConfig = utils_io.IniciarConfiguracion("io.config")
	if globals.IoConfig == nil {
		log.Fatal("No se pudo iniciar el config")
	}

	slog.SetLogLoggerLevel(utils_logger.Log_level_from_string(globals.IoConfig.LogLevel))

	// Canal para indicar que hemos terminado
	done := make(chan bool, 1)

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

	/* Goroutine que espera la señal
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println("Señal recibida:", sig)
		utils_io.Desconectar(globals.IoConfig.IpKernel, globals.IoConfig.PortKernel, globals.PidProcesoActual)
		done <- true
	}()*/

	// Servidor
	mux := http.NewServeMux()

	mux.HandleFunc("/solicitudDeIo", utils_io.RecibirSolicitudDeKernel)

	puerto := globals.IoConfig.PortIO
	srv := &http.Server{
		Addr:    ":" + strconv.Itoa(int(puerto)),
		Handler: mux,
	}

	// Ejecutar servidor en goroutine (no bloquea main)
	go func() {
		fmt.Println("Servidor escuchando en puerto", puerto)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error en servidor: %v\n", err)
		}
	}()

	// Goroutine para manejar señales
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println("Señal recibida:", sig)

		utils_io.Desconectar(globals.IoConfig.IpKernel, globals.IoConfig.PortKernel, globals.PidProcesoActual)

		// Shutdown con contexto para que ListenAndServe termine
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Error al apagar servidor: %v\n", err)
		} else {
			fmt.Println("Servidor apagado correctamente")
		}

		done <- true
	}()
	/*
		puerto := globals.IoConfig.PortIO
		err := http.ListenAndServe(":"+strconv.Itoa(int(puerto)), mux)
		if err != nil {
			panic(err)
		}
		<-done
	*/

	<-done
}
