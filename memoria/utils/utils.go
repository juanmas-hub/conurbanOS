package utils

import (
	//"encoding/json"
	//"fmt"
	"bufio"
	"log"

	//"net/http"
	"os"
	"strings"

	//globals "github.com/sisoputnfrba/tp-golang/globals/memoria"
	globals_memoria "github.com/sisoputnfrba/tp-golang/globals/memoria"
)

func CalcularMock() int {
	PAGE_SIZE := int(globals_memoria.MemoriaConfig.Page_size)
	libres := 0

	for _, estaOcupado := range globals_memoria.MemoriaMarcosOcupados {
		if !estaOcupado {
			libres++
		}
	}
	return libres * PAGE_SIZE
}

func abrirArchivo(filename string) *os.File {

	var rutaArchivo string = globals_memoria.MemoriaConfig.Scripts_path + filename

	log.Println("Intentando acceder a la direccion: ", rutaArchivo)

	file, err := os.Open(rutaArchivo)
	if err != nil {
		log.Println("No se pudo abrir el archivo: ", err)
		return nil
	}
	return file
}

func abrirArchivoBinario() *os.File {
	var ruta string = globals_memoria.MemoriaConfig.Swapfile_path

	archivo, err := os.OpenFile(ruta, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil
	}
	return archivo
}

func extraerInstrucciones(archivo *os.File) []string {
	var instrucciones []string
	scanner := bufio.NewScanner(archivo)
	for scanner.Scan() {
		linea := strings.TrimSpace(scanner.Text())
		if linea != "" {
			instrucciones = append(instrucciones, linea)
			log.Printf("Se extrajo la instruccion: %s", linea)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Println("Error al extraer las instrucciones del archivo")
		return nil
	}

	return instrucciones
}

func ObtenerInstruccionesDesdeArchivo(filename string) []string {

	var archivo *os.File = abrirArchivo(filename)
	if archivo == nil {
		return nil
	}

	var instrucciones []string = extraerInstrucciones(archivo)

	return instrucciones
}

/*
func verificarPIDUnico(pid int) int {
	if _, existe := (*globals_memoria.Tablas)[pid]; existe {
		return 1
	}
	return 0
}*/

func verificarPIDUnico(pid int) int {
	_, existeEntrada := globals_memoria.Procesos[pid]
	if !existeEntrada {
		return 0
	}
	return -1
}

/*
func IncrementarMetrica(metrica string, pid int, cantidad int) {
	switch metrica {
	case "ACCESOS_TABLAS":
		(*globals_memoria.Metricas)[pid].AccesosTablas += cantidad
	case "INSTRUCCIONES_SOLICITADAS":
		(*globals_memoria.Metricas)[pid].InstruccionesSolicitadas += cantidad
	case "BAJADAS_SWAP":
		(*globals_memoria.Metricas)[pid].BajadasSwap += cantidad
	case "SUBIDAS_MEMORIA":
		(*globals_memoria.Metricas)[pid].SubidasMemoria += cantidad
	case "LECTURAS_MEMORIA":
		(*globals_memoria.Metricas)[pid].LecturasMemoria += cantidad
	case "ESCRITURAS_MEMORIA":
		(*globals_memoria.Metricas)[pid].EscriturasMemoria += cantidad
	default:
		log.Printf("Métrica desconocida: %s\n", metrica)
	}
}
*/

func IncrementarMetrica(metrica string, pid int, cantidad int) {
	metricas := globals_memoria.MetricasMap[pid]
	switch metrica {
	case "ACCESOS_TABLAS":
		metricas.AccesosTablas += cantidad
	case "INSTRUCCIONES_SOLICITADAS":
		metricas.InstruccionesSolicitadas += cantidad
	case "BAJADAS_SWAP":
		metricas.BajadasSwap += cantidad
	case "SUBIDAS_MEMORIA":
		metricas.SubidasMemoria += cantidad
	case "LECTURAS_MEMORIA":
		metricas.LecturasMemoria += cantidad
	case "ESCRITURAS_MEMORIA":
		metricas.EscriturasMemoria += cantidad
	default:
		log.Printf("Métrica desconocida: %s\n", metrica)
	}
	globals_memoria.MetricasMap[pid] = metricas
}
