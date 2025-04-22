package main

import (
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"time"

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

	// Solicitud de IO (Es aproximadamente lo que un proceso usaria para solicitar a IO)
	/*Esto es para probar si funciona IO - espera 10 segundos (a que haga el handshake) y envia solicitud
	Problema: estoy mandando solicitud y espero la respuesta, bloqueando todo el modulo kernel
		- creo que el enunciado dice que:
			1. envio solicitud a IO (una API)
			2. recibo fin de IO (otra API)
		- si se puede hacer todo en una sola API (enviar solicitud, y esperar la respuesta):
			- Solucion (creo): que cada proceso sea un hilo, entonces en cada solicitud a IO
							   podes bloquear ese hilo tranqui, que el kernel sigue funcionando

	*/
	go func() {
		time.Sleep(10 * time.Second)
		if len(globals.HandshakesIO) > 0 {
			io := globals.HandshakesIO[0]
			ipIO := io.IP
			puertoIO := io.Puerto
			pid := int64(1)
			tiempo := int64(5000)

			utils_kernel.EnviarSolicitudIO(ipIO, puertoIO, pid, tiempo)
		} else {
			log.Println("No hay IOs registrados todavía")
		}
	}()

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
