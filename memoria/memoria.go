package main

import (
	"log"
	"log/slog"
	"net/http"
	"strconv"

	globals "github.com/sisoputnfrba/tp-golang/globals/memoria"
	utils_logger "github.com/sisoputnfrba/tp-golang/utils/loggers"
	utils_memoria "github.com/sisoputnfrba/tp-golang/utils/memoria"
)

func main() {

	utils_logger.ConfigurarLogger("memoria.log")

	globals.MemoriaConfig = utils_memoria.IniciarConfiguracion("config.json")
	if globals.MemoriaConfig == nil {
		log.Fatal("No se pudo iniciar el config")
	}

	slog.SetLogLoggerLevel(utils_logger.Log_level_from_string(globals.MemoriaConfig.Log_level))

	slog.Info(globals.MemoriaConfig.Log_level)

	// Servidor
	// Multiplexor de servidor HTTP
	// Examina la URL de cada solicitud HTTP y la dirige al controlador correspondiente
	mux := http.NewServeMux()

	// Maneja funciones segun URL de la solicitud
	mux.HandleFunc("/mensaje", utils_memoria.RecibirMensaje)

	// Inicia un servidor que escuche en el puerto del config
	puerto := globals.MemoriaConfig.Port_memory
	err := http.ListenAndServe(":"+strconv.Itoa(int(puerto)), mux)
	if err != nil {
		panic(err)
	}
}
