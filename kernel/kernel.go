package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/kernel"
	utils_general "github.com/sisoputnfrba/tp-golang/kernel/utils/general"
	utils_cp "github.com/sisoputnfrba/tp-golang/kernel/utils/planifCorto"
	utils_lp "github.com/sisoputnfrba/tp-golang/kernel/utils/planifLargo"
	utils_syscallController "github.com/sisoputnfrba/tp-golang/kernel/utils/syscallController"
	utils_logger "github.com/sisoputnfrba/tp-golang/utils/loggers"
)

func main() {

	// CONFIG
	utils_logger.ConfigurarLogger("kernel.log")
	globals.KernelConfig = utils_general.IniciarConfiguracion("config.json")
	if globals.KernelConfig == nil {
		log.Fatal("No se pudo iniciar el config")
	}
	slog.SetLogLoggerLevel(utils_logger.Log_level_from_string(globals.KernelConfig.Log_level))

	// INIT

	if len(os.Args) != 3 {
		log.Fatal("Uso: go run . archivo tamaño")
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

	// Prueba IO
	go func() {
		time.Sleep(10 * time.Second)
		if len(globals.ListaIOs) > 0 {
			io := globals.ListaIOs[0].Handshake
			ipIO := io.IP
			puertoIO := io.Puerto
			pid := int64(1)
			tiempo := int64(5000)
			syscallIO := globals.SyscallIO{
				Nombre: "TECLADO",
				Tiempo: tiempo,
				PID:    pid,
			}

			globals.ListaIOs[0].PidProcesoActual = pid
			globals.ListaIOs[0].ColaProcesosEsperando = append(globals.ListaIOs[0].ColaProcesosEsperando, syscallIO)
			utils_general.EnviarSolicitudIO(ipIO, puertoIO, pid, tiempo)
		} else {
			log.Println("No hay IOs registrados todavía")
		}
	}()

	// Servidor (recibir mensaje de CPU y IO)
	mux := http.NewServeMux()

	mux.HandleFunc("/mensajeDeCpu", utils_general.RecibirMensajeDeCpu)
	mux.HandleFunc("/mensajeDeIo", utils_general.RecibirMensajeDeIo)
	mux.HandleFunc("/handshakeIO", utils_general.RecibirHandshakeIO)
	mux.HandleFunc("/handshakeCPU", utils_general.RecibirHandshakeCPU)

	mux.HandleFunc("/finalizacionIO", utils_general.FinalizacionIO)
	mux.HandleFunc("/desconexionIO", utils_general.DesconexionIO)

	mux.HandleFunc("/devolucionProceso", utils_cp.DevolucionProceso)

	mux.HandleFunc("/syscallIO", utils_syscallController.ManejarIO)
	mux.HandleFunc("/syscallDUMP_MEMORY", utils_syscallController.ManejarDUMP_MEMORY)
	mux.HandleFunc("/syscallEXIT", utils_syscallController.ManejarEXIT)
	mux.HandleFunc("/syscallINIT_PROC", utils_syscallController.ManejarINIT_PROC)

	puerto := globals.KernelConfig.Port_kernel
	err = http.ListenAndServe(":"+strconv.Itoa(int(puerto)), mux)
	if err != nil {
		panic(err)
	}
}
