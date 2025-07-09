package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	utils_cpu "github.com/sisoputnfrba/tp-golang/cpu/utils"
	globals "github.com/sisoputnfrba/tp-golang/globals/cpu"
	globals_cpu "github.com/sisoputnfrba/tp-golang/globals/cpu"
	utils_logger "github.com/sisoputnfrba/tp-golang/utils/loggers"
)

func main() {

	if len(os.Args) != 3 {
		log.Fatal("No se paso como argumento el nombre de CPU") //por ej:  go run . nombreIO
	}
	nombreCPU := os.Args[1]
	prueba := os.Args[2]

	utils_logger.ConfigurarLogger("cpu.log")

	globals_cpu.CpuConfig = utils_cpu.IniciarConfiguracion(utils_logger.CONFIGS_DIRECTORY + "/" + prueba + "/" + nombreCPU + ".config")
	if globals_cpu.CpuConfig == nil {
		log.Fatal("No se pudo iniciar el config")
	}

	slog.SetLogLoggerLevel(utils_logger.Log_level_from_string(globals_cpu.CpuConfig.Log_level))

	slog.Info(globals_cpu.CpuConfig.Log_level)

	utils_cpu.HandshakeAKernel(
		globals_cpu.CpuConfig.Ip_kernel,
		globals_cpu.CpuConfig.Port_kernel,
		nombreCPU,
		globals_cpu.CpuConfig.Ip_cpu,
		globals_cpu.CpuConfig.Port_cpu,
	)

	go func() {
		for {
			log.Println("hola")
			utils_cpu.Wait(globals.Sem)
			for pcb := range utils_cpu.ColaDeEjecucion {
				log.Printf("Ejecutando PID %d en PC %d", pcb.Pid, pcb.PC)
				instruccion, err := utils_cpu.EnviarSolicitudInstruccion(pcb.Pid, pcb.PC) //solicitamos instruccion a memoria pasandole el pcb y pc
				if err != nil {
					log.Printf("Error al pedir instrucción: %s", err)
					continue
				}
				log.Printf("Instrucción: %s", instruccion)

				instruccionDeco, err := utils_cpu.Decode(instruccion) //decodificamos la instruccion
				if err != nil {
					log.Printf("Error al decodificar instrucción: %s", err)
					continue
				}
				log.Printf("Instrucción decodificada correctamente: %+v", instruccionDeco)

				resultadoEjecucion, err := utils_cpu.Execute(instruccionDeco, &pcb) //ejecutamos instruccion

				if err != nil {
					log.Printf("Error al ejecutar instruccion: %s", err)
					continue
				}
				log.Printf("Finalizado: nuevo PC = %d", pcb.PC)

				switch resultadoEjecucion {
				case utils_cpu.CONTINUAR_EJECUCION:
					utils_cpu.ColaDeEjecucion <- pcb
					continue // Volver al inicio del bucle para FETCH la siguiente instrucción del mismo PCB

				case utils_cpu.PONERSE_ESPERA:
					log.Printf("Proceso PID %d cede la CPU por Syscall: %s. PC actual: %d", pcb.Pid, instruccionDeco.Nombre, pcb.PC)

					//pcb a kernel

					break // Salir del switch, para que espere un nuevo PCB

				case utils_cpu.ERROR_EJECUCION:

					break
				}

			}
		}
	}()

	// Servidor
	// Cuando se ejecuta CPU, hay que mandar a kernel su puerto e IP para que kernel se pueda conectar (no esta hecho)
	mux := http.NewServeMux()

	mux.HandleFunc("/dispatchProceso", utils_cpu.RecibirProcesoAEjecutar)
	mux.HandleFunc("/recibirPCB", utils_cpu.RecibirPCBDeKernel)
	//mux.HandleFunc("/interrumpir", utils_cpu.Interrupcion)

	puerto := globals_cpu.CpuConfig.Port_cpu

	err := http.ListenAndServe(":"+strconv.Itoa(int(puerto)), mux)
	if err != nil {
		panic(err)
	}
}
