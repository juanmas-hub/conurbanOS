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

	// Multiplexor de servidor HTTP
	mux := http.NewServeMux()

	// Maneja funciones segun URL de la solicitud
	mux.HandleFunc("/mensajeDeKernel", utils_memoria.RecibirMensajeDeKernel)
	mux.HandleFunc("/mensajeDeCpu", utils_memoria.RecibirMensajeDeCpu)

	// General¿?
	mux.HandleFunc("/consultarMock", utils_memoria.ConsultarMock)

	// KERNEL
	mux.HandleFunc("/iniciarProceso", utils_memoria.IniciarProceso)
	mux.HandleFunc("/reanudarproceso", utils_memoria.ReanudarProceso) 
	mux.HandleFunc("/suspenderProceso", utils_memoria.SuspenderProceso)
	mux.HandleFunc("/finalizarProceso", utils_memoria.FinalizarProceso)
	mux.HandleFunc("/memoryDump", utils_memoria.MemoryDump)

	// CPU
	mux.HandleFunc("/obtenerMarcoProceso", utils_memoria.ObtenerMarcoProceso)
	mux.HandleFunc("/accederEspacioUsuarioLectura", utils_memoria.AccederEspacioUsuarioLectura)
	mux.HandleFunc("/accederEspacioUsuarioEscritura", utils_memoria.AccederEspacioUsuarioEscritura)
	mux.HandleFunc("/leerPagina", utils_memoria.LeerPagina)
	mux.HandleFunc("/actualizarPagina", utils_memoria.ActualizarPagina)
	mux.HandleFunc("/obtenerInstruccion", utils_memoria.EnviarInstruccion)
	mux.HandleFunc("actualizarTablaDePaginas", utils_memoria.ActualizarTablaDePaginas)

	// Inicia un servidor que escuche en el puerto del config
	puerto := globals.MemoriaConfig.Port_memory
	err := http.ListenAndServe(":"+strconv.Itoa(int(puerto)), mux)
	if err != nil {
		panic(err)
	}
}
