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
	// Handshake al kernel
	if len(os.Args) != 3 {
		log.Println("Error. El formato es: nombreInstanciaIO, prueba")
	}
	globals.NombreInstancia = os.Args[1]
	prueba := os.Args[2]

	// Configuración
	utils_logger.ConfigurarLogger(globals.NombreInstancia + ".log")
	//slog.Debug(fmt.Sprint(utils_logger.CONFIGS_DIRECTORY + "/" + prueba + "/" + globals.NombreInstancia + ".config"))
	globals.IoConfig = utils_io.IniciarConfiguracion(utils_logger.CONFIGS_DIRECTORY + "/" + prueba + "/" + globals.NombreInstancia + ".config")
	if globals.IoConfig == nil {
		log.Println("No se pudo iniciar el config")
	}

	slog.SetLogLoggerLevel(utils_logger.Log_level_from_string(globals.IoConfig.LogLevel))

	// Canal para indicar que hemos terminado
	done := make(chan bool, 1)

	utils_io.HandshakeAKernel(
		globals.IoConfig.IpKernel,
		globals.IoConfig.PortKernel,
		globals.IoConfig.NombreIO,
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
		slog.Debug(fmt.Sprint("Servidor escuchando en puerto", puerto))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Debug(fmt.Sprintf("Error en servidor: %v\n", err))
		}
	}()

	// Goroutine para manejar señales
	go func() {
		sig := <-sigs
		//fmt.Println()
		slog.Debug(fmt.Sprint("Señal recibida:", sig))

		utils_io.Desconectar(globals.IoConfig.IpKernel, globals.IoConfig.PortKernel, globals.PidProcesoActual)

		// Shutdown con contexto para que ListenAndServe termine
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			slog.Debug(fmt.Sprintf("Error al apagar servidor: %v\n", err))
		} else {
			slog.Debug(fmt.Sprintf("Servidor apagado correctamente"))
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
