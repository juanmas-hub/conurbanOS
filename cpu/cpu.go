package main

import (
	"fmt"
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
		log.Fatal("No se pudo iniciar el config de cpu")
	}
	globals_cpu.MemoriaConfig = utils_cpu.IniciarConfiguracionMemoria(utils_logger.CONFIGS_DIRECTORY + "/" + prueba + "/Memoria.config")
	if globals_cpu.MemoriaConfig == nil {
		log.Fatal("No se pudo iniciar el config de memoria")
	}

	slog.SetLogLoggerLevel(utils_logger.Log_level_from_string(globals_cpu.CpuConfig.Log_level))

	slog.Debug(globals_cpu.CpuConfig.Log_level)

	utils_cpu.HandshakeAKernel(
		globals_cpu.CpuConfig.Ip_kernel,
		globals_cpu.CpuConfig.Port_kernel,
		nombreCPU,
		globals_cpu.CpuConfig.Ip_cpu,
		globals_cpu.CpuConfig.Port_cpu,
	)

	if globals.CpuConfig.Cache_entries > 0 {
		utils_cpu.NuevaCache(globals.CpuConfig.Cache_entries, globals.CpuConfig.Cache_replacement)
	}

	if globals.CpuConfig.Tlb_entries > 0 {
		utils_cpu.NuevaTLB(globals.CpuConfig.Tlb_entries, globals.CpuConfig.Tlb_replacement)
	}

	go func() {
		for {
			slog.Debug(fmt.Sprintf("hola"))
			utils_cpu.Wait(globals.Sem)
			for pcb := range utils_cpu.ColaDeEjecucion {
				slog.Debug(fmt.Sprintf("Ejecutando PID %d en PC %d", pcb.Pid, pcb.PC))
				instruccion, err := utils_cpu.EnviarSolicitudInstruccion(pcb.Pid, pcb.PC) //solicitamos instruccion a memoria pasandole el pcb y pc

				slog.Info(fmt.Sprintf("## PID: %d - FETCH - Program Counter: %d", pcb.Pid, pcb.PC))

				if err != nil {
					slog.Debug(fmt.Sprintf("Error al pedir instrucción: %s", err))
					continue
				}
				slog.Debug(fmt.Sprintf("Instrucción: %s", instruccion))

				instruccionDeco, err := utils_cpu.Decode(instruccion) //decodificamos la instruccion
				if err != nil {
					slog.Debug(fmt.Sprintf("Error al decodificar instrucción: %s", err))
					continue
				}
				slog.Debug(fmt.Sprintf("Instrucción decodificada correctamente: %+v", instruccionDeco))

				slog.Info(fmt.Sprintf("## PID: %d- Ejecutando: %s - %v", pcb.Pid, instruccionDeco.Nombre, instruccionDeco.Parametros))

				resultadoEjecucion, err := utils_cpu.Execute(instruccionDeco, &pcb) //ejecutamos instruccion

				if err != nil {
					slog.Debug(fmt.Sprintf("Error al ejecutar instruccion: %s", err))
					continue
				}

				slog.Debug(fmt.Sprintf("Finalizado: nuevo PC = %d", pcb.PC))

				// Check interrupts
				if globals.HayInterrupcion {
					slog.Debug(fmt.Sprintf("Hay interrupcion"))
					globals.PC_Interrupcion = pcb.PC
					slog.Debug(fmt.Sprintf("PC: %d", globals.PC_Interrupcion))
					utils_cpu.Signal(globals.Sem_Interrupcion)
					slog.Debug(fmt.Sprintf("Señal enviada"))
					resultadoEjecucion = utils_cpu.PONERSE_ESPERA
				} else {
					slog.Debug(fmt.Sprintf("No hubo interrupcion"))
				}

				switch resultadoEjecucion {
				case utils_cpu.CONTINUAR_EJECUCION:
					utils_cpu.ColaDeEjecucion <- pcb
					continue // Volver al inicio del bucle para FETCH la siguiente instrucción del mismo PCB

				case utils_cpu.PONERSE_ESPERA:
					if globals.HayInterrupcion {

						if globals.EnvieSyscallBloqueante {
							slog.Debug(fmt.Sprintf("Proceso PID %d cede la CPU por Syscall: %s. PC actual: %d", pcb.Pid, instruccionDeco.Nombre, pcb.PC))
						} else {
							slog.Debug(fmt.Sprintf("Se cede la CPU con PID: (%d) por Interrupcion. PC actual: %d", pcb.Pid, pcb.PC))
						}
						globals.HayInterrupcion = false

					} else {
						slog.Debug(fmt.Sprintf("Proceso PID %d cede la CPU por Syscall: %s. PC actual: %d", pcb.Pid, instruccionDeco.Nombre, pcb.PC))

					}

					break // Salir del switch, para que espere un nuevo PCB

				case utils_cpu.ERROR_EJECUCION:

					break
				}
				globals.EnvieSyscallBloqueante = false
				//utils_cpu.EnviarPCBaKernel(pcb.Pid, pcb.Pid)
			}
		}
	}()

	// Servidor
	// Cuando se ejecuta CPU, hay que mandar a kernel su puerto e IP para que kernel se pueda conectar (no esta hecho)
	mux := http.NewServeMux()

	mux.HandleFunc("/dispatchProceso", utils_cpu.RecibirProcesoAEjecutar)
	mux.HandleFunc("/interrumpir", utils_cpu.RecibirInterrupcion)

	puerto := globals_cpu.CpuConfig.Port_cpu

	err := http.ListenAndServe(":"+strconv.Itoa(int(puerto)), mux)
	if err != nil {
		panic(err)
	}
}
