package utils

import (
	//"encoding/json"
	"fmt"
	"log"
	"log/slog"
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

/*
func escribirPaginas(pid int, paginasDTO []globals_memoria.PaginaDTO, marcos []int) {
	var direccion int
	for i := 0; i < len(marcos); i++ {
		direccion = marcos[i] * int(globals_memoria.MemoriaConfig.Page_size)

		if escribir(direccion, paginasDTO[i].Contenido) != 0 {
			log.Println("No se pudo escribir la pagina")
			return
		}
		log.Print(paginasDTO[i])
		if paginasDTO[i].Entrada != nil {
			paginasDTO[i].Entrada.Marco = marcos[i]
			paginasDTO[i].Entrada.Modificado = 0
			paginasDTO[i].Entrada.Presencia = 1
			paginasDTO[i].Entrada.Uso = 0
		}
		globals_memoria.MemoriaMarcosOcupados[marcos[i]] = true
	}
}*/

func escribirPaginas(pid int, paginas []globals_memoria.Pagina, marcos []int) *[]globals_memoria.PaginaLinkeada {

	var paginasLinkeadas []globals_memoria.PaginaLinkeada

	var direccion int
	for i := 0; i < len(marcos); i++ {
		direccion = marcos[i] * int(globals_memoria.MemoriaConfig.Page_size)

		if escribir(direccion, string(paginas[i].Contenido)) != 0 {
			log.Println("No se pudo escribir la pagina")
			return nil
		}

		paginaLinkeada := globals_memoria.PaginaLinkeada{
			NumeroDePagina: paginas[i].NumeroDePagina,
			NumeroDeFrame:  marcos[i],
		}
		paginasLinkeadas = append(paginasLinkeadas, paginaLinkeada)

		log.Print(paginas[i])

		globals_memoria.MemoriaMarcosOcupados[marcos[i]] = true
	}

	return &paginasLinkeadas
}

func actualizarPagina(indicePagina int, dato string) {
	// Se sobrescribe el dato
	for i := 0; i < len(dato); i++ {
		globals_memoria.Memoria[indicePagina+i] = dato[i]
	}

}

/*
func crearTabla(entradasPorPagina int64, nivel int) *globals_memoria.TablaDePaginas {
	tabla := &globals_memoria.TablaDePaginas{
		Entradas: make([]globals_memoria.EntradaTablaPagina, entradasPorPagina),
	}

	for i := 0; i < int(entradasPorPagina); i++ {
		tabla.Entradas[i].Nivel = nivel
	}

	return tabla
}*/

/*
func crearTabla(entradasPorPagina int64, nivel int) globals_memoria.TablaPaginas {
	tabla := globals_memoria.TablaDePaginas{
		Entradas: make([]globals_memoria.EntradaTablaPagina, entradasPorPagina),
	}

	for i := 0; i < int(entradasPorPagina); i++ {
		tabla.Entradas[i].Nivel = nivel
	}

	return tabla
}*/

func crearTabla(pid int, framesAsignados []int) globals_memoria.TablaPaginas {
	niveles := int(globals_memoria.MemoriaConfig.Number_of_levels)
	entradasPorPagina := int(globals_memoria.MemoriaConfig.Entries_per_page)

	frameIndex := 0

	return construirNivel(pid, niveles, entradasPorPagina, &frameIndex, framesAsignados)
}

func construirNivel(pid int, nivel int, entradasPorPagina int, frameIndex *int, framesAsignados []int) globals_memoria.TablaPaginas {
	cantPaginasRestantes := len(framesAsignados) - *frameIndex

	// ¿Cuántas entradas necesitamos en este nivel?
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
				fmt.Printf("⚠️  No hay suficientes frames asignados para el PID %d\n", pid)
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

	return tabla
}

/*
func construirArbolTablas(nivelActual, cantidadNiveles int, entradasPorTabla int64) *globals_memoria.TablaDePaginas {
    tabla := crearTabla(entradasPorTabla, nivelActual)

	if tabla == nil{
		log.Printf("Fallo al construir el arbol en el nivel: %d", nivelActual)
		return nil
	}

	log.Printf("Se creo una tabla de nivel: %d", nivelActual)

    for i := range tabla.Entradas {
        // Si no es el último nivel, crear recursivamente la siguiente tabla
        if nivelActual < cantidadNiveles {
            tabla.Entradas[i].SiguienteNivel = construirArbolTablas(nivelActual+1, cantidadNiveles, entradasPorTabla)
        }
    }

    return tabla
}
*/

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

/*
func actualizarTablaPaginas(pid int, indices []int) {

	var ENTRIES_PER_PAGE int64 = globals_memoria.MemoriaConfig.Entries_per_page
	var tabla *globals_memoria.TablaDePaginas = (*globals_memoria.Tablas)[pid]
	var cantidadMarcos int = len(globals_memoria.Procesos[pid].MarcosAsignados)
	var marco *globals_memoria.Pagina

	IncrementarMetrica("ACCESOS_TABLAS", pid, 1)
	slog.Debug(fmt.Sprintf("Me llego para actualizar la tabla de paginas del proceso %d", pid))
	slog.Debug(fmt.Sprintf("Me llegaron los siguientes indices: %+v", indices))

	// Mueve el primer marco al final del slice
	globals_memoria.Procesos[pid].MarcosAsignados =
		append(globals_memoria.Procesos[pid].MarcosAsignados[1:], globals_memoria.Procesos[pid].MarcosAsignados[0])

	marco = &globals_memoria.Procesos[pid].MarcosAsignados[cantidadMarcos-1]

	slog.Debug(fmt.Sprintf("Marco elegido para actualizar tabla de paginas: %+v", marco))

	var actual int
	var siguiente int
	for i := 0; i < len(indices)-1; i++ {
		actual = indices[i]
		siguiente = indices[i+1]
		slog.Debug(fmt.Sprintf("Indice Actual: %+v, Indice Siguiente: %+v", actual, siguiente))

		tabla.Entradas[actual].Marco = siguiente

		if tabla.Entradas[actual].SiguienteNivel == nil {
			tabla.Entradas[actual].SiguienteNivel = crearTabla(ENTRIES_PER_PAGE, i+1)
		}

		log.Printf("Nivel %d: pagina %d y marco %d", i+1, i, siguiente)
		IncrementarMetrica("ACCESOS_TABLAS", pid, 1)

		tabla = tabla.Entradas[actual].SiguienteNivel
	}

	// Asignar el marco fisico
	tabla.Entradas[siguiente].Marco = marco.IndiceAsignado

	marco.EntradaAsignada = &tabla.Entradas[siguiente]

	slog.Debug(fmt.Sprintf("Entrada asignada: %+v", marco.EntradaAsignada))

	fmt.Printf("Ruta de índices: %d->%d->%d->%d->%d\n", indices[0], indices[1], indices[2], indices[3], indices[4])
}*/

func actualizarTablaPaginas(pid int, paginasLinkeadas []globals_memoria.PaginaLinkeada) {

	IncrementarMetrica("ACCESOS_TABLAS", pid, 1)
	slog.Debug(fmt.Sprintf("Me llego para actualizar la tabla de paginas del proceso %d", pid))

	proceso := globals_memoria.Procesos[pid]

	// Crea un mapa para acceso rápido por número de página
	paginasMap := make(map[int]globals_memoria.PaginaLinkeada)
	for _, pl := range paginasLinkeadas {
		paginasMap[pl.NumeroDePagina] = pl
	}

	var recorrerYActualizar func(tabla *globals_memoria.TablaPaginas)
	recorrerYActualizar = func(tabla *globals_memoria.TablaPaginas) {
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

	// Actualizar el proceso en la tabla global
	globals_memoria.Procesos[pid] = proceso

	slog.Debug(fmt.Sprintf("Tabla de paginas actualizada: ", globals_memoria.Procesos[pid].TablaDePaginas))
}

/*
func asignarPaginasAProceso(pid int, indicesPaginas []int) {
	var paginaActual globals_memoria.Pagina
	paginaActual.IndiceSwapAsignado = -1

	for i := 0; i < len(indicesPaginas); i++ {
		paginaActual.IndiceAsignado = indicesPaginas[i]

		globals_memoria.Procesos[pid].MarcosAsignados = append(globals_memoria.Procesos[pid].MarcosAsignados, paginaActual)

		globals_memoria.MemoriaMarcosOcupados[indicesPaginas[i]] = true

		log.Printf("Pagina %+v asignada al proceso %d", paginaActual, pid)
	}
}*/

func asignarFramesAProceso(numerosDeFrame []int) {

	for _, frame := range numerosDeFrame {
		globals_memoria.MemoriaMarcosOcupados[frame] = true
	}

}

/*
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
	log.Print("Indices necesarios: ", indicesNecesarios)
	indicesDisponibles = buscarMarcosDisponibles(indicesNecesarios)
	log.Print("Indices disponibles: ", indicesDisponibles)

	if indicesDisponibles == nil {
		log.Printf("Error no hay suficiente espacio para almacenar el proceso %d", pid)
		return -1
	}

	globals_memoria.Procesos[pid] = &globals_memoria.Proceso{
		Pseudocodigo:    nil,
		MarcosAsignados: nil,
		Suspendido:      false,
		PaginasSWAP:     nil,
	}

	asignarPaginasAProceso(pid, indicesDisponibles)

	var ENTRIES_PER_PAGE int64 = globals_memoria.MemoriaConfig.Entries_per_page
	// var NUMBER_OF_LEVELS int = int(globals_memoria.MemoriaConfig.Number_of_levels)
	var instrucciones []string
	var tabla *globals_memoria.TablaDePaginas

	instrucciones = ObtenerInstruccionesDesdeArchivo(filename)

	globals_memoria.Procesos[pid].Pseudocodigo = instrucciones

	//tabla = construirArbolTablas(1, NUMBER_OF_LEVELS, ENTRIES_PER_PAGE)
	tabla = crearTabla(ENTRIES_PER_PAGE, 1)

	(*globals_memoria.Tablas)[pid] = tabla
	log.Printf("Se creo la tabla de paginas del proceso %d: %+v", pid, tabla)

	return 0
}
*/

func AlmacenarProceso(pid int, tamanio int, filename string) int {

	if verificarPIDUnico(pid) != 0 {
		log.Printf("el proceso con PID %d ya existia", pid)
		return -1
	}
	var pageSize int
	var indicesNecesarios int
	var indicesDisponibles []int

	pageSize = int(globals_memoria.MemoriaConfig.Page_size)
	indicesNecesarios = (tamanio + pageSize - 1) / pageSize
	log.Print("Indices necesarios: ", indicesNecesarios)
	indicesDisponibles = buscarMarcosDisponibles(indicesNecesarios)
	log.Print("Indices disponibles: ", indicesDisponibles)

	if indicesDisponibles == nil {
		log.Printf("Error no hay suficiente espacio para almacenar el proceso %d", pid)
		return -1
	}

	// Asigno los frames en la memoria
	asignarFramesAProceso(indicesDisponibles)

	// Obtengo las instrucciones
	instrucciones := ObtenerInstruccionesDesdeArchivo(filename)

	// Creo la tabla de paginas
	tabla := crearTabla(pid, indicesDisponibles)

	// Creo el proceso
	proceso := globals_memoria.Proceso{
		Pseudocodigo:      instrucciones,
		Suspendido:        false,
		TablaDePaginas:    tabla,
		CantidadDePaginas: len(indicesDisponibles),
	}

	// Lo añado al mapa de procesos
	globals_memoria.Procesos[pid] = proceso

	slog.Debug(fmt.Sprintf("Se creo la tabla de paginas del proceso %d: %+v", pid, tabla))

	return 0
}

func obtenerMarcoDesdeTabla(pid int, primerIndice int) int {
	(*globals_memoria.Metricas)[pid].AccesosTablas++

	var NUMBER_OF_LEVELS int = int(globals_memoria.MemoriaConfig.Number_of_levels)
	var tablaActual *globals_memoria.TablaDePaginas = (*globals_memoria.Tablas)[pid]
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

/*
func eliminarMarcosFisicos(pid int) []globals_memoria.PaginaDTO {

	slog.Debug(fmt.Sprint("Se entro a eliminarMarcosFisicos"))

	if globals_memoria.Procesos[pid].MarcosAsignados == nil {
		slog.Debug(fmt.Sprint("Marcos asignados nulo: ", globals_memoria.Procesos[pid].MarcosAsignados))
		return nil
	}

	var marcos *[]globals_memoria.Pagina = &globals_memoria.Procesos[pid].MarcosAsignados
	var pageSize int = int(globals_memoria.MemoriaConfig.Page_size)
	var paginasDTO []globals_memoria.PaginaDTO = []globals_memoria.PaginaDTO{}

	slog.Debug(fmt.Sprint("Length marcos: ", len(*marcos)))

	for i := 0; i < len(*marcos); i++ {
		var paginaDTO globals_memoria.PaginaDTO
		var inicio int = (*marcos)[i].IndiceAsignado * pageSize
		var contenido string

		slog.Debug(fmt.Sprintf("Marco: %+v", (*marcos)[i]))

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
		slog.Debug(fmt.Sprintf("Entrada: %+v", (*marcos)[i].EntradaAsignada))
		if paginaDTO.Entrada == nil {
			paginaDTO.Entrada = (*marcos)[i].EntradaAsignada
			paginasDTO = append(paginasDTO, paginaDTO)
		}

		// Marcar marco como disponible
		globals_memoria.MemoriaMarcosOcupados[(*marcos)[i].IndiceAsignado] = false

		// Eliminar marco asignado al proceso
		(*marcos)[i].IndiceAsignado = -1
	}

	slog.Debug(fmt.Sprint("PaginasDTO en eliminarMarcosFisicos: ", paginasDTO))
	return paginasDTO
}*/

func eliminarMarcosFisicos(pid int) []globals_memoria.Pagina {

	// Lista para devolver las páginas leídas
	var paginas []globals_memoria.Pagina

	// Obtener la tabla de páginas multinivel del proceso
	tabla := globals_memoria.Procesos[pid].TablaDePaginas

	// Obtengo las paginas
	recorrerTablaYLiberarMarcos(tabla, &paginas)

	slog.Debug(fmt.Sprint("Frames eliminados del proceso: ", pid))
	slog.Debug(fmt.Sprint("Paginas: ", paginas))

	return paginas
}

func recorrerTablaYLiberarMarcos(tabla globals_memoria.TablaPaginas, paginas *[]globals_memoria.Pagina) {
	pageSize := int(globals_memoria.MemoriaConfig.Page_size)
	for i := 0; i < len(tabla.Entradas); i++ {
		entrada := tabla.Entradas[i]

		if entrada.SiguienteNivel != nil {
			// Si hay siguiente nivel, seguimos recorriendo recursivamente
			recorrerTablaYLiberarMarcos(*entrada.SiguienteNivel, paginas)
		} else {
			// Último nivel: obtener frame asignado
			frame := entrada.NumeroDeFrame
			if frame >= 0 && frame < len(globals_memoria.MemoriaMarcosOcupados) && globals_memoria.MemoriaMarcosOcupados[frame] {
				inicio := frame * pageSize

				// Leer contenido de la memoria en ese marco
				contenido := make([]byte, pageSize)
				copy(contenido, globals_memoria.Memoria[inicio:inicio+pageSize])

				// Guardar página con su contenido
				pagina := globals_memoria.Pagina{
					NumeroDePagina: entrada.NumeroDePagina,
					Contenido:      contenido,
				}
				*paginas = append(*paginas, pagina)

				// Sobrescribir con ceros la memoria para liberar el marco
				for j := 0; j < pageSize; j++ {
					globals_memoria.Memoria[inicio+j] = 0
				}

				// Marcar marco como libre
				globals_memoria.MemoriaMarcosOcupados[frame] = false

			}
		}
	}
}

/*
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
*/

func generarMemoryDump(pid int) int {
	proceso, ok := globals_memoria.Procesos[pid]
	if !ok {
		log.Printf("PID %d no encontrado", pid)
		return -1
	}

	pageSize := int(globals_memoria.MemoriaConfig.Page_size)
	directorio := globals_memoria.MemoriaConfig.Dump_path + globals_memoria.Prueba

	timestamp := time.Now().Format("20060102-150405")
	nombreArchivo := fmt.Sprintf("%s/%d-%s.dmp", directorio, pid, timestamp)

	// Crear carpeta si no existe
	err := os.MkdirAll(directorio, os.ModePerm)
	if err != nil {
		log.Printf("❌ Error al crear directorio de dump: %v", err)
		return -1
	}

	archivo, err := os.Create(nombreArchivo)
	if err != nil {
		log.Printf("❌ Error al crear archivo dump: %v", err)
		return -1
	}
	defer archivo.Close()

	// Recorremos la tabla y buscamos las páginas que están en memoria
	var escribirContenido func(tabla *globals_memoria.TablaPaginas)
	escribirContenido = func(tabla *globals_memoria.TablaPaginas) {
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
						log.Printf("❌ Error al escribir dump de PID %d: %v", pid, err)
					}
				}
			}
		}
	}

	escribirContenido(&proceso.TablaDePaginas)

	log.Printf("✅ Memory dump generado correctamente: %s", nombreArchivo)
	return 0
}
