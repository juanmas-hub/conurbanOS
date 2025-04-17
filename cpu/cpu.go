package main

import (
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"os"

	utils_cpu "github.com/sisoputnfrba/tp-golang/cpu/utils"
	globals_cpu "github.com/sisoputnfrba/tp-golang/globals/cpu"
	utils_logger "github.com/sisoputnfrba/tp-golang/utils/loggers"
)

func main() {

	utils_logger.ConfigurarLogger("cpu.log")

	globals_cpu.CpuConfig = utils_cpu.IniciarConfiguracion("config.json")
	if globals_cpu.CpuConfig == nil {
		log.Fatal("No se pudo iniciar el config")
	}

	slog.SetLogLoggerLevel(utils_logger.Log_level_from_string(globals_cpu.CpuConfig.Log_level))

	slog.Info(globals_cpu.CpuConfig.Log_level)

	// Cliente (manda mensaje a kernel y memoria)
	mensaje := "Mensaje desde CPU"
	utils_cpu.EnviarMensaje(globals_cpu.CpuConfig.Ip_kernel, globals_cpu.CpuConfig.Port_kernel, mensaje)
	utils_cpu.EnviarMensaje(globals_cpu.CpuConfig.Ip_memory, globals_cpu.CpuConfig.Port_memory, mensaje)

	// Handshake al kernel
	if len(os.Args) != 2 {
		log.Fatal("No se paso como argumento el nombre de CPU") //por ej:  go run . nombreIO
	}
	nombreCPU := os.Args[1]

	utils_cpu.HandshakeAKernel(
		globals_cpu.CpuConfig.Ip_kernel,
		globals_cpu.CpuConfig.Port_kernel,
		nombreCPU,
		"127.0.0.1", // Esta es la IP que hay que mandarle a kernel? No se - tomytsa (yo juanma tampoco)
		globals_cpu.CpuConfig.Port_cpu,
	)

	// Ahora hay que recibir la petición del Kernel para que el modulo hago un usleep (no esta hecho)

	// Servidor
	// Cuando se ejecuta CPU, hay que mandar a kernel su puerto e IP para que kernel se pueda conectar (no esta hecho)
	mux := http.NewServeMux()

	mux.HandleFunc("/mensajeDeKernel", utils_cpu.RecibirMensajeDeKernel)

	puerto := globals_cpu.CpuConfig.Port_cpu
	err := http.ListenAndServe(":"+strconv.Itoa(int(puerto)), mux)
	if err != nil {
		panic(err)
	}
}
