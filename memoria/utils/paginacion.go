package utils

import (
	//"encoding/json"
	"fmt"
	"log"

	//"net/http"
	//"os"

	globals_memoria "github.com/sisoputnfrba/tp-golang/globals/memoria"
)

func leer(direccion int, tamanio int) string {
	var leido string = ""

	for i := 0; i < tamanio; i++ {
		leido += string(globals_memoria.Memoria[direccion+i])
	}
	return leido
}

func escribir(direccion int, dato string) int{

	var tamanioDePagina int = int(globals_memoria.MemoriaConfig.Page_size)
	var offset int = direccion % tamanioDePagina

	if len(dato) > (tamanioDePagina - offset){
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

		if escribir(direccion, paginasDTO[i].Contenido) != 0{
			log.Println("No se pudo escribir la pagina")
			return
		}
		paginasDTO[i].Entrada.Marco = marcos[i]
		globals_memoria.MemoriaMarcosOcupados[marcos[i]] = true

	}
}

func crearTabla(entradasPorPagina int64) *globals_memoria.TablaDePaginas {
	return &globals_memoria.TablaDePaginas{
		Entradas: make([]globals_memoria.EntradaTablaPagina, entradasPorPagina),
	}
}

func verificarPIDUnico(pid int) int {
	if _, existe := (*globals_memoria.ProcessManager)[pid]; existe {
		return 1
	}
	return 0
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

func actualizarTablaPaginas(pid int, indices []int) int{
	var ENTRIES_PER_PAGE int64 = globals_memoria.MemoriaConfig.Entries_per_page

	var tablaActual *globals_memoria.TablaDePaginas = (*globals_memoria.ProcessManager)[pid]
	var marcoDisponible []int = buscarMarcosDisponibles(1)

	if marcoDisponible == nil {
		log.Println("Error al actualizar paginas, no hay marco disponible para asignar")
		return 1
	}

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

		log.Printf("Nivel %d: pagina %d y marco %d", i+1, tablaActual.Entradas[indiceActual].Pagina, tablaActual.Entradas[indiceActual].Marco)

		tablaActual = tablaActual.Entradas[indiceActual].SiguienteNivel

	}

	// Asignar el marco fisico
	tablaActual.Entradas[indiceSiguiente].Marco = marcoDisponible[0]

	var marcoAsignado globals_memoria.Pagina

	marcoAsignado.IndiceAsignado = marcoDisponible[0]
	marcoAsignado.EntradaAsignada = &tablaActual.Entradas[indiceSiguiente]

	globals_memoria.Procesos[pid].MarcosAsignados = append(globals_memoria.Procesos[pid].MarcosAsignados, marcoAsignado)
	globals_memoria.MemoriaMarcosOcupados[marcoDisponible[0]] = true


	fmt.Printf("Ruta de índices: %d->%d->%d->%d->%d\n", indices[0], indices[1], indices[2], indices[3], indices[4])
	return 0
}

func AlmacenarProceso(pid int, filename string) error {

	if verificarPIDUnico(pid) != 0 {
		return fmt.Errorf("el proceso con PID %d ya existe", pid)
	}

	var ENTRIES_PER_PAGE int64 = globals_memoria.MemoriaConfig.Entries_per_page
	var instrucciones []string

	instrucciones = ObtenerInstruccionesDesdeArchivo(filename)

	globals_memoria.Procesos[pid] = &globals_memoria.Proceso{
		Pseudocodigo: instrucciones,
		Suspendido:   false, // supongo je
		PaginasSWAP: nil,
		MarcosAsignados: nil,
	}

	globals_memoria.Procesos[pid].Pseudocodigo = instrucciones
	log.Print(instrucciones)

	(*globals_memoria.ProcessManager)[pid] = crearTabla(ENTRIES_PER_PAGE)

	log.Println("Proceso almacenado")

	return nil
}

func obtenerMarcoDesdeTabla(pid int, primerIndice int) int {
	NUMBER_OF_LEVELS := int(globals_memoria.MemoriaConfig.Number_of_levels)

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
		(*marcos)[i].EntradaAsignada.Marco = -1

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
