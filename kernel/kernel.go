package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	utils_estados "github.com/sisoputnfrba/tp-golang/kernel/utils/estados"
	utils_general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
	utils_lp "github.com/sisoputnfrba/tp-golang/kernel/utils/planificadores/planifLargo"
	utils_syscallController "github.com/sisoputnfrba/tp-golang/kernel/utils/syscallController"
	utils_logger "github.com/sisoputnfrba/tp-golang/utils/loggers"
)

func main() {

	prueba := os.Args[3]

	// CONFIG
	utils_logger.ConfigurarLogger("kernel.log")
	globals.KernelConfig = utils_general.IniciarConfiguracion(utils_logger.CONFIGS_DIRECTORY + "/" + prueba + "/Kernel.config")
	if globals.KernelConfig == nil {
		log.Fatal("No se pudo iniciar el config")
	}
	slog.SetLogLoggerLevel(utils_logger.Log_level_from_string(globals.KernelConfig.Log_level))
	// INIT

	go func() {
		ahora := time.Now()
		time.Sleep(time.Millisecond * 2000)
		slog.Warn(strconv.Itoa(int(time.Now().Sub(ahora).Milliseconds())))

	}()

	if len(os.Args) != 4 {
		log.Fatal("Uso: go run . archivo tamaño prueba")
	}

	archivo := os.Args[1]
	tamanioStr := os.Args[2]
	tamanioProceso, err := strconv.ParseInt(tamanioStr, 10, 64)

	if err != nil {
		log.Fatalf("Error al convertir el tamaño a int64: %v", err)
	}

	go utils_lp.IniciarPlanificadorLargoPlazo(archivo, tamanioProceso)

	// Cliente (mandar mensaje a memoria)
	mensaje := "Mensaje desde Kernel"
	utils_general.EnviarMensajeAMemoria(globals.KernelConfig.Ip_memory, globals.KernelConfig.Port_memory, mensaje)

	// Servidor (recibir mensaje de CPU y IO)
	mux := http.NewServeMux()

	mux.HandleFunc("/mensajeDeCpu", utils_general.RecibirMensajeDeCpu)
	mux.HandleFunc("/mensajeDeIo", utils_general.RecibirMensajeDeIo)
	mux.HandleFunc("/handshakeIO", utils_general.RecibirHandshakeIO)
	mux.HandleFunc("/handshakeCPU", utils_general.RecibirHandshakeCPU)

	mux.HandleFunc("/finalizacionIO", utils_estados.FinalizacionIO)
	mux.HandleFunc("/desconexionIO", utils_general.DesconexionIO)

	mux.HandleFunc("/syscallIO", utils_syscallController.RecibirIO)
	mux.HandleFunc("/syscallDUMP_MEMORY", utils_syscallController.RecibirDUMP_MEMORY)
	mux.HandleFunc("/syscallEXIT", utils_syscallController.RecibirEXIT)
	mux.HandleFunc("/syscallINIT_PROC", utils_syscallController.RecibirINIT_PROC)

	puerto := globals.KernelConfig.Port_kernel
	err = http.ListenAndServe(":"+strconv.Itoa(int(puerto)), mux)
	if err != nil {
		panic(err)
	}
}
