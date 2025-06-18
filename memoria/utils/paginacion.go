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

func escribir(direccion int, dato string) {

	var tamanioDePagina int = int(globals_memoria.MemoriaConfig.Page_size)
	var marco int = direccion / tamanioDePagina // redondea hacia abajo

	globals_memoria.MemoriaMarcosOcupados[marco] = true

	for i := 0; i < len(dato); i++ {
		globals_memoria.Memoria[direccion+i] = dato[i]

		if i != 0 && i%tamanioDePagina == 0 {
			marco++
			globals_memoria.MemoriaMarcosOcupados[marco] = true
		}
	}
}

func escribirPaginas(paginas []string, marcos []int) {
	for i := 0; i < len(marcos); i++ {
		escribir(marcos[i], paginas[i])
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

func actualizarTablaPaginas(pid int, indices []int) {
	var ENTRIES_PER_PAGE int64 = globals_memoria.MemoriaConfig.Entries_per_page
	// var NUMBER_OF_LEVELS int = int(globals_memoria.MemoriaConfig.Number_of_levels)

	var tablaActual *globals_memoria.TablaDePaginas = (*globals_memoria.ProcessManager)[pid]

	for i := 0; i < len(indices)-1; i++ {
		var indiceActual int = indices[i]
		var indiceSiguiente int = indices[i+1]

		if tablaActual.Entradas[indiceActual].SiguienteNivel == nil {
			tablaActual.Entradas[indiceActual].SiguienteNivel = crearTabla(ENTRIES_PER_PAGE)
		}

		tablaActual.Entradas[indiceActual].Pagina = indiceActual
		tablaActual.Entradas[indiceActual].Marco = indiceSiguiente

		log.Printf("Nivel %d: pagina %d y marco %d", i+1, tablaActual.Entradas[indiceActual].Pagina, tablaActual.Entradas[indiceActual].Marco)

		tablaActual = tablaActual.Entradas[indiceActual].SiguienteNivel

	}

	fmt.Printf("Ruta de índices: %d->%d->%d->%d->%d\n", indices[0], indices[1], indices[2], indices[3], indices[4])

}

func AlmacenarProceso(pid int, filename string) error {

	if verificarPIDUnico(pid) != 0 {
		return fmt.Errorf("el proceso con PID %d ya existe", pid)
	}

	var ENTRIES_PER_PAGE int64 = globals_memoria.MemoriaConfig.Entries_per_page
	var instrucciones []string

	instrucciones = ObtenerInstruccionesDesdeArchivo(filename)

	// Inicializamos el proceso antes de asignar campos
	globals_memoria.Procesos[pid] = &globals_memoria.Proceso{
		Pseudocodigo: instrucciones,
		Suspendido:   false, // supongo je
		// Fijate que poner en paginas fisicas, swap
	}

	globals_memoria.Procesos[pid].Pseudocodigo = instrucciones
	log.Print(instrucciones)

	(*globals_memoria.ProcessManager)[pid] = crearTabla(ENTRIES_PER_PAGE)

	log.Println("Proceso almacenado")

	return nil
}

func obtenerMarcoDesdeTabla(pid int, pagina int) int {
	NUMBER_OF_LEVELS := int(globals_memoria.MemoriaConfig.Number_of_levels)

	tablaActual := (*globals_memoria.ProcessManager)[pid]
	indiceActual := pagina

	for i := 0; i < NUMBER_OF_LEVELS-1; i++ {

		entrada := tablaActual.Entradas[indiceActual]
		tablaActual = entrada.SiguienteNivel
		indiceActual = entrada.Marco
	}

	if indiceActual >= len(tablaActual.Entradas) {
		log.Printf("Error: índice fuera de rango en nivel final para PID %d", pid)
		return -1
	}

	return tablaActual.Entradas[indiceActual].Marco

}

func eliminarPaginasFisicas(pid int) []string {

	if globals_memoria.Procesos[pid].PaginasFisicas == nil {
		return nil
	}
	paginas := globals_memoria.Procesos[pid].PaginasFisicas
	pageSize := int(globals_memoria.MemoriaConfig.Page_size)

	contenidoPaginas := []string{}

	for i := 0; i < len(paginas); i++ {
		inicio := paginas[i] * pageSize

		// Leer contenido antes de sobrescribir
		contenido := leer(inicio, pageSize)
		contenidoPaginas = append(contenidoPaginas, contenido)

		// Sobrescribir con ceros en la memoria
		for j := 0; j < pageSize; j++ {
			globals_memoria.Memoria[inicio+j] = 0
		}

		globals_memoria.MemoriaMarcosOcupados[paginas[i]] = false
	}

	// Limpiar la lista de páginas físicas del proceso
	globals_memoria.Procesos[pid].PaginasFisicas = nil

	return contenidoPaginas
}
