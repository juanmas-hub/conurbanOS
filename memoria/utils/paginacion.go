package utils

import (
	//"encoding/json"
	"fmt"
	"log"
	"time"

	//"net/http"
	"os"

	globals_memoria "github.com/sisoputnfrba/tp-golang/globals/memoria"
)

func leer(direccion int, tamanio int) string {
	var leido string = ""

	for i := 0; i < tamanio; i++ {
		leido += string(globals_memoria.Memoria[direccion+i])
	}
	return leido
}

func escribir(direccion int, dato string) int {

	var tamanioDePagina int = int(globals_memoria.MemoriaConfig.Page_size)
	var offset int = direccion % tamanioDePagina

	if len(dato) > (tamanioDePagina - offset) {
		log.Printf("El dato no se pudo escribir porque excede la pagina")
		return -1
	}
	var marco int = direccion / tamanioDePagina // redondea hacia abajo

	if globals_memoria.MemoriaMarcosOcupados[marco] {
		log.Printf("No se pudo escribir en el marco %v porque ya estaba ocupado", marco)
		return -1
	}

	// Se escribe el dato
	for i := 0; i < len(dato); i++ {
		globals_memoria.Memoria[direccion+i] = dato[i]
	}
	return 0
}

func escribirPaginas(paginasDTO []globals_memoria.PaginaDTO, marcos []int) {
	var direccion int
	for i := 0; i < len(marcos); i++ {
		direccion = marcos[i] * int(globals_memoria.MemoriaConfig.Page_size)

		if escribir(direccion, paginasDTO[i].Contenido) != 0 {
			log.Println("No se pudo escribir la pagina")
			return
		}
		log.Print(paginasDTO[i])
		paginasDTO[i].Entrada.Marco = marcos[i]
		paginasDTO[i].Entrada.Modificado = 0
		paginasDTO[i].Entrada.Presencia = 1
		paginasDTO[i].Entrada.Uso = 0
		globals_memoria.MemoriaMarcosOcupados[marcos[i]] = true
	}
}

func actualizarPagina(indicePagina int, dato string) {
	// Se sobrescribe el dato
	for i := 0; i < len(dato); i++ {
		globals_memoria.Memoria[indicePagina+i] = dato[i]
	}

}

func crearTabla(entradasPorPagina int64) *globals_memoria.TablaDePaginas {
	return &globals_memoria.TablaDePaginas{
		Entradas: make([]globals_memoria.EntradaTablaPagina, entradasPorPagina),
	}
}

func buscarMarcosDisponibles(cantidad int) []int {
	var result []int = make([]int, 0, cantidad)

	for i := 0; i < len(globals_memoria.MemoriaMarcosOcupados); i++ {
		if !globals_memoria.MemoriaMarcosOcupados[i] {
			result = append(result, i)
			if len(result) >= cantidad {
				return result
			}
		}
	}

	log.Printf("no hay suficientes marcos libres: se encontraron %d de %d", len(result), cantidad)
	return nil
}

func actualizarTablaPaginas(pid int, indices []int) {

	IncrementarMetrica("ACCESOS_TABLAS", pid, 1)

	var ENTRIES_PER_PAGE int64 = globals_memoria.MemoriaConfig.Entries_per_page
	var tablaActual *globals_memoria.TablaDePaginas = (*globals_memoria.ProcessManager)[pid]
	var marcoDisponible *globals_memoria.Pagina = &globals_memoria.Procesos[pid].MarcosAsignados[1]

	// manda el marco recien tomado al final del slice
	globals_memoria.Procesos[pid].MarcosAsignados =
		append(globals_memoria.Procesos[pid].MarcosAsignados[1:], globals_memoria.Procesos[pid].MarcosAsignados[0])

	var indiceActual int
	var indiceSiguiente int
	for i := 0; i < len(indices)-1; i++ {
		indiceActual = indices[i]
		indiceSiguiente = indices[i+1]

		if tablaActual.Entradas[indiceActual].SiguienteNivel == nil {
			tablaActual.Entradas[indiceActual].SiguienteNivel = crearTabla(ENTRIES_PER_PAGE)
		}

		tablaActual.Entradas[indiceActual].Pagina = indiceActual
		tablaActual.Entradas[indiceActual].Marco = indiceSiguiente

		tablaActual.Entradas[indiceActual].Nivel = i + 1

		log.Printf("Nivel %d: pagina %d y marco %d", i+1, tablaActual.Entradas[indiceActual].Pagina, tablaActual.Entradas[indiceActual].Marco)
		IncrementarMetrica("ACCESOS_TABLAS", pid, 1)
		tablaActual = tablaActual.Entradas[indiceActual].SiguienteNivel

	}

	// Asignar el marco fisico
	tablaActual.Entradas[indiceSiguiente].Marco = marcoDisponible.IndiceAsignado

	marcoDisponible.EntradaAsignada = &tablaActual.Entradas[indiceSiguiente]

	fmt.Printf("Ruta de índices: %d->%d->%d->%d->%d\n", indices[0], indices[1], indices[2], indices[3], indices[4])
}

func asignarPaginasAProceso(pid int, indicesPaginas []int) {
	var paginaActual globals_memoria.Pagina
	paginaActual.IndiceSwapAsignado = -1

	for i := 0; i < len(indicesPaginas); i++ {
		paginaActual.IndiceAsignado = indicesPaginas[i]

		globals_memoria.Procesos[pid].MarcosAsignados = append(globals_memoria.Procesos[pid].MarcosAsignados, paginaActual)

		globals_memoria.MemoriaMarcosOcupados[indicesPaginas[i]] = true
	}
}

func AlmacenarProceso(pid int, tamanio int, filename string) int {

	if verificarPIDUnico(pid) != 0 {
		log.Printf("el proceso con PID %d ya existia", pid)
		return -1
	}
	var pageSize int
	var indicesNecesarios int
	var indicesDisponibles []int

	pageSize = int(globals_memoria.MemoriaConfig.Page_size)
	indicesNecesarios = (tamanio + pageSize - 1) / pageSize // antes estaba tamanio / pagesize que puede dar mal
	indicesDisponibles = buscarMarcosDisponibles(indicesNecesarios)

	if indicesDisponibles == nil {
		log.Printf("Error no hay suficiente espacio para almacenar el proceso %d", pid)
		return -1
	}

	globals_memoria.Procesos[pid] = &globals_memoria.Proceso{
		Suspendido:      false,
		PaginasSWAP:     nil,
		MarcosAsignados: nil,
	}

	asignarPaginasAProceso(pid, indicesDisponibles)

	var ENTRIES_PER_PAGE int64 = globals_memoria.MemoriaConfig.Entries_per_page
	var instrucciones []string

	instrucciones = ObtenerInstruccionesDesdeArchivo(filename)

	globals_memoria.Procesos[pid].Pseudocodigo = instrucciones
	log.Print(instrucciones)

	(*globals_memoria.ProcessManager)[pid] = crearTabla(ENTRIES_PER_PAGE)

	return 0
}

func obtenerMarcoDesdeTabla(pid int, primerIndice int) int {
	(*globals_memoria.Metricas)[pid].AccesosTablas++

	var NUMBER_OF_LEVELS int = int(globals_memoria.MemoriaConfig.Number_of_levels)
	var tablaActual *globals_memoria.TablaDePaginas = (*globals_memoria.ProcessManager)[pid]
	var indiceActual int = primerIndice

	for i := 0; i < NUMBER_OF_LEVELS-1; i++ {

		var entrada *globals_memoria.EntradaTablaPagina = &tablaActual.Entradas[indiceActual]
		tablaActual = entrada.SiguienteNivel
		indiceActual = entrada.Marco
	}

	if indiceActual >= len(tablaActual.Entradas) {
		log.Printf("Error: índice fuera de rango en nivel final para PID %d", pid)
		return -1
	}

	return tablaActual.Entradas[indiceActual].Marco
}

func eliminarMarcosFisicos(pid int) []globals_memoria.PaginaDTO {

	if globals_memoria.Procesos[pid].MarcosAsignados == nil {
		return nil
	}
	var marcos *[]globals_memoria.Pagina = &globals_memoria.Procesos[pid].MarcosAsignados
	log.Print(marcos)
	var pageSize int = int(globals_memoria.MemoriaConfig.Page_size)
	var paginasDTO []globals_memoria.PaginaDTO = []globals_memoria.PaginaDTO{}

	for i := 0; i < len(*marcos); i++ {
		var paginaDTO globals_memoria.PaginaDTO
		var inicio int = (*marcos)[i].IndiceAsignado * pageSize
		var contenido string

		// Leer contenido antes de sobrescribir
		contenido = leer(inicio, pageSize)
		paginaDTO.Contenido = contenido

		// Sobrescribir con ceros en la memoria
		for j := 0; j < pageSize; j++ {
			globals_memoria.Memoria[inicio+j] = 0
		}

		// Eliminar indice de la tabla de paginas
		if (*marcos)[i].EntradaAsignada != nil {
			(*marcos)[i].EntradaAsignada.Marco = -1
		}

		// Guardar entrada asignada
		paginaDTO.Entrada = (*marcos)[i].EntradaAsignada
		paginasDTO = append(paginasDTO, paginaDTO)

		// Marcar marco como disponible
		globals_memoria.MemoriaMarcosOcupados[(*marcos)[i].IndiceAsignado] = false

		// Eliminar marco asignado al proceso
		(*marcos)[i].IndiceAsignado = -1
	}

	return paginasDTO
}

func generarMemoryDump(pid int) int {
	var marcos []globals_memoria.Pagina = globals_memoria.Procesos[pid].MarcosAsignados
	var pageSize int = int(globals_memoria.MemoriaConfig.Page_size)
	var directorio string = globals_memoria.MemoriaConfig.Dump_path + globals_memoria.Prueba

	if marcos == nil || len(marcos) == 0 {
		log.Printf("No hay marcos asignados para el proceso %d. No se genera dump.", pid)
		return -1
	}

	var timestamp string = time.Now().Format("20060102-150405") // YYYYMMDD-HHMMSS
	var nombreArchivo string = fmt.Sprintf("%s/%d-%s.dmp", directorio, pid, timestamp)

	// Crear la carpeta si no existe
	err := os.MkdirAll(directorio, os.ModePerm)
	if err != nil {
		log.Printf("Error al crear el directorio de dump: %v", err)
		return -1
	}

	archivo, err := os.Create(nombreArchivo)
	if err != nil {
		log.Printf("Error al crear el archivo de dump: %v", err)
		return -1
	}
	defer archivo.Close()

	for i := 0; i < len(marcos); i++ {
		var inicio int = marcos[i].IndiceAsignado * pageSize
		var contenido string = leer(inicio, pageSize)
		_, err := archivo.WriteString(contenido)
		if err != nil {
			log.Printf("Error al escribir al archivo de dump: %v", err)
			return -1
		}
	}

	log.Printf("Memory dump generado correctamente en: %s", nombreArchivo)
	return 0
}
