# Layout adaptativo administrativo (decisión del propietario)
Una sola aplicación Flutter, estado y servicios compartidos. BPAdaptive
decide por restricciones locales: compacto <600, rail 600–1023, expandido
>=1024 px lógicos, con fallback compacto para texto ampliado. No inferir
modo a partir de Android/web. Rail visible y menú expandible, sidebar
alineado arriba; ancho proporcional acotado, objetivos táctiles >=48.
Conservar campos, filtros, borradores y acciones al redimensionar.
Tablas deben elegir presentación por ancho del contenido y número de
columnas; preferir tabla continua en amplio, lista legible en estrecho.
Probar 320–1920, bordes de los umbrales, teclado y 200% de texto.
