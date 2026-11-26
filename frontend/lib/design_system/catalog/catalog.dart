import "package:flutter/widgets.dart";

import "../design_system.dart";
import "../bp_choice_bar.dart";

class BPCatalog extends StatefulWidget {
  const BPCatalog({super.key});
  @override
  State<BPCatalog> createState() => _BPCatalogState();
}

class _BPCatalogState extends State<BPCatalog> {
  final controller = TextEditingController();
  int actions = 0;
  bool loading = false;
  String choice = "all";
  @override
  void dispose() {
    controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = BPVisuals.of(context);
    return ColoredBox(
      color: theme.background,
      child: SafeArea(
        child: LayoutBuilder(
          builder: (context, constraints) {
            final wide = constraints.maxWidth >= BPTokens.wideBreakpoint;
            final sections = [
              _section("Botones", "Primario, secundario y deshabilitado", [
                BPButton(
                  key: const Key("catalog-primary"),
                  label: "Guardar",
                  onPressed: () => setState(() => actions++),
                ),
                BPButton(
                  label: "Cancelar",
                  primary: false,
                  onPressed: () => setState(() => actions = 0),
                ),
                const BPButton(
                  key: Key("catalog-disabled"),
                  label: "Deshabilitado",
                  onPressed: null,
                ),
                Text("Acciones de ejemplo: $actions", style: theme.muted),
              ]),
              _section("Campos", "Edición local y etiqueta accesible", [
                BPInput(
                  controller: controller,
                  label: "Código de ejemplo",
                  placeholder: "Ej. ABC",
                  onChanged: (_) {},
                ),
                BPButton(
                  label: "Limpiar campo",
                  primary: false,
                  onPressed: controller.clear,
                ),
              ]),
              _section("Filtros", "Selección única controlada", [
                BPChoiceBar(
                  label: "Filtro de ejemplo",
                  keyPrefix: "catalog-filter",
                  choices: const [
                    BPChoice("all", "Todos"),
                    BPChoice("available", "Disponibles"),
                    BPChoice("attention", "Por revisar"),
                  ],
                  selectedId: choice,
                  onChanged: (id) => setState(() => choice = id),
                ),
                Text("Filtro de ejemplo: $choice", style: theme.muted),
              ]),
              _section(
                "Carga y tarjeta",
                "Simulación manual; no hay consultas de red",
                [
                  Wrap(
                    spacing: BPTokens.sm,
                    runSpacing: BPTokens.sm,
                    children: [
                      BPButton(
                        key: const Key("catalog-loading"),
                        label: "Cargando",
                        primary: loading,
                        onPressed: () => setState(() => loading = true),
                      ),
                      BPButton(
                        key: const Key("catalog-loaded"),
                        label: "Cargado",
                        primary: !loading,
                        onPressed: () => setState(() => loading = false),
                      ),
                    ],
                  ),
                  Semantics(
                    liveRegion: true,
                    label: loading ? "Cargando ejemplo" : "Ejemplo cargado",
                    child: BPCard(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          SizedBox(
                            height: BPTokens.touchTarget * 2,
                            child: loading
                                ? const Column(
                                    key: Key("catalog-skeleton"),
                                    crossAxisAlignment:
                                        CrossAxisAlignment.start,
                                    mainAxisAlignment: MainAxisAlignment.center,
                                    children: [
                                      BPSkeleton(
                                        width: double.infinity,
                                        height: BPTokens.xl,
                                      ),
                                      SizedBox(height: BPTokens.sm),
                                      BPSkeleton(
                                        width: 96,
                                        height: BPTokens.lg,
                                      ),
                                    ],
                                  )
                                : Center(
                                    child: Text(
                                      "Producto de ejemplo",
                                      style: theme.title,
                                    ),
                                  ),
                          ),
                        ],
                      ),
                    ),
                  ),
                ],
              ),
            ];
            return ListView(
              padding: EdgeInsets.all(wide ? BPTokens.xxl : BPTokens.lg),
              children: [
                Text("Catálogo de componentes", style: theme.heading),
                const SizedBox(height: BPTokens.sm),
                Text(
                  "BalanzaPro3.0 · Entorno visual de desarrollo · Datos ficticios",
                  style: theme.muted,
                ),
                const SizedBox(height: BPTokens.xl),
                if (wide)
                  Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Expanded(
                        child: Column(
                          children: [
                            sections[0],
                            const SizedBox(height: BPTokens.lg),
                            sections[1],
                          ],
                        ),
                      ),
                      const SizedBox(width: BPTokens.lg),
                      Expanded(
                        child: Column(
                          children: [
                            sections[3],
                            const SizedBox(height: BPTokens.lg),
                            sections[2],
                          ],
                        ),
                      ),
                    ],
                  )
                else
                  for (final section in [
                    sections[0],
                    sections[1],
                    sections[3],
                    sections[2],
                  ])
                    Padding(
                      padding: const EdgeInsets.only(bottom: BPTokens.lg),
                      child: section,
                    ),
              ],
            );
          },
        ),
      ),
    );
  }

  Widget _section(String title, String description, List<Widget> examples) {
    final theme = BPVisuals.of(context);
    return BPCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(title, style: theme.title),
          const SizedBox(height: BPTokens.xs),
          Text(description, style: theme.muted),
          const SizedBox(height: BPTokens.lg),
          for (final example in examples)
            Padding(
              padding: const EdgeInsets.only(bottom: BPTokens.sm),
              child: example,
            ),
        ],
      ),
    );
  }
}
