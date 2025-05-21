package main

import (
	"log"
	"log/slog"
	"net/http"
	"strconv"

	globals "github.com/sisoputnfrba/tp-golang/globals/memoria"
	utils_memoria "github.com/sisoputnfrba/tp-golang/memoria/utils"
	utils_logger "github.com/sisoputnfrba/tp-golang/utils/loggers"
)

func main() {

	utils_logger.ConfigurarLogger("memoria.log")

	globals.MemoriaConfig = utils_memoria.IniciarConfiguracion("config.json")
	if globals.MemoriaConfig == nil {
		log.Fatal("No se pudo iniciar el config")
	}

	slog.SetLogLoggerLevel(utils_logger.Log_level_from_string(globals.MemoriaConfig.Log_level))

	utils_memoria.InicializarMemoria()

	var instruccionesPrueba []string = []string{"HOLAA", "RAMPLAA", "PANSANSON BANANINI", "QEUW xAASDADF", "ASADDASASASAAAAAAAAAAA", "444abcdsdfsfffQQUEUEUEUEUEUEEUEUEUUEEUEUEUEUEUUEUEUEUEUUUEUEUEUEU"}

	utils_memoria.AlmacenarProceso(5, instruccionesPrueba)

	var primerInstruccion string = ""

	for i := 0; i < 5; i++ {
		primerInstruccion += string(globals.Memoria[i])
	}

	log.Printf("Primer instruccion: %s", primerInstruccion)

	log.Printf("Mock despues de guardar un proceso: %d", utils_memoria.CalcularMock())


	// Multiplexor de servidor HTTP
	mux := http.NewServeMux()

	// Maneja funciones segun URL de la solicitud
	mux.HandleFunc("/mensajeDeKernel", utils_memoria.RecibirMensajeDeKernel)
	mux.HandleFunc("/mensajeDeCpu", utils_memoria.RecibirMensajeDeCpu)

	// General¿?
	mux.HandleFunc("/consultarMock", utils_memoria.ConsultarMock)

	// KERNEL
	mux.HandleFunc("/iniciarProceso", utils_memoria.IniciarProceso)
	mux.HandleFunc("/suspenderProceso", utils_memoria.SuspenderProceso) // ya hice la funcion desde kernel en /kernel/utils/planifMedio (avisarSwappeo)
	mux.HandleFunc("/finalizarProceso", utils_memoria.FinalizarProceso)
	mux.HandleFunc("/memoryDump", utils_memoria.MemoryDump)

	// CPU
	// mux.HandleFunc("/obtenerMarcoProceso", utils_memoria.obtenerMarcoProceso)
	// mux.HandleFunc("/accederEspacioUsuario", utils_memoria.accederEspacioUsuario)
	// mux.HandleFunc("/leerPagina", utils_memoria.leerPagina)
	// mux.HandleFunc("/actualizarPagina", utils_memoria.actualizarPagina)


	// Inicia un servidor que escuche en el puerto del config
	puerto := globals.MemoriaConfig.Port_memory
	err := http.ListenAndServe(":"+strconv.Itoa(int(puerto)), mux)
	if err != nil {
		panic(err)
	}
}
