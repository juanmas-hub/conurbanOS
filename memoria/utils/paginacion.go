package utils

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	globals "github.com/sisoputnfrba/tp-golang/globals/memoria"
	globals_memoria "github.com/sisoputnfrba/tp-golang/globals/memoria"
)

func leer(direccion int, tamanio int) string {
	var leido string = ""

	globals_memoria.MemoriaMutex.Lock()
	for i := 0; i < tamanio; i++ {
		leido += string(globals_memoria.Memoria[direccion+i])
	}
	globals_memoria.MemoriaMutex.Unlock()
	return leido
}

func escribir(direccion int, dato string) int {

	var tamanioDePagina int = int(globals_memoria.MemoriaConfig.Page_size)
	var offset int = direccion % tamanioDePagina

	if len(dato) > (tamanioDePagina - offset) {
		slog.Debug(fmt.Sprintf("El dato no se pudo escribir porque excede la pagina"))
		return -1
	}

	globals_memoria.MemoriaMutex.Lock()
	for i := 0; i < len(dato); i++ {
		globals_memoria.Memoria[direccion+i] = dato[i]
	}
	globals_memoria.MemoriaMutex.Unlock()
	return 0
}

func escribirPaginas(pid int, paginas []globals_memoria.Pagina, marcos []int) *[]globals_memoria.PaginaLinkeada {

	var paginasLinkeadas []globals_memoria.PaginaLinkeada

	var direccion int
	for i := 0; i < len(marcos); i++ {
		direccion = marcos[i] * int(globals_memoria.MemoriaConfig.Page_size)

		if escribir(direccion, string(paginas[i].Contenido)) != 0 {
			slog.Debug(fmt.Sprint("No se pudo escribir la pagina"))
			return nil
		}

		paginaLinkeada := globals_memoria.PaginaLinkeada{
			NumeroDePagina: paginas[i].NumeroDePagina,
			NumeroDeFrame:  marcos[i],
		}
		paginasLinkeadas = append(paginasLinkeadas, paginaLinkeada)

		slog.Debug(fmt.Sprint(paginas[i]))
		globals_memoria.MemoriaMarcosOcupadosMutex.Lock()
		globals_memoria.MemoriaMarcosOcupados[marcos[i]] = true
		globals_memoria.MemoriaMarcosOcupadosMutex.Unlock()

		IncrementarMetrica("ESCRITURAS_MEMORIA", pid, 1)
	}

	return &paginasLinkeadas
}

func actualizarPagina(indicePagina int, dato string) {
	globals_memoria.MemoriaMutex.Lock()
	for i := 0; i < len(dato); i++ {
		globals_memoria.Memoria[indicePagina+i] = dato[i]
	}
	globals_memoria.MemoriaMutex.Unlock()
}

func crearTabla(pid int, framesAsignados []int) globals_memoria.TablaPaginas {
	niveles := int(globals_memoria.MemoriaConfig.Number_of_levels)
	entradasPorPagina := int(globals_memoria.MemoriaConfig.Entries_per_page)

	frameIndex := 0

	return construirNivel(pid, niveles, entradasPorPagina, &frameIndex, framesAsignados)
}

func construirNivel(pid int, nivel int, entradasPorPagina int, frameIndex *int, framesAsignados []int) globals_memoria.TablaPaginas {
	cantPaginasRestantes := len(framesAsignados) - *frameIndex
	
	cantEntradas := (cantPaginasRestantes + entradasPorPagina - 1) / entradasPorPagina
	if nivel == 1 {
		if cantPaginasRestantes > entradasPorPagina {
			cantEntradas = entradasPorPagina
		} else {
			cantEntradas = cantPaginasRestantes
		}
	}

	tabla := globals_memoria.TablaPaginas{
		Entradas: make([]globals_memoria.EntradaTP, 0, cantEntradas),
	}

	for i := 0; i < cantEntradas; i++ {
		if nivel == 1 {
			// Nivel más bajo: asignar página y frame real
			if *frameIndex < len(framesAsignados) {
				entrada := globals_memoria.EntradaTP{
					NumeroDePagina: *frameIndex,
					NumeroDeFrame:  framesAsignados[*frameIndex],
					SiguienteNivel: nil,
				}
				tabla.Entradas = append(tabla.Entradas, entrada)
				*frameIndex++
			} else {
				fmt.Printf("No hay suficientes frames asignados para el PID %d\n", pid)
			}
		} else {
			// Nivel intermedio: crear subtabla
			subtabla := construirNivel(pid, nivel-1, entradasPorPagina, frameIndex, framesAsignados)
			entrada := globals_memoria.EntradaTP{
				NumeroDePagina: -1,
				NumeroDeFrame:  -1,
				SiguienteNivel: &subtabla,
			}
			tabla.Entradas = append(tabla.Entradas, entrada)
		}
	}

	//IncrementarMetrica("ACCESOS_TABLAS", pid, 1)
	return tabla
}

func buscarMarcosDisponibles(cantidad int) []int {
	var result []int = make([]int, 0, cantidad)

	globals_memoria.MemoriaMarcosOcupadosMutex.Lock()

	for i := 0; i < len(globals_memoria.MemoriaMarcosOcupados); i++ {
		if !globals_memoria.MemoriaMarcosOcupados[i] {
			result = append(result, i)
			if len(result) >= cantidad {
				globals_memoria.MemoriaMarcosOcupadosMutex.Unlock()
				return result
			}
		}
	}
	globals_memoria.MemoriaMarcosOcupadosMutex.Unlock()

	slog.Debug(fmt.Sprintf("no hay suficientes marcos libres: se encontraron %d de %d", len(result), cantidad))
	return nil
}

func actualizarTablaPaginas(pid int, paginasLinkeadas []globals_memoria.PaginaLinkeada) {
	globals_memoria.ProcesosMutex[pid].Lock()
	proceso := globals_memoria.Procesos[pid]

	// Crea un mapa para acceso rápido por número de página
	paginasMap := make(map[int]globals_memoria.PaginaLinkeada)
	for _, pl := range paginasLinkeadas {
		paginasMap[pl.NumeroDePagina] = pl
	}

	var recorrerYActualizar func(tabla *globals_memoria.TablaPaginas)
	recorrerYActualizar = func(tabla *globals_memoria.TablaPaginas) {
		IncrementarMetrica("ACCESOS_TABLAS", pid, 1)
		for i := 0; i < len(tabla.Entradas); i++ {
			entrada := &tabla.Entradas[i]

			if entrada.SiguienteNivel != nil {
				recorrerYActualizar(entrada.SiguienteNivel)
			} else {
				pl, existe := paginasMap[entrada.NumeroDePagina]
				if !existe {
					continue
				}

				// Actualizar frame y marcarlo como ocupado
				entrada.NumeroDeFrame = pl.NumeroDeFrame
				globals_memoria.MemoriaMarcosOcupados[pl.NumeroDeFrame] = true
			}
		}
	}

	recorrerYActualizar(&proceso.TablaDePaginas)

	globals_memoria.Procesos[pid] = proceso
	globals_memoria.ProcesosMutex[pid].Unlock()

	slog.Debug(fmt.Sprint("Tabla de paginas actualizada del proceso: ", pid))
	logTablaDePaginas(pid)

}

func asignarFramesAProceso(numerosDeFrame []int) {

	globals_memoria.MemoriaMarcosOcupadosMutex.Lock()
	for _, frame := range numerosDeFrame {
		globals_memoria.MemoriaMarcosOcupados[frame] = true
	}
	globals_memoria.MemoriaMarcosOcupadosMutex.Unlock()
}

func AlmacenarProceso(pid int, tamanio int, filename string) int {
	var pageSize int
	var indicesNecesarios int
	var indicesDisponibles []int

	slog.Debug(fmt.Sprint("Tamaño: ", tamanio))
	pageSize = int(globals_memoria.MemoriaConfig.Page_size)
	indicesNecesarios = (tamanio + pageSize - 1) / pageSize
	slog.Debug(fmt.Sprintf("Resultado del cálculo: (%d + %d - 1) / %d = %d", tamanio, pageSize, pageSize, indicesNecesarios))
	slog.Debug(fmt.Sprint("Indices necesarios: ", indicesNecesarios))
	indicesDisponibles = buscarMarcosDisponibles(indicesNecesarios)
	slog.Debug(fmt.Sprint("Indices disponibles: ", indicesDisponibles))

	if indicesDisponibles == nil {
		slog.Debug(fmt.Sprintf("Error no hay suficiente espacio para almacenar el proceso %d", pid))
		return -1
	}

	var instrucciones []string
	var tabla globals_memoria.TablaPaginas

	if indicesNecesarios != 0 {
		// Asigno los frames en la memoria
		asignarFramesAProceso(indicesDisponibles)

		tabla = crearTabla(pid, indicesDisponibles)
	} else {
		tabla = globals_memoria.TablaPaginas{
			Entradas: []globals_memoria.EntradaTP{},
		}
	}

	instrucciones = ObtenerInstruccionesDesdeArchivo(filename)

	proceso := globals_memoria.Proceso{
		Pseudocodigo:      instrucciones,
		Suspendido:        false,
		TablaDePaginas:    tabla,
		CantidadDePaginas: len(indicesDisponibles),
	}

	m, ok := globals_memoria.ProcesosMutex[pid]
	if !ok {
		m = &sync.Mutex{}
		globals_memoria.ProcesosMutex[pid] = m
	}
	globals_memoria.ProcesosMutex[pid].Lock()
	globals_memoria.Procesos[pid] = proceso
	globals_memoria.ProcesosMutex[pid].Unlock()

	var metrica globals_memoria.Memoria_Metrica
	globals_memoria.MetricasMutex.Lock()
	globals_memoria.MetricasMap[pid] = metrica
	globals_memoria.MetricasMutex.Unlock()
	slog.Debug(fmt.Sprintf("Metricas al crear el proceso %+v", globals_memoria.MetricasMap[pid]))

	slog.Debug(fmt.Sprintf("Se creo la tabla de paginas del proceso %d:", pid))
	logTablaDePaginas(pid)
	slog.Debug(fmt.Sprintf("Se creo el proceso %d correctamente", pid))

	return 0
}

func obtenerMarcoDesdeTabla(pid int, entradas []int64) int {
	var NUMBER_OF_LEVELS int = int(globals_memoria.MemoriaConfig.Number_of_levels)
	globals_memoria.ProcesosMutex[pid].Lock()
	defer globals_memoria.ProcesosMutex[pid].Unlock()
	tabla := globals.Procesos[pid].TablaDePaginas
	var indiceActual int

	IncrementarMetrica("ACCESOS_TABLAS", pid, NUMBER_OF_LEVELS)

	for i := 0; i < NUMBER_OF_LEVELS-1; i++ {

		indiceActual = int(entradas[i])
		entrada := tabla.Entradas[indiceActual]
		tabla = *entrada.SiguienteNivel

	}

	indiceActual = int(entradas[len(entradas)-1])

	if indiceActual >= len(tabla.Entradas) {
		slog.Debug(fmt.Sprintf("Error: índice fuera de rango en nivel final para PID %d", pid))
		return -1
	}

	return tabla.Entradas[indiceActual].NumeroDeFrame
}

func eliminarMarcosFisicos(pid int) []globals_memoria.Pagina {

	var paginas []globals_memoria.Pagina

	globals_memoria.ProcesosMutex[pid].Lock()
	tabla := globals_memoria.Procesos[pid].TablaDePaginas

	recorrerTablaYLiberarMarcos(pid, tabla, &paginas)
	globals_memoria.ProcesosMutex[pid].Unlock()

	slog.Debug(fmt.Sprint("Frames eliminados del proceso: ", pid))
	slog.Debug(fmt.Sprint("Paginas: ", paginas))

	return paginas
}

func recorrerTablaYLiberarMarcos(pid int, tabla globals_memoria.TablaPaginas, paginas *[]globals_memoria.Pagina) {
	IncrementarMetrica("ACCESOS_TABLAS", pid, 1)
	pageSize := int(globals_memoria.MemoriaConfig.Page_size)
	for i := 0; i < len(tabla.Entradas); i++ {
		entrada := tabla.Entradas[i]

		if entrada.SiguienteNivel != nil {
			recorrerTablaYLiberarMarcos(pid, *entrada.SiguienteNivel, paginas)
		} else {
			frame := entrada.NumeroDeFrame
			if frame >= 0 && frame < len(globals_memoria.MemoriaMarcosOcupados) && globals_memoria.MemoriaMarcosOcupados[frame] {
				inicio := frame * pageSize

				globals_memoria.MemoriaMutex.Lock()
				contenido := make([]byte, pageSize)
				copy(contenido, globals_memoria.Memoria[inicio:inicio+pageSize])

				pagina := globals_memoria.Pagina{
					NumeroDePagina: entrada.NumeroDePagina,
					Contenido:      contenido,
				}
				*paginas = append(*paginas, pagina)

				for j := 0; j < pageSize; j++ {
					globals_memoria.Memoria[inicio+j] = 0
				}
				globals_memoria.MemoriaMutex.Unlock()

				globals_memoria.MemoriaMarcosOcupadosMutex.Lock()
				globals_memoria.MemoriaMarcosOcupados[frame] = false
				globals_memoria.MemoriaMarcosOcupadosMutex.Unlock()
			}
		}
	}
}

func generarMemoryDump(pid int) int {
	globals_memoria.ProcesosMutex[pid].Lock()
	defer globals_memoria.ProcesosMutex[pid].Unlock()

	proceso, ok := globals_memoria.Procesos[pid]
	if !ok {
		slog.Debug(fmt.Sprintf("PID %d no encontrado", pid))
		return -1
	}

	pageSize := int(globals_memoria.MemoriaConfig.Page_size)
	directorio := globals_memoria.MemoriaConfig.Dump_path + globals_memoria.Prueba

	timestamp := time.Now().Format("20060102-150405")
	nombreArchivo := fmt.Sprintf("%s/%d-%s.dmp", directorio, pid, timestamp)

	err := os.MkdirAll(directorio, os.ModePerm)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al crear directorio de dump: %v", err))
		return -1
	}

	archivo, err := os.Create(nombreArchivo)
	if err != nil {
		slog.Debug(fmt.Sprintf("Error al crear archivo dump: %v", err))
		return -1
	}
	defer archivo.Close()

	// Recorremos la tabla y buscamos las páginas que están en memoria
	var escribirContenido func(tabla *globals_memoria.TablaPaginas)
	escribirContenido = func(tabla *globals_memoria.TablaPaginas) {
		IncrementarMetrica("ACCESOS_TABLAS", pid, 1)
		for i := 0; i < len(tabla.Entradas); i++ {
			entrada := &tabla.Entradas[i]

			if entrada.SiguienteNivel != nil {
				escribirContenido(entrada.SiguienteNivel)
			} else if entrada.NumeroDeFrame >= 0 {
				inicio := entrada.NumeroDeFrame * pageSize
				if inicio+pageSize <= len(globals_memoria.Memoria) {
					contenido := globals_memoria.Memoria[inicio : inicio+pageSize]
					_, err := archivo.Write(contenido)
					if err != nil {
						slog.Debug(fmt.Sprintf("Error al escribir dump de PID %d: %v", pid, err))
					}
				}
			}
		}
	}

	escribirContenido(&proceso.TablaDePaginas)

	slog.Debug(fmt.Sprintf("Memory dump generado correctamente: %s", nombreArchivo))
	return 0
}
