import "package:flutter/widgets.dart";

import "../design_system/design_system.dart";

/// Experimental UI adapter, not the final design system.
class PreviewApp extends StatelessWidget {
  const PreviewApp({super.key});
  @override
  Widget build(BuildContext context) => const BPApp(home: _InventoryPreview());
}

class _InventoryPreview extends StatefulWidget {
  const _InventoryPreview();
  @override
  State<_InventoryPreview> createState() => _InventoryPreviewState();
}

class _InventoryPreviewState extends State<_InventoryPreview> {
  static const products = [
    ("Arroz 1 kg", "ALI-001", "48", "Disponible"),
    ("Café 250 g", "ALI-002", "6", "Stock bajo"),
    ("Aceite 1 L", "ALI-003", "0", "Agotado"),
  ];
  final search = TextEditingController();
  String query = "";
  int selected = 0;
  @override
  void dispose() {
    search.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = BPVisuals.of(context);
    final visible = products
        .where(
          (p) =>
              p.$1.toLowerCase().contains(query) ||
              p.$2.toLowerCase().contains(query),
        )
        .toList();
    return ColoredBox(
      color: theme.background,
      child: SafeArea(
        child: LayoutBuilder(
          builder: (context, constraints) {
            final wide = constraints.maxWidth >= BPTokens.wideBreakpoint;
            return Column(
              children: [
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: BPTokens.lg,
                    vertical: BPTokens.md,
                  ),
                  decoration: BoxDecoration(
                    border: Border(bottom: BorderSide(color: theme.border)),
                  ),
                  child: Row(
                    children: [
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text("BalanzaPro3.0", style: theme.title),
                            Text(
                              "Entorno de desarrollo · Datos ficticios",
                              style: theme.muted,
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),
                if (!wide) _navigation(false),
                Expanded(
                  child: Row(
                    children: [
                      if (wide)
                        SizedBox(
                          width: BPTokens.sidebarWidth,
                          child: _navigation(true),
                        ),
                      Expanded(
                        child: ListView(
                          padding: EdgeInsets.all(
                            wide ? BPTokens.xxl : BPTokens.lg,
                          ),
                          children: [
                            Text(
                              selected == 0 ? "Productos" : "Movimientos",
                              style: theme.heading,
                            ),
                            const SizedBox(height: BPTokens.sm),
                            Text(
                              selected == 0
                                  ? "Consulta de existencias de ejemplo"
                                  : "Vista de demostración; sin operaciones reales",
                              style: theme.muted,
                            ),
                            const SizedBox(height: BPTokens.xl),
                            if (selected == 0) ...[
                              Text(
                                "Buscar por nombre o código",
                                style: theme.label,
                              ),
                              const SizedBox(height: BPTokens.sm),
                              BPInput(
                                label: "Buscar productos",
                                controller: search,
                                placeholder: "Ej. café o ALI-002",
                                onChanged: (value) => setState(
                                  () => query = value.trim().toLowerCase(),
                                ),
                              ),
                              const SizedBox(height: BPTokens.lg),
                              Text(
                                "${visible.length} productos",
                                style: theme.muted,
                              ),
                              const SizedBox(height: BPTokens.md),
                              if (visible.isEmpty)
                                BPCard(
                                  child: Column(
                                    children: [
                                      const Text("No encontramos productos"),
                                      const SizedBox(height: BPTokens.md),
                                      BPButton(
                                        primary: false,
                                        label: "Limpiar búsqueda",
                                        onPressed: () {
                                          search.clear();
                                          setState(() => query = "");
                                        },
                                      ),
                                    ],
                                  ),
                                )
                              else
                                ...visible.map(
                                  (p) => Padding(
                                    padding: const EdgeInsets.only(
                                      bottom: BPTokens.sm,
                                    ),
                                    child: BPCard(
                                      child: wide
                                          ? Row(
                                              children: [
                                                Expanded(
                                                  flex: 3,
                                                  child: Text(
                                                    p.$1,
                                                    style: theme.title,
                                                  ),
                                                ),
                                                Expanded(
                                                  flex: 2,
                                                  child: Text(p.$2),
                                                ),
                                                Expanded(
                                                  child: Text("${p.$3} und."),
                                                ),
                                                Expanded(
                                                  flex: 2,
                                                  child: Text(p.$4),
                                                ),
                                              ],
                                            )
                                          : Column(
                                              crossAxisAlignment:
                                                  CrossAxisAlignment.start,
                                              children: [
                                                Text(p.$1, style: theme.title),
                                                const SizedBox(
                                                  height: BPTokens.xs,
                                                ),
                                                Text(p.$2, style: theme.muted),
                                                const SizedBox(
                                                  height: BPTokens.sm,
                                                ),
                                                Text("${p.$3} und. · ${p.$4}"),
                                              ],
                                            ),
                                    ),
                                  ),
                                ),
                            ] else
                              const BPCard(
                                child: Text(
                                  "Los movimientos estarán disponibles al conectar la API protegida.",
                                ),
                              ),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            );
          },
        ),
      ),
    );
  }

  Widget _navigation(bool wide) {
    final items = [_navButton("Productos", 0), _navButton("Movimientos", 1)];
    return Padding(
      key: Key(wide ? "desktop-navigation" : "compact-navigation"),
      padding: const EdgeInsets.all(BPTokens.md),
      child: wide
          ? Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: items,
            )
          : Wrap(
              spacing: BPTokens.sm,
              runSpacing: BPTokens.sm,
              children: items,
            ),
    );
  }

  Widget _navButton(String label, int index) {
    void activate() => setState(() => selected = index);
    return Padding(
      padding: const EdgeInsets.only(bottom: BPTokens.xs),
      child: Semantics(
        selected: selected == index,
        child: BPButton(
          label: label,
          primary: selected == index,
          onPressed: activate,
        ),
      ),
    );
  }
}
