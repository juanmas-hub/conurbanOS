package utils

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"

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

	slog.Debug(fmt.Sprint("Intentando acceder a la direccion: ", rutaArchivo))

	file, err := os.Open(rutaArchivo)
	if err != nil {
		slog.Debug(fmt.Sprint("No se pudo abrir el archivo: ", err))
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
			slog.Debug(fmt.Sprintf("Se extrajo la instruccion: %s", linea))
		}
	}
	if err := scanner.Err(); err != nil {
		slog.Debug(fmt.Sprintf("Error al extraer las instrucciones del archivo"))
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
		slog.Debug(fmt.Sprintf("Métrica desconocida: %s\n", metrica))
	}
	globals_memoria.MetricasMap[pid] = metricas
}

func logTablaDePaginas(pid int) {
	proceso, ok := globals_memoria.Procesos[pid]
	if !ok {
		slog.Debug(fmt.Sprintf("❌ Proceso con PID %d no encontrado.", pid))
		return
	}

	slog.Debug(fmt.Sprintf("-------- Tabla de páginas del proceso %d:", pid))
	recorrerYLoguearTabla(&proceso.TablaDePaginas, 0, "")
}

func recorrerYLoguearTabla(tabla *globals_memoria.TablaPaginas, nivel int, prefijo string) {
	for i, entrada := range tabla.Entradas {
		indent := strings.Repeat("  ", nivel) // dos espacios por nivel

		slog.Debug(fmt.Sprintf(
			"%s[%d] Nivel %d - Página: %d | Frame: %d | SiguienteNivel: %v",
			indent,
			i,
			nivel,
			entrada.NumeroDePagina,
			entrada.NumeroDeFrame,
			entrada.SiguienteNivel != nil,
		))

		if entrada.SiguienteNivel != nil {
			recorrerYLoguearTabla(entrada.SiguienteNivel, nivel+1, prefijo+"  ")
		}
	}
}
