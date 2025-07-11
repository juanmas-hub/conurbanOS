package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	utils_general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
	handlers "github.com/sisoputnfrba/tp-golang/kernel/utils/handlers"
	planificadores "github.com/sisoputnfrba/tp-golang/kernel/utils/planificadores"
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

	if len(os.Args) != 4 {
		log.Fatal("Uso: go run . archivo tamaño prueba")
	}

	slog.Debug("    ")
	slog.Debug("    ")
	slog.Debug("    ")

	archivo := os.Args[1]
	tamanioStr := os.Args[2]
	tamanioProceso, err := strconv.ParseInt(tamanioStr, 10, 64)

	if err != nil {
		log.Fatalf("Error al convertir el tamaño a int64: %v", err)
	}

	go planificadores.IniciarPlanificadorLargoPlazo(archivo, tamanioProceso)

	// Servidor (recibir mensaje de CPU y IO)
	mux := http.NewServeMux()

	mux.HandleFunc("/handshakeIO", handlers.RecibirHandshakeIO)
	mux.HandleFunc("/handshakeCPU", handlers.RecibirHandshakeCPU)

	mux.HandleFunc("/finalizacionIO", handlers.FinalizacionIO)
	mux.HandleFunc("/desconexionIO", handlers.DesconexionIO)

	mux.HandleFunc("/syscallIO", handlers.RecibirIO)
	mux.HandleFunc("/syscallDUMP_MEMORY", handlers.RecibirDUMP_MEMORY)
	mux.HandleFunc("/syscallEXIT", handlers.RecibirEXIT)
	mux.HandleFunc("/syscallINIT_PROC", handlers.RecibirINIT_PROC)

	puerto := globals.KernelConfig.Port_kernel
	err = http.ListenAndServe(":"+strconv.Itoa(int(puerto)), mux)
	if err != nil {
		panic(err)
	}
}
